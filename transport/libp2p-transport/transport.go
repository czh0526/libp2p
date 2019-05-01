package p2ptransport

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"sync"
	"testing"

	peer "github.com/libp2p/go-libp2p-peer"
	tpt "github.com/libp2p/go-libp2p-transport"
	ma "github.com/multiformats/go-multiaddr"
)

var testData = []byte("this is some test data")

func SubtestProtocols(t *testing.T, ta, tb tpt.Transport, maddr ma.Multiaddr, peerA peer.ID) {
	rawIPAddr, _ := ma.NewMultiaddr("/ip4/1.2.3.4")
	if ta.CanDial(rawIPAddr) || tb.CanDial(rawIPAddr) {
		t.Error("nothing should be able to dial raw IP")
	}

	tprotos := make(map[int]bool)
	for _, p := range ta.Protocols() {
		tprotos[p] = true
		fmt.Printf("ta protocol: %v \n", p)
	}

	if !ta.Proxy() {
		protos := maddr.Protocols()
		for _, p := range protos {
			fmt.Printf("maddr protocols: %v,%v \n", p.Code, p.Name)
		}
		proto := protos[len(protos)-1]
		if !tprotos[proto.Code] {
			t.Errorf("transport should have reported that it supports protocol '%s' (%d)", proto.Name, proto.Code)
		}
	} else {
		found := false
		for _, proto := range maddr.Protocols() {
			if tprotos[proto.Code] {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("didn't find any matching proxy protocols in maddr: %s", maddr)
		}
	}
}

func SubtestBasic(t *testing.T, ta, tb tpt.Transport, maddr ma.Multiaddr, peerA peer.ID) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 从 tcp transport 中获取 Listener
	list, err := ta.Listen(maddr)
	if err != nil {
		t.Fatal(err)
	}
	defer list.Close()
	fmt.Printf("'transport A' begin listening at: %s \n", maddr)

	var (
		connA, connB tpt.Conn
		done         = make(chan struct{})
	)
	defer func() {
		<-done
		if connA != nil {
			connA.Close()
		}
		if connB != nil {
			connB.Close()
		}
	}()

	// 被动监听的例程
	go func() {
		defer close(done)
		var err error

		// 等待连接
		connB, err = list.Accept()
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Printf("'Transport A' got a conn from %s \n", connB.RemoteMultiaddr())

		// 等待建立 Stream
		s, err := connB.AcceptStream()
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Printf("'Transport A' got a stream \n")

		// 从 Stream 读取数据
		buf, err := ioutil.ReadAll(s)
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Printf("'Transport A' read data from stream <= %s \n", buf)

		if !bytes.Equal(testData, buf) {
			t.Errorf("expected %s, got %s", testData, buf)
		}

		// 向 Stream 里写数据
		n, err := s.Write(testData)
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Printf("'Transport A' write data to stream => %s \n", testData)
		if n != len(testData) {
			t.Error(err)
			return
		}

		err = s.Close()
		if err != nil {
			t.Error(err)
		}
	}()

	if !tb.CanDial(list.Multiaddr()) {
		t.Error("CanDial should have returned true")
	}

	// 建立 Conn 连接
	connA, err = tb.Dial(ctx, list.Multiaddr(), peerA)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("'Transport B' dial to %s: %s \n", list.Multiaddr(), peerA.String())

	// 主动打开流
	s, err := connA.OpenStream()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("'Transport B' open stream\n")

	n, err := s.Write(testData)
	if err != nil {
		t.Fatal(err)
		return
	}
	fmt.Printf("'Transport B' write data to stream ==> %s \n", testData)

	if n != len(testData) {
		t.Fatalf("failed to write enough data (a->b)")
		return
	}

	err = s.Close()
	if err != nil {
		t.Fatal(err)
		return
	}
	fmt.Printf("'Transport B' close stream \n")

	// 从 Stream 流中读数据
	buf, err := ioutil.ReadAll(s)
	if err != nil {
		t.Fatal(err)
		return
	}
	fmt.Printf("'Transport B' read data from stream <== %s \n", buf)

	if !bytes.Equal(testData, buf) {
		t.Errorf("expected %s, got %s", testData, buf)
	}
}

func SubtestPingPong(t *testing.T, ta, tb tpt.Transport, maddr ma.Multiaddr, peerA peer.ID) {
	streams := 100

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	list, err := ta.Listen(maddr)
	if err != nil {
		t.Fatal(err)
	}
	defer list.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c, err := list.Accept()
		if err != nil {
			t.Error(err)
			return
		}
		defer c.Close()

		var sWg sync.WaitGroup
		for i := 0; i < streams; i++ {
			s, err := c.AcceptStream()
			if err != nil {
				t.Error(err)
				return
			}

			sWg.Add(1)
			go func() {
				defer sWg.Done()

				data, err := ioutil.ReadAll(s)
				if err != nil {
					s.Reset()
					t.Error(err)
					return
				}
				fmt.Printf("[a <- b] %s \n", data)

				if !bytes.HasPrefix(data, testData) {
					t.Errorf("expected %q to have prefix %q", string(data), string(testData))
				}

				n, err := s.Write(data)
				if err != nil {
					s.Reset()
					t.Error(err)
					return
				}
				fmt.Printf("[a -> b] %s \n", data)

				if n != len(data) {
					s.Reset()
					t.Error(err)
					return
				}

				s.Close()
			}()
		}
		sWg.Wait()
	}()

	if !tb.CanDial(list.Multiaddr()) {
		t.Error("CanDial should have returned true")
	}

	c, err := tb.Dial(ctx, list.Multiaddr(), peerA)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	for i := 0; i < streams; i++ {
		s, err := c.OpenStream()
		if err != nil {
			t.Error(err)
			continue
		}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			data := []byte(fmt.Sprintf("%s - %d", testData, i))
			n, err := s.Write(data)
			if err != nil {
				s.Reset()
				t.Error(err)
				return
			}
			fmt.Printf("[b -> a] %s \n", data)

			if n != len(data) {
				s.Reset()
				t.Error("failed to write enough data (a->b)")
				return
			}
			s.Close()

			ret, err := ioutil.ReadAll(s)
			if err != nil {
				s.Reset()
				t.Error(err)
				return
			}
			fmt.Printf("[b <- a] %s \n", ret)
			if !bytes.Equal(data, ret) {
				t.Errorf("expected %q, got %q", string(data), string(ret))
			}
		}(i)
	}
	wg.Wait()
}

func SubtestCancel(t *testing.T, ta, tb tpt.Transport, maddr ma.Multiaddr, peerA peer.ID) {
	list, err := ta.Listen(maddr)
	if err != nil {
		t.Fatal(err)
	}
	defer list.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c, err := tb.Dial(ctx, list.Multiaddr(), peerA)
	if err == nil {
		c.Close()
		t.Fatal("dial should have failed.")
	}
}
