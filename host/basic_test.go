package test_host

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	swarmt "github.com/czh0526/libp2p/swarm"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	protocol "github.com/libp2p/go-libp2p-protocol"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
)

func TestHostSimple(t *testing.T) {
	ctx := context.Background()
	// 生成 Host
	h1 := bhost.New(swarmt.GenSwarm(t, ctx))
	h2 := bhost.New(swarmt.GenSwarm(t, ctx))
	defer h1.Close()
	defer h2.Close()

	// 建立长连接：h1 ==> h2
	h2pi := h2.Peerstore().PeerInfo(h2.ID())
	if err := h1.Connect(ctx, h2pi); err != nil {
		t.Fatal(err)
	}

	// 给 h2 增加协议消息 (protocol.TestingID) 处理器
	piper, pipew := io.Pipe()
	h2.SetStreamHandler(protocol.TestingID, func(s inet.Stream) {
		defer s.Close()
		// 复制协议消息进入管道
		w := io.MultiWriter(s, pipew)
		io.Copy(w, s)
	})

	// 为 protocol.TestingID 打开 Stream
	s, err := h1.NewStream(ctx, h2pi.ID, protocol.TestingID)
	if err != nil {
		t.Fatal(err)
	}

	// --> h1 stream
	buf1 := []byte("abcdefghijkl")
	if _, err := s.Write(buf1); err != nil {
		t.Fatal(err)
	}

	// h2 stream -->
	buf2 := make([]byte, len(buf1))
	if _, err := io.ReadFull(s, buf2); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf1, buf2) {
		t.Fatalf("buf1 != buf2 -- %x != %x", buf1, buf2)
	}

	// piper -->
	buf3 := make([]byte, len(buf1))
	if _, err := io.ReadFull(piper, buf3); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf1, buf3) {
		t.Fatalf("buf1 != buf3 --%x != %x", buf1, buf3)
	}
}

func getHostPair(ctx context.Context, t *testing.T) (host.Host, host.Host) {
	h1 := bhost.New(swarmt.GenSwarm(t, ctx))
	h2 := bhost.New(swarmt.GenSwarm(t, ctx))

	h2pi := h2.Peerstore().PeerInfo(h2.ID())
	if err := h1.Connect(ctx, h2pi); err != nil {
		t.Fatal(err)
	}

	return h1, h2
}

func assertWait(t *testing.T, c chan protocol.ID, exp protocol.ID) {
	select {
	case proto := <-c:
		if proto != exp {
			t.Fatal("should have connected on ", exp)
		}
	case <-time.After(time.Second * 5):
		t.Fatal("timeout waiting for stream")
	}
}

func TestHostProtoPreference(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h1, h2 := getHostPair(ctx, t)
	defer h1.Close()
	defer h2.Close()

	protoOld := protocol.ID("/testing")
	protoNew := protocol.ID("/testsing/1.1.0")
	protoMinor := protocol.ID("/testing/1.2.0")

	connectedOn := make(chan protocol.ID)
	handler := func(s inet.Stream) {
		connectedOn <- s.Protocol()
		s.Close()
	}

	//
	// "/testing"
	//
	h1.SetStreamHandler(protoOld, handler)
	// /testing ==> /testing
	s, err := h2.NewStream(ctx, h1.ID(), protoOld)
	if err != nil {
		t.Fatal(err)
	}
	assertWait(t, connectedOn, protoOld)
	s.Close()

	// /testing/1.1.0 ==> /testing
	s, err = h2.NewStream(ctx, h1.ID(), protoNew)
	if err == nil {
		t.Fatalf("%s ==> %s should got error", protoNew, protoOld)
	}
	fmt.Printf("%s ==> %s error: %s \n", protoNew, protoOld, err)

	//
	// protocol is higher than /testing/1.1.0
	//
	mfunc, err := host.MultistreamSemverMatcher(protoMinor)
	if err != nil {
		t.Fatal(err)
	}
	h1.SetStreamHandlerMatch(protoMinor, mfunc, handler)

	// /testing ==> /testing/1.1.0
	s2, err := h2.NewStream(ctx, h1.ID(), protoOld)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s2.Write([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	assertWait(t, connectedOn, protoOld)
	s2.Close()

	// /testing/1.1.0 ==> /testing/1.1.0
	s2, err = h2.NewStream(ctx, h1.ID(), protoMinor)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s2.Write([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	assertWait(t, connectedOn, protoMinor)
	s2.Close()

	// /testing/1.2.0 ==> /testsing/1.1.0
	s2, err = h2.NewStream(ctx, h1.ID(), protoNew)
	if err == nil {
		t.Fatalf("%s ==> %s should got error", protoNew, protoMinor)
	}
	fmt.Printf("%s ==> %s error: %s \n", protoNew, protoMinor, err)
}
