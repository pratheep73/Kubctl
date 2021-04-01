package network

import (
	"fmt"
	"io/ioutil"
	"os"
	"text/template"

	"github.com/go-logr/logr"
	firewallv1 "github.com/metal-stack/firewall-controller/api/v1"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/metal-stack/metal-networker/pkg/netconf"
	"github.com/rakyll/statik/fs"

	_ "github.com/metal-stack/firewall-controller/pkg/network/statik"
)

const (
	MetalKnowledgeBase = "/etc/metal/install.yaml"
	FrrConfig          = "/etc/frr/frr.conf"
	TmpPath            = "/var/tmp"
)

// ReconcileNetwork reconciles the network settings for a firewall
// in the current stage it only changes the FRR-Configuration when network prefixes or FRR template changes
func ReconcileNetwork(f firewallv1.Firewall, log logr.Logger) error {
	kb := netconf.NewKnowledgeBase(MetalKnowledgeBase)

	networkMap := map[string]firewallv1.FirewallNetwork{}
	for _, n := range f.Spec.FirewallNetworks {
		if n.Networktype == nil {
			continue
		}
		networkMap[*n.Networkid] = n
	}

	newNetworks := []models.V1MachineNetwork{}
	for _, n := range kb.Networks {
		newNet := n
		newNet.Prefixes = networkMap[*n.Networkid].Prefixes
		newNetworks = append(newNetworks, newNet)
	}
	kb.Networks = newNetworks

	tmpFile, err := tmpFile("frr.conf")
	if err != nil {
		return fmt.Errorf("error during network reconcilation %v: %w", tmpFile, err)
	}
	defer func() {
		os.Remove(tmpFile)
	}()

	a := netconf.NewFrrConfigApplier(netconf.Firewall, kb, tmpFile)
	tpl, err := readTpl(netconf.TplFirewallFRR)
	if err != nil {
		return fmt.Errorf("error during network reconcilation: %v: %w", tmpFile, err)
	}

	err = a.Apply(*tpl, tmpFile, FrrConfig, true)
	if err != nil {
		return fmt.Errorf("error during network reconcilation: %v: %w", tmpFile, err)
	}

	return nil
}

func tmpFile(prefix string) (string, error) {
	f, err := ioutil.TempFile(TmpPath, prefix)
	if err != nil {
		return "", err
	}

	err = f.Close()
	if err != nil {
		return "", err
	}

	return f.Name(), nil
}

func readTpl(tplName string) (*template.Template, error) {
	statikFS, err := fs.NewWithNamespace("networker")
	if err != nil {
		return nil, fmt.Errorf("could not open statik namespace tpl: %w", err)
	}

	r, err := statikFS.Open("/" + tplName)
	if err != nil {
		return nil, fmt.Errorf("could not open template %v from statik: %w", tplName, err)
	}
	defer r.Close()

	s, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("could not read template %v from statik: %w", tplName, err)
	}

	t, err := template.New(tplName).Parse(string(s))
	if err != nil {
		return nil, fmt.Errorf("could not parse template %v from statik: %w", tplName, err)
	}

	return t, nil
}
