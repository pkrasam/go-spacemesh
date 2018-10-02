package p2p

import (
	"encoding/hex"
	"testing"
	"time"

	"errors"
	"github.com/gogo/protobuf/proto"
	"github.com/spacemeshos/go-spacemesh/crypto"
	"github.com/spacemeshos/go-spacemesh/p2p/config"
	"github.com/spacemeshos/go-spacemesh/p2p/dht"
	"github.com/spacemeshos/go-spacemesh/p2p/net"
	"github.com/spacemeshos/go-spacemesh/p2p/node"
	"github.com/spacemeshos/go-spacemesh/p2p/pb"
	"github.com/spacemeshos/go-spacemesh/p2p/timesync"
	"github.com/stretchr/testify/assert"
	"github.com/spacemeshos/go-spacemesh/p2p/service"
	"fmt"
	"math/rand"
)

func p2pTestInstance(t testing.TB, config config.Config) *swarm {
	port, err := node.GetUnboundedPort()
	assert.NoError(t, err, "Error getting a port", err)
	config.TCPPort = port
	p, err := newSwarm(config, true)//false)
	assert.NoError(t, err, "Error creating p2p stack, err: %v", err)
	return p
}

const exampleProtocol = "EX"
const examplePayload = "Example"

func TestNew(t *testing.T) {
	s, err := New(config.DefaultConfig())
	assert.NoError(t, err, err)
	assert.NotNil(t, s, "its nil")
	s.Shutdown()
}

func Test_newSwarm(t *testing.T) {
	s, err := newSwarm(config.DefaultConfig(), false)
	assert.NoError(t, err)
	assert.NotNil(t, s)
	s.Shutdown()
}

func TestSwarm_signMessage(t *testing.T) {
	n, _ := node.GenerateTestNode(t)
	pm := &pb.ProtocolMessage{
		Metadata: newProtocolMessageMetadata(n.PublicKey(), exampleProtocol, false),
		Payload:  []byte(examplePayload),
	}

	bin, err := proto.Marshal(pm)
	assert.NoError(t, err)

	asig, err := n.PrivateKey().Sign(bin)

	assert.NoError(t, err)

	err = signMessage(n.PrivateKey(), pm)
	assert.NoError(t, err, err)
	assert.Equal(t, pm.Metadata.AuthorSign, hex.EncodeToString(asig))

}

func TestSwarm_Shutdown(t *testing.T) {
	s, err := newSwarm(config.DefaultConfig(), false)
	assert.NoError(t, err)
	s.Shutdown()

	select {
	case _, ok := <-s.shutdown:
		assert.False(t, ok)
	case <-time.After(1 * time.Second):
		t.Error("Failed to shutdown")
	}
}

func TestSwarm_processMessage(t *testing.T) {
	s := swarm{}
	s.lNode, _ = node.GenerateTestNode(t)
	r := node.GenerateRandomNodeData()
	c := &net.ConnectionMock{}
	c.SetRemotePublicKey(r.PublicKey())
	ime := net.IncomingMessageEvent{Message: nil, Conn: c}
	s.processMessage(ime) // should error

	assert.True(t, c.Closed())
}

func TestSwarm_updateConnection(t *testing.T) {
	p := &swarm{}
	p.lNode, _ = node.GenerateTestNode(t)
	dhtmock := &dht.MockDHT{}
	p.dht = dhtmock
	c := &net.ConnectionMock{}

	p.updateConnection(c)

	assert.Equal(t, dhtmock.UpdateCount(), 0)

	r := node.GenerateRandomNodeData()
	c.SetRemotePublicKey(r.PublicKey())

	p.updateConnection(c)

	assert.Equal(t, dhtmock.UpdateCount(), 1)
}

func TestSwarm_authAuthor(t *testing.T) {
	// create a message

	priv, pub, err := crypto.GenerateKeyPair()

	assert.NoError(t, err, err)
	assert.NotNil(t, priv)
	assert.NotNil(t, pub)

	pm := &pb.ProtocolMessage{
		Metadata: newProtocolMessageMetadata(pub, exampleProtocol, false),
		Payload:  []byte(examplePayload),
	}
	ppm, err := proto.Marshal(pm)
	assert.NoError(t, err, "cant marshal msg ", err)

	// sign it
	s, err := priv.Sign(ppm)
	assert.NoError(t, err, "cant sign ", err)
	ssign := hex.EncodeToString(s)

	pm.Metadata.AuthorSign = ssign

	vererr := authAuthor(pm)
	assert.NoError(t, vererr)

	priv2, pub2, err := crypto.GenerateKeyPair()

	assert.NoError(t, err, err)
	assert.NotNil(t, priv2)
	assert.NotNil(t, pub2)

	s, err = priv2.Sign(ppm)
	assert.NoError(t, err, "cant sign ", err)
	ssign = hex.EncodeToString(s)

	pm.Metadata.AuthorSign = ssign

	vererr = authAuthor(pm)
	assert.Error(t, vererr)
}

func TestSwarm_SignAuth(t *testing.T) {
	n, _ := node.GenerateTestNode(t)
	pm := &pb.ProtocolMessage{
		Metadata: newProtocolMessageMetadata(n.PublicKey(), exampleProtocol, false),
		Payload:  []byte(examplePayload),
	}

	err := signMessage(n.PrivateKey(), pm)
	assert.NoError(t, err)

	err = authAuthor(pm)

	assert.NoError(t, err)
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func sendDirectMessage(t *testing.T, sender *swarm, recvPub string, inChan chan service.Message) {
	//payload := []byte(RandString(10))
	payload := []byte(examplePayload)
	err := sender.SendMessage(recvPub, exampleProtocol, payload)
	assert.NoError(t, err)
	select {
	case msg := <-inChan:
		//assert.Equal(t, msg.Data(), payload)
		//assert.Equal(t, msg.Sender().String(), sender.lNode.String())
		fmt.Println("GOT ", string(msg.Data()[:]))
		break
	case <-time.After(1 * time.Second):
		t.Error("Took too much time to recieve")
	}
}

func TestSwarm_RoundTrip(t *testing.T) {
	//t.Skip()
	rand.Seed(time.Now().UnixNano())

	p1 := p2pTestInstance(t, config.DefaultConfig())
	p2 := p2pTestInstance(t, config.DefaultConfig())

	exchan1 := p1.RegisterProtocol(exampleProtocol)
	assert.Equal(t, exchan1, p1.protocolHandlers[exampleProtocol])
	exchan2 := p2.RegisterProtocol(exampleProtocol)
	assert.Equal(t, exchan2, p2.protocolHandlers[exampleProtocol])

	err := p2.SendMessage(p1.lNode.String(), exampleProtocol, []byte(examplePayload))
	assert.Error(t, err, "ERR") // should'nt be in routing table
	p2.dht.Update(p1.lNode.Node)

	//sendDirectMessage(t, p2, p1.localNode().String(), exchan1)
	//sendDirectMessage(t, p1, p2.localNode().String(), exchan2)

	//rand.Seed(time.Now().UnixNano())
	for i := 0; i < 200; i++ {
		//r := rand.Int31n(2)
		//if r == 0 {
			go sendDirectMessage(t, p2, p1.localNode().String(), exchan1)
			//time.Sleep(100 * time.Millisecond)
		//} else {
		//	go sendDirectMessage(t, p1, p2.localNode().String(), exchan2)
		//}
	}
	time.Sleep(2 * time.Second)
}

func TestSwarm_RoundTrip2(t *testing.T) {
	//t.Skip()
	p1 := p2pTestInstance(t, config.DefaultConfig())
	p2 := p2pTestInstance(t, config.DefaultConfig())

	exchan1 := p1.RegisterProtocol(exampleProtocol)
	assert.Equal(t, exchan1, p1.protocolHandlers[exampleProtocol])
	exchan2 := p2.RegisterProtocol(exampleProtocol)
	assert.Equal(t, exchan2, p2.protocolHandlers[exampleProtocol])

	err := p2.SendMessage(p1.lNode.String(), exampleProtocol, []byte(examplePayload))
	assert.Error(t, err, "ERR") // should'nt be in routing table
	p2.dht.Update(p1.lNode.Node)

	e1, conn1, msg1 := p2.SendMessageTest(p1.localNode().String(), exampleProtocol, []byte(examplePayload))
	assert.NoError(t, e1)

	e2, conn2, msg2 := p2.SendMessageTest(p1.localNode().String(), exampleProtocol, []byte(examplePayload))
	assert.NoError(t, e2)

	conn1.Send(msg1)
	select {
	case msg := <-exchan1:
		assert.Equal(t, msg.Data(), []byte(examplePayload))
		assert.Equal(t, msg.Sender().String(), p2.lNode.String())
		fmt.Println("GOT ", string(msg.Data()[:]))
		break
	case <-time.After(1 * time.Second):
		t.Error("Took too much time to recieve")
	}

	conn2.Send(msg2)
	select {
	case msg := <-exchan1:
		assert.Equal(t, msg.Data(), []byte(examplePayload))
		assert.Equal(t, msg.Sender().String(), p2.lNode.String())
		fmt.Println("GOT ", string(msg.Data()[:]))
		break
	case <-time.After(1 * time.Second):
		t.Error("Took too much time to recieve")
	}

	//sendDirectMessage(t, p2, p1.localNode().String(), exchan1)
	//sendDirectMessage(t, p1, p2.localNode().String(), exchan2)
	//
	////rand.Seed(time.Now().UnixNano())
	//for i := 0; i < 20; i++ {
	//	//r := rand.Int31n(2)
	//	//if r == 0 {
	//	go sendDirectMessage(t, p2, p1.localNode().String(), exchan1)
	//	//time.Sleep(100 * time.Millisecond)
	//	//} else {
	//	//	go sendDirectMessage(t, p1, p2.localNode().String(), exchan2)
	//	//}
	//}
	time.Sleep(1 * time.Second)
}

func Test_newProtocolMessageMeatadata(t *testing.T) {
	//newProtocolMessageMetadata()
	_, pk, _ := crypto.GenerateKeyPair()
	const gossip = false

	assert.NotNil(t, pk)

	meta := newProtocolMessageMetadata(pk, exampleProtocol, gossip)

	assert.NotNil(t, meta, "should be a metadata")
	assert.Equal(t, meta.Timestamp, time.Now().Unix())
	assert.Equal(t, meta.ClientVersion, config.ClientVersion)
	assert.Equal(t, meta.AuthPubKey, pk.Bytes())
	assert.Equal(t, meta.Protocol, exampleProtocol)
	assert.Equal(t, meta.Gossip, gossip)
	assert.Equal(t, meta.AuthorSign, "")

}

func TestSwarm_onRemoteClientMessage(t *testing.T) {
	p := p2pTestInstance(t, config.DefaultConfig())
	nmock := &net.ConnectionMock{}
	nmock.SetRemotePublicKey(node.GenerateRandomNodeData().PublicKey())

	// Test bad format

	msg := []byte("badbadformat")
	imc := net.IncomingMessageEvent{nmock, msg}
	err := p.onRemoteClientMessage(imc)
	assert.Equal(t, err, ErrBadFormat)

	// Test out of sync

	realmsg := &pb.CommonMessageData{
		SessionId: []byte("test"),
		Payload:   []byte("test"),
		Timestamp: time.Now().Add(timesync.MaxAllowedMessageDrift + time.Minute).Unix(),
	}
	bin, _ := proto.Marshal(realmsg)

	imc.Message = bin

	err = p.onRemoteClientMessage(imc)
	assert.Equal(t, err, ErrOutOfSync)

	// Test no payload

	cmd := &pb.CommonMessageData{
		SessionId: []byte("test"),
		Payload:   []byte(""),
		Timestamp: time.Now().Unix(),
	}

	bin, _ = proto.Marshal(cmd)
	imc.Message = bin
	err = p.onRemoteClientMessage(imc)

	assert.Equal(t, err, ErrNoPayload)

	// Test No Session

	cmd.Payload = []byte("test")

	bin, _ = proto.Marshal(cmd)
	imc.Message = bin

	err = p.onRemoteClientMessage(imc)
	assert.Equal(t, err, ErrNoSession)

	// Test bad session

	session := &net.SessionMock{}
	session.SetDecrypt(nil, errors.New("fail"))
	imc.Conn.SetSession(session)

	err = p.onRemoteClientMessage(imc)
	assert.Equal(t, err, ErrFailDecrypt)

	// Test bad format again

	session.SetDecrypt([]byte("wont_format_fo_protocol_message"), nil)

	err = p.onRemoteClientMessage(imc)
	assert.Equal(t, err, ErrBadFormat)

	// Test bad auth sign

	goodmsg := &pb.ProtocolMessage{
		Metadata: newProtocolMessageMetadata(p.lNode.PublicKey(), exampleProtocol, false), // not signed
		Payload:  []byte(examplePayload),
	}

	goodbin, _ := proto.Marshal(goodmsg)

	cmd.Payload = goodbin
	bin, _ = proto.Marshal(cmd)
	imc.Message = bin
	session.SetDecrypt(goodbin, nil)

	err = p.onRemoteClientMessage(imc)
	assert.Equal(t, err, ErrAuthAuthor)

	// Test no protocol

	err = signMessage(p.lNode.PrivateKey(), goodmsg)
	assert.NoError(t, err, err)

	goodbin, _ = proto.Marshal(goodmsg)
	cmd.Payload = goodbin
	bin, _ = proto.Marshal(cmd)
	imc.Message = bin
	session.SetDecrypt(goodbin, nil)

	err = p.onRemoteClientMessage(imc)
	assert.Equal(t, err, ErrNoProtocol)

	// Test no err

	c := p.RegisterProtocol(exampleProtocol)
	go func() { <-c }()

	err = p.onRemoteClientMessage(imc)

	assert.NoError(t, err)
}

func TestSession(t *testing.T) {
	type sp struct {
		s      *swarm
		protoC chan service.Message
	}

	numPeers, connections := 10, 5

	nodes := make([]*swarm, numPeers)
	chans := make([]chan service.Message, numPeers)
	nchan := make(chan *sp, numPeers)

	cfg := config.DefaultConfig()
	cfg.SwarmConfig.RandomConnections = connections
	cfg.SwarmConfig.Bootstrap = false
	bn := p2pTestInstance(t, cfg)
	// TODO: write protocol matching. so we won't crash connections because bad protocol messages.
	// if we're after protocol matching then we can crash the connection since its probably malicious
	//bn.RegisterProtocol("gossip") // or else it will crash connections

	//err := bn.Start()
	//assert.NoError(t, err, "Bootnode didnt work")

	cfg2 := config.DefaultConfig()
	cfg2.SwarmConfig.RandomConnections = connections
	cfg2.SwarmConfig.Bootstrap = true
	cfg2.SwarmConfig.BootstrapNodes = []string{node.StringFromNode(bn.lNode.Node)}
	for i := 0; i < numPeers; i++ {
		go func() {
			nod := p2pTestInstance(t, cfg2)
			if nod == nil {
				t.Error("ITS NIL WTF")
			}
			nodchan := nod.RegisterProtocol("gossip") // this is example
			//err := nod.Start()
			//assert.NoError(t, err, err)
			nchan <- &sp{nod, nodchan}
		}()
	}

	i := 0
	for n := range nchan {
		nodes[i] = n.s
		chans[i] = n.protoC

		i++
		if i >= numPeers {
			close(nchan)
		}
	}

	fmt.Println(" ################################################ ALL PEERS BOOTSTRAPPED ################################################")
/*
	msg := []byte("gossip")
	fmt.Println(" ################################################ GOSSIPING ################################################")
	b := time.Now()
	err = nodes[0].Broadcast("gossip", msg)

	fmt.Printf("%v GOSSIPED TO %v, err=%v\r\n", nodes[0].lNode.String(), err)

	var got int32 = 0
	didntget := make([]*swarm, 0)
	//var wg sync.WaitGroup

	for c := range chans {
		var resp service.Message
		timeout := time.NewTimer(time.Second * 10)
		select {
		case resp = <-chans[c]:
		case <-timeout.C:
			didntget = append(didntget, nodes[c])
			continue
		}

		if bytes.Equal(resp.Data(), msg) {
			nodes[c].lNode.Info("GOT THE gossip MESSAge ", atomic.AddInt32(&got, 1))
		}
	}
	//wg.Wait()
	bn.LocalNode().Info("THIS IS GOT ", got)
	assert.Equal(t, got, int32(numPeers))
	bn.lNode.Info("message spread to %v peers in %v", got, time.Since(b))
	didnt := ""
	for i := 0; i < len(didntget); i++ {
		didnt += fmt.Sprintf("%v\r\n", didntget[i].lNode.String())
	}
	bn.lNode.Info("didnt get : %v", didnt)
	time.Sleep(time.Millisecond * 1000) // to see the log
*/
}