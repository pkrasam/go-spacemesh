package hare

import (
	"time"
)

const ProtoName = "HARE_PROTOCOL"
const RoundDuration = time.Second * time.Duration(15)
const (
	Status   = 0 // round 0
	Proposal = 1 // round 1
	Commit   = 2 // round 2
	Notify   = 3 // round 3
)

type BLockId uint32 // TODO: replace with import core

type NetworkConnection interface {
	Broadcast(message []byte) (int, error)
	RegisterProtocol(protoName string) chan Byteable
}

type Byteable interface {
	Bytes() []byte
}

type Rolacle interface {
	Role(i uint32, r uint32) uint32
}

type Certificate interface {
	// TODO: change to struct
	Validate() bool
}
