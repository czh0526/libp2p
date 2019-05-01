package test_dht

import (
	"context"
	"strings"
	"testing"
	"time"

	swarmt "github.com/czh0526/libp2p/swarm"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	opts "github.com/libp2p/go-libp2p-kad-dht/opts"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
)

func TestGetSetPluggedProtocol(t *testing.T) {
	t.Run("PutValue/GetValue - same protocol", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		os := []opts.Option{
			// 设置相同的协议名
			opts.Protocols("/esh/dht"),
			opts.Client(false),
			opts.NamespacedValidator("v", blankValidator{}),
		}

		dhtA, err := dht.New(ctx, bhost.New(swarmt.GenSwarm(t, ctx, swarmt.OptDisableReuseport)), os...)
		if err != nil {
			t.Fatal(err)
		}

		dhtB, err := dht.New(ctx, bhost.New(swarmt.GenSwarm(t, ctx, swarmt.OptDisableReuseport)), os...)
		if err != nil {
			t.Fatal(err)
		}

		connect(t, ctx, dhtA, dhtB)
		ctxT, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := dhtA.PutValue(ctxT, "/v/cat", []byte("meow")); err != nil {
			t.Fatal(err)
		}

		value, err := dhtB.GetValue(ctxT, "/v/cat")
		if err != nil {
			t.Fatal(err)
		}

		if string(value) != "meow" {
			t.Fatalf("Expected 'meow' got '%s'", string(value))
		}
	})

	t.Run("DHT routing table for peer A won't contain B if A and B don't use same protocol", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		dhtA, err := dht.New(ctx,
			bhost.New(swarmt.GenSwarm(t, ctx, swarmt.OptDisableReuseport)),
			[]opts.Option{
				// 设置不同的协议名
				opts.Protocols("/esh/dht"),
				opts.Client(false),
				opts.NamespacedValidator("v", blankValidator{}),
			}...)
		if err != nil {
			t.Fatal(err)
		}

		dhtB, err := dht.New(ctx,
			bhost.New(swarmt.GenSwarm(t, ctx, swarmt.OptDisableReuseport)),
			[]opts.Option{
				// 设置不同的协议名
				opts.Protocols("/lsr/dht"),
				opts.Client(false),
				opts.NamespacedValidator("v", blankValidator{}),
			}...)
		if err != nil {
			t.Fatal(err)
		}

		connectNoSync(t, ctx, dhtA, dhtB)

		time.Sleep(time.Second * 2)

		err = dhtA.PutValue(ctx, "/v/cat", []byte("meow"))
		if err == nil || !strings.Contains(err.Error(), "failed to find any peer in table") {
			t.Fatalf("put should not have been able to find any peers in routing table, err: %v", err)
		}

		_, err = dhtB.GetValue(ctx, "/v/cat")
		if err == nil || !strings.Contains(err.Error(), "failed to find any peer in table") {
			t.Fatalf("got should not have been able to find any peers in routing table, err: %v", err)
		}
	})
}
