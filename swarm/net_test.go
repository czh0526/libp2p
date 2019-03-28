package test_swarm

import (
	"context"
	"fmt"
	"testing"
	"time"

	inet "github.com/libp2p/go-libp2p-net"
)

func TestConnectednessCorrect(t *testing.T) {
	ctx := context.Background()

	nets := make([]inet.Network, 4)
	for i := 0; i < 4; i++ {
		nets[i] = GenSwarm(t, ctx)
	}

	dial := func(a, b inet.Network) {
		DivulgeAddresses(b, a)
		if _, err := a.DialPeer(ctx, b.LocalPeer()); err != nil {
			t.Fatalf("Failed to dial: %s", err)
		}
	}

	dial(nets[0], nets[1])
	dial(nets[0], nets[3])
	dial(nets[1], nets[2])
	dial(nets[3], nets[2])

	time.Sleep(time.Millisecond * 100)

	expectConnectedness(t, nets[0], nets[1], inet.Connected)
	expectConnectedness(t, nets[0], nets[3], inet.Connected)
	expectConnectedness(t, nets[1], nets[2], inet.Connected)
	expectConnectedness(t, nets[3], nets[2], inet.Connected)
	expectConnectedness(t, nets[0], nets[2], inet.NotConnected)
	expectConnectedness(t, nets[1], nets[3], inet.NotConnected)

	// 检查与 0 相连的节点
	if len(nets[0].Peers()) != 2 {
		t.Fatal("expected net 0 to have two peers")
	}

	// 检查 2 的连接数
	if len(nets[2].Conns()) != 2 {
		t.Fatal("expected net 2 to have tow conns")
	}

	// 检查 1->3 的连接
	if len(nets[1].ConnsToPeer(nets[3].LocalPeer())) != 0 {
		t.Fatal("net 1 should have no connections to net 3")
	}

	// 关闭 2->1 的连接
	if err := nets[2].ClosePeer(nets[1].LocalPeer()); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 50)

	expectConnectedness(t, nets[2], nets[1], inet.NotConnected)

	for _, n := range nets {
		n.Close()
	}

	for _, n := range nets {
		<-n.Process().Closed()
	}
}

func expectConnectedness(t *testing.T, a, b inet.Network, expected inet.Connectedness) {
	es := "%s is connected to %s, but Connectedness incorrect. %s %s %s"
	atob := a.Connectedness(b.LocalPeer())
	btoa := b.Connectedness(a.LocalPeer())

	if atob != expected {
		t.Errorf(es, a, b, printConns(a), printConns(b), atob)
	}

	if btoa != expected {
		t.Errorf(es, b, a, printConns(b), printConns(a), btoa)
	}
}

func printConns(n inet.Network) string {
	s := fmt.Sprintf("Connections in %s: \n", n)
	for _, c := range n.Conns() {
		s = s + fmt.Sprintf("-%s\n", c)
	}
	return s
}
