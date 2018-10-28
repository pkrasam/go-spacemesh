package hare

import (
	"github.com/golang/protobuf/proto"
	"github.com/spacemeshos/go-spacemesh/hare/pb"
	"github.com/spacemeshos/go-spacemesh/log"
	"sync"
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
	oracle    Rolacle // roles oracle
	network   NetworkConnection
	startTime time.Time
	inbox     chan Byteable
	abort     chan struct{}
	round     uint32
	mutex     *sync.Mutex
}

func (algo *Algo) NewAlgo(s Set, oracle Rolacle) {
	algo.k = 0
	algo.ki = 0
	algo.s = s
	algo.mutex = &sync.Mutex{}
}

func (algo *Algo) Start() {
	algo.startTime = time.Now()

	go algo.Listen()

	algo.do()
}

func (algo *Algo) Listen() {
	log.Info("Start listening")
	algo.network.RegisterProtocol(ProtoName)

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

func (algo *Algo) handleMessage(msg Byteable) {
	hareMsg := &pb.HareMessage{}
	proto.Unmarshal(msg.Bytes(), hareMsg)

	/*if time.Now().After(algo.startTime.Add(time.Duration(algo.k) * RoundDuration)) {
		return // ignore late msg
	}
	*/

	algo.mutex.Lock()

	if getRoundOf(hareMsg) != algo.round { // verify round
		return
	}

	// TODO: verify role

	// add to list of msg
	//
	//

	algo.mutex.Unlock()
}

func (algo *Algo) do() {
	ticker := time.NewTicker(RoundDuration)
	for t := range ticker.C { // WHEN DOES IT STOP?
		log.Info("Next round: %d, %d", algo.round, t)

		algo.mutex.Lock()

		// switch round
		// do round

		algo.round++
		//init data

		algo.mutex.Unlock()
	}

}

func getRoundOf(msg *pb.HareMessage) uint32 {
	if msg.Type > 3 {
		panic("Unknown message type")
	}

	return msg.Type
}

type PrePost interface {
	DoPre(d Knowledge)
	DoPost(d Knowledge)
}

type Round0 struct {
	pb.HareMessage
}

func (*Round0) DoPre() {

}
