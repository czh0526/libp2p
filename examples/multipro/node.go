package main

import (
	"context"
	"fmt"
	"time"

	pb "github.com/czh0526/libp2p/multipro/pb"
	ggio "github.com/gogo/protobuf/io"
	proto "github.com/gogo/protobuf/proto"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	protocol "github.com/libp2p/go-libp2p-protocol"
)

const clientVersion = "go-p2p-node/0.0.1"

type Node struct {
	host.Host
	*PingProtocol
	*EchoProtocol
}

func NewNode(host host.Host, done chan bool) *Node {
	node := &Node{Host: host}
	node.PingProtocol = NewPingProtocol(node, done)
	node.EchoProtocol = NewEchoProtocol(node, done)
	return node
}

func (n *Node) NewMessageData(messageId string, gossip bool) *pb.MessageData {
	nodePubKey, err := n.Peerstore().PubKey(n.ID()).Bytes()
	if err != nil {
		panic("Failed to get public key for sender from local peer store.")
	}

	return &pb.MessageData{
		ClientVersion: clientVersion,
		NodeId:        peer.IDB58Encode(n.ID()),
		NodePubKey:    nodePubKey,
		Timestamp:     time.Now().Unix(),
		Id:            messageId,
		Gossip:        gossip,
	}
}

func (n *Node) sendProtoMessage(id peer.ID, p protocol.ID, data proto.Message) bool {
	// 与相连的 peer 建立消息流信道
	s, err := n.NewStream(context.Background(), id, p)
	if err != nil {
		fmt.Println(err)
		return false
	}

	// 向信道里面写消息
	writer := ggio.NewFullWriter(s)
	err = writer.WriteMsg(data)
	if err != nil {
		fmt.Println(err)
		s.Reset()
		return false
	}

	// 关闭信道
	err = inet.FullClose(s)
	if err != nil {
		fmt.Println(err)
		s.Reset()
		return false
	}
	return true
}

func (n *Node) signProtoMessage(message proto.Message) ([]byte, error) {
	data, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}
	return n.signData(data)
}

func (n *Node) signData(data []byte) ([]byte, error) {
	key := n.Peerstore().PrivKey(n.ID())
	res, err := key.Sign(data)
	return res, err
}

func (n *Node) authenticateMessage(message proto.Message, data *pb.MessageData) bool {
	sign := data.Sign
	data.Sign = nil

	bin, err := proto.Marshal(message)
	if err != nil {
		fmt.Println(err, "Failed to marshal pb message")
		return false
	}

	data.Sign = sign

	peerId, err := peer.IDB58Decode(data.NodeId)
	if err != nil {
		fmt.Println(err, "Failed to decode node id from base58")
		return false
	}

	return n.verifyData(bin, []byte(sign), peerId, data.NodePubKey)
}

func (n *Node) verifyData(data []byte, signature []byte, peerId peer.ID, pubKeyData []byte) bool {
	key, err := crypto.UnmarshalPublicKey(pubKeyData)
	if err != nil {
		fmt.Println(err, "Failed to extract key from message key data.")
		return false
	}

	// 验证 public key <==> peer id
	idFromKey, err := peer.IDFromPublicKey(key)
	if err != nil {
		fmt.Println(err, "Failed to extract peer id from public key")
		return false
	}

	if idFromKey != peerId {
		fmt.Println(err, "Node id and provided public key mismatch.")
		return false
	}

	// data <==> sugnature
	res, err := key.Verify(data, signature)
	if err != nil {
		fmt.Println(err, "Error authenticating data")
	}

	return res
}
