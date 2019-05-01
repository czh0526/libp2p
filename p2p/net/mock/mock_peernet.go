package test_mocknet

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/jbenet/goprocess"
	goprocessctx "github.com/jbenet/goprocess/context"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

type peernet struct {
	mocknet       *mocknet
	peer          peer.ID
	ps            pstore.Peerstore
	connsByPeer   map[peer.ID]map[*conn]struct{}
	connsByLink   map[*link]map[*conn]struct{}
	streamHandler inet.StreamHandler
	connHandler   inet.ConnHandler
	notifmu       sync.Mutex
	notifs        map[inet.Notifiee]struct{}
	proc          goprocess.Process
	sync.RWMutex
}

func newPeernet(ctx context.Context, m *mocknet, p peer.ID, ps pstore.Peerstore) (*peernet, error) {
	n := &peernet{
		mocknet: m,
		peer:    p,
		ps:      ps,

		connsByPeer: map[peer.ID]map[*conn]struct{}{},
		connsByLink: map[*link]map[*conn]struct{}{},

		notifs: make(map[inet.Notifiee]struct{}),
	}

	n.proc = goprocessctx.WithContextAndTeardown(ctx, n.teardown)
	return n, nil
}

func (pn *peernet) teardown() error {
	for _, c := range pn.allConns() {
		c.Close()
	}
	return nil
}

func (pn *peernet) allConns() []*conn {
	pn.RLock()
	var cs []*conn
	for _, csl := range pn.connsByPeer {
		for c, _ := range csl {
			cs = append(cs, c)
		}
	}

	pn.RUnlock()
	return cs
}

func (pn *peernet) Close() error {
	return pn.proc.Close()
}

func (pn *peernet) Peerstore() pstore.Peerstore {
	return pn.ps
}

func (pn *peernet) String() string {
	return fmt.Sprintf("<mock.peernet %s - %d conns>", pn.peer, len(pn.allConns()))
}

func (pn *peernet) DialPeer(ctx context.Context, p peer.ID) (inet.Conn, error) {
	return pn.connect(p)
}

func (pn *peernet) connect(p peer.ID) (*conn, error) {
	if p == pn.peer {
		return nil, fmt.Errorf("attempled to dial self %s", p)
	}

	pn.RLock()
	cs, found := pn.connsByPeer[p]
	if found && len(cs) > 0 {
		var chosen *conn
		for c := range cs {
			chosen = c
			break
		}
		pn.RUnlock()
		return chosen, nil
	}
	pn.RUnlock()

	links := pn.mocknet.LinksBetweenPeers(pn.peer, p)
	if len(links) < 1 {
		return nil, fmt.Errorf("%s cannot connect to %s", pn.peer, p)
	}

	l := links[rand.Intn(len(links))]
	c := pn.openConn(p, l.(*link))
	return c, nil
}

func (pn *peernet) openConn(r peer.ID, l *link) *conn {
	lc, rc := l.newConnPair(pn)
	pn.addConn(lc)
	pn.notifyAll(func(n inet.Notifiee) {
		n.Connected(pn, lc)
	})
	rc.net.remoteOpenedConn(rc)
	return lc
}

func (pn *peernet) remoteOpenedConn(c *conn) {
	pn.addConn(c)
	pn.handleNewConn(c)
	pn.notifyAll(func(n inet.Notifiee) {
		n.Connected(pn, c)
	})
}

func (pn *peernet) addConn(c *conn) {
	pn.Lock()
	defer pn.Unlock()

	// add conn to peer map
	cs, found := pn.connsByPeer[c.RemotePeer()]
	if !found {
		cs = map[*conn]struct{}{}
		pn.connsByPeer[c.RemotePeer()] = cs
	}
	pn.connsByPeer[c.RemotePeer()][c] = struct{}{}

	// add conn to link peer
	cs, found = pn.connsByLink[c.link]
	if !found {
		cs = map[*conn]struct{}{}
		pn.connsByLink[c.link] = cs
	}
	pn.connsByLink[c.link][c] = struct{}{}
}

func (pn *peernet) removeConn(c *conn) {
	pn.Lock()
	defer pn.Unlock()

	cs, found := pn.connsByLink[c.link]
	if !found || len(cs) < 1 {
		panic(fmt.Sprintf("attempting to remove a conn that doesn't exist %v", c.link))
	}
	delete(cs, c)

	cs, found = pn.connsByPeer[c.remote]
	if !found {
		panic(fmt.Sprintf("attempting to remove a conn that doesn't exist %v", c.remote))
	}
	delete(cs, c)
}

func (pn *peernet) LocalPeer() peer.ID {
	return pn.peer
}

func (pn *peernet) Process() goprocess.Process {
	return pn.proc
}

func (pn *peernet) Peers() []peer.ID {
	pn.RLock()
	defer pn.RUnlock()

	peers := make([]peer.ID, 0, len(pn.connsByPeer))
	for _, cs := range pn.connsByPeer {
		for c := range cs {
			peers = append(peers, c.remote)
			break
		}
	}
	return peers
}

func (pn *peernet) Conns() []inet.Conn {
	pn.RLock()
	defer pn.RUnlock()

	var out []inet.Conn
	for _, cs := range pn.connsByPeer {
		for c := range cs {
			out = append(out, c)
		}
	}
	return out
}

func (pn *peernet) ConnsToPeer(p peer.ID) []inet.Conn {
	pn.RLock()
	defer pn.RUnlock()

	cs, found := pn.connsByPeer[p]
	if !found || len(cs) == 0 {
		return nil
	}

	var cs2 []inet.Conn
	for c := range cs {
		cs2 = append(cs2, c)
	}
	return cs2
}

func (pn *peernet) ClosePeer(p peer.ID) error {
	pn.RLock()
	cs, found := pn.connsByPeer[p]
	if !found {
		pn.RUnlock()
		return nil
	}

	var conns []*conn
	for c := range cs {
		conns = append(conns, c)
	}
	pn.RUnlock()
	for _, c := range conns {
		c.Close()
	}
	return nil
}

func (pn *peernet) Listen(addrs ...ma.Multiaddr) error {
	pn.Peerstore().AddAddrs(pn.LocalPeer(), addrs, pstore.PermanentAddrTTL)
	return nil
}

func (pn *peernet) ListenAddresses() []ma.Multiaddr {
	return pn.Peerstore().Addrs(pn.LocalPeer())
}

func (pn *peernet) InterfaceListenAddresses() ([]ma.Multiaddr, error) {
	return pn.ListenAddresses(), nil
}

func (pn *peernet) Connectedness(p peer.ID) inet.Connectedness {
	pn.Lock()
	defer pn.Unlock()

	cs, found := pn.connsByPeer[p]
	if found && len(cs) > 0 {
		return inet.Connected
	}
	return inet.NotConnected
}

func (pn *peernet) NewStream(ctx context.Context, p peer.ID) (inet.Stream, error) {
	c, err := pn.DialPeer(ctx, p)
	if err != nil {
		return nil, err
	}
	return c.NewStream()
}

func (pn *peernet) SetStreamHandler(h inet.StreamHandler) {
	pn.Lock()
	pn.streamHandler = h
	pn.Unlock()
}

func (pn *peernet) SetConnHandler(h inet.ConnHandler) {
	pn.Lock()
	pn.connHandler = h
	pn.Unlock()
}

func (pn *peernet) Notify(f inet.Notifiee) {
	pn.notifmu.Lock()
	pn.notifs[f] = struct{}{}
	pn.notifmu.Unlock()
}

func (pn *peernet) StopNotify(f inet.Notifiee) {
	pn.notifmu.Lock()
	delete(pn.notifs, f)
	pn.notifmu.Unlock()
}

func (pn *peernet) notifyAll(notification func(f inet.Notifiee)) {
	pn.notifmu.Lock()
	var wg sync.WaitGroup
	for n := range pn.notifs {
		wg.Add(1)
		go func(n inet.Notifiee) {
			defer wg.Done()
			notification(n)
		}(n)
	}
	wg.Wait()
	pn.notifmu.Unlock()
}

func (pn *peernet) handleNewConn(c inet.Conn) {
	pn.RLock()
	handler := pn.connHandler
	pn.RUnlock()
	if handler != nil {
		go handler(c)
	}
}

func (pn *peernet) handleNewStream(s inet.Stream) {
	pn.RLock()
	handler := pn.streamHandler
	pn.RUnlock()
	if handler != nil {
		go handler(s)
	}
}
