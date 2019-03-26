package tcptransport

import (
	"testing"

	p2pt "github.com/czh0526/p2p/transport/libp2p-transport"

	insecure "github.com/libp2p/go-conn-security/insecure"
	tptu "github.com/libp2p/go-libp2p-transport-upgrader"
	tcpt "github.com/libp2p/go-tcp-transport"
	mplex "github.com/whyrusleeping/go-smux-multiplex"
)

func TestTcpTransport(t *testing.T) {
	for i := 0; i < 2; i++ {
		ta := tcpt.NewTCPTransport(&tptu.Upgrader{
			Secure: insecure.New("peerA"),
			Muxer:  new(mplex.Transport),
		})
		tb := tcpt.NewTCPTransport(&tptu.Upgrader{
			Secure: insecure.New("peerB"),
			Muxer:  new(mplex.Transport),
		})

		zero := "/ip4/127.0.0.1/tcp/0"
		p2pt.SubtestTransport(t, ta, tb, zero, "peerA")
	}
}
