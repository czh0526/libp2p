package main

import (
	"context"
	"flag"
	"fmt"

	ipfslog "github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	inet "github.com/libp2p/go-libp2p-net"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	logging "github.com/whyrusleeping/go-logging"
)

func init() {
	ipfslog.SetAllLoggers(logging.DEBUG)
	ipfslog.SetLogLevel("addrutil", "ERROR")
}

func main() {
	var relayAddrString string
	flag.StringVar(&relayAddrString, "relay_addr", "", "this is relay node address")
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

	// 构建 dest host
	h3, err := libp2p.New(context.Background(), libp2p.ListenAddrs(), libp2p.EnableRelay())
	if err != nil {
		panic(fmt.Sprintf("libp2p.New() error: %s", err))
	}

	// src node --> relay node <-- h3
	if err := h3.Connect(context.Background(), *relayInfo); err != nil {
		panic(fmt.Sprintf("Connect() error: %s", err))
	}

	// set dest stream handler
	h3.SetStreamHandler("/cats", func(s inet.Stream) {
		fmt.Println("Meow! It worked!")
		s.Close()
	})

	relayPath, err := ma.NewMultiaddr("/p2p-circuit/ipfs/" + h3.ID().Pretty())
	if err != nil {
		panic(fmt.Sprintf("NewMultiaddr() error: %s", err))
	}

	fmt.Printf("peer id = %s \n", h3.ID())
	fmt.Printf("relay path = %s \n", relayPath)

	fmt.Println()
	fmt.Println("在源节点终端执行下列命令 ==> ")
	fmt.Printf("\tgo run main.go --relay_addr %s --dest_id %s \n", relayAddr, h3.ID())
	select {}
}
