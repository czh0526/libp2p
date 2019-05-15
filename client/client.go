package main

import (
	"context"

	discovery "github.com/libp2p/go-libp2p-discovery"
	host "github.com/libp2p/go-libp2p-host"
	kad_dht "github.com/libp2p/go-libp2p-kad-dht"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
)

type Client struct {
	groups           []string
	friends          map[peer.ID]inet.Stream
	host             host.Host
	dht              *kad_dht.IpfsDHT
	routingDiscovery *discovery.RoutingDiscovery
	groupPeerChan    <-chan pstore.PeerInfo
}

func NewClient(ctx context.Context,
	groups []string,
	host host.Host,
	dht *kad_dht.IpfsDHT) *Client {

	// 构建 Discovery
	routingDiscovery := discovery.NewRoutingDiscovery(dht)
	for _, group := range groups {
		// 并行宣布本节点的存在
		discovery.Advertise(ctx, routingDiscovery, group)
	}

	client := &Client{
		groups:           groups,
		friends:          make(map[peer.ID]inet.Stream),
		host:             host,
		dht:              dht,
		routingDiscovery: routingDiscovery,
	}

	return client
}
