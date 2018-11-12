package hare

import (
	"github.com/gogo/protobuf/proto"
	"github.com/spacemeshos/go-spacemesh/hare/pb"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/go-spacemesh/p2p"
	"time"
)

const InitialKnowledgeSize = 1000

type Set struct {
	blocks map[BlockId]struct{}
}

type State struct {
	k    uint32          // the iteration number
	ki   uint32          // ?
	s    Set             // the set of blocks
	cert *pb.Certificate // the certificate
}

type ConsensusProcess struct {
	State
	*Stopper // the consensus is stoppable
	layerId   LayerId
	oracle    Rolacle // roles oracle
	network   NetworkService
	startTime time.Time
	inbox     chan *pb.HareMessage
	knowledge []*pb.HareMessage
	isProcessed map[uint32]bool // TODO: could be empty struct
}

func (proc *ConsensusProcess) NewConsensusProcess(layer LayerId, s Set, oracle Rolacle, p2p NetworkService, broker *Broker) {
	proc.State = State{0, 0, s, nil}
	proc.Stopper = NewStopper(2)
	proc.layerId = layer
	proc.oracle = oracle
	proc.network = p2p
	proc.inbox = broker.Inbox(layer)
	proc.knowledge = make([]*pb.HareMessage, 0)
	proc.isProcessed = make(map[uint32]bool)
}

func (proc *ConsensusProcess) Start() {
	proc.startTime = time.Now()

	go proc.eventLoop()
}

func (proc *ConsensusProcess) WaitForCompletion() {
	select {
	case <- proc.StopChannel():
		return
	}
}

func (proc *ConsensusProcess) eventLoop() {
	log.Info("Start listening")

	ticker := time.NewTicker(RoundDuration)

	for {
		select {
		case msg := <-proc.inbox:
			proc.handleMessage(msg) // TODO: should be go handle (?)
		case <-ticker.C:
			proc.nextRound()
		case <-proc.StopChannel():
			log.Info("Stop event loop, instance aborted")
			return
		}
	}
}

func (proc *ConsensusProcess) handleMessage(hareMsg *pb.HareMessage) {
	// verify iteration
	if hareMsg.Message.K != proc.k {
		return
	}

	// TODO: validate sig

	// TODO: verify role
	// proc.oracle.

	proc.knowledge = append(proc.knowledge, hareMsg)
}

func (proc *ConsensusProcess) nextRound() {
	log.Info("Next round: %d", proc.k + 1)

	// process the round
	go proc.processMessages(proc.knowledge)

	// advance to next round
	proc.k++

	// reset knowledge
	proc.knowledge = make([]*pb.HareMessage, InitialKnowledgeSize)
}

func (proc *ConsensusProcess) processMessages(msg []*pb.HareMessage) {
	// check if process of messages has already been made for this round
	if _, exist :=proc.isProcessed[msg[0].Message.K]; exist {
		return
	}

	// TODO: process to create suitable hare message

	m := proc.round0()
	buff, err := proto.Marshal(m)
	if err != nil {
		return
	}

	// send message
	proc.network.Broadcast(ProtoName, buff)


	// mark as processed
	proc.isProcessed[m.Message.K] = true
}


func (proc *ConsensusProcess) round0() *pb.HareMessage {
	// TODO: build SVP msg

	// send proposal
}