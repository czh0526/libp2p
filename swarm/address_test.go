package test_swarm

import (
	"context"
	"fmt"
	"testing"

	peerstoremem "github.com/libp2p/go-libp2p-peerstore/pstoremem"
	swarm "github.com/libp2p/go-libp2p-swarm"
	tcp "github.com/libp2p/go-tcp-transport"

	tu "github.com/czh0526/libp2p/testutil"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
)

func TestSwarmInterfaceAddress(t *testing.T) {
	// 构建公/私钥
	privKey, pubKey, err := tu.RandTestKeyPair(2048)
	// 根据密钥构建 peer.ID
	peerId, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		t.Fatalf("generate peer id error: %s ", err)
	}
	// 构建并填充 peerstore
	pstore := peerstoremem.NewPeerstore()
	pstore.AddPrivKey(peerId, privKey)
	pstore.AddPubKey(peerId, pubKey)

	// 构建 swarm 对象
	swarm := swarm.NewSwarm(context.Background(), peerId, pstore, nil)

	// 为 swarm 关联 Transport
	tcpTransport := tcp.NewTCPTransport(GenUpgrader(swarm))
	swarm.AddTransport(tcpTransport)

	// 构建监听地址
	listenAddr, err := ma.NewMultiaddr("/ip4/0.0.0.0/tcp/13002")
	if err != nil {
		t.Fatalf("parse listen address error: %s ", err)
	}

	// 校验是否有匹配的 Transport
	tpt := swarm.TransportForListening(listenAddr)
	if tpt == nil {
		t.Fatalf("query transport for listen, but got nil")
	} else {
		fmt.Printf("query transport for listen: %T \n", tpt)
	}

	// Swarm Listen
	if err := swarm.Listen(listenAddr); err != nil {
		t.Fatalf("swarm listen error: %s", err)
	}

	// 打印 Swarm 的 Listen Address
	fmt.Printf("swarm's listen addresses = %s \n", swarm.ListenAddresses())
	interfaceListenAddresses, err := swarm.InterfaceListenAddresses()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("swarm's interface listen addresses = %s \n", interfaceListenAddresses)
}
