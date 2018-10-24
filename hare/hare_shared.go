package hare

import(
	"time"
)

const PROTO_NAME = "HARE_PROTOCOL"
const ROUND_DURATION = time.Second * time.Duration(15)

type BLockId uint32 // TODO: replace with import core

type NetworkConnection interface {
	Broadcast(message []byte) (int, error)
	RegisterProtocol(protoName string) chan IByteable
}

type IByteable interface {
	Bytes() []byte
}

type IRolacle interface {
	Role(i uint32, r uint32) uint32
}

type Certificate interface { // TODO: change to struct
	Validate() bool
}