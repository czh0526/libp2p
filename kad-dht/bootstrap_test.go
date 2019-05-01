package test_dht

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
)

func TestBootstrap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nDHTs := 30
	dhts := setupDHTS(t, ctx, nDHTs)
	defer func() {
		for i := 0; i < nDHTs; i++ {
			dhts[i].Close()
			defer dhts[i].Host().Close()
		}
	}()

	fmt.Printf("connecting %d dhts in a ring", nDHTs)
	for i := 0; i < nDHTs; i++ {
		connect(t, ctx, dhts[i], dhts[(i+1)%len(dhts)])
	}

	<-time.After(100 * time.Millisecond)
	stop := make(chan struct{})
	go func() {
		for {
			fmt.Printf("bootstrapping them so they find each other %d", nDHTs)
			ctxT, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			bootstrap(t, ctxT, dhts)

			select {
			case <-time.After(50 * time.Millisecond):
				continue
			case <-stop:
				return
			}
		}
	}()

	waitForWellFormedTables(t, dhts, 7, 10, 20*time.Second)
	close(stop)
}

// 从随机位置开始，顺序启动 DHT
func bootstrap(t *testing.T, ctx context.Context, dhts []*dht.IpfsDHT) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	fmt.Println("Bootstrapping DHTs ...")

	cfg := dht.DefaultBootstrapConfig
	cfg.Queries = 3

	start := rand.Intn(len(dhts))
	for i := range dhts {
		dht := dhts[(start+i)%len(dhts)]
		dht.RunBootstrap(ctx, cfg)
	}
}

func waitForWellFormedTables(t *testing.T, dhts []*dht.IpfsDHT, minPeers, avgPeers int, timeout time.Duration) bool {

	checkTables := func() bool {
		totalPeers := 0
		for _, dht := range dhts {
			rtlen := dht.RoutingTable().Size()
			totalPeers += rtlen
			// 检查到任意一个 DHT 的 routingTable 中节点数量少于 minPeers 时
			if minPeers > 0 && rtlen < minPeers {
				return false
			}
		}
		actualAvgPeers := totalPeers / len(dhts)
		fmt.Printf("avg rt size: %d", actualAvgPeers)
		if avgPeers > 0 && actualAvgPeers < avgPeers {
			fmt.Printf("avg rt size: %d < %d", actualAvgPeers, avgPeers)
			return false
		}

		return true
	}

	timeoutA := time.After(timeout)
	for {
		select {
		case <-timeoutA:
			fmt.Printf("did not reach well-formed routing tables by %s", timeout)
			return false // failed
		case <-time.After(5 * time.Millisecond):
			if checkTables() {
				return true // succeeded
			}
		}
	}

}
