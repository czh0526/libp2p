package test_dht

import (
	"context"
	"fmt"
	"testing"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	record "github.com/libp2p/go-libp2p-record"
)

func TestSearchValue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dhtA := setupDHT(ctx, t, false)
	dhtB := setupDHT(ctx, t, false)

	defer dhtA.Close()
	defer dhtB.Close()
	defer dhtA.Host().Close()
	defer dhtB.Host().Close()

	connect(t, ctx, dhtA, dhtB)

	dhtA.Validator.(record.NamespacedValidator)["v"] = testValidator{}
	dhtB.Validator.(record.NamespacedValidator)["v"] = testValidator{}

	ctxT, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	// put value into dhtA
	fmt.Printf("===> put('/v/hello', 'valid') into %s \n", dhtA.Self().ShortString())
	err := dhtA.PutValue(ctxT, "/v/hello", []byte("valid"))
	if err != nil {
		t.Error(err)
	}

	// 从 dhtA 开始，发起一次查找过程
	ctxT, cancel = context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	valCh, err := dhtA.SearchValue(ctx, "/v/hello", dht.Quorum(-1))
	if err != nil {
		t.Fatal(err)
	}

	// 读取查找的结果
	select {
	case v := <-valCh:
		if string(v) != "valid" {
			t.Errorf("expected 'valid', got '%s'", string(v))
		}
	case <-ctxT.Done():
		t.Fatal(ctxT.Err())
	}

	<-time.After(time.Microsecond * 100)

	// 在查找过程中，修改 dhtB 中的数据
	fmt.Printf("===> put('/v/hello', 'newer') into %s \n", dhtB.Self().ShortString())
	err = dhtB.PutValue(ctxT, "/v/hello", []byte("newer"))
	if err != nil {
		t.Error(err)
	}

	select {
	case v, ok := <-valCh:
		if !ok {
			fmt.Println("==> valCh is closed.")
		}
		if string(v) != "newer" {
			t.Errorf("expect 'newer', got '%s'", string(v))
		}
	case <-ctxT.Done():
	}
}
