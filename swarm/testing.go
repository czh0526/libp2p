package test_swarm

import (
	"context"
	"testing"

	secio "github.com/libp2p/go-libp2p-secio"

	csms "github.com/libp2p/go-conn-security-multistream"

	tcp "github.com/libp2p/go-tcp-transport"

	tu "github.com/czh0526/p2p/testutil"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	pstoremem "github.com/libp2p/go-libp2p-peerstore/pstoremem"
	swarm "github.com/libp2p/go-libp2p-swarm"
	tptu "github.com/libp2p/go-libp2p-transport-upgrader"
	msmux "github.com/whyrusleeping/go-smux-multistream"
	yamux "github.com/whyrusleeping/go-smux-yamux"
)

type config struct {
	disableReuseport bool
	dialOnly         bool
}

type Option func(*testing.T, *config)

var OptDisableReuseport Option = func(_ *testing.T, c *config) {
	c.disableReuseport = true
}

var OptDialOnly Option = func(_ *testing.T, c *config) {
	c.dialOnly = true
}

func GenUpgrader(n *swarm.Swarm) *tptu.Upgrader {
	id := n.LocalPeer()
	pk := n.Peerstore().PrivKey(id)
	secMuxer := new(csms.SSMuxer)
	secMuxer.AddTransport(secio.ID, &secio.Transport{
		LocalID:    id,
		PrivateKey: pk,
	})

	stMuxer := msmux.NewBlankTransport()
	stMuxer.AddTransport("/yamux/1.0.0", yamux.DefaultTransport)

	return &tptu.Upgrader{
		Secure:  secMuxer,
		Muxer:   stMuxer,
		Filters: n.Filters,
	}
}

func GenSwarm(t *testing.T, ctx context.Context, opts ...Option) *swarm.Swarm {
	var cfg config
	for _, o := range opts {
		o(t, &cfg)
	}

	// 构造节点的身份 priv/pub key, ID, multiaddr
	p := tu.RandPeerNetParamsOrFatal(t)
	// 构建 Peerstore
	ps := pstoremem.NewPeerstore()
	ps.AddPubKey(p.ID, p.PubKey)
	ps.AddPrivKey(p.ID, p.PrivKey)
	// 构建 swarm 网络
	s := swarm.NewSwarm(ctx, p.ID, ps, nil)

	tcpTransport := tcp.NewTCPTransport(GenUpgrader(s))
	tcpTransport.DisableReuseport = cfg.disableReuseport

	if err := s.AddTransport(tcpTransport); err != nil {
		t.Fatal(err)
	}

	if !cfg.dialOnly {
		if err := s.Listen(); err != nil {
			t.Fatal(err)
		}
		s.Peerstore().AddAddrs(p.ID, s.ListenAddresses(), pstore.PermanentAddrTTL)
	}

	return s
}
