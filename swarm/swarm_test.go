package test_swarm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	swarm "github.com/libp2p/go-libp2p-swarm"
	ma "github.com/multiformats/go-multiaddr"
)

func EchoStreamHandler(stream inet.Stream) {
	go func() {
		defer stream.Close()

		c := stream.Conn()
		fmt.Printf("%s ponging to %s  \n", c.LocalPeer(), c.RemotePeer())
		buf := make([]byte, 4)

		for {
			if _, err := stream.Read(buf); err != nil {
				if err != io.EOF {
					fmt.Printf("ping receive error: %s \n", err)
				}
				return
			}

			if !bytes.Equal(buf, []byte("ping")) {
				fmt.Printf("ping receive error: ping != %s \n", buf)
				return
			}

			if _, err := stream.Write([]byte("pong")); err != nil {
				fmt.Printf("pong send error: %s \n", err)
				return
			}
		}
	}()
}

func makeSwarms(ctx context.Context, t *testing.T, num int, opts ...Option) []*swarm.Swarm {
	swarms := make([]*swarm.Swarm, 0, num)

	for i := 0; i < num; i++ {
		swarm := GenSwarm(t, ctx, opts...)
		swarm.SetStreamHandler(EchoStreamHandler)
		swarms = append(swarms, swarm)
	}

	return swarms
}

func connectSwarms(t *testing.T, ctx context.Context, swarms []*swarm.Swarm) {
	var wg sync.WaitGroup
	connect := func(s *swarm.Swarm, dst peer.ID, addr ma.Multiaddr) {
		s.Peerstore().AddAddr(dst, addr, pstore.PermanentAddrTTL)
		if _, err := s.DialPeer(ctx, dst); err != nil {
			t.Fatal("error swarm dialing to peer", err)
		}
		wg.Done()
	}

	fmt.Println("Connecting swarms simultaneously.")
	for i, s1 := range swarms {
		fmt.Printf("peer %v \n", i+1)
		for _, s2 := range swarms[i+1:] {
			wg.Add(1)
			connect(s1, s2.LocalPeer(), s2.ListenAddresses()[0])
		}
	}
	wg.Wait()

	for _, s := range swarms {
		fmt.Printf("%s swarm routing table: %s \n", s.LocalPeer(), s.Peers())
	}
}

func SubtestSwarm(t *testing.T, SwarmNum int, MsgNum int) {
	ctx := context.Background()
	swarms := makeSwarms(ctx, t, SwarmNum, OptDisableReuseport)
	fmt.Println("\n------ finish making swarm -------")

	connectSwarms(t, ctx, swarms)
	fmt.Println("\n------ finish connecting swarm -------")

	// 遍历每个 swarm 点
	for _, s1 := range swarms {
		fmt.Println("----------------------------------------")
		fmt.Printf("%s ping/pong round \n", s1.LocalPeer())
		fmt.Println("----------------------------------------")

		_, cancel := context.WithCancel(ctx)
		got := map[peer.ID]int{}
		errChan := make(chan error, MsgNum*len(swarms))
		streamChan := make(chan inet.Stream, MsgNum)

		// 集中发送 ping
		go func() {
			defer close(streamChan)

			// 定义一个发送 ping 的函数
			var wg sync.WaitGroup
			send := func(p peer.ID) {
				defer wg.Done()

				stream, err := s1.NewStream(ctx, p)
				if err != nil {
					errChan <- err
					return
				}

				// 向邻居发送 MsgNum 个 ping 消息
				for k := 0; k < MsgNum; k++ {
					msg := "ping"
					fmt.Printf("%s %s %s (%d) \n", s1.LocalPeer(), msg, p, k)
					if _, err := stream.Write([]byte(msg)); err != nil {
						errChan <- err
						continue
					}
				}

				streamChan <- stream
			}

			// 让每个盒子向邻居发送 ping 消息
			for _, s2 := range swarms {
				if s2.LocalPeer() == s1.LocalPeer() {
					continue
				}

				wg.Add(1)
				go send(s2.LocalPeer())
				<-time.After(time.Second)
			}
			wg.Wait()
		}()

		// 集中接收 pong
		go func() {
			defer close(errChan)
			count := 0
			countShouldBe := MsgNum * (len(swarms) - 1)
			for stream := range streamChan {
				defer stream.Close()

				p := stream.Conn().RemotePeer()

				msgCount := 0
				msg := make([]byte, 4)
				for k := 0; k < MsgNum; k++ {
					if _, err := stream.Read(msg); err != nil {
						errChan <- err
						continue
					}
					if string(msg) != "pong" {
						errChan <- fmt.Errorf("unexpected message: %s", msg)
						continue
					}

					fmt.Printf("%s %s %s (%d) \n", s1.LocalPeer(), msg, p, k)
					msgCount++
				}

				got[p] = msgCount
				count += msgCount
			}

			if count != countShouldBe {
				errChan <- fmt.Errorf("count mismatch: %d != %d", count, countShouldBe)
			}
		}()

		// 检查结果
		for err := range errChan {
			if err != nil {
				t.Error(err.Error())
			}
		}

		fmt.Printf("%s got pongs \n", s1.LocalPeer())
		if (len(swarms) - 1) != len(got) {
			t.Errorf("got (%d) less messages than sent(%d).", len(got), len(swarms))
		}

		for p, n := range got {
			if n != MsgNum {
				t.Error("peer did not get all msgs", p, n, "/", MsgNum)
			}
		}

		cancel()
		<-time.After(10 * time.Millisecond)
	}
}

func TestSwarm(t *testing.T) {
	t.Parallel()

	msgs := 1
	swarms := 5
	SubtestSwarm(t, swarms, msgs)
}
