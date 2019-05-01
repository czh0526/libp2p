package test_discovery

import (
	"context"
	"sync"
	"testing"

	swarmt "github.com/czh0526/libp2p/swarm"
	cid "github.com/ipfs/go-cid"
	disc "github.com/libp2p/go-libp2p-discovery"
	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
)

type mockRoutingTable struct {
	mx        sync.Mutex
	providers map[string]map[peer.ID]pstore.PeerInfo
}

func NewMockRoutingTable() *mockRoutingTable {
	return &mockRoutingTable{
		providers: make(map[string]map[peer.ID]pstore.PeerInfo),
	}
}

type mockRouting struct {
	h   host.Host
	tab *mockRoutingTable
}

func NewMockRouting(h host.Host, tab *mockRoutingTable) *mockRouting {
	return &mockRouting{h: h, tab: tab}
}

func (m *mockRouting) Provide(ctx context.Context, cid cid.Cid, bcast bool) error {
	m.tab.mx.Lock()
	defer m.tab.mx.Unlock()

	pmap, ok := m.tab.providers[cid.String()]
	if !ok {
		pmap = make(map[peer.ID]pstore.PeerInfo)
		m.tab.providers[cid.String()] = pmap
	}
	pmap[m.h.ID()] = pstore.PeerInfo{ID: m.h.ID(), Addrs: m.h.Addrs()}

	return nil
}

func (m *mockRouting) FindProvidersAsync(ctx context.Context, cid cid.Cid, limit int) <-chan pstore.PeerInfo {
	ch := make(chan pstore.PeerInfo)
	go func() {
		defer close(ch)
		m.tab.mx.Lock()
		defer m.tab.mx.Unlock()

		pmap, ok := m.tab.providers[cid.String()]
		if !ok {
			return
		}
		for _, pi := range pmap {
			select {
			case ch <- pi:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}

func TestRoutingDiscovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h1, err := bhost.NewHost(ctx, swarmt.GenSwarm(t, ctx), &bhost.HostOpts{})
	if err != nil {
		t.Fatal(err)
	}
	h2, err := bhost.NewHost(ctx, swarmt.GenSwarm(t, ctx), &bhost.HostOpts{})
	if err != nil {
		t.Fatal(err)
	}

	// 构建 routingTable
	mtab := NewMockRoutingTable()

	// 构建 ContentRouting
	mr1 := NewMockRouting(h1, mtab)
	mr2 := NewMockRouting(h2, mtab)

	// 构建 RoutingDiscovery
	d1 := disc.NewRoutingDiscovery(mr1)
	d2 := disc.NewRoutingDiscovery(mr2)

	// 声明 Provider
	_, err = d1.Advertise(ctx, "/test")
	if err != nil {
		t.Fatal(err)
	}

	pis, err := disc.FindPeers(ctx, d2, "/test", 20)
	if err != nil {
		t.Fatal(err)
	}

	if len(pis) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(pis))
	}

	pi := pis[0]
	if pi.ID != h1.ID() {
		t.Fatalf("Unexpected peer: %s", pi.ID)
	}
}
