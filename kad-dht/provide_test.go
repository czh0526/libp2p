package test_dht

import (
	"context"
	"fmt"
	"testing"
	"time"

	cid "github.com/ipfs/go-cid"
	mhash "github.com/multiformats/go-multihash"
)

var testCaseCids []cid.Cid

func init() {
	for i := 0; i < 1; i++ {
		v := fmt.Sprintf("%d -- value", i)

		mhv, err := mhash.Sum([]byte(v), mhash.SHA2_256, -1)
		if err != nil {
			panic(fmt.Sprintf("init error: %s ", err))
		}
		testCaseCids = append(testCaseCids, cid.NewCidV0(mhv))
	}
}

// 测试 ContentRouting 的 Provide(), FindProvidersAsync() 方法
func TestProvides(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dhts := setupDHTS(t, ctx, 4)
	defer func() {
		for i := 0; i < 4; i++ {
			dhts[i].Close()
			defer dhts[i].Host().Close()
		}
	}()

	connect(t, ctx, dhts[0], dhts[1])
	connect(t, ctx, dhts[1], dhts[2])
	connect(t, ctx, dhts[1], dhts[3])

	for _, k := range testCaseCids {
		// 消息扩散
		if err := dhts[3].Provide(ctx, k, true); err != nil {
			t.Fatal(err)
		}
	}

	time.Sleep(time.Millisecond * 6)

	n := 0
	for _, c := range testCaseCids {
		n = (n + 1) % 3

		ctxT, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		// 在 DHT 网络中查找
		provchan := dhts[n].FindProvidersAsync(ctxT, c, 1)

		select {
		case prov := <-provchan:
			if prov.ID == "" {
				t.Fatal("Got back nil provider")
			}
			if prov.ID != dhts[3].Self() {
				t.Fatal("Got back wrong provider.")
			}
		case <-ctxT.Done():
			t.Fatal("Did not get a provider back.")
		}
	}
}

func TestLocalProvides(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dhts := setupDHTS(t, ctx, 4)
	defer func() {
		for i := 0; i < len(dhts); i++ {
			dhts[i].Close()
			dhts[i].Host().Close()
		}
	}()

	connect(t, ctx, dhts[0], dhts[1])
	connect(t, ctx, dhts[1], dhts[2])
	connect(t, ctx, dhts[1], dhts[3])

	for _, k := range testCaseCids {
		// 消息不扩散
		if err := dhts[3].Provide(ctx, k, false); err != nil {
			t.Fatal(err)
		}
	}

	time.Sleep(time.Millisecond * 10)

	for _, c := range testCaseCids {
		for i := 0; i < 3; i++ {
			// 在 [0-2] 节点进行本地查找,验证 [3] 的数据是否扩散到其他节点
			provs := dhts[i].Providers().GetProviders(ctx, c)
			if len(provs) > 0 {
				t.Fatal("shouldn't know this")
			}
		}
	}
}
