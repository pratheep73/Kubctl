package network

import (
	"fmt"
	"io/ioutil"
	"os"
	"text/template"

	"github.com/go-logr/logr"
	firewallv1 "github.com/metal-stack/firewall-controller/api/v1"
	"github.com/metal-stack/firewall-controller/pkg/nftables"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/metal-stack/metal-networker/pkg/netconf"

	"embed"
)

const (
	MetalKnowledgeBase = "/etc/metal/install.yaml"
	FrrConfig          = "/etc/frr/frr.conf"
	TmpPath            = "/var/tmp"
)

//go:embed *.tpl
var templates embed.FS

// ReconcileNetwork reconciles the network settings for a firewall
// Changes both the FRR-Configuration and Nftable rules when network prefixes or FRR template changes
func ReconcileNetwork(f firewallv1.Firewall, enableDNSProxy bool, log logr.Logger) (bool, error) {
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

	// Reconcile nftables
	firewall := nftables.NewDefaultFirewall()
	if err := firewall.ReconcileNetconfTables(kb, enableDNSProxy); err != nil {
		return false, fmt.Errorf("failed to reconcile network: %w", err)
	}

	return reconcileFRR(kb)
}

func reconcileFRR(kb netconf.KnowledgeBase) (changed bool, err error) {
	tmpFile, err := tmpFile("frr.conf")
	if err != nil {
		return false, fmt.Errorf("error during network reconcilation %v: %w", tmpFile, err)
	}
	defer func() {
		os.Remove(tmpFile)
	}()

	a := netconf.NewFrrConfigApplier(netconf.Firewall, kb, tmpFile)
	tpl, err := readTpl(netconf.TplFirewallFRR)
	if err != nil {
		return false, fmt.Errorf("error during network reconcilation: %v: %w", tmpFile, err)
	}

	changed, err = a.Apply(*tpl, tmpFile, FrrConfig, true)
	if err != nil {
		return changed, fmt.Errorf("error during network reconcilation: %v: %w", tmpFile, err)
	}

	return
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
	contents, err := templates.ReadFile(tplName)
	if err != nil {
		return nil, err
	}

	t, err := template.New(tplName).Parse(string(contents))
	if err != nil {
		return nil, fmt.Errorf("could not parse template %v from embed.FS: %w", tplName, err)
	}

	return t, nil
}
