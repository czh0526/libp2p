package test_dht

import (
	"context"
	"fmt"
	"gx/ipfs/QmPVkJMTeRC6iBByPWdrRkD3BE5UXsj5HPzb4kPqL186mS/testify/assert"
	"math/rand"
	"sort"
	"testing"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"

	kb "github.com/libp2p/go-libp2p-kbucket"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
)

func TestFindPeersConnectedToPeer(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dhts := setupDHTS(t, ctx, 4)
	defer func() {
		for i := 0; i < 4; i++ {
			dhts[i].Close()
			dhts[i].Host().Close()
		}
	}()

	// 0 -> 1, 2, 3
	connect(t, ctx, dhts[0], dhts[1])
	connect(t, ctx, dhts[1], dhts[2])
	connect(t, ctx, dhts[1], dhts[3])
	connect(t, ctx, dhts[2], dhts[3])

	ctxT, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	// 通过 0 查找与 2 连接的 peer
	pchan, err := dhts[0].FindPeersConnectedToPeer(ctxT, dhts[2].Self())
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%s ==> \n", dhts[2].Self().ShortString())
	var found []*pstore.PeerInfo
	for nextp := range pchan {
		found = append(found, nextp)
		fmt.Printf("\t ==> %s \n", nextp.ID.ShortString())
	}

	fmt.Println("好像结果不对，应该返回1，3")
}

func TestClientModeFindPeer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	a := setupDHT(ctx, t, false)
	b := setupDHT(ctx, t, true)
	c := setupDHT(ctx, t, true)
	fmt.Printf("a). %s \n", a.Host().Addrs())
	fmt.Printf("b). %s \n", b.Host().Addrs())
	fmt.Printf("c). %s \n", c.Host().Addrs())

	connectNoSync(t, ctx, b, a)
	connectNoSync(t, ctx, c, a)

	wait(t, ctx, b, a)
	wait(t, ctx, c, a)

	pi, err := c.FindPeer(ctx, b.Self())
	if err != nil {
		t.Fatal(err)
	}

	if len(pi.Addrs) == 0 {
		t.Fatal("should have found addresses for node b")
	}

	fmt.Printf("c find b's Addrs = %s, \n", pi.Addrs)
	fmt.Println("one port is listening port, the other port is the client port connecting to node a.")

	err = c.Host().Connect(ctx, pi)
	if err != nil {
		t.Fatal(err)
	}

	<-time.After(time.Minute * 5)
}

func TestFindPeerQuery(t *testing.T) {
	testFindPeerQuery(t, 20, 80, 16)
}

func minInt(a, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

func testFindPeerQuery(t *testing.T,
	bootstrappers, // Number of nodes connected to the querying node
	leafs, // Number of nodes that might be connected to from the bootstrappers
	bootstrapperLeafConns int, // Number of connections each bootstrapper has to the leaf nodes
) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dhts := setupDHTS(t, ctx, 1+bootstrappers+leafs)
	defer func() {
		for _, d := range dhts {
			d.Close()
			d.Host().Close()
		}
	}()

	mrand := rand.New(rand.NewSource(42))
	guy := dhts[0]
	others := dhts[1:]
	// 构建 bootstrappers
	for i := 0; i < bootstrappers; i++ {
		for j := 0; j < bootstrapperLeafConns; j++ {
			v := mrand.Intn(leafs)
			connect(t, ctx, others[i], others[bootstrappers+v])
		}
	}
	// 构建 guy
	for i := 0; i < bootstrappers; i++ {
		connect(t, ctx, guy, others[i])
	}

	var reachableIds []peer.ID
	for i, d := range dhts {
		lp := len(d.Host().Network().Peers())
		if i != 0 && lp > 0 {
			reachableIds = append(reachableIds, d.Self())
		}
	}
	fmt.Println("guy routing table ==> ")
	guy.RoutingTable().Print()

	// 测试 RoutingTable.NearestPeers(...)
	val := "foobar"
	rtval := kb.ConvertKey(val)
	rtablePeers := guy.RoutingTable().NearestPeers(rtval, dht.AlphaValue)
	// 在 guy 上的节点查找，如果 dht.AlphaValue 足够大，返回节点的数量不会少于直连的节点数
	assert.Len(t, rtablePeers, minInt(bootstrappers, dht.AlphaValue))
	// 验证 Network().Peers() == 直连的节点数
	assert.Len(t, guy.Host().Network().Peers(), bootstrappers)

	// 测试 IpfsDHT.GetClosestPeers(...)
	out, err := guy.GetClosestPeers(ctx, val)
	if err != nil {
		t.Fatalf("IpfsDHT.GetClosestPeers() error: %s", err)
	}

	var outpeers []peer.ID
	for p := range out {
		outpeers = append(outpeers, p)
	}

	sort.Sort(peer.IDSlice(outpeers))
	exp := kb.SortClosestPeers(reachableIds, rtval)[:minInt(dht.KValue, len(reachableIds))]
	got := kb.SortClosestPeers(outpeers, rtval)
	assert.EqualValues(t, exp, got)
}
