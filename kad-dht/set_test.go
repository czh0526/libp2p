package test_dht

import (
	"context"
	"fmt"
	"testing"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
)

func TestValueSet(t *testing.T) {

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
		fmt.Printf("%d). %s \n", i, dhts[i].Self())
		defer dhts[i].Close()
		defer dhts[i].Host().Close()
	}

	/*
					/------ 0
		4 - 3  - 2			|
					\_______1
	*/
	connect(t, ctx, dhts[0], dhts[1])
	connect(t, ctx, dhts[2], dhts[0])
	connect(t, ctx, dhts[2], dhts[1])
	connect(t, ctx, dhts[3], dhts[2])
	connect(t, ctx, dhts[4], dhts[3])

	fmt.Printf("adding value on: %s \n", dhts[0].Self())
	ctxT, cancel := context.WithTimeout(ctx, 100*time.Second)
	defer cancel()
	err := dhts[0].PutValue(ctxT, "/v/hello", []byte("world"))
	if err != nil {
		t.Fatal(err)
	}

	<-time.After(5 * time.Second)
}
