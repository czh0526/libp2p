package main

import (
	"context"
	"encoding/hex"
	"fmt"

	libp2p "github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
)

func makeSourceNode(privKeyStr string) (*host.Host, error) {

	// 构建私钥 privKey
	privKeyBytes, err := hex.DecodeString(privKeyStr)
	if err != nil {
		return nil, err
	}
	privKey, err := crypto.UnmarshalPrivateKey(privKeyBytes)
	if err != nil {
		panic(err)
	}

	// 使用私钥构建 Host
	h1, err := libp2p.New(context.Background(),
		libp2p.Identity(privKey),
		libp2p.EnableRelay(circuit.OptDiscovery))
	if err != nil {
		panic(err)
	}
	fmt.Printf("Source Node ==> %v \n", h1.ID().Pretty())
	for _, addr := range h1.Addrs() {
		fmt.Printf("\t %s \n", addr)
	}
	fmt.Println()

	return &h1, err
}
