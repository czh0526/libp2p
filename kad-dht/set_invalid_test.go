package test_dht

import (
	"context"
	"testing"
	"time"

	record "github.com/libp2p/go-libp2p-record"
	routing "github.com/libp2p/go-libp2p-routing"
)

func TestValueSetInvalid(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dhtA := setupDHT(ctx, t, false)
	dhtB := setupDHT(ctx, t, false)

	defer dhtA.Close()
	defer dhtB.Close()
	defer dhtA.Host().Close()
	defer dhtB.Host().Close()

	dhtA.Validator.(record.NamespacedValidator)["v"] = testValidator{}
	dhtB.Validator.(record.NamespacedValidator)["v"] = blankValidator{}

	connect(t, ctx, dhtA, dhtB)

	testSetGet := func(val string, failset bool, exp string, experr error) {
		ctxT, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		// 测试 put value
		err := dhtA.PutValue(ctxT, "/v/hello", []byte(val))
		if failset {
			if err == nil {
				t.Fatal("expected set to fail")
			}
		} else {
			if err != nil {
				t.Fatal(err)
			}
		}

		// 测试 get value
		ctxT, cancel = context.WithTimeout(ctx, time.Second*2)
		defer cancel()
		valb, err := dhtB.GetValue(ctxT, "/v/hello")
		if err != experr {
			t.Fatalf("Set/Get %v: Expected '%v' error but got %v", val, experr, err)
		} else if err == nil && string(valb) != exp {
			t.Fatalf("Expected '%v' got '%s'", exp, string(valb))
		}
	}

	testSetGet("expired", true, "", routing.ErrNotFound)
	testSetGet("valid", false, "valid", nil)
	testSetGet("newer", false, "newer", nil)
	testSetGet("valid", true, "newer", nil)
}
