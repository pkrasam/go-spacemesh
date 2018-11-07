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

type Communicator interface {
	Broadcast(msg Byteable)
	Inbox() chan *pb.HareMessage
}

type Aborter struct {
	count uint8
	abort chan struct{}
}

func NewAborter(abortCount uint8) *Aborter {
	return &Aborter{abortCount, make(chan struct{})}
}

func (aborter *Aborter) Abort() {
	for i := uint8(0); i < aborter.count; i++ {
		aborter.abort <- struct{}{}
	}
}

func (aborter *Aborter) AbortChannel() chan struct{} {
	return aborter.abort
}

func (aborter *Aborter) Increase() {
	aborter.count++
}

type LayerCommunicator struct {
	layer LayerId
	p2p p2p.Service
	inbox chan *pb.HareMessage
}

type Broker struct {
	p2p p2p.Service
	inbox chan service.Message
	outbox map[LayerId]chan *pb.HareMessage
	mutex sync.Mutex
	*Aborter
}

func NewLayerCommunicator(layer LayerId, p2p p2p.Service, inbox chan *pb.HareMessage) Communicator {
	return &LayerCommunicator{layer, p2p, inbox}
}

func (comm *LayerCommunicator) Broadcast(msg Byteable) {
	comm.p2p.Broadcast(ProtoName, msg.Bytes())
}

func (comm *LayerCommunicator) Inbox() chan *pb.HareMessage {
	return comm.inbox
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

// Gets a new Communicator for the given layer
func (broker *Broker) Communicator(layer LayerId) Communicator {
	broker.mutex.Lock()
	defer broker.mutex.Unlock()

	if _ , exist := broker.outbox[layer]; !exist {
		broker.outbox[layer] = make(chan *pb.HareMessage, InboxCapacity)
	}

	return NewLayerCommunicator(layer, broker.p2p, broker.outbox[layer])
}
