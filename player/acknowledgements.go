package player

import (
	"math/rand"
	"sync"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// @whoever.the.fuck.touching.networkstacklatency.packet how about instead
// of touching my packet, you touch the issue I sent that's almost a year old on the bug tracker.
// https://bugs.mojang.com/browse/MCPE-158716
// pls :(((
const (
	DefaultNetworkStackLatencyDivider     = 1_000_000
	PlaystationNetworkStackLatencyDivider = 1_000
)

type Acknowledgements struct {
	AcknowledgeMap   map[int64][]func()
	CurrentTimestamp int64

	HasTicked  bool
	LegacyMode bool

	acknowledgementOrder []int64
	awaitResTicks        uint64
	mu                   sync.Mutex
}

func (a *Acknowledgements) UseLegacy(b bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.LegacyMode = b
}

// Add adds an acknowledgement to run in the future to the map of acknowledgements.
func (a *Acknowledgements) Add(f func()) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.AcknowledgeMap[a.CurrentTimestamp] = append(a.AcknowledgeMap[a.CurrentTimestamp], f)
}

// AddMap adds a list of functions in the AcknowledgeMap with a specified timestamp.
func (a *Acknowledgements) AddMap(m []func(), t int64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.AcknowledgeMap[t] = m
}

// GetMap returns the list of functions in the AcknowledgeMap with the specified timestamp.
func (a *Acknowledgements) GetMap(t int64) ([]func(), bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	m, ok := a.AcknowledgeMap[t]
	if !ok {
		return nil, false
	}

	return m, true
}

// tryHandle checks if an acknowledgement ID has a map of functions, and if so, executes them.
// If tryHandle() ends up finding a map, it returns true, and if not, returns false.
func (a *Acknowledgements) tryHandle(i int64) bool {
	a.mu.Lock()
	calls, ok := a.AcknowledgeMap[i]
	a.mu.Unlock()

	if ok {
		a.awaitResTicks = 0
		a.Remove(i)
		for _, f := range calls {
			f()
		}
	}

	return ok
}

// Remove removes an acknowledgement from the map of acknowledgements.
func (a *Acknowledgements) Remove(i int64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.AcknowledgeMap, i)
}

// Refresh sets a new timestamp for the acknowledgements.
func (a *Acknowledgements) Refresh() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create a random timestamp, and ensure that it is not already being used.
	for {
		a.CurrentTimestamp = int64(rand.Uint32())

		// On clients supposedly <1.20, the timestamp is rounded to the thousands.
		if a.LegacyMode {
			a.CurrentTimestamp *= 1000
		}

		if _, ok := a.AcknowledgeMap[a.CurrentTimestamp]; !ok {
			break
		}
	}
}

// Create creats a new acknowledgement packet and returns it.
func (a *Acknowledgements) Create() *packet.NetworkStackLatency {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.AcknowledgeMap[a.CurrentTimestamp]) == 0 {
		return nil
	}

	return &packet.NetworkStackLatency{
		Timestamp:     a.CurrentTimestamp,
		NeedsResponse: true,
	}
}

// Validate checks if the client is still responding to acknowledgements sent to it. If it's determined that
// the client is not responding despite ticking, this function will return false. This is to prevent modified
// clients from breaking certain systems by simply ignoring acknowledgements we send.
func (a *Acknowledgements) Validate() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.HasTicked {
		return true
	}

	a.HasTicked = false

	if len(a.AcknowledgeMap) == 0 {
		a.awaitResTicks = 0
		return true
	}

	a.awaitResTicks++
	return a.awaitResTicks < 200
}

// Clear clears all existing acknowledgements.
func (a *Acknowledgements) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.AcknowledgeMap = make(map[int64][]func())
	a.awaitResTicks = 0
	a.acknowledgementOrder = []int64{}
}

// Handle gets the acknowledgement in the map with the timestamp given in the function. If there is no acknowledgement
// found, then false is returned. If there is an acknowledgement, then it is removed from the map and the function is ran.
// "awaitResTicks" will also bet set to 0, as the client has responded to an acknowledgement.
func (p *Player) handleNetworkStackLatency(i int64, ps4 bool) bool {
	a := p.Acknowledgements()
	if a == nil {
		return false
	}

	var ok bool
	if a.LegacyMode {
		ok = a.tryHandle(i)
		if ps4 && !ok {
			i /= 1000
			ok = a.tryHandle(i)
		} else if ps4 {
			delete(a.AcknowledgeMap, i/1000)
		}

		return ok
	}

	if ps4 {
		i /= PlaystationNetworkStackLatencyDivider
	} else {
		i /= DefaultNetworkStackLatencyDivider
	}

	ok = a.tryHandle(i)
	if ps4 && ok {
		i /= PlaystationNetworkStackLatencyDivider
		ok = a.tryHandle(i)
	}

	return ok
}

// SendAck sends an acknowledgement packet to the client.
func (p *Player) SendAck() {
	acks := p.Acknowledgements()
	if pk := acks.Create(); pk != nil {
		defer acks.Refresh()

		buf, ok := acks.GetMap(acks.CurrentTimestamp)
		if !ok {
			return
		}

		if len(buf) == 0 {
			acks.Remove(acks.CurrentTimestamp)
			return
		}

		// It seems that when the client is changing dimensions, they do not send back NetworkStackLatency.
		if p.inDimensionChange {
			acks.tryHandle(acks.CurrentTimestamp)
			acks.Remove(acks.CurrentTimestamp)
			return
		}

		// NetworkStackLatency behavior on Playstation devices sends the original timestamp
		// back to the server for a certain period of time (?) but then starts dividing the timestamp later on.
		if p.ClientData().DeviceOS == protocol.DeviceOrbis && acks.LegacyMode {
			acks.AddMap(buf, acks.CurrentTimestamp/1000)
		}

		//acks.acknowledgementOrder = append(acks.acknowledgementOrder, expectedTimestamp)
		p.conn.WritePacket(pk)
	}
}
