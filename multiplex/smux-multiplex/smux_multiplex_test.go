package smux_multiplex

import (
	"testing"

	test "github.com/czh0526/libp2p/multiplex/stream-muxer"
	smux_mplex "github.com/whyrusleeping/go-smux-multiplex"
)

func TestMultiplexTransport(t *testing.T) {
	test.SubtestAll(t, smux_mplex.DefaultTransport)
}
