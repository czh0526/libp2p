package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"

	ymux "github.com/whyrusleeping/go-smux-yamux"
)

func main() {
	dial()
}

func dial() {
	nconn, _ := net.Dial("tcp", "localhost:13002")
	sconn, _ := ymux.DefaultTransport.NewConn(nconn, false)

	go func() {
		for {
			sconn.AcceptStream()
		}
	}()

	go func() {
		length := 20
		buf := make([]byte, length)

		// 打开一个 Stream
		s1, _ := sconn.OpenStream()
		s1.Write([]byte("hello"))
		s1.Read(buf)
		fmt.Printf("received %s as a response \n", string(buf))
		s1.Close()
	}()

	go func() {
		<-time.After(time.Second)
		length := 20
		buf := make([]byte, length)

		s2, _ := sconn.OpenStream()
		s2.Write([]byte("world"))
		s2.Read(buf)
		fmt.Printf("received %s as a response \n", string(buf))
		s2.Close()
	}()

	<-time.After(time.Second * 3)
	s3, _ := sconn.OpenStream()
	buf3 := make([]byte, 20)
	io.ReadFull(os.Stdin, buf3)
	fmt.Printf("get data <= %s \n", buf3)
	s3.Write(buf3)
	fmt.Printf("send data ==> %s\n", buf3)
}
