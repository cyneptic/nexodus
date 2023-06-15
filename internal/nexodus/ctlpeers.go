package nexodus

import (
	"encoding/json"
	"fmt"
)

func (ac *NexdCtl) ListPeers(_ string, result *string) error {
	peers, err := ac.nx.DumpPeersDefault()
	if err != nil {
		return fmt.Errorf("error getting list of peers: %w", err)
	}

	ac.nx.deviceCacheIterRead(func(d deviceCacheEntry) {
		if d.device.PublicKey == ac.nx.wireguardPubKey {
			return
		}
		p, ok := peers[d.device.PublicKey]
		if !ok {
			return
		}
		p.Healthy = d.peerHealthy
		peers[d.device.PublicKey] = p
	})

	peersJSON, err := json.Marshal(peers)
	if err != nil {
		return fmt.Errorf("error marshalling list of peers: %w", err)
	}

	*result = string(peersJSON)

	return nil
}
