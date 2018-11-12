package hare

import (
	"github.com/gogo/protobuf/proto"
	"github.com/spacemeshos/go-spacemesh/hare/pb"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/go-spacemesh/p2p"
	"github.com/spacemeshos/go-spacemesh/p2p/service"
	"sync"
)

const InboxCapacity = 100

// Aborter is used to add abortability
type Aborter struct {
	count uint8 // the number of listeners
	abort chan struct{}
}

func NewAborter(abortCount uint8) *Aborter {
	return &Aborter{abortCount, make(chan struct{})}
}

// Aborts all listening instances
func (aborter *Aborter) Abort() {
	for i := uint8(0); i < aborter.count; i++ {
		aborter.abort <- struct{}{} // signal abort
	}
}

// AbortChannel returns the abort channel to wait on
func (aborter *Aborter) AbortChannel() chan struct{} {
	return aborter.abort
}

// Increment the number of listening instances
// Should not be called after aborting
func (aborter *Aborter) Increment() {
	aborter.count++
}

// Broker is responsible for dispatching hare messages to the matching layer listeners
type Broker struct {
	p2p    p2p.Service
	inbox  chan service.Message
	outbox map[LayerId]chan *pb.HareMessage
	mutex  sync.Mutex
	*Aborter
}

func NewBroker(p2p p2p.Service) *Broker {
	p := new(Broker)
	p.p2p = p2p
	p.outbox = make(map[LayerId]chan *pb.HareMessage)
	p.Aborter = NewAborter(1)

	return p
}

// Start listening to protocol messages and dispatching messages
func (broker *Broker) Start() {
	broker.inbox = broker.p2p.RegisterProtocol(ProtoName)

	go broker.dispatcher()
}

// Dispatch incoming messages to the matching layer instance
func (broker *Broker) dispatcher() {
	for {
		select {
		case msg := <-broker.inbox:
			hareMsg := &pb.HareMessage{}
			err := proto.Unmarshal(msg.Data(), hareMsg)
			if err != nil {
				log.Error("Could not unmarshal message: ", err)
			}

			broker.outbox[LayerId(hareMsg.GetLayer())] <- hareMsg

		case <-broker.AbortChannel():
			return
		}
	}
}

// Inbox returns the messages channel associated with the layer
func (broker *Broker) Inbox(layer LayerId) chan *pb.HareMessage {
	broker.mutex.Lock()
	defer broker.mutex.Unlock()

	if _, exist := broker.outbox[layer]; !exist { // TODO: maybe we want to create a new one every time (?)
		broker.outbox[layer] = make(chan *pb.HareMessage, InboxCapacity)
	}

	return broker.outbox[layer]
}
