package main

import (
	"context"
	"encoding/hex"
	"fmt"

	libp2p "github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
)

func makeRelayNode(privKeyStr string) (*host.Host, error) {

	// 构建 private key
	privKeyBytes, err := hex.DecodeString(privKeyStr)
	if err != nil {
		return nil, err
	}
	privKey, err := crypto.UnmarshalPrivateKey(privKeyBytes)
	if err != nil {
		return nil, err
	}

	// 在 13002 端口上构建 Host
	h2, err := libp2p.New(context.Background(),
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/13002"),
		libp2p.Identity(privKey),
		libp2p.EnableRelay(circuit.OptHop))
	if err != nil {
		return nil, err
	}
	fmt.Printf("relay Node ==> %s \n", h2.ID().Pretty())
	for _, addr := range h2.Addrs() {
		fmt.Printf("\t %s \n", addr)
	}
	fmt.Println()

	// Host 3 设置 StreamHandler
	h2.SetStreamHandler("/cats", func(s inet.Stream) {
		fmt.Println("Stream handler on relay node is worked!")
		s.Close()
	})
	fmt.Println("SetStreamHandler ==> /cats")

	return &h2, nil
}
