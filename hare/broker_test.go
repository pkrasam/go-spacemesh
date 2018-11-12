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
	inbox := broker.Inbox(1)

	serMsg := createMessage(t, 1)
	n2.Broadcast(ProtoName, serMsg)

	recv := <- inbox

	assert.True(t, recv.Layer == 1)
}

// test that aborting the broker aborts
func TestBroker_Abort(t *testing.T) {
	sim := simulator.New()
	n1 := sim.NewNode()

	broker := NewBroker(n1)
	broker.Increment()
	broker.Start()
	broker.Inbox(1)

	timer := time.NewTimer(3 * time.Second)

	go broker.Stop()

	select {
	case <-broker.StopChannel():
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

func waitForMessages(t *testing.T, inbox chan *pb.HareMessage, layer LayerId, msgCount int) {
	for i := 0; i < msgCount; i++ {
		x := <-inbox
		assert.True(t, x.Layer == uint32(layer))
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
	inbox1 := broker.Inbox(1)
	inbox2 := broker.Inbox(2)
	inbox3 := broker.Inbox(3)

	go sendMessages(t, 1, n2, msgCount)
	go sendMessages(t, 2, n2, msgCount)
	go sendMessages(t, 3, n2, msgCount)

	waitForMessages(t, inbox1, 1, msgCount)
	waitForMessages(t, inbox2, 2, msgCount)
	waitForMessages(t, inbox3, 3, msgCount)

	assert.True(t, true)
}
