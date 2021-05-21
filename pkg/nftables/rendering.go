package nftables

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"text/template"

	firewallv1 "github.com/metal-stack/firewall-controller/api/v1"
)

// firewallRenderingData holds the data available in the nftables template
type firewallRenderingData struct {
	ForwardingRules  forwardingRules
	RateLimitRules   nftablesRules
	SnatRules        nftablesRules
	Sets             []firewallv1.IPSet
	InternalPrefixes string
	PrivateVrfID     uint
}

func newFirewallRenderingData(f *Firewall) (*firewallRenderingData, error) {
	ingress, egress := nftablesRules{}, nftablesRules{}
	for ind, np := range f.clusterwideNetworkPolicies.Items {
		err := np.Spec.Validate()
		if err != nil {
			continue
		}

		i, e, u := clusterwideNetworkPolicyRules(f.cache, np)
		ingress = append(ingress, i...)
		egress = append(egress, e...)
		f.clusterwideNetworkPolicies.Items[ind] = u
	}

	for _, svc := range f.services.Items {
		ingress = append(ingress, serviceRules(svc)...)
	}

	snatRules, err := snatRules(f)
	if err != nil {
		return &firewallRenderingData{}, err
	}

	return &firewallRenderingData{
		PrivateVrfID:     uint(*f.primaryPrivateNet.Vrf),
		InternalPrefixes: strings.Join(f.spec.InternalPrefixes, ", "),
		ForwardingRules: forwardingRules{
			Ingress: ingress,
			Egress:  egress,
		},
		RateLimitRules: rateLimitRules(f),
		SnatRules:      snatRules,
		Sets:           f.cache.GetSetsForRendering(),
	}, nil
}

func (d *firewallRenderingData) write(file string) error {
	c, err := d.renderString()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(file, []byte(c), 0644)
	if err != nil {
		return fmt.Errorf("error writing to nftables file '%s': %w", file, err)
	}
	return nil
}

func (d *firewallRenderingData) renderString() (string, error) {
	var b bytes.Buffer

	tplString, err := d.readTpl()
	if err != nil {
		return "", err
	}

	tpl := template.Must(
		template.New("v4").
			Funcs(template.FuncMap{"StringsJoin": strings.Join}).
			Parse(tplString),
	)

	err = tpl.Execute(&b, d)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

func (d *firewallRenderingData) readTpl() (string, error) {
	r, err := templates.Open("nftables.tpl")
	if err != nil {
		return "", err
	}
	defer r.Close()
	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
