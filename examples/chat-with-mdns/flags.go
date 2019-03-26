package main

import (
	"flag"
)

type config struct {
	RendezvousString string
	ProtocolID       string
	listenHost       string
	listenPort       int
}

func parseFlags() *config {
	c := &config{}

	// 设置命令行参数的接收变量
	flag.StringVar(&c.RendezvousString, "rendezvous", "meetme", "Unique string to identify group of nodes. Share this with your friends to let them connect with you")
	flag.StringVar(&c.listenHost, "host", "0.0.0.0", "THis bootstrap node host listen address\n")
	flag.StringVar(&c.ProtocolID, "pid", "/char/1.1.0", "Sets a protocol id for stream headers")
	flag.IntVar(&c.listenPort, "port", 4001, "node listen port")

	// 解析命令行参数
	flag.Parse()
	return c
}
