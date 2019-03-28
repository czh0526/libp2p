package test_swarm

import (
	"context"
	"fmt"
	"testing"
	"time"

	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	swarm "github.com/libp2p/go-libp2p-swarm"
	ma "github.com/multiformats/go-multiaddr"
)

func TestNotifications(t *testing.T) {
	const swarmSize = 5

	ctx := context.Background()
	swarms := makeSwarms(ctx, t, swarmSize)
	defer func() {
		for _, s := range swarms {
			s.Close()
		}
	}()

	timeout := 5 * time.Second
	// 为 Swarm 设置 Notifiee
	notifiees := make([]*netNotifiee, len(swarms))
	for i, swarm := range swarms {
		n := newNetNotifiee(swarmSize)
		swarm.Notify(n)
		notifiees[i] = n
	}

	// 连接各个 Swarm, 监控 Connected 事件
	connectSwarms(t, ctx, swarms)

	// 处理 Connected 事件
	<-time.After(time.Millisecond)
	for i, s := range swarms {
		n := notifiees[i]
		notifs := make(map[peer.ID][]inet.Conn)
		for j, s2 := range swarms {
			if i == j {
				continue
			}

			for len(s.ConnsToPeer(s2.LocalPeer())) != len(notifs[s2.LocalPeer()]) {
				select {
				case c := <-n.connected:
					nfp := notifs[c.RemotePeer()]
					notifs[c.RemotePeer()] = append(nfp, c)
				case <-time.After(timeout):
					t.Fatal("timeout")
				}
			}
		}

		// Swarm s 现在存在的连接，和接到通知的连接必须匹配
		for p, cons := range notifs {
			expect := s.ConnsToPeer(p)
			if len(expect) != len(cons) {
				t.Fatal("got different number of connections")
			}

			for _, c := range cons {
				var found bool
				for _, c2 := range expect {
					if c == c2 {
						found = true
						break
					}
				}

				if !found {
					t.Fatal("connection not found!")
				}
			}
		}
	}

	// 遍历全部 Swarm, 找到与 Conn c 远端的 Swarm, netNotifiee, Conn
	complement := func(c inet.Conn) (*swarm.Swarm, *netNotifiee, *swarm.Conn) {
		for i, s := range swarms {
			for _, c2 := range s.Conns() {
				if c.LocalMultiaddr().Equal(c2.RemoteMultiaddr()) &&
					c2.LocalMultiaddr().Equal(c.RemoteMultiaddr()) {
					return s, notifiees[i], c2.(*swarm.Conn)
				}
			}
		}
		t.Fatal("complementary conn not found", c)
		return nil, nil, nil
	}

	// 判断 Notifiee 检测到的变化对象是不是 s
	testOCStream := func(n *netNotifiee, s inet.Stream) {
		var s2 inet.Stream
		// 检测流打开
		select {
		case s2 = <-n.openedStream:
			fmt.Println("got notif for opened stream.")
		case <-time.After(timeout):
			t.Fatal("timeout")
		}
		if s != s2 {
			t.Fatal("got incorrect stream", s.Conn(), s2.Conn())
		}

		// 检测流关闭
		select {
		case s2 = <-n.closedStream:
			fmt.Println("got notif for closed stream.")
		case <-time.After(timeout):
			t.Fatal("timeout")
		}
		if s != s2 {
			t.Fatal("got incorrect stream", s.Conn(), s2.Conn())
		}
	}

	// 汇集 Stream ???
	streams := make(chan inet.Stream)
	for _, s := range swarms {
		s.SetStreamHandler(func(s inet.Stream) {
			streams <- s
			s.Reset()
		})
	}

	// open a sreams in each conn
	for i, s := range swarms {
		for _, c := range s.Conns() {
			_, n2, _ := complement(c)
			st1, err := c.NewStream()
			if err != nil {
				t.Error(err)
			} else {
				st1.Write([]byte("hello"))
				st1.Reset()
				testOCStream(notifiees[i], st1)
				st2 := <-streams
				testOCStream(n2, st2)
			}
		}
	}

	// 关闭连接
	for i, s := range swarms {
		n := notifiees[i]
		for _, c := range s.Conns() {
			_, n2, c2 := complement(c)
			c.Close()
			c2.Close()

			var c3, c4 inet.Conn
			select {
			case c3 = <-n.disconnected:
			case <-time.After(timeout):
				t.Fatal("c3 timeout")
			}
			if c != c3 {
				t.Fatal("got incorrect conn", c, c3)
			}

			select {
			case c4 = <-n2.disconnected:
			case <-time.After(timeout):
				t.Fatal("c4 timeout")
			}
			if c2 != c4 {
				t.Fatal("got incorrect conn.", c, c3)
			}
		}
	}
}

type netNotifiee struct {
	listen       chan ma.Multiaddr
	listenClose  chan ma.Multiaddr
	connected    chan inet.Conn
	disconnected chan inet.Conn
	openedStream chan inet.Stream
	closedStream chan inet.Stream
}

func newNetNotifiee(buffer int) *netNotifiee {
	return &netNotifiee{
		listen:       make(chan ma.Multiaddr, buffer),
		listenClose:  make(chan ma.Multiaddr, buffer),
		connected:    make(chan inet.Conn, buffer),
		disconnected: make(chan inet.Conn, buffer),
		openedStream: make(chan inet.Stream, buffer),
		closedStream: make(chan inet.Stream, buffer),
	}
}

func (nn *netNotifiee) Listen(n inet.Network, a ma.Multiaddr) {
	fmt.Println("swarm Listen")
	nn.listen <- a
}

func (nn *netNotifiee) ListenClose(n inet.Network, a ma.Multiaddr) {
	fmt.Println("swarm Listen Close")
	nn.listenClose <- a
}

func (nn *netNotifiee) Connected(n inet.Network, v inet.Conn) {
	fmt.Println("swarm Connected")
	nn.connected <- v
}

func (nn *netNotifiee) Disconnected(n inet.Network, v inet.Conn) {
	fmt.Println("swarm Disconnected")
	nn.disconnected <- v
}

func (nn *netNotifiee) OpenedStream(n inet.Network, v inet.Stream) {
	fmt.Println("swarm Open Stream")
	nn.openedStream <- v
}

func (nn *netNotifiee) ClosedStream(n inet.Network, v inet.Stream) {
	fmt.Println("swarm Close Stream")
	nn.closedStream <- v
}
