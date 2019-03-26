package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"os"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	inet "github.com/libp2p/go-libp2p-net"
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
		// 逐行读取网络上的字符
		str, err := rw.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from buffer")
			panic(err)
		}

		if str == "" {
			return
		}

		// 输出信息到终端上
		if str != "\n" {
			fmt.Printf("\x1b[32m%s\x1b[0m>", str)
		}
	}
}

func writeData(rw *bufio.ReadWriter) {
	stdReader := bufio.NewReader(os.Stdin)
	for {
		// 打印提示符
		fmt.Print("> ")
		// 逐行读取终端上的输入信息
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from stdin")
			panic(err)
		}

		// 将信息发送到网络上
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
	cfg := parseFlags()
	fmt.Printf("[*] Listening on: %s with ports: %d\n", cfg.listenHost, cfg.listenPort)

	ctx := context.Background()
	r := rand.Reader

	// 生成私钥
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.ECDSA, 2048, r)
	if err != nil {
		panic(err)
	}

	// 生成源地址
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(
		fmt.Sprintf("/ip4/%s/tcp/%d", cfg.listenHost, cfg.listenPort))

	// 构建主机
	host, err := libp2p.New(ctx, libp2p.ListenAddrs(sourceMultiAddr), libp2p.Identity(prvKey))
	if err != nil {
		panic(err)
	}

	// 为主机设置 StreamHandler
	host.SetStreamHandler(protocol.ID(cfg.ProtocolID), handleStream)
	fmt.Printf("\n[*] Your Multiaddress Is: /ip4/%s/tcp/%v/p2p/%s\n", cfg.listenHost, cfg.listenPort, host.ID().Pretty())

	// 等待新的 PeerInfo 通知
	peerChan := initMDNS(ctx, host, cfg.RendezvousString)
	peer := <-peerChan
	fmt.Println("Found oeer:", peer, ", connecting")

	if err := host.Connect(ctx, peer); err != nil {
		fmt.Println("Connection failed:", err)
	}
	stream, err := host.NewStream(ctx, peer.ID, protocol.ID(cfg.ProtocolID))
	if err != nil {
		fmt.Println("Stream open failed", err)
	}

	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	go writeData(rw)
	go readData(rw)
	fmt.Println("Connected to:", peer)

	select {}
}
