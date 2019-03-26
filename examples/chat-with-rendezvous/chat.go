package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sync"

	libp2p "github.com/libp2p/go-libp2p"
	discovery "github.com/libp2p/go-libp2p-discovery"
	libp2pdht "github.com/libp2p/go-libp2p-kad-dht"
	inet "github.com/libp2p/go-libp2p-net"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	protocol "github.com/libp2p/go-libp2p-protocol"
	multiaddr "github.com/multiformats/go-multiaddr"
)

func handleStream(stream inet.Stream) {
	fmt.Println("Got a new stream!")

	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	go readData(rw)
	go writeData(rw)
}

func readData(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from buffer")
			panic(err)
		}

		if str == "" {
			return
		}

		if str != "\n" {
			fmt.Printf("\x1b[32m%s\x1b[0m> ", str)
		}
	}
}

func writeData(rw *bufio.ReadWriter) {
	stdReader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("> ")
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from stdin")
			panic(err)
		}

		_, err = rw.WriteString(fmt.Sprintf("%s\n", sendData))
		if err != nil {
			fmt.Println("Error writing to buffer")
			panic(err)
		}

		err = rw.Flush()
		if err != nil {
			fmt.Println("Error flushing buffer")
			panic(err)
		}
	}
}

func main() {
	// 解析命令行参数
	config, err := ParseFlags()
	if err != nil {
		panic(err)
	}

	// 构建主机对象
	ctx := context.Background()
	host, err := libp2p.New(ctx, libp2p.ListenAddrs([]multiaddr.Multiaddr(config.ListenAddresses)...))
	if err != nil {
		panic(err)
	}
	fmt.Printf("Host created. We are: %s \n", host.ID())
	fmt.Println(host.Addrs())

	// 设置 StreamHandler
	host.SetStreamHandler(protocol.ID(config.ProtocolID), handleStream)

	// 构建 DHT 对象
	kademliaDHT, err := libp2pdht.New(ctx, host)
	if err != nil {
		panic(err)
	}

	// 启动 DHT 客户端
	fmt.Println("Bootstrapping the DHT")
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	// 连接 Bootstrap 节点
	var wg sync.WaitGroup
	for _, peerAddr := range config.BootstrapPeers {
		peerinfo, _ := peerstore.InfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := host.Connect(ctx, *peerinfo); err != nil {
				fmt.Printf("[error]: %s \n", err)
			} else {
				fmt.Printf("Connection established with bootstrap node: %s \n", *peerinfo)
			}
		}()
	}
	wg.Wait()

	// 宣布自己上线了，并宣布自己作为 RendezvousString 的内容提供者
	fmt.Println("Announcing ourselves...")
	routingDiscovery := discovery.NewRoutingDiscovery(kademliaDHT)
	discovery.Advertise(ctx, routingDiscovery, config.RendezvousString)
	fmt.Println("Successfully announced!")

	// 查找 RendezvousString 的其它内容提供者
	fmt.Println("Searching for other peers ...")
	peerChan, err := routingDiscovery.FindPeers(ctx, config.RendezvousString)
	if err != nil {
		panic(err)
	}

	// 逐个与其他内容提供者建立通讯流
	for peer := range peerChan {
		if peer.ID == host.ID() {
			continue
		}

		fmt.Printf("Found peer: %s \n", peer)
		fmt.Printf("Connecting to: %s \n", peer)
		stream, err := host.NewStream(ctx, peer.ID, protocol.ID(config.ProtocolID))

		if err != nil {
			fmt.Printf("Connect failed: %s \n", err)
			continue
		}

		rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

		go writeData(rw)
		go readData(rw)

		fmt.Printf("Connected to: %s \n", peer)
	}

	select {}
}
