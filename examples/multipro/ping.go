package main

import (
	"fmt"
	"io/ioutil"
	"log"

	pb "github.com/czh0526/libp2p/multipro/pb"
	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
)

const pingRequest = "/ping/pingreq/0.0.1"
const pingResponse = "/ping/pingresp/0.0.1"

type PingProtocol struct {
	node     *Node
	requests map[string]*pb.PingRequest
	done     chan bool
}

func NewPingProtocol(node *Node, done chan bool) *PingProtocol {
	p := &PingProtocol{
		node:     node,
		requests: make(map[string]*pb.PingRequest),
		done:     done,
	}
	node.SetStreamHandler(pingRequest, p.onPingRequest)
	node.SetStreamHandler(pingResponse, p.onPingResponse)
	return p
}

func (p *PingProtocol) onPingRequest(s inet.Stream) {
	data := &pb.PingRequest{}
	buf, err := ioutil.ReadAll(s)
	if err != nil {
		s.Reset()
		fmt.Println(err)
		return
	}
	s.Close()

	proto.Unmarshal(buf, data)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%s: Received ping request from %s. Message: %s \n", s.Conn().LocalPeer(), s.Conn().RemotePeer(), data.Message)
	valid := p.node.authenticateMessage(data, data.MessageData)
	if !valid {
		fmt.Println("Failed to authrnticate message")
		return
	}

	log.Printf("%s: Sending ping response to %s. Message id: %s... \n", s.Conn().LocalPeer(), s.Conn().RemotePeer(), data.MessageData.Id)

	resp := &pb.PingResponse{
		MessageData: p.node.NewMessageData(data.MessageData.Id, false),
		Message:     fmt.Sprintf("Ping response from %s", p.node.ID()),
	}

	signature, err := p.node.signProtoMessage(resp)
	if err != nil {
		fmt.Println("failed to sign response")
		return
	}

	resp.MessageData.Sign = signature
	ok := p.node.sendProtoMessage(s.Conn().RemotePeer(), pingResponse, resp)
	if ok {
		fmt.Printf("%s: Ping response to %s sent. \n", s.Conn().LocalPeer().String(), s.Conn().RemotePeer().String())
	}
}

func (p *PingProtocol) onPingResponse(s inet.Stream) {
	data := &pb.PingResponse{}
	buf, err := ioutil.ReadAll(s)
	if err != nil {
		s.Reset()
		fmt.Println(err)
		return
	}
	s.Close()

	// unmarshal it
	proto.Unmarshal(buf, data)
	if err != nil {
		fmt.Println(err)
		return
	}

	valid := p.node.authenticateMessage(data, data.MessageData)

	if !valid {
		fmt.Println("Failed to authenticate message")
		return
	}

	// locate request data and remove it if found
	_, ok := p.requests[data.MessageData.Id]
	if ok {
		// remove request from map as we have processed it here
		delete(p.requests, data.MessageData.Id)
	} else {
		fmt.Println("Failed to locate request data boject for response")
		return
	}

	fmt.Printf("%s: Received ping response from %s. Message id:%s. Message: %s. \n", s.Conn().LocalPeer(), s.Conn().RemotePeer(), data.MessageData.Id, data.Message)
	p.done <- true
}

func (p *PingProtocol) Ping(host host.Host) bool {
	fmt.Printf("%s: Sending ping to: %s... \n", p.node.ID(), host.ID())

	req := &pb.PingRequest{
		MessageData: p.node.NewMessageData(uuid.New().String(), false),
		Message:     fmt.Sprintf("Ping from %s", p.node.ID()),
	}

	signature, err := p.node.signProtoMessage(req)
	if err != nil {
		fmt.Println("failed to sign pb data")
		return false
	}

	req.MessageData.Sign = signature
	ok := p.node.sendProtoMessage(host.ID(), pingRequest, req)
	if !ok {
		return false
	}

	p.requests[req.MessageData.Id] = req
	fmt.Printf("%s: Ping to: %s was sent. Message Id: %s, Message: %s \n", p.node.ID(), host.ID(), req.MessageData.Id, req.Message)
	return true
}
