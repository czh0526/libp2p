package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"

	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
)

const help = `
This example creates a simple HTTP Proxy using two libp2p peers. The first peer
provides an HTTP server locally which tunnels the HTTP requests with libp2p
to a remote peer. The remote peer performs the requests and 
send the sends the response back.
Usage: Start remote peer first with:   ./proxy
       Then start the local peer with: ./proxy -d <remote-peer-multiaddress>
Then you can do something like: curl -x "localhost:9900" "http://ipfs.io".
This proxies sends the request through the local peer, which proxies it to
the remote peer, which makes it and sends the response back.
`

const Protocol = "/proxy-example/0.0.1"

// 构建主机
func makeRandomHost(port int) host.Host {
	host, err := libp2p.New(
		context.Background(),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)))
	if err != nil {
		fmt.Println(err)
	}
	return host
}

type ProxyService struct {
	host      host.Host
	dest      peer.ID
	proxyAddr ma.Multiaddr
}

// 构建代理服务器
func NewProxyService(h host.Host, proxyAddr ma.Multiaddr, dest peer.ID) *ProxyService {
	h.SetStreamHandler(Protocol, streamHandler)

	fmt.Println("Proxy server is ready.")
	fmt.Println("libp2p-peer addresses: ")
	for _, a := range h.Addrs() {
		fmt.Printf("%s/ipfs/%s\n", a, peer.IDB58Encode(h.ID()))
	}

	return &ProxyService{
		host:      h,
		dest:      dest,
		proxyAddr: proxyAddr,
	}
}

// remote host 使用
func streamHandler(stream inet.Stream) {
	defer stream.Close()

	// 从 stream 中读 http request
	buf := bufio.NewReader(stream)
	req, err := http.ReadRequest(buf)
	if err != nil {
		stream.Reset()
		fmt.Println(err)
		return
	}
	defer req.Body.Close()

	req.URL.Scheme = "http"
	hp := strings.Split(req.Host, ":")
	if len(hp) > 1 && hp[1] == "443" {
		req.URL.Scheme = "https"
	} else {
		req.URL.Scheme = "http"
	}
	req.URL.Host = req.Host

	outreq := new(http.Request)
	*outreq = *req

	fmt.Printf("Making request to %s \n", req.URL)
	resp, err := http.DefaultTransport.RoundTrip(outreq)
	if err != nil {
		stream.Reset()
		fmt.Println(err)
		return
	}

	resp.Write(stream)
}

func (p *ProxyService) Serve() {
	_, serveArgs, _ := manet.DialArgs(p.proxyAddr)
	fmt.Println("proxy addr = ", p.proxyAddr)
	fmt.Println("proxy listening on ", serveArgs)
	if p.dest != "" {
		http.ListenAndServe(serveArgs, p)
	}
}

/*
 * /ip4/<x.x.x.x>/tcp/<port>/ipfs/<peerid> ---- <peerid>
 *                                          \__ /ip4/<x.x.x.x>/tcp/<port>
 */
func addAddrToPeerstore(h host.Host, addr string) peer.ID {
	// /ip4/<x.x.x.x>/tcp/<port>/ipfs/<peerid> => ma.Multiaddr
	ipfsaddr, err := ma.NewMultiaddr(addr)
	if err != nil {
		fmt.Println(err)
	}
	// <peerid>
	pid, err := ipfsaddr.ValueForProtocol(ma.P_IPFS)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("pid = %s \n", pid)

	// <peerid> => peer.ID
	peerid, err := peer.IDB58Decode(pid)
	if err != nil {
		fmt.Println(err)
	}

	// /ip4/<x.x.x.x>tcp/<port>
	targetPeerAddr, _ := ma.NewMultiaddr(
		fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)))
	targetAddr := ipfsaddr.Decapsulate(targetPeerAddr)

	// ADD record: <peerid> ==> /ip4/<x.x.x.x>/tcp/<port>
	h.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)
	return peerid
}

// local peer 使用
func (p *ProxyService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("proxying request for %s to peer %s \n", r.URL, p.dest.Pretty())
	// Stream: host ==> dest
	stream, err := p.host.NewStream(context.Background(), p.dest, Protocol)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stream.Close()

	// ==> request
	err = r.Write(stream)
	if err != nil {
		stream.Reset()
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	buf := bufio.NewReader(stream)
	resp, err := http.ReadResponse(buf, r)
	if err != nil {
		stream.Reset()
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}

	for k, v := range resp.Header {
		for _, s := range v {
			w.Header().Add(k, s)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	resp.Body.Close()
}

func main() {
	flag.Usage = func() {
		fmt.Println(help)
		flag.PrintDefaults()
	}

	destPeer := flag.String("d", "", "destination peer address")
	port := flag.Int("p", 9900, "proxy port")
	p2pport := flag.Int("l", 12000, "libp2p listen port")
	flag.Parse()

	if *destPeer != "" {
		// 构建主机对象
		host := makeRandomHost(*p2pport + 1)
		destPeerID := addAddrToPeerstore(host, *destPeer)
		// 代理进程的端口
		proxyAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port))
		if err != nil {
			panic(err)
		}

		// 启动本地代理服务
		proxy := NewProxyService(host, proxyAddr, destPeerID)
		proxy.Serve()

	} else {
		// 构建主机对象
		host := makeRandomHost(*p2pport)
		// 启动远程代理服务
		_ = NewProxyService(host, nil, "")
		<-make(chan struct{})
	}
}
