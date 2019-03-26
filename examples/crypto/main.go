package main

import (
	"crypto/rand"
	"fmt"

	crypto "github.com/libp2p/go-libp2p-crypto"
)

func main() {
	privKey, _, err := crypto.GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		panic(err)
	}

	keyBytes, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		panic(err)
	}
	fmt.Printf("key = %x \n", keyBytes)

	privKey2, err := crypto.UnmarshalPrivateKey(keyBytes)
	if err != nil {
		panic(err)
	}

	if privKey2.Equals(privKey) {
		fmt.Println("Marshal & Unmarshal is OK !")
	} else {
		fmt.Println("Marshal & Unmarshal is bad !")
	}
}
