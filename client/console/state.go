package console

import (
	peer "github.com/libp2p/go-libp2p-peer"
)

type CurrentState struct {
	peerId peer.ID
}

func (s CurrentState) IsValidatePID() bool {
	if err := s.peerId.Validate(); err != nil {
		return false
	}

	return true
}
