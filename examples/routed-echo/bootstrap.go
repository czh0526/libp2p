package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	host "github.com/libp2p/go-libp2p-host"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

var (
	IPFS_PEERS = convertPeers([]string{
		"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		"/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
		"/ip4/128.199.219.111/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
		"/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
		"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		"/ip6/2604:a880:1:20::203:d001/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
		"/ip6/2400:6180:0:d0::151:6001/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
		"/ip6/2604:a880:800:10::4a:5001/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
		"/ip6/2a03:b0c0:0:1010::23:1001/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
	})
	LOCAL_PEER_ENDPOINT = "http://localhost:13002/api/v0/id"
)

func convertPeers(peers []string) []pstore.PeerInfo {
	pinfos := make([]pstore.PeerInfo, len(peers))
	for i, peer := range peers {
		maddr := ma.StringCast(peer)
		p, err := pstore.InfoFromP2pAddr(maddr)
		if err != nil {
			fmt.Println(err)
		}
		pinfos[i] = *p
	}
	return pinfos
}

type IdOutput struct {
	ID              string
	PublicKey       string
	Addresses       []string
	AgentVersion    string
	ProtocolVersion string
}

func getLocalPeerInfo() []pstore.PeerInfo {
	resp, err := http.Get(LOCAL_PEER_ENDPOINT)
	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var js IdOutput
	err = json.Unmarshal(body, &js)
	if err != nil {
		panic(err)
	}

	for _, addr := range js.Addresses {
		if addr[0:8] == "/ip4/127" {
			return convertPeers([]string{addr})
		}
	}
	return make([]pstore.PeerInfo, 1)
}

// ph ==> peers
func bootstrapConnect(ctx context.Context, ph host.Host, peers []pstore.PeerInfo) error {
	if len(peers) < 1 {
		return errors.New("not enough bootstrap peers")
	}

	// 逐个连接 peer
	errs := make(chan error, len(peers))
	var wg sync.WaitGroup
	for _, p := range peers {
		wg.Add(1)
		go func(p pstore.PeerInfo) {
			defer wg.Done()
			defer fmt.Printf("bootstrapDial: %s, %s \n", ph.ID(), p.ID)

			// ph ==> p
			ph.Peerstore().AddAddrs(p.ID, p.Addrs, pstore.PermanentAddrTTL)
			if err := ph.Connect(ctx, p); err != nil {
				fmt.Printf("bootstrapDialFailed: %s \n", p.ID)
				fmt.Printf("failed to bootstrap with %v: %s", p.ID, err)
				errs <- err
				return
			}
			fmt.Printf("bootstrapDialSuccess: %s \n", p.ID)
			fmt.Printf("bootstrapped with %v", p.ID)
		}(p)
	}
	wg.Wait()

	close(errs)
	count := 0
	var err error
	for err = range errs {
		if err != nil {
			count++
		}
	}
	if count == len(peers) {
		return fmt.Errorf("failed to bootstrap. %s \n", err)
	}
	return nil
}
