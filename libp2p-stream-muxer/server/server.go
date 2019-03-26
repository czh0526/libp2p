package main

import (
	"fmt"
	"io"
	"net"

	smux "github.com/libp2p/go-stream-muxer"
	ymux "github.com/whyrusleeping/go-smux-yamux"
)

func main() {
	listen()
}

func listen() {
	// 获取默认的 Transport 对象
	tr := ymux.DefaultTransport
	// 获取 tcp/ip 的 listener
	l, _ := net.Listen("tcp", "localhost:13002")

	for {
		// 获取服务器端的连接
		c, _ := l.Accept()
		fmt.Println("accepted connection")
		// net.Conn ==> smux.Conn
		sc, _ := tr.NewConn(c, true)

		go func() {
			fmt.Println("serving connection")
			for {
				// 接受 Stream 请求
				s, _ := sc.AcceptStream()
				buf := make([]byte, 5)
				echoStream(s, buf)
			}
		}()
	}
}

func echoStream(s smux.Stream, buf []byte) {
	defer s.Close()

	fmt.Println("accepted stream")

	io.ReadFull(s, buf)
	fmt.Printf("read data <== %s \n", buf)
	s.Write(buf)
	fmt.Printf("echo data ==> %s \n", buf)

	fmt.Println("closing stream")
}
