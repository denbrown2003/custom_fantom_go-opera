/*
Copyright 2017 Mosaic Networks Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package hashgraph

import (
	"crypto/ecdsa"
	"fmt"
	"testing"

	"github.com/arrivets/go-swirlds/crypto"
)

/*
|   e12 |
|   | \ |
|   |   e20
|   | / |
|   /   |
| / |   |
e01 |   |
| \ |   |
e0  e1  e2
0   1   2
*/
func initHashgraph() (Hashgraph, map[string]string) {
	n := 3
	index := make(map[string]string)

	nodes := []struct {
		Pub    []byte
		PubHex string
		Key    *ecdsa.PrivateKey
		Events []Event
	}{}

	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		pub := crypto.FromECDSAPub(&key.PublicKey)
		pubHex := fmt.Sprintf("0x%X", pub)
		event := NewEvent([][]byte{}, []string{"", ""}, pub)
		event.Sign(key)
		name := fmt.Sprintf("e%d", i)
		index[name] = event.Hex()
		events := []Event{event}
		node := struct {
			Pub    []byte
			PubHex string
			Key    *ecdsa.PrivateKey
			Events []Event
		}{Pub: pub, PubHex: pubHex, Key: key, Events: events}
		nodes = append(nodes, node)
	}

	event01 := NewEvent([][]byte{},
		[]string{nodes[0].Events[0].Hex(), nodes[1].Events[0].Hex()}, //e0 and e1
		nodes[0].Pub)
	event01.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, event01)
	index["e01"] = event01.Hex()

	event20 := NewEvent([][]byte{},
		[]string{nodes[2].Events[0].Hex(), nodes[0].Events[1].Hex()}, //e2 and e01
		nodes[2].Pub)
	event20.Sign(nodes[2].Key)
	nodes[2].Events = append(nodes[2].Events, event20)
	index["e20"] = event20.Hex()

	event12 := NewEvent([][]byte{},
		[]string{nodes[1].Events[0].Hex(), nodes[2].Events[1].Hex()}, //e1 and e20
		nodes[1].Pub)
	event12.Sign(nodes[1].Key)
	nodes[1].Events = append(nodes[1].Events, event12)
	index["e12"] = event12.Hex()

	hashgraph := NewHashgraph()
	for _, node := range nodes {
		for _, ev := range node.Events {
			hashgraph.Events[ev.Hex()] = ev
		}
	}
	return hashgraph, index
}

func TestAncestor(t *testing.T) {
	h, index := initHashgraph()

	//1 generation
	if !h.Ancestor(index["e01"], index["e0"]) {
		t.Fatal("e0 should be ancestor of e01")
	}
	if !h.Ancestor(index["e01"], index["e1"]) {
		t.Fatal("e1 should be ancestor of e01")
	}
	if !h.Ancestor(index["e20"], index["e01"]) {
		t.Fatal("e01 should be ancestor of e20")
	}
	if !h.Ancestor(index["e20"], index["e2"]) {
		t.Fatal("e2 should be ancestor of e20")
	}
	if !h.Ancestor(index["e12"], index["e20"]) {
		t.Fatal("e20 should be ancestor of e12")
	}
	if !h.Ancestor(index["e12"], index["e1"]) {
		t.Fatal("e1 should be ancestor of e12")
	}

	//2 generations
	if !h.Ancestor(index["e20"], index["e0"]) {
		t.Fatal("e0 should be ancestor of e20")
	}
	if !h.Ancestor(index["e20"], index["e1"]) {
		t.Fatal("e1 should be ancestor of e20")
	}
	if !h.Ancestor(index["e12"], index["e01"]) {
		t.Fatal("e01 should be ancestor of e12")
	}
	if !h.Ancestor(index["e12"], index["e2"]) {
		t.Fatal("e2 should be ancestor of e12")
	}

	//3 generations
	if !h.Ancestor(index["e12"], index["e0"]) {
		t.Fatal("e0 should be ancestor of e12")
	}
	if !h.Ancestor(index["e12"], index["e1"]) {
		t.Fatal("e1 should be ancestor of e12")
	}

	//false positive
	if h.Ancestor(index["e01"], index["e2"]) {
		t.Fatal("e2 should not be ancestor of e01")
	}

}

func TestSelfAncestor(t *testing.T) {
	h, index := initHashgraph()

	//1 generation
	if !h.SelfAncestor(index["e01"], index["e0"]) {
		t.Fatal("e0 should be self ancestor of e01")
	}
	if !h.SelfAncestor(index["e20"], index["e2"]) {
		t.Fatal("e2 should be self ancestor of e20")
	}
	if !h.SelfAncestor(index["e12"], index["e1"]) {
		t.Fatal("e1 should be self ancestor of e12")
	}

	//1 generation false negatives
	if h.SelfAncestor(index["e01"], index["e1"]) {
		t.Fatal("e1 should not be self ancestor of e01")
	}
	if h.SelfAncestor(index["e20"], index["e01"]) {
		t.Fatal("e01 should not be self ancestor of e20")
	}
	if h.SelfAncestor(index["e12"], index["e20"]) {
		t.Fatal("e20 should not be self ancestor of e12")
	}

	//2 generation false negative
	if h.SelfAncestor(index["e20"], index["e0"]) {
		t.Fatal("e0 should not be self ancestor of e20")
	}
	if h.SelfAncestor(index["e12"], index["e2"]) {
		t.Fatal("e2 should not be self ancestor of e12")
	}

}

/*
|   e12    |
|    | \   |
|    |   \ |
|    |    e20
|    |   / |
|    | /   |
|    /     |
|  / |     |
e01  |     |
| \  |     |
|   \|     |
|    |\    |
|    |  \  |
e0   e1 (a)e2
0    1     2

Node 2 Forks; events a and e2 are both created by node2, they are not self-parents
and yet they are both ancestors of event e20
*/
func initForkHashgraph() (Hashgraph, map[string]string) {
	n := 3
	index := make(map[string]string)

	nodes := []struct {
		Pub    []byte
		PubHex string
		Key    *ecdsa.PrivateKey
		Events []Event
	}{}

	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		pub := crypto.FromECDSAPub(&key.PublicKey)
		pubHex := fmt.Sprintf("0x%X", pub)
		event := NewEvent([][]byte{}, []string{"", ""}, pub)
		event.Sign(key)
		name := fmt.Sprintf("e%d", i)
		index[name] = event.Hex()
		events := []Event{event}
		node := struct {
			Pub    []byte
			PubHex string
			Key    *ecdsa.PrivateKey
			Events []Event
		}{Pub: pub, PubHex: pubHex, Key: key, Events: events}
		nodes = append(nodes, node)
	}

	//a and e2 need to have different hashes
	eventA := NewEvent([][]byte{[]byte("yo")}, []string{"", ""}, nodes[2].Pub)
	eventA.Sign(nodes[2].Key)
	nodes[2].Events = append(nodes[2].Events, eventA)
	index["a"] = eventA.Hex()

	event01 := NewEvent([][]byte{},
		[]string{nodes[0].Events[0].Hex(), nodes[2].Events[1].Hex()}, //e0 and A
		nodes[0].Pub)
	event01.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, event01)
	index["e01"] = event01.Hex()

	event20 := NewEvent([][]byte{},
		[]string{nodes[2].Events[0].Hex(), nodes[0].Events[1].Hex()}, //e2 and e01
		nodes[2].Pub)
	event20.Sign(nodes[2].Key)
	nodes[2].Events = append(nodes[2].Events, event20)
	index["e20"] = event20.Hex()

	event12 := NewEvent([][]byte{},
		[]string{nodes[1].Events[0].Hex(), nodes[2].Events[2].Hex()}, //e1 and e20
		nodes[1].Pub)
	event12.Sign(nodes[1].Key)
	nodes[1].Events = append(nodes[1].Events, event12)
	index["e12"] = event12.Hex()

	hashgraph := NewHashgraph()
	for _, node := range nodes {
		for _, ev := range node.Events {
			hashgraph.Events[ev.Hex()] = ev
		}
	}
	return hashgraph, index
}

func TestDetectFork(t *testing.T) {
	h, index := initForkHashgraph()

	//1 generation
	fork := h.DetectFork(index["e20"], index["a"])
	if !fork {
		t.Fatal("e20 should detect a fork under a")
	}
	fork = h.DetectFork(index["e20"], index["e2"])
	if !fork {
		t.Fatal("e20 should detect a fork under e2")
	}
	fork = h.DetectFork(index["e12"], index["e20"])
	if !fork {
		t.Fatal("e12 should detect a fork under e20")
	}

	//2 generations
	fork = h.DetectFork(index["e12"], index["a"])
	if !fork {
		t.Fatal("e12 should detect a fork under a")
	}
	fork = h.DetectFork(index["e12"], index["e2"])
	if !fork {
		t.Fatal("e12 should detect a fork under e2")
	}

	//false negatives
	fork = h.DetectFork(index["e01"], index["e0"])
	if fork {
		t.Fatal("e01 should not detect a fork under e0")
	}
	fork = h.DetectFork(index["e01"], index["a"])
	if fork {
		t.Fatal("e01 should not detect a fork under 'a'")
	}
	fork = h.DetectFork(index["e01"], index["e2"])
	if fork {
		t.Fatal("e01 should not detect a fork under e2")
	}
	fork = h.DetectFork(index["e20"], index["e01"])
	if fork {
		t.Fatal("e20 should not detect a fork under e01")
	}
	fork = h.DetectFork(index["e12"], index["e01"])
	if fork {
		t.Fatal("e12 should not detect a fork under e01")
	}
}

func TestSee(t *testing.T) {
	h, index := initForkHashgraph()

	if !h.See(index["e01"], index["e0"]) {
		t.Fatal("e01 should see e0")
	}
	if !h.See(index["e01"], index["a"]) {
		t.Fatal("e01 should see 'a'")
	}
	if !h.See(index["e20"], index["e0"]) {
		t.Fatal("e20 should see e0")
	}
	if !h.See(index["e20"], index["e01"]) {
		t.Fatal("e20 should see e01")
	}
	if !h.See(index["e12"], index["e01"]) {
		t.Fatal("e12 should see e01")
	}
	if !h.See(index["e12"], index["e0"]) {
		t.Fatal("e12 should see e0")
	}
	if !h.See(index["e12"], index["e1"]) {
		t.Fatal("e12 should see e1")
	}

	//fork
	if h.See(index["e20"], index["a"]) {
		t.Fatal("e20 should not see 'a' because of fork")
	}
	if h.See(index["e20"], index["e2"]) {
		t.Fatal("e20 should not see e2 because of fork")
	}
	if h.See(index["e12"], index["a"]) {
		t.Fatal("e12 should not see 'a' because of fork")
	}
	if h.See(index["e12"], index["e2"]) {
		t.Fatal("e12 should not see e2 because of fork")
	}
	if h.See(index["e12"], index["e20"]) {
		t.Fatal("e12 should not see e20 because of fork")
	}

}
