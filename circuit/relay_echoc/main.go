package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	security "github.com/libp2p/go-conn-security"
	csms "github.com/libp2p/go-conn-security-multistream"
	"github.com/libp2p/go-conn-security/insecure"
	circuit "github.com/libp2p/go-libp2p-circuit"
	crypto "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	pstoremem "github.com/libp2p/go-libp2p-peerstore/pstoremem"
	swrm "github.com/libp2p/go-libp2p-swarm"
	transport "github.com/libp2p/go-libp2p-transport"
	tptu "github.com/libp2p/go-libp2p-transport-upgrader"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	mux "github.com/libp2p/go-stream-muxer"
	tcp "github.com/libp2p/go-tcp-transport"
	ws "github.com/libp2p/go-ws-transport"
	ma "github.com/multiformats/go-multiaddr"
	mplex "github.com/whyrusleeping/go-smux-multiplex"
	msmux "github.com/whyrusleeping/go-smux-multistream"
	yamux "github.com/whyrusleeping/go-smux-yamux"
)

const Proto = "/relay/test/echo"

func init() {
	//ipfslog.SetAllLoggers(logging.DEBUG)
	//ipfslog.SetLogLevel("addrutil", "ERROR")
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <echod-address> <msg> \n", os.Args[0])
		os.Exit(1)
	}

	// 分析 relay 节点的地址
	paddr, err := ma.NewMultiaddr(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("paddr = %s \n", paddr)

	// 分析 relay 节点的 peer info
	pinfo, err := pstore.InfoFromP2pAddr(paddr)
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

	// 构建 swarm
	ctx := context.Background()
	swarm := swrm.NewSwarm(ctx, id, peerstore, nil)

	// 构建 Upgrader, 两类 Transport
	upgrader := new(tptu.Upgrader)
	upgrader.Secure = makeInsecureTransport(id) // 利用 peer.ID 构建加密的 Transport
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

	// 增加 Relay Transport
	host := bhost.New(swarm)
	err = circuit.AddRelayTransport(ctx, host, upgrader, circuit.OptHop)

	rctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	err = host.Connect(
		rctx,
		*pinfo)
	if err != nil {
		log.Fatal(fmt.Sprintf("host.Connect() error: %s", err))
	}
	for i := 0; i < 5; i++ {
		time.Sleep(time.Second)
		fmt.Println("...")
	}

	fmt.Printf("host.NewStream ==> %s(%s) \n", pinfo.ID, Proto)
	s, err := host.NewStream(rctx, pinfo.ID, Proto)
	if err != nil {
		log.Fatal(fmt.Sprintf("host.NewStream() error: %s", err))
	}

	msg := []byte(os.Args[2])
	s.Write(msg)

	data := make([]byte, len(msg))
	_, err = s.Read(data)
	if err != nil && err != io.EOF {
		log.Fatal(err)
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
