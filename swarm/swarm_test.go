package test_swarm

import (
	"context"
	"fmt"
	"testing"

	swarm "github.com/libp2p/go-libp2p-swarm"
)

func makeSwarms(ctx context.Context, t *testing.T, num int, opts ...Option) []*swarm.Swarm {
	swarms := make([]*swarm.Swarm, 0, num)

	for i := 0; i < num; i++ {
		swarm := GenSwarm(t, ctx, opts...)
		swarm.SetStreamHandler(EchoStreamHandler)
		swarms = append(swarms, swarm)
	}
}

func SubtestSwarm(t *testing.T, SwarmNum int, MsgNum int) {
	ctx := context.Background()
	swarms := makeSwarms(ctx, t, SwarmNum, OptDisableReuseport)

	connectSwarms(t, ctx, swarms)

	for _, s1 := range swarms {
		fmt.Println("----------------------------------------")
		fmt.Printf("%s ping/pong round", s1.LocalPeer())
		fmt.Println("----------------------------------------")
	}
}

func TestSwarm(t *testing.T) {
	t.Parallel()

	msgs := 100
	swarms := 5
	subtestSwarm(t, swarms, msgs)
}
