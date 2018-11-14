package hare

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"math/rand"
	"strconv"
)

const (
	Passive = 0
	Active = 1
	Leader = 2
)

type Rolacle interface {
	Role(rq RoleRequest) RoleProof
	ValidateRole(role uint8, sig []byte) bool
}

type RoleRequest struct {
	layerId LayerId
	k uint32
	nodeId string
}

func (roleRequest *RoleRequest) bytes() []byte {
	var binBuf bytes.Buffer
	binary.Write(&binBuf, binary.BigEndian, roleRequest)

	return binBuf.Bytes()
}


type RoleProof struct {
	role uint8
	sig []byte
}

type MockOracle struct {
	roles map[int]uint8
	isLeaderTaken bool
}

func (mockOracle *MockOracle) NewMockOracle() {
	mockOracle.roles = make(map[int]uint8)
	mockOracle.isLeaderTaken = false
}



// TODO:
// check round
// assign first leader OR randomly assign actives
// keep knowledge for later
// return result

func (mockOracle *MockOracle) Role(rq RoleRequest) RoleProof {
	i, _ := strconv.Atoi(string(rq.bytes()))
	sha := sha1.Sum(rq.bytes())

	// check if exist
	if role, exist := mockOracle.roles[i]; exist {
		return RoleProof{role, sha[:]}
	}

	// randomize new role
	r := uint8(rand.Intn(100))

	role := uint8(0)
	if r == 100 && !mockOracle.isLeaderTaken { // one leader
		role = Leader
		mockOracle.isLeaderTaken = true
	} else if r > 80 { // ~20% active
		role = Active
	}

	mockOracle.roles[i] = role

	return RoleProof{role, sha[:]}
}

func (mockOracle *MockOracle) ValidateRole(role uint8, sig []byte) bool {
	return true
}

