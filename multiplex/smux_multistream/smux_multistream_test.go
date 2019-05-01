package stream_multistream

import (
	"fmt"
	"testing"

	test "github.com/czh0526/libp2p/multiplex/stream-muxer"
	smux_mplex "github.com/whyrusleeping/go-smux-multiplex"
	smux_mstream "github.com/whyrusleeping/go-smux-multistream"
	yamux "github.com/whyrusleeping/go-smux-yamux"
)

func TestMultiplexTransport(t *testing.T) {
	var tpt *smux_mstream.Transport
	tpt = smux_mstream.NewBlankTransport()
	tpt.AddTransport("/yamux/1.0.0", yamux.DefaultTransport)
	test.SubtestAll(t, tpt)

	fmt.Println("-----------------------------------------")

	tpt = smux_mstream.NewBlankTransport()
	tpt.AddTransport("/smux_mplex/1.0.0", smux_mplex.DefaultTransport)
	test.SubtestAll(t, tpt)
}
