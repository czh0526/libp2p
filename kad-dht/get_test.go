package test_dht

import (
	"context"
	"sort"
	"testing"
	"time"
)

func TestGetValues(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dhtA := setupDHT(ctx, t, false)
	dhtB := setupDHT(ctx, t, false)

	defer dhtA.Close()
	defer dhtB.Close()
	defer dhtA.Host().Close()
	defer dhtB.Host().Close()

	connect(t, ctx, dhtA, dhtB)

	ctxT, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	err := dhtB.PutValue(ctxT, "/v/hello", []byte("newer"))
	if err != nil {
		t.Error(err)
	}

	err = dhtA.PutValue(ctxT, "/v/hello", []byte("valid"))
	if err != nil {
		t.Error(err)
	}

	ctxT, cancel = context.WithTimeout(ctx, time.Second*2)
	defer cancel()

	// 指定数量一次获取，不通过 channel 获取
	vals, err := dhtA.GetValues(ctx, "/v/hello", 16)
	if err != nil {
		t.Fatal(err)
	}

	// 校验结果
	if len(vals) != 2 {
		t.Fatalf("expected to get 2 values, got %d", len(vals))
	}

	sort.Slice(vals, func(i, j int) bool {
		return string(vals[i].Val) < string(vals[j].Val)
	})

	if string(vals[0].Val) != "valid" {
		t.Errorf("unexpected vals[0]: %s", string(vals[0].Val))
	}

	if string(vals[1].Val) != "valid" {
		t.Errorf("unexpected vals[1]: %s", string(vals[1].Val))
	}
}
