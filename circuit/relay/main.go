package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	ipfslog "github.com/ipfs/go-log"
	security "github.com/libp2p/go-conn-security"
	csms "github.com/libp2p/go-conn-security-multistream"
	insecure "github.com/libp2p/go-conn-security/insecure"
	circuit "github.com/libp2p/go-libp2p-circuit"
	crypto "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	pstoremem "github.com/libp2p/go-libp2p-peerstore/pstoremem"
	swrm "github.com/libp2p/go-libp2p-swarm"
	transport "github.com/libp2p/go-libp2p-transport"
	tptu "github.com/libp2p/go-libp2p-transport-upgrader"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	mux "github.com/libp2p/go-stream-muxer"
	tcp "github.com/libp2p/go-tcp-transport"
	ws "github.com/libp2p/go-ws-transport"
	ma "github.com/multiformats/go-multiaddr"
	logging "github.com/whyrusleeping/go-logging"
	mplex "github.com/whyrusleeping/go-smux-multiplex"
	msmux "github.com/whyrusleeping/go-smux-multistream"
	yamux "github.com/whyrusleeping/go-smux-yamux"
)

func init() {
	ipfslog.SetAllLoggers(logging.WARNING)
	ipfslog.SetLogLevel("net/identify", "DEBUG")
}

func main() {
	port := flag.Int("l", 9001, "Relay TCP listen port")
	wsport := flag.Int("ws", 9002, "Relay WS listen port")
	flag.Parse()

	ip4addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port))
	if err != nil {
		log.Fatal(err)
	}

	ip6addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip6/::/tcp/%d", *port))
	if err != nil {
		log.Fatal(err)
	}

	wsaddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d/ws", *wsport))
	if err != nil {
		log.Fatal(err)
	}

	privKey, pubKey, err := crypto.GenerateKeyPair(crypto.ECDSA, 2048)
	if err != nil {
		log.Fatal(err)
	}

	id, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		log.Fatal(err)
	}

	peerstore := pstoremem.NewPeerstore()
	peerstore.AddPrivKey(id, privKey)
	peerstore.AddPubKey(id, pubKey)

	ctx := context.Background()
	swarm := swrm.NewSwarm(ctx, id, peerstore, nil)
	host, err := bhost.NewHost(ctx, swarm, &bhost.HostOpts{})
	if err != nil {
		swarm.Close()
		log.Fatal(err)
	}

	upgrader := new(tptu.Upgrader)
	upgrader.Secure = makeInsecureTransport(id)
	upgrader.Muxer, err = makeMuxer()
	if err != nil {
		log.Fatal(err)
	}
	tpts, err := makeTransports(upgrader)
	if err != nil {
		swarm.Close()
		log.Fatal(err)
	}
	for _, t := range tpts {
		err = swarm.AddTransport(t)
		if err != nil {
			swarm.Close()
			log.Fatal(err)
		}
	}
	swarm.AddListenAddr(ip4addr)
	swarm.AddListenAddr(ip6addr)
	swarm.AddListenAddr(wsaddr)

	err = circuit.AddRelayTransport(ctx, host, upgrader, circuit.OptHop)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Relay addresses: \n")
	for _, addr := range host.Addrs() {
		_, err := addr.ValueForProtocol(circuit.P_CIRCUIT)
		if err == nil {
			continue
		}
		fmt.Printf("%s/ipfs/%s\n", addr.String(), id.Pretty())
	}

	select {}
}

func makeInsecureTransport(id peer.ID) security.Transport {
	secMuxer := new(csms.SSMuxer)
	secMuxer.AddTransport(insecure.ID, insecure.New(id))
	return secMuxer
}

func makeMuxer() (mux.Transport, error) {
	muxMuxer := msmux.NewBlankTransport()
	muxMuxer.AddTransport("/yamux/1.0.0", yamux.DefaultTransport)
	muxMuxer.AddTransport("/mplex/6.7.0", mplex.DefaultTransport)
	return muxMuxer, nil
}

func makeTransports(upgrader *tptu.Upgrader) ([]transport.Transport, error) {
	transports := make([]transport.Transport, 2)
	transports[0] = tcp.NewTCPTransport(upgrader)
	transports[1] = ws.New(upgrader)
	return transports, nil
}
