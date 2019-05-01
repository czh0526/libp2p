package test_mocknet

import (
	"io"
	"time"

	ic "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

type Mocknet interface {
	GenPeer() (host.Host, error)
	AddPeer(ic.PrivKey, ma.Multiaddr) (host.Host, error)
	AddPeerWithPeerstore(peer.ID, pstore.Peerstore) (host.Host, error)

	Peers() []peer.ID
	Net(peer.ID) inet.Network
	Nets() []inet.Network
	Host(peer.ID) host.Host
	Hosts() []host.Host
	Links() LinkMap
	LinksBetweenPeers(a, b peer.ID) []Link
	LinksBetweenNets(a, b inet.Network) []Link

	LinkPeers(peer.ID, peer.ID) (Link, error)
	LinkNets(inet.Network, inet.Network) (Link, error)
	Unlink(Link) error
	UnlinkPeers(peer.ID, peer.ID) error
	UnlinkNets(inet.Network, inet.Network) error

	SetLinkDefaults(LinkOptions)
	LinkDefaults() LinkOptions

	ConnectPeers(peer.ID, peer.ID) (inet.Conn, error)
	ConnectNets(inet.Network, inet.Network) (inet.Conn, error)
	DisconnectPeers(peer.ID, peer.ID) error
	DisconnectNets(inet.Network, inet.Network) error
	LinkAll() error
	ConnectAllButSelf() error
}

type LinkOptions struct {
	Latency   time.Duration
	Bandwidth float64
}

type Link interface {
	Networks() []inet.Network
	Peers() []peer.ID
	SetOptions(LinkOptions)
	Options() LinkOptions
}

type LinkMap map[string]map[string]map[Link]struct{}

type Printer interface {
	MocknetLinks(mn Mocknet)
	NetworkConns(ni inet.Network)
}

func PrinterTo(w io.Writer) Printer {
	return &printer{w}
}
