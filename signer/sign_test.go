package test_sign

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"testing"

	btcec "github.com/btcsuite/btcd/btcec"
	libp2pcrypto "github.com/libp2p/go-libp2p-crypto"
	sha256 "github.com/minio/sha256-simd"
)

func TestLibp2pSign(t *testing.T) {
	message := []byte("test message for sign")

	testSign(t, libp2pcrypto.ECDSA, message)
	fmt.Println("--------------------------")
	fmt.Println("--------------------------")

	testSign(t, libp2pcrypto.Ed25519, message)
	fmt.Println("-------------------------")
	fmt.Println("-------------------------")

	testSign(t, libp2pcrypto.Secp256k1, message)
}

func testSign(t *testing.T, typ int, message []byte) {
	sk, _, err := libp2pcrypto.GenerateKeyPairWithReader(typ, 2048, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		s, err := sk.Sign(message)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("%d) %x \n", i, s)
	}
}

func TestP256Sign(t *testing.T) {
	sk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	message := []byte("test message for sign")
	hash := sha256.Sum256(message)
	for i := 0; i < 10; i++ {
		r, s, err := ecdsa.Sign(rand.Reader, sk, hash[:])
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("%d) %x - %x \n", i, r, s)
	}
}

func TestBtcS256Sign(t *testing.T) {
	sk, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		t.Fatal(err)
	}

	message := []byte("test message for sign")
	hash := sha256.Sum256(message)
	for i := 0; i < 10; i++ {
		signature, err := sk.Sign(hash[:])
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("%d) %x \n", i, signature.Serialize())
	}
}
