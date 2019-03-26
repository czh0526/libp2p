package peer

import (
	"fmt"
	"io"
	"math/rand"
	"testing"
	"time"

	ci "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	b58 "github.com/mr-tron/base58/base58"
	mh "github.com/multiformats/go-multihash"
)

func TestRandPeerID(t *testing.T) {
	// 构建 Reader
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// 构建 buf
	buf := make([]byte, 16)
	// 填充 buf
	if _, err := io.ReadFull(r, buf); err != nil {
		t.Fatalf("read buf error: %s", err)
	}
	// 散列 buf
	h, _ := mh.Sum(buf, mh.SHA2_256, -1)
	fmt.Printf("hash = %s \n", h.String())
	// b58 编码散列数据
	fmt.Printf("hash base58 encode = %s \n", b58.Encode(h))
	// 构建 PeerID
	peerid := peer.ID(h)
	fmt.Printf("peer id \t   = %s \n", peerid)
}

func TestKeypair2PeerID(t *testing.T) {
	// 构建 Reader
	reader := rand.New(rand.NewSource(time.Now().UnixNano()))
	// 构建密钥对
	priv, pub, err := ci.GenerateKeyPairWithReader(ci.ECDSA, 512, reader)
	if err != nil {
		t.Fatalf("generate key pair error: %s", err)
	}

	// 公钥字节数组
	pubBytes, err := pub.Bytes()
	if err != nil {
		t.Fatalf("public key => bytes error: %s", err)
	}
	// 公钥取 hash
	hash, err := mh.Sum(pubBytes, mh.SHA2_256, -1)
	if err != nil {
		t.Fatalf("public key bytes => hash error: %s", err)
	}
	// 编码 hash 为 peer.ID
	peerid, err := peer.IDB58Decode(b58.Encode(hash))
	if err != nil {
		t.Fatalf("public key hash => peerId error: %s", err)
	}

	// 检验 public key 与 peer.ID 是否匹配
	if !peerid.MatchesPublicKey(pub) {
		t.Fatalf("peerid doesn't match with public key")
	}
	if !peerid.MatchesPrivateKey(priv) {
		t.Fatalf("peerid doesn't match with private key")
	}

	peerid2, err := peer.IDFromPublicKey(pub)
	if peerid.String() != peerid2.String() {
		t.Fatalf("peerid doesn't match with peerid2")
	}
}
