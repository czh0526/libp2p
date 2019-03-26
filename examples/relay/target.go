package main

import (
	"context"
	"encoding/hex"
	"fmt"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
)

func makeTargetNode(privKeyStr string) (*host.Host, error) {

	// 构建 private key
	privBytes, err := hex.DecodeString(privKeyStr)
	if err != nil {
		return nil, err
	}
	privKey, err := crypto.UnmarshalPrivateKey(privBytes)
	if err != nil {
		return nil, err
	}

	// 在 13002 端口上构建 Host
	h3, err := libp2p.New(context.Background(),
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/13002"),
		libp2p.EnableRelay())
	if err != nil {
		return nil, err
	}
	fmt.Printf("Target Node ==> %s \n", h3.ID().Pretty())
	for _, addr := range h3.Addrs() {
		fmt.Printf("\t %s \n", addr)
	}
	fmt.Println()

	// Host 3 设置 StreamHandler
	h3.SetStreamHandler("/cats", func(s inet.Stream) {
		fmt.Println("Stream handler on target node is worked!")
		s.Close()
	})
	fmt.Println("SetStreamHandler ==> /cats")

	return &h3, nil
}
