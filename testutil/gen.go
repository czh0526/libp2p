package testutil

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ci "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
)

var generatedPairs int64 = 0

var ZeroLocalTCPAddress ma.Multiaddr

func init() {
	maddr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0")
	if err != nil {
		panic(err)
	}
	ZeroLocalTCPAddress = maddr
}

type PeerNetParams struct {
	ID      peer.ID
	PrivKey ci.PrivKey
	PubKey  ci.PubKey
	Addr    ma.Multiaddr
}

func RandPeerNetParamsOrFatal(t *testing.T) PeerNetParams {
	p, err := RandPeerNetParams()
	if err != nil {
		t.Fatal(err)
		return PeerNetParams{}
	}

	return *p
}

func RandTestKeyPair(bits int) (ci.PrivKey, ci.PubKey, error) {
	seed := time.Now().UnixNano()

	seed += atomic.AddInt64(&generatedPairs, 1) << 32
	r := rand.New(rand.NewSource(seed))
	return ci.GenerateKeyPairWithReader(ci.ECDSA, bits, r)
}

var lastPort = struct {
	port int
	sync.Mutex
}{}

func RandLocalTCPAddress() ma.Multiaddr {
	lastPort.Lock()
	if lastPort.port == 0 {
		lastPort.port = 1000 + SeededRand.Intn(50000)
	}
	port := lastPort.port
	lastPort.port++
	lastPort.Unlock()

	addr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port)
	maddr, _ := ma.NewMultiaddr(addr)
	return maddr
}

func RandPeerNetParams() (*PeerNetParams, error) {
	var p PeerNetParams
	var err error
	p.Addr = ZeroLocalTCPAddress
	p.PrivKey, p.PubKey, err = RandTestKeyPair(1024)
	if err != nil {
		return nil, err
	}

	p.ID, err = peer.IDFromPublicKey(p.PubKey)
	if err != nil {
		return nil, err
	}
	if err := p.checkKeys(); err != nil {
		return nil, err
	}
	return &p, nil
}

func (p *PeerNetParams) checkKeys() error {
	if !p.ID.MatchesPrivateKey(p.PrivKey) {
		return errors.New("p.ID does not match p.PrivKey")
	}

	if !p.ID.MatchesPublicKey(p.PubKey) {
		return errors.New("p.ID does not match p.PubKey")
	}

	buf := new(bytes.Buffer)
	buf.Write([]byte("hello world. this is me, I swear."))
	b := buf.Bytes()

	sig, err := p.PrivKey.Sign(b)
	if err != nil {
		return fmt.Errorf("sig signing failed: %s", err)
	}

	sigok, err := p.PubKey.Verify(b, sig)
	if err != nil {
		return fmt.Errorf("sig verify failed: %s", err)
	}
	if !sigok {
		return fmt.Errorf("sig verify failed: sig invalid")
	}

	return nil // ok. move along.
}
