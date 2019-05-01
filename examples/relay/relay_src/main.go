package main

import (
	"context"
	"flag"
	"fmt"

	peer "github.com/libp2p/go-libp2p-peer"

	ipfslog "github.com/ipfs/go-log"
	swarm "github.com/libp2p/go-libp2p-swarm"
	logging "github.com/whyrusleeping/go-logging"

	libp2p "github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

func init() {
	ipfslog.SetAllLoggers(logging.DEBUG)
	ipfslog.SetLogLevel("addrutil", "ERROR")
}

func main() {
	var relayAddrString string
	var destIDString string
	flag.StringVar(&relayAddrString, "relay_addr", "", "this is relay node address")
	flag.StringVar(&destIDString, "dest_id", "", "this is target node id")
	flag.Parse()

	// 处理 relay address
	relayAddr, err := ma.NewMultiaddr(relayAddrString)
	if err != nil {
		panic(fmt.Sprintf("newMultiaddr() error: %s ", err))
	}

	relayInfo, err := pstore.InfoFromP2pAddr(relayAddr)
	if err != nil {
		panic(fmt.Sprintf("InfoFromP2pAddr() error: %s ", err))
	}

	// 处理 dest id
	destID, err := peer.IDB58Decode(destIDString)
	if err != nil {
		panic(fmt.Sprintf("IDB58Decode() error: %s ", err))
	}

	// 构建 src host
	h1, err := libp2p.New(context.Background(), libp2p.EnableRelay(circuit.OptDiscovery))
	if err != nil {
		panic(err)
	}

	// h1 --> relay node <-- dest node
	if err := h1.Connect(context.Background(), *relayInfo); err != nil {
		panic(fmt.Sprintf("Connect() error: %s ", err))
	}

	// 确保 h1 --> dest node 无直接通路
	_, err = h1.NewStream(context.Background(), destID, "/cats")
	if err == nil {
		fmt.Println("Didn't actually expect to get a stream here. What happened?")
		return
	}
	fmt.Println("Okay, no connection from SRC to TARGET: ", err)
	fmt.Println("Just as we suspected")

	// 使用 Relay 通道传输数据
	h1.Network().(*swarm.Swarm).Backoff().Clear(destID)

	circuitAddrString := fmt.Sprintf("%s/p2p-circuit/ipfs/%s", relayAddr, destID)
	circuitAddr, err := ma.NewMultiaddr(circuitAddrString)
	if err != nil {
		panic(fmt.Sprintf("NewMultiaddr('%s') error: %s", circuitAddrString, err))
	}
	circuitInfo := pstore.PeerInfo{
		ID:    destID,
		Addrs: []ma.Multiaddr{circuitAddr},
	}

	if err := h1.Connect(context.Background(), circuitInfo); err != nil {
		panic(fmt.Sprintf("Connect() error: %s", err))
	}

	s, err := h1.NewStream(context.Background(), destID, "/cats")
	if err != nil {
		fmt.Println("huh, this should have worked: ", err)
		return
	}

	s.Read(make([]byte, 1))
	select {}
}
