package hare

import (
	"github.com/spacemeshos/go-spacemesh/p2p"
	"github.com/spacemeshos/go-spacemesh/p2p/service"
)

const InboxCapacity = 100

type Communicator interface {
	Broadcast(msg Byteable)
	Inbox() chan Byteable
}

type LayerCommunicator struct {
	layer LayerId
	p2p p2p.Service
	inbox chan Byteable
}

// TODO: consider adding logging
type Broker struct {
	p2p p2p.Service
	inbox chan service.Message
	outbox map[LayerId]chan Byteable
	abort chan struct{}
}

func NewLayerCommunicator(layer LayerId, p2p p2p.Service, inbox chan Byteable) Communicator {
	return &LayerCommunicator{layer, p2p, inbox}
}

func (comm *LayerCommunicator) Broadcast(msg Byteable) {
	//p2p.Broadcast....
}

func (comm *LayerCommunicator) Inbox() chan Byteable {
	return comm.inbox
}

func NewBroker(p2p p2p.Service) *Broker {
	p := new(Broker)
	p.p2p = p2p
	p.abort = make(chan struct{})

	return p
}

// Start listening to protocol messages and dispatching messages
func (broker *Broker) Start() {
	// TODO: p2p register
	broker.inbox = broker.p2p.RegisterProtocol(ProtoName)

	go broker.dispatcher()
}

// Dispatch incoming messages to the matching layer instance
func (broker *Broker) dispatcher() {
	for {
		select {
		case msg := <-broker.inbox:
			msg.Data()
		// TODO: protobuf to our type and pass to matching layer
		// TODO: should probably pass struct msg and not Byteable

		case <-broker.abort:
			return
		}
	}
}

// TODO: should it be thread safe?
func (broker *Broker) Communicator(layer LayerId) Communicator {
	if _ , exist := broker.outbox[layer]; !exist {
		broker.outbox[layer] = make(chan Byteable, InboxCapacity)
	}

	return NewLayerCommunicator(layer, broker.p2p, broker.outbox[layer])
}
