package handler

import (
	"math/rand"

	"github.com/oomph-ac/oomph/player"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const HandlerIDAcknowledgements = "oomph:acknowledgements"

const (
	AckDivider = 1_000
)

// AcknowledgementHandler handles acknowledgements to the client, so that the anti-cheat knows the precise
// tick the client processed a certain action.
type AcknowledgementHandler struct {
	// LegacyMode is set to true if the client's protocol version is 1.20.0 or lower.
	LegacyMode bool
	// Playstation is set to true if the client is a Playstation client.
	Playstation bool

	// AckMap is a map of timestamps associated with a list of callbacks.
	// The callbacks are called when NetworkStackLatency is received from the client.
	AckMap map[int64][]func()
	// CurrentTimestamp is the current timestamp for acks, which is refreshed every server tick
	// where the connections are flushed.
	CurrentTimestamp int64
}

func NewAcknowledgementHandler() *AcknowledgementHandler {
	return &AcknowledgementHandler{
		AckMap: make(map[int64][]func()),
	}
}

func (AcknowledgementHandler) ID() string {
	return HandlerIDAcknowledgements
}

func (a *AcknowledgementHandler) HandleClientPacket(pk packet.Packet, p *player.Player) bool {
	switch pk := pk.(type) {
	case *packet.TickSync:
		a.Playstation = p.Conn().ClientData().DeviceOS == protocol.DeviceOrbis
		a.Refresh()
	case *packet.NetworkStackLatency:
		return !a.Execute(pk.Timestamp)
	}

	return true
}

func (*AcknowledgementHandler) HandleServerPacket(pk packet.Packet, p *player.Player) bool {
	return true
}

func (a *AcknowledgementHandler) OnTick(p *player.Player) {
	if pk := a.CreatePacket(); pk != nil {
		p.Conn().WritePacket(pk)
	}

	a.Refresh()
}

func (*AcknowledgementHandler) Defer() {
}

// AddCallback adds a callback to AckMap.
func (a *AcknowledgementHandler) AddCallback(callback func()) {
	a.AckMap[a.CurrentTimestamp] = append(a.AckMap[a.CurrentTimestamp], callback)
}

// Execute takes a timestamp, and looks for callbacks associated with it.
func (a *AcknowledgementHandler) Execute(timestamp int64) bool {
	if a.LegacyMode {
		return a.tryExecute(timestamp)
	}

	timestamp /= AckDivider
	if !a.Playstation {
		timestamp /= AckDivider
	}
	return a.tryExecute(timestamp)
}

// Refresh updates the AcknowledgementHandler's current timestamp with a random value.
func (a *AcknowledgementHandler) Refresh() {
	// Create a random timestamp, and ensure that it is not already being used.
	for {
		a.CurrentTimestamp = int64(rand.Uint32())

		// On clients supposedly <1.20, the timestamp is rounded to the thousands.
		if a.LegacyMode {
			a.CurrentTimestamp *= 1000
		}

		// Check if the timestamp is already being used, if not, break out of the loop.
		if _, ok := a.AckMap[a.CurrentTimestamp]; !ok {
			break
		}
	}
}

// CreatePacket creates a NetworkStackLatency packet with the current timestamp.
func (a *AcknowledgementHandler) CreatePacket() *packet.NetworkStackLatency {
	if len(a.AckMap[a.CurrentTimestamp]) == 0 {
		delete(a.AckMap, a.CurrentTimestamp)
		return nil
	}

	timestamp := a.CurrentTimestamp
	if a.LegacyMode && a.Playstation {
		timestamp /= AckDivider
	}

	return &packet.NetworkStackLatency{
		Timestamp:     timestamp,
		NeedsResponse: true,
	}
}

// tryExecute takes a timestamp, and looks for callbacks associated with it.
func (a *AcknowledgementHandler) tryExecute(timestamp int64) bool {
	callables, ok := a.AckMap[timestamp]
	if !ok {
		return false
	}

	for _, callable := range callables {
		callable()
	}

	delete(a.AckMap, timestamp)
	return true
}
