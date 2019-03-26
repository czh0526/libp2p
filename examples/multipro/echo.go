package main

import (
	"fmt"
	"io/ioutil"

	pb "github.com/czh0526/libp2p/multipro/pb"
	"github.com/google/uuid"

	"github.com/gogo/protobuf/proto"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
)

const echoRequest = "/echo/echoreq/0.0.1"
const echoResponse = "/echo/echoresp/0.0.1"

type EchoProtocol struct {
	node     *Node
	requests map[string]*pb.EchoRequest
	done     chan bool
}

func NewEchoProtocol(node *Node, done chan bool) *EchoProtocol {
	e := EchoProtocol{
		node:     node,
		requests: make(map[string]*pb.EchoRequest),
		done:     done,
	}
	node.SetStreamHandler(echoRequest, e.onEchoRequest)
	node.SetStreamHandler(echoResponse, e.onEchoResponse)
	return &e
}

func (e *EchoProtocol) onEchoRequest(s inet.Stream) {
	data := &pb.EchoRequest{}
	// stream ==> buf
	buf, err := ioutil.ReadAll(s)
	if err != nil {
		s.Reset()
		fmt.Println(err)
		return
	}
	s.Close()

	// Unmarshal
	proto.Unmarshal(buf, data)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%s: Received echo request from %s. Message: %s \n", s.Conn().LocalPeer(), s.Conn().RemotePeer(), data.Message)
	valid := e.node.authenticateMessage(data, data.MessageData)
	if !valid {
		fmt.Println("Failed to authenticate message")
		return
	}

	fmt.Printf("%s: Sending echo response to %s. Message id: %s... \n", s.Conn().LocalPeer(), s.Conn().RemotePeer(), data.Message)

	resp := &pb.EchoResponse{
		MessageData: e.node.NewMessageData(data.MessageData.Id, false),
		Message:     data.Message,
	}

	signature, err := e.node.signProtoMessage(resp)
	if err != nil {
		fmt.Println("failed to sign response")
		return
	}

	resp.MessageData.Sign = signature

	ok := e.node.sendProtoMessage(s.Conn().RemotePeer(), echoResponse, resp)

	if ok {
		fmt.Printf("%s: Echo response to %s sent. \n", s.Conn().LocalPeer().String(), s.Conn().RemotePeer().String())
	}
}

func (e *EchoProtocol) onEchoResponse(s inet.Stream) {
	data := &pb.EchoResponse{}
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

	valid := e.node.authenticateMessage(data, data.MessageData)
	if !valid {
		fmt.Println("Failed to authenticate message")
		return
	}

	req, ok := e.requests[data.MessageData.Id]
	if ok {
		delete(e.requests, data.MessageData.Id)
	} else {
		fmt.Println("Failed to locate request data object for response")
		return
	}

	if req.Message != data.Message {
		panic("Expected echo to respond with request message.")
	}

	fmt.Printf("%s: Received echo response from %s. Message id: %s, Message: %s \n", s.Conn().LocalPeer(), s.Conn().RemotePeer(), data.MessageData.Id, data.Message)
	e.done <- true
}

func (e *EchoProtocol) Echo(host host.Host) bool {
	fmt.Printf("%s: Sending echo to: %s... \n", e.node.ID(), host.ID())

	req := &pb.EchoRequest{
		MessageData: e.node.NewMessageData(uuid.New().String(), false),
		Message:     fmt.Sprintf("Echo from %s", e.node.ID()),
	}

	// 计算签名
	signature, err := e.node.signProtoMessage(req)
	if err != nil {
		fmt.Println("failed to sign message")
		return false
	}

	// 设置签名字段
	req.MessageData.Sign = signature

	// 发送签名消息
	ok := e.node.sendProtoMessage(host.ID(), echoRequest, req)
	if !ok {
		return false
	}

	e.requests[req.MessageData.Id] = req
	fmt.Printf("%s: Echo to: %s was sent. Message Id: %s, Message: %s \n", e.node.ID(), host.ID(), req.MessageData.Id, req.Message)
	return true
}
