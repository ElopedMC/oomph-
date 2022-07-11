package check

import (
	"math"

	"github.com/oomph-ac/oomph/entity"
	"github.com/oomph-ac/oomph/game"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// KillAuraB checks if a player is attacking too many entities at once.
type KillAuraB struct {
	entities map[uint64]*entity.Entity
	basic
}

// NewKillAuraB creates a new KillAuraB check.
func NewKillAuraB() *KillAuraB {
	return &KillAuraB{entities: make(map[uint64]*entity.Entity)}
}

// Name ...
func (*KillAuraB) Name() (string, string) {
	return "Killaura", "B"
}

// Description ...
func (*KillAuraB) Description() string {
	return "This checks if a player is attacking more than one entity at once."
}

// MaxViolations ...
func (*KillAuraB) MaxViolations() float64 {
	return 15
}

// Process ...
func (k *KillAuraB) Process(p Processor, pk packet.Packet) bool {
	switch pk := pk.(type) {
	case *packet.InventoryTransaction:
		if data, ok := pk.TransactionData.(*protocol.UseItemOnEntityTransactionData); ok && data.ActionType == protocol.UseItemOnEntityActionAttack {
			if e, ok := p.SearchEntity(data.TargetEntityRuntimeID); ok {
				k.entities[data.TargetEntityRuntimeID] = e
			}
		}
	case *packet.PlayerAuthInput:
		if len(k.entities) > 1 {
			minDist := math.MaxFloat64
			for id, data := range k.entities {
				for subId, subData := range k.entities {
					if subId != id {
						minDist = math.Min(minDist, game.AABBVectorDistance(data.AABB().Grow(0.1).Translate(data.LastPosition()), subData.LastPosition()))
					}
				}
			}
			if minDist < math.MaxFloat64 && minDist > 1.5 {
				p.Flag(k, k.violationAfterTicks(p.ClientTick(), 40), map[string]any{
					"Minimum Distance": game.Round(minDist, 2),
					"Entities":         len(k.entities),
				})
				return true
			}
		}
		k.entities = make(map[uint64]*entity.Entity)
	}

	return false
}
