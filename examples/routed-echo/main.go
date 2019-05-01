package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"io/ioutil"

	mrand "math/rand"

	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	net "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	ma "github.com/multiformats/go-multiaddr"
)

func makeRoutedHost(listenPort int, randseed int64, bootstrapPeers []pstore.PeerInfo, globalFlag string) (host.Host, error) {
	// 构建随机数流
	var r io.Reader
	if randseed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(randseed))
	}

	// 构建密钥对
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.ECDSA, 2048, r)
	if err != nil {
		panic(err)
	}

	// 构建配置参数
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/00.0.0.0/tcp/%d", listenPort)),
		libp2p.Identity(priv),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
		libp2p.NATPortMap(),
	}

	fmt.Println("构建 BasicHost")
	ctx := context.Background()
	basicHost, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	fmt.Println("构建 IpfsDHT( 实现 IpfsRouting 接口 )")
	dstore := dsync.MutexWrap(ds.NewMapDatastore())
	dht := dht.NewDHT(ctx, basicHost, dstore)

	fmt.Println("BasicHost + IpfsRouting => RoutedHost")
	routedHost := rhost.Wrap(basicHost, dht)

	// 让 routedHost 连接 bootstrapPeers 节点
	err = bootstrapConnect(ctx, routedHost, bootstrapPeers)
	if err != nil {
		return nil, err
	}

	// 打印信息
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", routedHost.ID().Pretty()))
	addrs := routedHost.Addrs()
	fmt.Println("I can reached at: ")
	for _, addr := range addrs {
		fmt.Println(addr.Encapsulate(hostAddr))
	}

	return routedHost, nil
}

func main() {
	// 解析命令行参数
	listenF := flag.Int("l", 0, "wait for incoming connections")
	target := flag.String("d", "", "target peer to dial")
	seed := flag.Int64("seed", 0, "set random seed for id generation")
	global := flag.Bool("global", false, "use global ipfs peers for bootstrapping")
	flag.Parse()

	if *listenF == 0 {
		panic("Please provide a port to bind on with -l")
	}

	// 设置 bootstrap 节点集合
	var bootstrapPeers []pstore.PeerInfo
	var globalFlag string
	if *global {
		fmt.Println("using global bootstrap")
		bootstrapPeers = IPFS_PEERS
		globalFlag = " -global"
	} else {
		fmt.Println("using local bootstrap")
		bootstrapPeers = getLocalPeerInfo()
		globalFlag = ""
	}

	// 构建 RoutedHost
	ha, err := makeRoutedHost(*listenF, *seed, bootstrapPeers, globalFlag)
	if err != nil {
		panic(err)
	}

	// 为 RoutedHost 设置 StreamHandler
	ha.SetStreamHandler("/echo/1.0.0", func(s net.Stream) {
		fmt.Println("Got a new stram!")
		if err := doEcho(s); err != nil {
			panic(err)
			s.Reset()
		} else {
			s.Close()
		}
	})

	if *target == "" {
		fmt.Println("listening for connections")
		select {}
	}

	peerid, err := peer.IDB58Decode(*target)
	if err != nil {
		panic(err)
	}

	// 通过 ID 连接远程的 Peer
	fmt.Println("opening stream")
	s, err := ha.NewStream(context.Background(), peerid, "/echo/1.0.0")
	if err != nil {
		panic(err)
	}

	// 向 Stream 写内容
	_, err = s.Write([]byte("Hello, world! \n"))
	if err != nil {
		panic(err)
	}

	// 从 Stream 读内容
	out, err := ioutil.ReadAll(s)
	if err != nil {
		panic(err)
	}

	fmt.Printf("read reply: %q \n", out)
}

func doEcho(s net.Stream) error {
	buf := bufio.NewReader(s)
	str, err := buf.ReadString('\n')
	if err != nil {
		return err
	}
	fmt.Printf("read data <== %s\n", str)

	_, err = s.Write([]byte(str))
	fmt.Printf("write data ==> %s\n", str)
	return err
}
