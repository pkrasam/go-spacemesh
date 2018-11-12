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

type BlockId uint32 // TODO: replace with import
type LayerId uint32 // TODO: replace with import

type Byteable interface {
	Bytes() []byte
}
