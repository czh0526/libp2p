package main

import (
	"context"
	"fmt"
	"sync"

	chat "github.com/czh0526/libp2p/client/chat"
	console "github.com/czh0526/libp2p/client/console"
	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	kad_dht "github.com/libp2p/go-libp2p-kad-dht"
	pstore "github.com/libp2p/go-libp2p-peerstore"
)

func init() {
	//ipfslog.SetAllLoggers(logging.DEBUG)
	//ipfslog.SetLogLevel("net/identify", "ERROR")
	//ipfslog.SetLogLevel("addrutil", "ERROR")
}

func main() {
	cfg, err := ParseFlags()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	host, dht, err := makeHostAndDHT(ctx, cfg)
	if err != nil {
		panic(err)
	}
	fmt.Printf("host listening on: %s@%s \n", host.ID(), host.Network().ListenAddresses())

	// 启动 Client
	client := chat.New(ctx, []string{}, host, dht)
	fmt.Printf("Client <%s> started ... \n", host.ID())

	// 启动 Console
	consoleCfg := console.Config{}
	console, err := console.New(consoleCfg, client)
	if err != nil {
		panic(err)
	}
	console.Welcome()
	go console.Interactive()

	select {}
}

func makeHostAndDHT(ctx context.Context, cfg Config) (host.Host, *kad_dht.IpfsDHT, error) {

	// 构建 Host
	host, err := libp2p.New(ctx, libp2p.Identity(cfg.PrivKey), libp2p.ListenAddrs(cfg.ListenAddrs...))
	if err != nil {
		return nil, nil, err
	}

	// 构建&启动 DHT
	dht, err := kad_dht.New(ctx, host)
	if err != nil {
		return nil, nil, err
	}
	if err := dht.Bootstrap(ctx); err != nil {
		return nil, nil, err
	}

	// 将 Host 接入网络
	var wg sync.WaitGroup
	for _, paddr := range cfg.BootstrapPeers {
		pi, err := pstore.InfoFromP2pAddr(paddr)
		if err != nil {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := host.Connect(ctx, *pi); err != nil {
				fmt.Printf("connect to %s error: %s \n", pi.String(), err)
			}
		}()
	}
	wg.Wait()

	return host, dht, nil
}
