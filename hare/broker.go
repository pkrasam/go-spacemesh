package hare

import (
	"github.com/gogo/protobuf/proto"
	"github.com/spacemeshos/go-spacemesh/hare/pb"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/go-spacemesh/p2p/service"
	"sync"
)

const InboxCapacity = 100

// Stopper is used to add stoppability to an object
type Stopper struct {
	count   uint8         // the number of listeners
	channel chan struct{} // listeners listen to this channel
}

func NewStopper(listenersCount uint8) *Stopper {
	return &Stopper{listenersCount, make(chan struct{})}
}

// Stops all listening instances (should be called only once)
func (stopper *Stopper) Stop() {
	for i := uint8(0); i < stopper.count; i++ {
		stopper.channel <- struct{}{} // signal listener through channel
	}
}

// StopChannel returns the channel channel to wait on
func (stopper *Stopper) StopChannel() chan struct{} {
	return stopper.channel
}

// Increment the number of listening instances
// Should not be called after Stop()
func (stopper *Stopper) Increment() {
	stopper.count++
}

// Broker is responsible for dispatching hare messages to the matching layer listener
type Broker struct {
	*Stopper
	network NetworkService
	inbox   chan service.Message
	outbox  map[LayerId]chan *pb.HareMessage
	mutex   sync.Mutex
}

func NewBroker(networkService NetworkService) *Broker {
	p := new(Broker)
	p.Stopper = NewStopper(1)
	p.network = networkService
	p.outbox = make(map[LayerId]chan *pb.HareMessage)

	return p
}

// Start listening to protocol messages and dispatch messages
func (broker *Broker) Start() {
	broker.inbox = broker.network.RegisterProtocol(ProtoName)

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

		case <-broker.StopChannel():
			return
		}
	}
}

// Inbox returns the message channel associated with the given layer
func (broker *Broker) Inbox(layer LayerId) chan *pb.HareMessage {
	broker.mutex.Lock()
	defer broker.mutex.Unlock()

	if _, exist := broker.outbox[layer]; exist {
		panic("Inbox called more than once per layer")
	}

	broker.outbox[layer] = make(chan *pb.HareMessage, InboxCapacity) // create new channel

	return broker.outbox[layer]
}
