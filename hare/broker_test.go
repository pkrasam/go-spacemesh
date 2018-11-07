package hare

import (
	"github.com/gogo/protobuf/proto"
	"github.com/spacemeshos/go-spacemesh/hare/pb"
	"github.com/spacemeshos/go-spacemesh/p2p/simulator"
	"github.com/stretchr/testify/assert"
	"testing"
)

func createMessage(t *testing.T, layer LayerId) []byte {
	hareMsg := &pb.HareMessage{}
	hareMsg.Layer = uint32(layer)
	serMsg, err := proto.Marshal(hareMsg)

	if err != nil {
		assert.Error(t, err, "Failed to marshal data")
	}

	return serMsg
}

func TestBroker_Received(t *testing.T) {
	sim := simulator.New()
	n1 := sim.NewNode()
	n2 := sim.NewNode()

	broker1 := NewBroker(n1)
	broker1.Start()
	comm1 := broker1.Communicator(1)

	serMsg := createMessage(t, 1)
	n2.Broadcast(ProtoName, serMsg)

	recv := <- comm1.Inbox()

	broker1.Abort()

	assert.True(t, recv.Layer == 1)
}