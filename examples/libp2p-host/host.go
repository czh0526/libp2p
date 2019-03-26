package main

import (
	"context"
	"crypto/rand"
	"fmt"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ec_priv, _, err := crypto.GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		panic(err)
	}

	h, err := libp2p.New(ctx,
		libp2p.Identity(ec_priv),
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/13002"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("Hello => my ECDSA hosts ID is %s \n", h.ID())

	ed_priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		panic(err)
	}

	h2, err := libp2p.New(ctx,
		libp2p.Identity(ed_priv),
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/13002"))
	if err != nil {
		panic(err)
	}

	fmt.Printf("Hello => my Ed25519 hosts ID is %s \n", h2.ID())
}
