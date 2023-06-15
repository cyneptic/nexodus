package nexodus

import (
	"fmt"

	"github.com/bytedance/gopkg/util/logger"

	"go.uber.org/zap"
)

type NexdCtl struct {
	nx *Nexodus
}

func (ac *NexdCtl) Status(_ string, result *string) error {
	var statusStr string
	switch ac.nx.status {
	case NexdStatusStarting:
		statusStr = "Starting"
	case NexdStatusAuth:
		statusStr = "WaitingForAuth"
	case NexdStatusRunning:
		statusStr = "Running"
	default:
		statusStr = "Unknown"
	}
	res := fmt.Sprintf("Status: %s\n", statusStr)
	if len(ac.nx.statusMsg) > 0 {
		res += ac.nx.statusMsg
	}
	*result = res
	return nil
}

func (ac *NexdCtl) Version(_ string, result *string) error {
	*result = ac.nx.version
	return nil
}

func (ac *NexdCtl) GetTunnelIPv4(_ string, result *string) error {
	*result = ac.nx.TunnelIP
	return nil
}

func (ac *NexdCtl) GetTunnelIPv6(_ string, result *string) error {
	*result = ac.nx.TunnelIpV6
	return nil
}

func (ac *NexdCtl) ProxyList(_ string, result *string) error {
	*result = ""
	ac.nx.proxyLock.RLock()
	defer ac.nx.proxyLock.RUnlock()
	for _, proxy := range ac.nx.proxies {
		proxy.mu.RLock()
		for _, rule := range proxy.rules {
			*result += fmt.Sprintf("%s\n", rule.AsFlag())
		}
		proxy.mu.RUnlock()
	}
	return nil
}

func (ac *NexdCtl) proxyAdd(proxyType ProxyType, rule string, result *string) error {

	proxyRule, err := ParseProxyRule(rule, proxyType)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Failed to parse %s proxy rule (%s): %v", proxyType, rule, err))
	}
	proxyRule.stored = true

	proxy, err := ac.nx.UserspaceProxyAdd(proxyRule)
	if err != nil {
		return err
	}
	proxy.Start(ac.nx.nexCtx, ac.nx.nexWg, ac.nx.userspaceNet)

	err = ac.nx.StoreProxyRules()
	if err != nil {
		return err
	}
	*result = fmt.Sprintf("Added %s proxy rule: %s\n", proxyType, rule)
	return nil
}

func (ac *NexdCtl) ProxyAddIngress(rule string, result *string) error {
	return ac.proxyAdd(ProxyTypeIngress, rule, result)
}

func (ac *NexdCtl) ProxyAddEgress(rule string, result *string) error {
	return ac.proxyAdd(ProxyTypeEgress, rule, result)
}

func (ac *NexdCtl) proxyRemove(proxyType ProxyType, rule string, result *string) error {
	proxyRule, err := ParseProxyRule(rule, proxyType)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Failed to parse %s proxy rule (%s): %v", proxyType, rule, err))
	}
	proxyRule.stored = true

	_, err = ac.nx.UserspaceProxyRemove(proxyRule)
	if err != nil {
		return err
	}
	err = ac.nx.StoreProxyRules()
	if err != nil {
		return err
	}

	*result = fmt.Sprintf("Removed ingress proxy rule: %s\n", rule)
	return nil
}
func (ac *NexdCtl) ProxyRemoveIngress(rule string, result *string) error {
	return ac.proxyRemove(ProxyTypeIngress, rule, result)
}

func (ac *NexdCtl) ProxyRemoveEgress(rule string, result *string) error {
	return ac.proxyRemove(ProxyTypeEgress, rule, result)
}

func (ac *NexdCtl) SetDebugOn(_ string, result *string) error {
	ac.nx.logLevel.SetLevel(zap.DebugLevel)
	*result = "Debug logging enabled"
	return nil
}

func (ac *NexdCtl) SetDebugOff(_ string, result *string) error {
	ac.nx.logLevel.SetLevel(zap.InfoLevel)
	*result = "Debug logging disabled"
	return nil
}

func (ac *NexdCtl) GetDebug(_ string, result *string) error {
	if ac.nx.logLevel.Level() == zap.DebugLevel {
		*result = "on"
	} else {
		*result = "off"
	}
	return nil
}
