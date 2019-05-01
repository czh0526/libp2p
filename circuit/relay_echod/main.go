package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	ipfslog "github.com/ipfs/go-log"
	security "github.com/libp2p/go-conn-security"
	csms "github.com/libp2p/go-conn-security-multistream"
	"github.com/libp2p/go-conn-security/insecure"
	circuit "github.com/libp2p/go-libp2p-circuit"
	crypto "github.com/libp2p/go-libp2p-crypto"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	pstoremem "github.com/libp2p/go-libp2p-peerstore/pstoremem"
	swarm "github.com/libp2p/go-libp2p-swarm"
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

const Proto = "/relay/test/echo"

func init() {
	ipfslog.SetAllLoggers(logging.WARNING)
	ipfslog.SetLogLevel("net/identify", "DEBUG")
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <relay-address> \n", os.Args[0])
		os.Exit(1)
	}

	// 分析 relay 节点的地址
	raddr, err := ma.NewMultiaddr(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	/*
		fmt.Printf("raddr = %s \n", raddr)
		protos := raddr.Protocols()
		for _, proto := range protos {
			value, _ := raddr.ValueForProtocol(proto.Code)
			fmt.Printf("%s = %s \n", proto.Name, value)
		}
	*/

	// 分析 relay 节点的 peer info
	rinfo, err := pstore.InfoFromP2pAddr(raddr)
	if err != nil {
		log.Fatal(err)
	}

	// 构建本地的 websocket 地址
	wsaddr, err := ma.NewMultiaddr("/ip4/0.0.0.0/tcp/0/ws")
	if err != nil {
		log.Fatal(err)
	}

	// 构建证书
	privKey, pubKey, err := crypto.GenerateKeyPair(crypto.ECDSA, 2048)
	if err != nil {
		log.Fatal(err)
	}

	// 构建 ID
	id, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		log.Fatal(err)
	}

	// 构建 Peerstore
	peerstore := pstoremem.NewPeerstore()
	peerstore.AddPrivKey(id, privKey)
	peerstore.AddPubKey(id, pubKey)

	// 构建 Swarm
	ctx := context.Background()
	swarm := swarm.NewSwarm(ctx, id, peerstore, nil)

	// 构建 Upgrader
	upgrader := new(tptu.Upgrader)
	upgrader.Secure = makeInsecureTransport(id)
	upgrader.Muxer, err = makeMuxer()
	if err != nil {
		log.Fatal(err)
	}

	// 构建各类 transport
	tpts, err := makeTransports(upgrader)
	if err != nil {
		swarm.Close()
		log.Fatal(err)
	}
	// 关联 transport & swarm
	for _, t := range tpts {
		err = swarm.AddTransport(t)
		if err != nil {
			swarm.Close()
			log.Fatal(err)
		}
	}

	// 监听
	swarm.AddListenAddr(wsaddr)
	interfaceListenAddr, err := swarm.InterfaceListenAddresses()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("swarm %s listen on: %s \n", id.Pretty(), interfaceListenAddr)

	// 构建 Host
	host, err := bhost.NewHost(ctx, swarm, &bhost.HostOpts{})
	if err != nil {
		log.Fatal(err)
	}

	// 增加 Relay Transport
	err = circuit.AddRelayTransport(ctx, host, upgrader)
	if err != nil {
		log.Fatal(err)
	}

	// 设置 Stream Handler
	host.SetStreamHandler(Proto, handleStream)

	// 连接 relay 节点
	rctx, cancel := context.WithTimeout(ctx, time.Second)
	err = host.Connect(rctx, *rinfo)
	if err != nil {
		log.Fatal(err)
	}
	cancel()

	fmt.Printf("Listening at /p2p-circuit/ipfs/%s\n", id.Pretty())
	fmt.Printf("host.addrs = %s \n", host.AllAddrs())
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

func handleStream(s inet.Stream) {
	fmt.Printf("New echo stream from %s \n", s.Conn().RemoteMultiaddr().String())
	defer s.Close()
	count, err := io.Copy(s, s)
	if err != nil {
		fmt.Printf("Error echoing: %s", err.Error())
	}
	fmt.Printf("echoed %d bytes", count)
}
