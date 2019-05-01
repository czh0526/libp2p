package test_dht

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	routing "github.com/libp2p/go-libp2p-routing"

	dht "github.com/libp2p/go-libp2p-kad-dht"
)

type testValidator struct{}

func (testValidator) Select(_ string, bs [][]byte) (int, error) {
	index := -1
	for i, b := range bs {
		if bytes.Equal(b, []byte("newer")) {
			index = i
		} else if bytes.Equal(b, []byte("valid")) {
			if index == -1 {
				index = i
			}
		}
	}

	if index == -1 {
		return -1, errors.New("no rec found")
	}
	return index, nil
}

func (testValidator) Validate(_ string, b []byte) error {
	if bytes.Equal(b, []byte("expired")) {
		return errors.New("expired")
	}
	return nil
}

func TestValueGetSet(t *testing.T) {

	//logging.SetLogLevel("basichost", "DEBUG")
	//logging.SetLogLevel("swarm2", "DEBUG")
	//logging.SetLogLevel("dht", "DEBUG")
	//logging.SetLogLevel("table", "DEBUG")
	//logging.SetDebugLogging()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var dhts [5]*dht.IpfsDHT
	for i := range dhts {
		dhts[i] = setupDHT(ctx, t, false)
		fmt.Printf("%d). %s \n", i, dhts[i].Self().ShortString())
		defer dhts[i].Close()
		defer dhts[i].Host().Close()
	}

	// 0 -> 1
	connect(t, ctx, dhts[0], dhts[1])

	fmt.Printf("PutValue('/v/hello', 'world') to %s \n", dhts[0].Self().ShortString())
	ctxT, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	err := dhts[0].PutValue(ctxT, "/v/hello", []byte("world"))
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("requesting value on dhts: ", dhts[1].Self())
	ctxT, cancel = context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	val, err := dhts[1].GetValue(ctxT, "/v/hello")
	if err != nil {
		t.Fatal(err)
	}

	if string(val) != "world" {
		t.Fatalf("expected 'world' got '%s'", string(val))
	}
	fmt.Println("put value into a, got value on b")
	fmt.Println("--------------------------------------------------")

	// 2 -> 0, 1
	connect(t, ctx, dhts[2], dhts[0])
	connect(t, ctx, dhts[2], dhts[1])

	fmt.Println("requesting value (offline) on dhts: ", dhts[2].Self())
	vala, err := dhts[2].GetValue(ctxT, "/v/hello", dht.Quorum(0))
	if vala != nil {
		t.Fatalf("offline get should have failed, got %s", string(vala))
	}
	if err != routing.ErrNotFound {
		t.Fatalf("offline get should have failed with ErrNotFound, got: %s", err)
	}
	fmt.Println("when the offline dht become online, it has no value.")

	fmt.Println("requesting value (online) on dhts: ", dhts[2].Self())
	vala, err = dhts[2].GetValue(ctxT, "/v/hello")
	if err != nil {
		t.Fatal(err)
	}
	if string(val) != "world" {
		t.Fatalf("Expected 'world' got '%s'", string(val))
	}
	fmt.Println("the new online dhts got value by query to others")
	fmt.Println("--------------------------------------------------")

	// 3 -> 0,1,2
	for _, d := range dhts[:3] {
		connect(t, ctx, dhts[3], d)
	}
	// 4 -> 3 -> 0,1,2
	connect(t, ctx, dhts[4], dhts[3])

	fmt.Println("requesting value(requires peer routing) on dhts: ", dhts[4].Self())
	val, err = dhts[4].GetValue(ctxT, "/v/hello")
	if err != nil {
		t.Fatal(err)
	}

	if string(val) != "world" {
		t.Fatalf("Expected 'world' got '%s'", string(val))
	}
	fmt.Println("query path: 4 -> 3 -> 0,1,2")
}

func TestValueGetInNetwork(t *testing.T) {

	//logging.SetLogLevel("basichost", "DEBUG")
	//logging.SetLogLevel("swarm2", "DEBUG")
	//logging.SetLogLevel("dht", "DEBUG")
	//logging.SetLogLevel("table", "DEBUG")
	//logging.SetDebugLogging()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var dhts [5]*dht.IpfsDHT
	for i := range dhts {
		dhts[i] = setupDHT(ctx, t, false)
		fmt.Printf("%d). %s \n", i, dhts[i].Self().ShortString())
		defer dhts[i].Close()
		defer dhts[i].Host().Close()
	}

	// 0 -> 1
	connect(t, ctx, dhts[0], dhts[1])

	// set value in node 0
	fmt.Printf("PutValue('/v/hello', 'world') to %s \n", dhts[0].Self().ShortString())
	ctxT, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	err := dhts[0].PutValue(ctxT, "/v/hello", []byte("world"))
	if err != nil {
		t.Fatal(err)
	}

	// 2 -> 0, 1
	connect(t, ctx, dhts[2], dhts[0])
	connect(t, ctx, dhts[2], dhts[1])
	// 3 -> 0,1,2
	for _, d := range dhts[:3] {
		connect(t, ctx, dhts[3], d)
	}
	// 4 -> 3 -> 0,1,2
	connect(t, ctx, dhts[4], dhts[3])

	fmt.Println("requesting value(requires peer routing) on dhts: ", dhts[4].Self())
	val, err := dhts[4].GetValue(ctxT, "/v/hello")
	if err != nil {
		t.Fatal(err)
	}

	if string(val) != "world" {
		t.Fatalf("Expected 'world' got '%s'", string(val))
	}
	fmt.Println("query path: 4 -> 3 -> 0,1,2")
}
