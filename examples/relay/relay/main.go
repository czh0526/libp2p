package main

import (
	"context"
	"fmt"

	ipfslog "github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	ma "github.com/multiformats/go-multiaddr"
	logging "github.com/whyrusleeping/go-logging"
)

func init() {
	ipfslog.SetAllLoggers(logging.DEBUG)
	ipfslog.SetLogLevel("addrutil", "ERROR")
}

func main() {
	h2, err := libp2p.New(context.Background(), libp2p.EnableRelay(circuit.OptHop))
	if err != nil {
		panic(err)
	}

	var relayAddr ma.Multiaddr
	addrs, err := h2.Network().InterfaceListenAddresses()
	for _, addr := range addrs {
		fmt.Printf("\t%s/ipfs/%s \n", addr, h2.ID())
		ip, err := addr.ValueForProtocol(ma.P_IP4)
		if err == nil && ip == "127.0.0.1" {
			relayAddr = addr
		}
	}

	fmt.Println()
	fmt.Println("在目标节点的终端执行下列命令 ==> ")

	fmt.Printf("\tgo run main.go --relay_addr %s/ipfs/%s \n", relayAddr, h2.ID())
	select {}
}
