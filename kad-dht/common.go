package test_dht

import (
	"context"
	"fmt"
	"testing"
	"time"

	swarmt "github.com/czh0526/libp2p/swarm"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	opts "github.com/libp2p/go-libp2p-kad-dht/opts"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
)

type blankValidator struct{}

func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

func setupDHTS(t *testing.T, ctx context.Context, n int) []*dht.IpfsDHT {
	dhts := make([]*dht.IpfsDHT, n)
	for i := 0; i < n; i++ {
		dhts[i] = setupDHT(ctx, t, false)
		fmt.Printf("%d). %s \n", i, dhts[i].Self().ShortString())
	}
	return dhts
}

func setupDHT(ctx context.Context, t *testing.T, client bool) *dht.IpfsDHT {
	d, err := dht.New(ctx,
		bhost.New(swarmt.GenSwarm(t, ctx, swarmt.OptDisableReuseport)),
		opts.Client(client),
		opts.NamespacedValidator("v", blankValidator{}),
	)
	if err != nil {
		t.Fatal(err)
	}

	return d
}

// 等待 a 的节点数据库中包含了 b 节点的信息
func wait(t *testing.T, ctx context.Context, a, b *dht.IpfsDHT) {
	for a.RoutingTable().Find(b.Self()) == "" {
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(time.Millisecond * 5):
		}
	}
	fmt.Printf("%s has connected to %s \n", a.Self().ShortString(), b.Self().ShortString())
}

func connect(t *testing.T, ctx context.Context, a, b *dht.IpfsDHT) {
	connectNoSync(t, ctx, a, b)
	wait(t, ctx, a, b)
	wait(t, ctx, b, a)
}

func connectNoSync(t *testing.T, ctx context.Context, a, b *dht.IpfsDHT) {
	// 获取 b 的 peer ID
	idB := b.Self()
	// 获取 b 的地址
	addrB := b.Peerstore().Addrs(idB)
	if len(addrB) == 0 {
		t.Fatal("peers setup incorrectly: no local address")
	}
	fmt.Printf("%s ==> %s \n", a.Host().Addrs(), b.Host().Network().ListenAddresses())

	// 将 b 的地址加入到 a 的节点库中
	a.Peerstore().AddAddrs(idB, addrB, pstore.PermanentAddrTTL)
	// a ==> b
	pi := pstore.PeerInfo{ID: idB}
	if err := a.Host().Connect(ctx, pi); err != nil {
		t.Fatal(err)
	}
}
