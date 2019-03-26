package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	mrand "math/rand"
	"os"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	multiaddr "github.com/multiformats/go-multiaddr"

	net "github.com/libp2p/go-libp2p-net"

	peerstore "github.com/libp2p/go-libp2p-peerstore"
)

func handleStream(s net.Stream) {
	fmt.Println("Got a new stream!")

	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	go readData(rw)
	go writeData(rw)
}

func readData(rw *bufio.ReadWriter) {
	for {
		str, _ := rw.ReadString('\n')

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
		fmt.Print("> ")
		sendData, err := stdReader.ReadString('\n')

		if err != nil {
			panic(err)
		}

		rw.WriteString(fmt.Sprintf("%s\n", sendData))
		rw.Flush()
	}
}

func main() {

	sourcePort := flag.Int("sp", 0, "Source port number")
	dest := flag.String("d", "", "Destination multiaddr string")
	debug := flag.Bool("debug", false, "Debug generates the same node ID on every execution")

	flag.Parse()

	var r io.Reader
	if *debug {
		r = mrand.New(mrand.NewSource(int64(*sourcePort)))
	} else {
		r = rand.Reader
	}

	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Secp256k1, 2048, r)
	if err != nil {
		panic(err)
	}

	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *sourcePort))
	fmt.Printf("sourceMultiAddr = %s\n", sourceMultiAddr)

	host, err := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)

	if err != nil {
		panic(err)
	}

	if *dest == "" {
		host.SetStreamHandler("/chat/1.0.0", handleStream)

		var port string
		for _, la := range host.Network().ListenAddresses() {
			fmt.Printf("la = %s\n", la)
			if p, err := la.ValueForProtocol(multiaddr.P_TCP); err == nil {
				port = p
				break
			}
		}

		if port == "" {
			panic("was not able to find actual local port")
		}

		fmt.Printf("Run './chat -d /ip4/127.0.0.1/tcp/%v/p2p/%s' on another console. \n", port, host.ID().Pretty())
		fmt.Println("You can replace 127.0.0.1 with public IP as well.")
		fmt.Printf("\nWaiting for incoming connection\n\n")

		<-make(chan struct{})

	} else {
		fmt.Println("This node is multiaddrsses:")
		for _, la := range host.Addrs() {
			fmt.Printf(" - %v \n", la)
		}
		fmt.Println()

		// 构建远程节点地址
		maddr, err := multiaddr.NewMultiaddr(*dest)
		if err != nil {
			panic(err)
		}
		fmt.Printf("dest = %s \n", maddr)

		// 获得远程节点ID
		info, err := peerstore.InfoFromP2pAddr(maddr)
		if err != nil {
			panic(err)
		}
		fmt.Printf("peerInfo ==> \n\t ID: %s\n\t Addrs: %s\n", info.ID.String(), info.Addrs)

		// 将远程节点加入地址簿
		host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

		s, err := host.NewStream(context.Background(), info.ID, "/chat/1.0.0")
		if err != nil {
			panic(err)
		}

		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

		go writeData(rw)
		go readData(rw)

		select {}
	}
}
