package hare

import (
	"github.com/gogo/protobuf/proto"
	"github.com/spacemeshos/go-spacemesh/hare/pb"
	"github.com/spacemeshos/go-spacemesh/p2p/simulator"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func createMessage(t *testing.T, layer LayerId) []byte {
	hareMsg := &pb.HareMessage{}
	hareMsg.Layer = uint32(layer)
	serMsg, err := proto.Marshal(hareMsg)

	if err != nil {
		assert.Fail(t, "Failed to marshal data")
	}

	return serMsg
}

// test that a message to a specific layer is delivered by the broker
func TestBroker_Received(t *testing.T) {
	sim := simulator.New()
	n1 := sim.NewNode()
	n2 := sim.NewNode()

	broker := NewBroker(n1)
	broker.Start()
	comm1 := broker.Communicator(1)

	serMsg := createMessage(t, 1)
	n2.Broadcast(ProtoName, serMsg)

	recv := <-comm1.Inbox()

	assert.True(t, recv.Layer == 1)
}

// test that aborting the broker aborts
func TestBroker_Abort(t *testing.T) {
	sim := simulator.New()
	n1 := sim.NewNode()

	broker := NewBroker(n1)
	broker.Increase()
	broker.Start()
	broker.Communicator(1)

	timer := time.NewTimer(3 * time.Second)

	go broker.Abort()

	select {
	case <-broker.AbortChannel():
		assert.True(t, true)
	case <-timer.C:
		assert.Fail(t, "timeout")
	}
}

func sendMessages(t *testing.T, layer LayerId, n *simulator.Node, count int) {
	for i := 0; i < count; i++ {
		n.Broadcast(ProtoName, createMessage(t, layer))
	}
}

func waitForMessages(comm Communicator, msgCount int) {
	for i := 0; i < msgCount; i++ {
		<-comm.Inbox()
	}
}

// test flow for multiple layers
func TestBroker_MultipleLayers(t *testing.T) {
	sim := simulator.New()
	n1 := sim.NewNode()
	n2 := sim.NewNode()
	const msgCount = 100

	broker := NewBroker(n1)
	broker.Start()
	comm1 := broker.Communicator(1)
	comm2 := broker.Communicator(2)
	comm3 := broker.Communicator(3)

	go sendMessages(t, 1, n2, msgCount)
	go sendMessages(t, 2, n2, msgCount)
	go sendMessages(t, 3, n2, msgCount)

	waitForMessages(comm1, msgCount)
	waitForMessages(comm2, msgCount)
	waitForMessages(comm3, msgCount)

	assert.True(t, true)
}
