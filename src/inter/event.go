package inter

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/golang/protobuf/proto"

	"github.com/Fantom-foundation/go-lachesis/src/crypto"
	"github.com/Fantom-foundation/go-lachesis/src/hash"
	"github.com/Fantom-foundation/go-lachesis/src/inter/wire"
)

// Event is a poset event.
type Event struct {
	Index                uint64
	Creator              hash.Peer
	Parents              hash.Events
	LamportTime          Timestamp
	InternalTransactions []*InternalTransaction
	ExternalTransactions ExtTxns
	Sign                 string
	CreatorSeq           int64
	FirstDescendantsSeq  []int64 // 0 <= FirstDescendantsSeq[i] <= 9223372036854775807
	LastAncestorsSeq     []int64 // -1 <= LastAncestorsSeq[i] <= 9223372036854775807
	FirstDescendants     []hash.Event
	LastAncestors        []hash.Event

	hash hash.Event // cache for .Hash()
}

// SignBy signs event by private key.
func (e *Event) SignBy(priv *crypto.PrivateKey) error {
	eventHash := e.Hash()

	R, S, err := priv.Sign(eventHash.Bytes())
	if err != nil {
		return err
	}

	e.Sign = crypto.EncodeSignature(R, S)
	return nil
}

// Verify sign event by public key.
func (e *Event) Verify(pubKey *crypto.PublicKey) bool {
	if pubKey == nil {
		log.Fatal("can't verify without key")
	}

	if e.Sign == "" {
		return false
	}

	eventHash := e.Hash()
	r, s, err := crypto.DecodeSignature(string(e.Sign))
	if err != nil {
		log.Fatal(err)
	}

	return pubKey.Verify(eventHash.Bytes(), r, s)
}

// Hash calcs hash of event.
func (e *Event) Hash() hash.Event {
	if e.hash.IsZero() {
		e.hash = EventHashOf(e)
	}
	return e.hash
}

// FindInternalTxn find transaction in event's internal transactions list.
// TODO: use map
func (e *Event) FindInternalTxn(idx hash.Transaction) *InternalTransaction {
	for _, txn := range e.InternalTransactions {
		if TransactionHashOf(e.Creator, txn.Index) == idx {
			return txn
		}
	}
	return nil
}

// String returns string representation.
func (e *Event) String() string {
	return fmt.Sprintf("Event{%s, %s, t=%d}", e.Hash().String(), e.Parents.String(), e.LamportTime)
}

// ToWire converts to proto.Message.
func (e *Event) ToWire() (*wire.Event, *wire.Event_ExtTxnsValue) {
	if e == nil {
		return nil, nil
	}

	extTxns, extTxnsHash := e.ExternalTransactions.ToWire()

	return &wire.Event{
		Index:                e.Index,
		Creator:              e.Creator.Hex(),
		CreatorSeq:           e.CreatorSeq,
		Parents:              e.Parents.ToWire(),
		LamportTime:          uint64(e.LamportTime),
		InternalTransactions: InternalTransactionsToWire(e.InternalTransactions),
		ExternalTransactions: extTxnsHash,
		Sign:                 e.Sign,
	}, extTxns
}

// SelfParent returns self parent from event. If it returns "false" then a self
// parent is missing.
func (e *Event) SelfParent() (hash.Event, bool) {
	for parent := range e.Parents {
		if bytes.Equal(e.Creator.Bytes(), parent.Bytes()) {
			return parent, true
		}
	}

	return hash.Event{}, false
}

// OtherParents returns "other parents" sorted slice.
func (e *Event) OtherParents() hash.EventsSlice {
	parents := e.Parents.Copy()

	sp, ok := e.SelfParent()
	if ok {
		delete(parents, sp)
	}

	events := parents.Slice()
	sort.Sort(events)

	return events
}

// WireToEvent converts from wire.
func WireToEvent(w *wire.Event) *Event {
	if w == nil {
		return nil
	}
	return &Event{
		Index:                w.Index,
		Creator:              hash.HexToPeer(w.Creator),
		CreatorSeq:           w.CreatorSeq,
		Parents:              hash.WireToEventHashes(w.Parents),
		LamportTime:          Timestamp(w.LamportTime),
		InternalTransactions: WireToInternalTransactions(w.InternalTransactions),
		ExternalTransactions: WireToExtTxns(w),
		Sign:                 w.Sign,
	}
}

/*
 * Utils:
 */

// EventHashOf calcs hash of event.
func EventHashOf(e *Event) hash.Event {
	w, _ := e.ToWire()
	w.Sign = ""

	buf, err := proto.Marshal(w)
	if err != nil {
		log.Fatal(err)
	}

	return hash.Event(hash.Of(buf))
}

// FakeFuzzingEvents generates random independent events for test purpose.
func FakeFuzzingEvents() (res []*Event) {
	creators := []hash.Peer{
		{},
		hash.FakePeer(),
		hash.FakePeer(),
		hash.FakePeer(),
	}
	parents := []hash.Events{
		hash.FakeEvents(0),
		hash.FakeEvents(1),
		hash.FakeEvents(8),
	}
	extTxns := [][][]byte{
		nil,
		[][]byte{
			[]byte("fake external transaction 1"),
			[]byte("fake external transaction 2"),
		},
	}
	i := 0
	for c := 0; c < len(creators); c++ {
		seq := int64(0)
		for p := 0; p < len(parents); p++ {
			e := &Event{
				Index:      uint64(p),
				Creator:    creators[c],
				CreatorSeq: seq,
				Parents:    parents[p],
				InternalTransactions: []*InternalTransaction{
					{
						Amount:   999,
						Receiver: creators[c],
					},
				},
				ExternalTransactions: ExtTxns{
					Value: extTxns[i%len(extTxns)],
				},
			}
			res = append(res, e)
			i++
			seq++
		}
	}
	return
}
