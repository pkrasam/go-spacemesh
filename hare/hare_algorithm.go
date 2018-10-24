package hare

import (
	"github.com/spacemeshos/go-spacemesh/log"
	"time"
)

type Set struct {
	blocks map[BLockId]struct{}
}

type Parti struct {
	k    uint32 // the iteration number
	ki   uint32
	s    Set // the set of blocks
	cert Certificate
}

type Algo struct {
	Parti
	oracle    IRolacle // roles oracle
	network   NetworkConnection
	startTime time.Time
	inbox     chan IByteable
	abort     chan struct{}
}

func (algo *Algo) Start() {
	algo.startTime = time.Now()
	algo.k = 0
	algo.ki = 0
	algo.network.RegisterProtocol(PROTO_NAME)

	go algo.Listen()
	algo.do()
}

func (algo *Algo) Listen() {
	log.Info("Start listening")
	for {
		select {
		case msg := <-algo.inbox:
			algo.handleMessage(msg) // should be go handle (?)
		case <-algo.abort:
			log.Info("Listen aborted")
			return
		}
	}
}

func (algo *Algo) handleMessage(msg IByteable) {
	if time.Now().After(algo.startTime.Add(algo.k * ROUND_DURATION)) {
		return // ignore late msg
	}

	// classify to current round
	algo.round0(msg)
}

func (algo *Algo) do() {
	
}
