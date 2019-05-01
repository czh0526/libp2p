package test_host

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p/p2p/host/relay"

	ggio "github.com/gogo/protobuf/io"
	autonat "github.com/libp2p/go-libp2p-autonat"
	autonatpb "github.com/libp2p/go-libp2p-autonat/pb"
	inet "github.com/libp2p/go-libp2p-net"

	cid "github.com/ipfs/go-cid"
	libp2p "github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	routing "github.com/libp2p/go-libp2p-routing"
	ma "github.com/multiformats/go-multiaddr"
)

func init() {
	logging.SetLogLevel("basichost", "DEBUG")
	logging.SetLogLevel("swarm2", "DEBUG")
	logging.SetLogLevel("autonat", "DEBUG")
	logging.SetLogLevel("autorelay", "DEBUG")
	autonat.AutoNATIdentifyDelay = 1 * time.Second
	autonat.AutoNATBootDelay = 2 * time.Second
	relay.BootDelay = 1 * time.Second
	relay.AdvertiseBootDelay = 100 * time.Millisecond
}

type mockRoutingTable struct {
	mx        sync.Mutex
	providers map[string]map[peer.ID]pstore.PeerInfo
}

func newMockRoutingTable() *mockRoutingTable {
	return &mockRoutingTable{
		providers: make(map[string]map[peer.ID]pstore.PeerInfo),
	}
}

type mockRouting struct {
	h   host.Host
	tab *mockRoutingTable
}

func newMockRouting(h host.Host, tab *mockRoutingTable) *mockRouting {
	return &mockRouting{h: h, tab: tab}
}

func (m *mockRouting) FindPeer(ctx context.Context, p peer.ID) (pstore.PeerInfo, error) {
	return pstore.PeerInfo{}, routing.ErrNotFound
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

func makeAutoNATServicePrivate(ctx context.Context, t *testing.T) host.Host {
	h, err := libp2p.New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	h.SetStreamHandler(autonat.AutoNATProto, sayAutoNATPrivate)
	return h
}

func TestAutoRelay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mtab := newMockRoutingTable()
	makeRouting := func(h host.Host) (routing.PeerRouting, error) {
		mr := newMockRouting(h, mtab)
		return mr, nil
	}

	h1 := makeAutoNATServicePrivate(ctx, t)
	for _, addr := range h1.Addrs() {
		fmt.Printf("h1 address = %s \n", addr)
	}
	h2, err := libp2p.New(ctx, libp2p.EnableRelay(circuit.OptHop), libp2p.EnableAutoRelay(), libp2p.Routing(makeRouting))
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range h2.Addrs() {
		fmt.Printf("h2 address = %s \n", addr)
	}
	h3, err := libp2p.New(ctx, libp2p.EnableRelay(), libp2p.EnableAutoRelay(), libp2p.Routing(makeRouting))
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range h3.Addrs() {
		fmt.Printf("h3 address = %s \n", addr)
	}
	h4, err := libp2p.New(ctx, libp2p.EnableRelay())
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range h4.Addrs() {
		fmt.Printf("h4 address = %s \n", addr)
	}

	connect(t, h1, h3)
	time.Sleep(5 * time.Second)

	unspecificRelay, err := ma.NewMultiaddr("/p2p-circuit")
	if err != nil {
		t.Fatal(err)
	}

	haveRelay := false
	for _, addr := range h3.Addrs() {
		fmt.Printf("addr = %s \n", addr.String())
		if addr.Equal(unspecificRelay) {
			t.Fatal("unspecific relay addr advertised")
		}

		_, err := addr.ValueForProtocol(circuit.P_CIRCUIT)
		if err == nil {
			haveRelay = true
		}
	}

	if !haveRelay {
		t.Fatal("No relay addrs advertised.")
	}
}

func sayAutoNATPrivate(s inet.Stream) {
	defer s.Close()
	w := ggio.NewDelimitedWriter(s)
	res := autonatpb.Message{
		Type:         autonatpb.Message_DIAL_RESPONSE.Enum(),
		DialResponse: newDialResponseError(autonatpb.Message_E_DIAL_ERROR, "no dialable addresses"),
	}
	w.WriteMsg(&res)
}

func newDialResponseError(status autonatpb.Message_ResponseStatus, text string) *autonatpb.Message_DialResponse {
	dr := new(autonatpb.Message_DialResponse)
	dr.Status = status.Enum()
	dr.StatusText = &text
	return dr
}

func connect(t *testing.T, a, b host.Host) {
	pinfo := pstore.PeerInfo{ID: a.ID(), Addrs: a.Addrs()}
	err := b.Connect(context.Background(), pinfo)
	if err != nil {
		t.Fatal(err)
	}
}
