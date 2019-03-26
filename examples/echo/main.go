package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"

	libp2p "github.com/libp2p/go-libp2p"

	mrand "math/rand"

	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	net "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

func makeBasicHost(listenPort int, insecure bool, randseed int64) (host.Host, error) {
	// 构建 random reader
	var r io.Reader
	if randseed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(randseed))
	}

	// 构建私钥
	priv, _, err := crypto.GenerateKeyPairWithReader(
		crypto.ECDSA, 2048, r)
	if err != nil {
		return nil, err
	}

	// 构建必要的处理 Config 对象的 Option 函数集合
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
	}
	if insecure {
		opts = append(opts, libp2p.NoSecurity)
	}

	// 构建 Host 对象
	basicHost, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	// 获取 Host 的 multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", basicHost.ID().Pretty()))
	fmt.Printf("hostAddr = %s \n", hostAddr)
	addr := basicHost.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr)
	fmt.Printf("I am %s \n", fullAddr)
	if insecure {
		fmt.Printf("Now run \",/echo -l %d -d %s\" on a different terminal\n", listenPort+1, fullAddr)
	} else {
		fmt.Printf("Now run \"./echo -l %d -d %s\" on a different terminal\n", listenPort+1, fullAddr)
	}

	return basicHost, nil
}

func main() {
	// 解析命令行参数
	listenF := flag.Int("l", 0, "wait for incoming connections")
	target := flag.String("d", "", "target peer to dial")
	insecure := flag.Bool("insecure", false, "use an unencrypted connection")
	seed := flag.Int64("seed", 0, "set random seed for id generation")
	flag.Parse()

	if *listenF == 0 {
		panic("Please provide a port to bind on with -l")
	}

	// 构建 BasicHost
	ha, err := makeBasicHost(*listenF, *insecure, *seed)
	if err != nil {
		panic(err)
	}

	// 设置协议流处理器，被动等待连接进来
	ha.SetStreamHandler("/echo/1.0.0", func(s net.Stream) {
		fmt.Println("Got a new stream!")
		if err := doEcho(s); err != nil {
			log.Println(err)
			s.Reset()
		} else {
			s.Close()
		}
	})

	if *target == "" {
		fmt.Println("listening for connections.")
		select {}
	}

	// string ==> multiaddr
	ipfsaddr, err := ma.NewMultiaddr(*target)
	if err != nil {
		panic(err)
	}

	// 提取 ipfs 的 pid
	pid, err := ipfsaddr.ValueForProtocol(ma.P_IPFS)
	if err != nil {
		panic(err)
	}

	// Base58 解码 pid
	peerid, err := peer.IDB58Decode(pid)
	if err != nil {
		panic(err)
	}

	// 将 pid => targetAddr 加入本地节点库
	targetPeerAddr, _ := ma.NewMultiaddr(
		fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)))
	targetAddr := ipfsaddr.Decapsulate(targetPeerAddr)
	ha.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)

	// 主动连接 Target
	fmt.Println("opening stream")
	s, err := ha.NewStream(context.Background(), peerid, "/echo/1.0.0")
	if err != nil {
		panic(err)
	}

	// 发送字符串
	_, err = s.Write([]byte("Hello, world!\n"))
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("send data ==> Hello, world!\n")

	// 读取字符串
	out, err := ioutil.ReadAll(s)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("reply data ==> %q\n", out)
}

func doEcho(s net.Stream) error {
	buf := bufio.NewReader(s)
	str, err := buf.ReadString('\n')
	if err != nil {
		return err
	}
	fmt.Printf("read data <== %s \n", str)

	_, err = s.Write([]byte(str))
	fmt.Printf("reply data <== %s \n", str)
	return err
}
