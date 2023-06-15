package nexodus

import (
	"encoding/json"
	"fmt"
	"net"

	"go.uber.org/zap"
)

const (
	batchSize = 10
	v4        = "v4"
	v6        = "v6"
)

// ConnectivityV4 pings all peers via IPv4
func (ac *NexdCtl) ConnectivityV4(_ string, keepaliveResults *string) error {
	res := ac.nx.connectivityProbe(v4)
	var err error

	// Marshal the map into a JSON string.
	keepaliveJson, err := json.Marshal(res)
	if err != nil {
		return fmt.Errorf("error marshalling connectivty results")
	}

	*keepaliveResults = string(keepaliveJson)

	return nil
}

// ConnectivityV6 pings all peers via IPv6
func (ac *NexdCtl) ConnectivityV6(_ string, keepaliveResults *string) error {
	res := ac.nx.connectivityProbe(v6)
	var err error

	// Marshal the map into a JSON string.
	keepaliveJson, err := json.Marshal(res)
	if err != nil {
		return fmt.Errorf("error marshalling connectivty results")
	}

	*keepaliveResults = string(keepaliveJson)

	return nil
}

func (nx *Nexodus) connectivityProbe(family string) map[string]KeepaliveStatus {
	peerStatusMap := make(map[string]KeepaliveStatus)

	if !nx.relay {
		for _, value := range nx.deviceCache {
			// skip the node sourcing the probe
			if nx.wireguardPubKey == value.device.PublicKey {
				continue
			}
			var nodeAddr string
			pubKey := value.device.PublicKey
			if family == v6 {
				nodeAddr = value.device.TunnelIpV6
				if net.ParseIP(value.device.TunnelIpV6) == nil {
					nx.logger.Debugf("failed parsing an ipv6 address from %s", value.device.TunnelIp)
					continue
				}
			} else {
				nodeAddr = value.device.TunnelIp
				if net.ParseIP(value.device.TunnelIp) == nil {
					nx.logger.Debugf("failed parsing an ipv4 address from %s", value.device.TunnelIp)
					continue
				}
			}

			hostname := value.device.Hostname
			peerStatusMap[pubKey] = KeepaliveStatus{
				WgIP:        nodeAddr,
				IsReachable: false,
				Hostname:    hostname,
			}
		}
	}
	connResults := nx.probeConnectivity(peerStatusMap, nx.logger)

	return connResults
}

// probeConnectivity check connectivity in batches to limit excessive traffic in the case of a large number of peers
func (nx *Nexodus) probeConnectivity(peers map[string]KeepaliveStatus, logger *zap.SugaredLogger) map[string]KeepaliveStatus {
	peerConnResultsMap := make(map[string]KeepaliveStatus)

	peerKeys := make([]string, 0, len(peers))
	for key := range peers {
		peerKeys = append(peerKeys, key)
	}

	for i := 0; i < len(peerKeys); i += batchSize {
		end := i + batchSize
		if end > len(peerKeys) {
			end = len(peerKeys)
		}

		batch := peerKeys[i:end]

		c := make(chan struct {
			KeepaliveStatus
			IsReachable bool
		})

		for _, pubKey := range batch {
			go nx.runProbe(peers[pubKey], c)
		}

		for range batch {
			result := <-c
			ip := result.WgIP

			if result.IsReachable {
				logger.Debugf("connectivty probe [ %s ] is reachable", ip)
			} else {
				logger.Debugf("connectivty probe [ %s ] is not reachable", ip)
			}

			peerConnResultsMap[ip] = KeepaliveStatus{
				WgIP:        result.WgIP,
				IsReachable: result.IsReachable,
				Hostname:    result.Hostname,
			}
		}
	}

	return peerConnResultsMap
}
