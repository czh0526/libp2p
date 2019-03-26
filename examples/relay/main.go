package main

import (
	"context"
	"fmt"

	host "github.com/libp2p/go-libp2p-host"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	swarm "github.com/libp2p/go-libp2p-swarm"
	ma "github.com/multiformats/go-multiaddr"
)

var (
	privString1 = "080312793077020101042021e746cdf0ecbeadf369457fc37d09cc1decfb53dc0843b38521636c410f39aea00a06082a8648ce3d030107a144034200042457fa493e518413e18299a1f1884cef47ed2be3196bcc87a18a20b9e186c80358664eb7d78885ecba736022b1fe0a28870098608f9be7cd93f523cfe63479e1"
	privString2 = "0803127930770201010420c10c49391d7313bafe551aad57f3a0e79e1c030035c1c0194f0ee283f8b66992a00a06082a8648ce3d030107a14403420004239c6d1667f1adf403eea1a48028a58daea7fd28e6e803ca480071f979a32d197be88b2cf95a9fb81f05b85acb925ee29b42e05b27bac301333db7a1fcb47cf4"
	privString3 = "0803127930770201010420b6fef0a136624d6f46d07236c12827d3407849e8bfaeba6a2095a9d96ff74dbfa00a06082a8648ce3d030107a144034200047ab011a60868d9220bb7fb32a5a9a7d467ebf2fa1379caa1f6a6f1daf7477905e6a111ff98c2005e279ffaf9596b17c3ee3922b1c6c9b6296171982c8667839c"
)

func main() {
	var n int
	var relayAddr string
	var targetAddr string
	parseFlags(&n, &relayAddr, &targetAddr)

	if n == 1 {

		// 构造 source node
		sourceHost, err := makeSourceNode(privString1)
		if err != nil {
			panic(err)
		}

		relayInfo, err := multiaddr2PeerInfo(relayAddr)
		if err != nil {
			panic(err)
		}

		targetInfo, err := multiaddr2PeerInfo(targetAddr)
		if err != nil {
			panic(err)
		}

		// source node ==> relay node
		if err := connect(*sourceHost, *relayInfo); err != nil {
			panic(fmt.Sprintf("1) %s", err))
		}
		fmt.Printf("Connect ==> %s \n", (*relayInfo).ID.Pretty())

		// 测试 NewStream: sourceHost ==> targetAddr
		_, err = (*sourceHost).NewStream(context.Background(), targetInfo.ID, "/cats")
		if err == nil {
			fmt.Println("Didnt actually expect to get a stream here. What happened?")
			return
		}
		fmt.Println("和我们想的一样，source ==> target 不能直连: ", err)

		// 构建一个中继地址 p2p-circuit|target
		relayaddr, err := ma.NewMultiaddr("/p2p-circuit/ipfs/" + targetInfo.ID.Pretty())
		if err != nil {
			panic(fmt.Sprintf("2) %s", err))
		}
		fmt.Printf("构建中继地址：%s \n", relayaddr)

		// 清除缓存
		(*sourceHost).Network().(*swarm.Swarm).Backoff().Clear(targetInfo.ID)

		// 构建一个中继的 PeerInfo
		h3relayInfo := pstore.PeerInfo{
			ID:    targetInfo.ID,
			Addrs: []ma.Multiaddr{relayaddr},
		}

		// source node ==> relay peer
		if err := (*sourceHost).Connect(context.Background(), h3relayInfo); err != nil {
			panic(fmt.Sprintf("3) %s", err))
		}
		fmt.Printf("Connect ==> %s \n", h3relayInfo.ID.Pretty())

		// NewStream source node ==> target node
		s, err := (*sourceHost).NewStream(context.Background(), h3relayInfo.ID, "/cats")
		if err != nil {
			fmt.Println("huh, this should have worked: ", err)
			return
		}
		fmt.Printf("NewStream ==> %s \n", h3relayInfo.ID.Pretty())

		s.Read(make([]byte, 1))

		select {}

	} else if n == 2 {

		// 构造 relay node
		relayHost, err := makeRelayNode(privString2)
		if err != nil {
			panic(err)
		}
		fmt.Printf("relay host 启动成功，下一步是在 target host 上执行 ./relay -n 3 -raddr /ip4/x.x.x.x/tcp/13002/ipfs/%s \n", (*relayHost).ID().Pretty())
		select {}

	} else if n == 3 {

		// 构造 target node
		targetHost, err := makeTargetNode(privString3)
		if err != nil {
			panic(err)
		}

		relayInfo, err := multiaddr2PeerInfo(relayAddr)
		if err != nil {
			panic(err)
		}

		// target node ==> relay node
		if err := connect(*targetHost, *relayInfo); err != nil {
			panic(err)
		}
		fmt.Printf("Connect ==> %s\n", relayInfo.ID.Pretty())

		if _, err := (*targetHost).NewStream(context.Background(), relayInfo.ID, "/cats"); err != nil {
			panic(err)
		}
		fmt.Printf("NewStream ==> %s\n", relayInfo.ID.Pretty())

		select {}
	}
}

func multiaddr2PeerInfo(addrString string) (*pstore.PeerInfo, error) {
	// Host 3 的 PeerInfo
	multiaddr, err := ma.NewMultiaddr(addrString)
	if err != nil {
		return nil, err
	}
	peerInfo, err := pstore.InfoFromP2pAddr(multiaddr)
	if err != nil {
		return nil, err
	}
	return peerInfo, nil
}

func connect(h1 host.Host, h2PeerInfo pstore.PeerInfo) error {

	// Host 1 ==> Host 2
	if err := h1.Connect(context.Background(), h2PeerInfo); err != nil {
		return err
	}

	return nil
}
