package stream_multiplex

import (
	"testing"

	test "github.com/czh0526/p2p/multiplex/stream-muxer"
	ps_mplex "github.com/whyrusleeping/go-smux-multiplex"
)

func TestMultiplexTransport(t *testing.T) {
	test.SubtestAll(t, ps_mplex.DefaultTransport)
}
