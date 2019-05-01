package test_dht

import (
	"context"
	"testing"

	dht "github.com/libp2p/go-libp2p-kad-dht"
)

// 不知道在测试什么？？？
func TestNotifieeMultipleConn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d1 := setupDHT(ctx, t, false)
	d2 := setupDHT(ctx, t, false)

	nn1 := dht.NewNetNotifiee(d1)
	nn2 := dht.NewNetNotifiee(d2)

	connect(t, ctx, d1, d2)
	c12 := d1.Host().Network().ConnsToPeer(d2.Self())[0]
	c21 := d2.Host().Network().ConnsToPeer(d1.Self())[0]

	nn1.Connected(d1.Host().Network(), c12)
	nn2.Connected(d1.Host().Network(), c21)

	if !checkRoutingTable(d1, d2) {
		t.Fatal("no routes")
	}

	nn1.Disconnected(d1.Host().Network(), c12)
	nn2.Disconnected(d2.Host().Network(), c21)
	if !checkRoutingTable(d1, d2) {
		t.Fatal("no routes")
	}

	for _, conn := range d1.Host().Network().ConnsToPeer(d2.Self()) {
		conn.Close()
	}
	for _, conn := range d2.Host().Network().ConnsToPeer(d1.Self()) {
		conn.Close()
	}

	if checkRoutingTable(d1, d2) {
		t.Fatal("should have no routes")
	}
}

func checkRoutingTable(a, b *dht.IpfsDHT) bool {
	return a.RoutingTable().Find(b.Self()) != "" && b.RoutingTable().Find(a.Self()) != ""
}
