package test_swarm

import (
	"context"
	"fmt"
	"testing"

	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	swarm "github.com/libp2p/go-libp2p-swarm"
	ma "github.com/multiformats/go-multiaddr"
)

func TestPeers(t *testing.T) {
	ctx := context.Background()
	swarms := makeSwarms(ctx, t, 2)
	s1 := swarms[0]
	s2 := swarms[1]

	connect := func(s *swarm.Swarm, dst peer.ID, addr ma.Multiaddr) {
		s.Peerstore().AddAddr(dst, addr, pstore.PermanentAddrTTL)
		if _, err := s.DialPeer(ctx, dst); err != nil {
			t.Fatal("error swarm dialing to peer", err)
		}
	}

	s1GotConn := make(chan struct{}, 0)
	s2GotConn := make(chan struct{}, 0)
	s1.SetConnHandler(func(c inet.Conn) {
		s1GotConn <- struct{}{}
		fmt.Println("s1 finished handle conn.")
	})
	s2.SetConnHandler(func(c inet.Conn) {
		s2GotConn <- struct{}{}
		fmt.Println("s2 finished handle conn.")
	})

	connect(s1, s2.LocalPeer(), s2.ListenAddresses()[0])
	<-s2GotConn
	connect(s2, s1.LocalPeer(), s1.ListenAddresses()[0])
	<-s1GotConn

	for i := 0; i < 100; i++ {
		connect(s1, s2.LocalPeer(), s2.ListenAddresses()[0])
		connect(s2, s1.LocalPeer(), s1.ListenAddresses()[0])
	}

	for _, s := range swarms {
		fmt.Printf("%s swarm routing table: %s \n", s.LocalPeer(), s.Peers())
	}
}
