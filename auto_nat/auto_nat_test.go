package test_auto_nat

import (
	"context"
	"testing"
	"time"

	swarmt "github.com/czh0526/libp2p/swarm"
	ggio "github.com/gogo/protobuf/io"
	autonat "github.com/libp2p/go-libp2p-autonat"
	pb "github.com/libp2p/go-libp2p-autonat/pb"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	ma "github.com/multiformats/go-multiaddr"
)

func init() {
	autonat.AutoNATBootDelay = 1 * time.Second
	autonat.AutoNATRefreshInterval = 1 * time.Second
	autonat.AutoNATRetryInterval = 1 * time.Second
	autonat.AutoNATIdentifyDelay = 100 * time.Millisecond
}

// 构建被动响应方
func makeAutoNATServicePrivate(ctx context.Context, t *testing.T) host.Host {
	h, err := bhost.NewHost(ctx, swarmt.GenSwarm(t, ctx), &bhost.HostOpts{})
	if err != nil {
		t.Fatalf("create BasicHost error: %s", err)
	}
	h.SetStreamHandler(autonat.AutoNATProto, sayAutoNATPrivate)
	return h
}

func sayAutoNATPrivate(s inet.Stream) {
	defer s.Close()
	w := ggio.NewDelimitedWriter(s)
	res := pb.Message{
		Type:         pb.Message_DIAL_RESPONSE.Enum(),
		DialResponse: newDialResponseError(pb.Message_E_DIAL_ERROR, "no diabable address"),
	}
	w.WriteMsg(&res)
}

// 构建被动响应方
func makeAutoNATServicePublic(ctx context.Context, t *testing.T) host.Host {
	h, err := bhost.NewHost(ctx, swarmt.GenSwarm(t, ctx), &bhost.HostOpts{})
	if err != nil {
		t.Fatalf("Create BasicHost error: %s", err)
	}
	h.SetStreamHandler(autonat.AutoNATProto, sayAutoNATPublic)
	return h
}

func sayAutoNATPublic(s inet.Stream) {
	defer s.Close()
	w := ggio.NewDelimitedWriter(s)
	res := pb.Message{
		Type:         pb.Message_DIAL_RESPONSE.Enum(),
		DialResponse: newDialResponseOK(s.Conn().RemoteMultiaddr()),
	}
	w.WriteMsg(&res)
}

func newDialResponseOK(addr ma.Multiaddr) *pb.Message_DialResponse {
	dr := new(pb.Message_DialResponse)
	dr.Status = pb.Message_OK.Enum()
	dr.Addr = addr.Bytes()
	return dr
}

func newDialResponseError(status pb.Message_ResponseStatus, text string) *pb.Message_DialResponse {
	dr := new(pb.Message_DialResponse)
	dr.Status = status.Enum()
	dr.StatusText = &text
	return dr
}

// 构建主动连接方
func makeAutoNAT(ctx context.Context, t *testing.T, ash host.Host) (host.Host, autonat.AutoNAT) {
	// 构建 AutoNat 实例
	h, err := bhost.NewHost(ctx, swarmt.GenSwarm(t, ctx), &bhost.HostOpts{})
	if err != nil {
		t.Fatalf("Create BasicHost error: %s", err)
	}
	a := autonat.NewAutoNAT(ctx, h, nil)

	// 修改 peers 列表，影响 DialBack() 过程
	a.(*autonat.AmbientAutoNAT).Mx().Lock()
	a.(*autonat.AmbientAutoNAT).Peers()[ash.ID()] = ash.Addrs()
	a.(*autonat.AmbientAutoNAT).Mx().Unlock()
	return h, a
}

func connect(t *testing.T, a, b host.Host) {
	pinfo := pstore.PeerInfo{ID: a.ID(), Addrs: a.Addrs()}
	err := b.Connect(context.Background(), pinfo)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAutoNATPrivate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 构建相应端
	hs := makeAutoNATServicePrivate(ctx, t)
	// 构建拨号端
	hc, an := makeAutoNAT(ctx, t, hs)

	status := an.Status()
	if status != autonat.NATStatusUnknown {
		t.Fatalf("unexpected NAT status: %d", status)
	}

	// 连接
	connect(t, hs, hc)
	time.Sleep(2 * time.Second)

	status = an.Status()
	if status != autonat.NATStatusPrivate {
		t.Fatalf("unexpected NAT status: %d", status)
	}
}
