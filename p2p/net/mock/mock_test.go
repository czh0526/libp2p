package test_mocknet

import (
	"context"
	"fmt"
	"testing"

	"github.com/czh0526/libp2p/testutil"
)

func TestNetworkSetup(t *testing.T) {
	ctx := context.Background()
	sk1, _, err := testutil.RandTestKeyPair(512)
	if err != nil {
		t.Fatal(t)
	}
	sk2, _, err := testutil.RandTestKeyPair(512)
	if err != nil {
		t.Fatal(t)
	}
	sk3, _, err := testutil.RandTestKeyPair(512)
	if err != nil {
		t.Fatal(t)
	}

	mn := New(ctx)
	a1 := testutil.RandLocalTCPAddress()
	a2 := testutil.RandLocalTCPAddress()
	a3 := testutil.RandLocalTCPAddress()

	h1, err := mn.AddPeer(sk1, a1)
	if err != nil {
		t.Fatal(err)
	}
	p1 := h1.ID()

	h2, err := mn.AddPeer(sk2, a2)
	if err != nil {
		t.Fatal(err)
	}
	p2 := h2.ID()

	h3, err := mn.AddPeer(sk3, a3)
	if err != nil {
		t.Fatal(err)
	}
	p3 := h3.ID()

	n1 := h1.Network()
	n2 := h2.Network()
	n3 := h3.Network()

	// link p1 <--> p2, p1 <--> p1, p2 <--> p3, p3 <--> p2
	l12, err := mn.LinkPeers(p1, p2)
	l11, err := mn.LinkPeers(p1, p1)
	l23, err := mn.LinkPeers(p2, p3)
	l32, err := mn.LinkPeers(p3, p2)

	links12 := mn.LinksBetweenPeers(p1, p2)
	if len(links12) != 1 {
		t.Errorf("should be 1 link between p1 and p2 (foutd %d)", len(links12))
	}
	if links12[0] != l12 {
		t.Error("links 1-2 should be l12.")
	}

	links11 := mn.LinksBetweenPeers(p1, p1)
	if len(links11) != 1 {
		t.Errorf("sould be 1 link between p1 and p1 (found %d)", len(links11))
	}
	if links11[0] != l11 {
		t.Error("links 1-1 should be l11.")
	}

	links23 := mn.LinksBetweenPeers(p2, p3)
	if len(links23) != 2 {
		t.Errorf("sould be 2 link between p2 and p3 (found %d)", len(links23))
	}
	if !(links23[0] == l23 && links23[1] == l32) &&
		!(links23[0] == l32 && links23[1] == l23) {
		t.Error("links 2-3 should be l23 and l32.")
	}

	fmt.Printf("n1 has %d conns. \n", len(n1.Conns()))
	fmt.Printf("n2 has %d conns. \n", len(n2.Conns()))
	fmt.Printf("n3 has %d conns. \n", len(n3.Conns()))

	if _, err := n2.DialPeer(ctx, p3); err != nil {
		t.Error(err)
	}
	fmt.Println("n2 dial to n3")
	fmt.Printf("n2 has %d conns. \n", len(n2.Conns()))
	fmt.Printf("n3 has %d conns. \n", len(n3.Conns()))

	if _, err := n2.NewStream(ctx, p3); err != nil {
		t.Error(err)
	}

}
