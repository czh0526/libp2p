package p2ptransport

import (
	"reflect"
	"runtime"
	"testing"

	peer "github.com/libp2p/go-libp2p-peer"
	tpt "github.com/libp2p/go-libp2p-transport"
	ma "github.com/multiformats/go-multiaddr"
)

var Subtests = []func(t *testing.T, ta, tb tpt.Transport, maddr ma.Multiaddr, peerA peer.ID){
	SubtestProtocols,
	SubtestBasic,
	SubtestPingPong,
	SubtestCancel,
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func SubtestTransport(t *testing.T, ta, tb tpt.Transport, addr string, peerA peer.ID) {
	maddr, err := ma.NewMultiaddr(addr)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range Subtests {
		t.Run(getFunctionName(f), func(t *testing.T) {
			f(t, ta, tb, maddr, peerA)
		})
	}
}
