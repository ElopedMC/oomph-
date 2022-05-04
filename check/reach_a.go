package check

import (
	"math"

	"github.com/go-gl/mathgl/mgl64"
	"github.com/oomph-ac/oomph/game"

	"github.com/df-mc/dragonfly/server/entity/physics/trace"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// ReachA checks if a player has an abnormal amount of reach.
type ReachA struct {
	awaitingTick   bool
	inputMode      uint32
	attackedEntity uint64
	attackPos      mgl64.Vec3
	basic
}

// NewReachA creates a new ReachA check.
func NewReachA() *ReachA {
	return &ReachA{}
}

// Name ...
func (*ReachA) Name() (string, string) {
	return "Reach", "A"
}

// Description ...
func (*ReachA) Description() string {
	return "This checks if a player has an abnormal amount of reach."
}

// MaxViolations ...
func (*ReachA) MaxViolations() float64 {
	return 15
}

// Process ...
func (r *ReachA) Process(p Processor, pk packet.Packet) {
	switch pk := pk.(type) {
	case *packet.InventoryTransaction:
		if data, ok := pk.TransactionData.(*protocol.UseItemOnEntityTransactionData); ok && data.ActionType == protocol.UseItemOnEntityActionAttack {
			if p.GameMode() == packet.GameTypeSurvival || p.GameMode() == packet.GameTypeAdventure {
				r.attackedEntity = data.TargetEntityRuntimeID
				r.attackPos = game.Vec32To64(data.Position)
				if p.Sneaking() {
					r.attackPos[1] -= 0.08
				}
				if t, ok := p.SearchEntity(data.TargetEntityRuntimeID); ok && t.TeleportationTicks() >= 40 {
					if r.inputMode == packet.InputModeTouch {
						dist := game.AABBVectorDistance(t.AABB().Translate(t.Position()), r.attackPos)
						if dist > 3.15 {
							if r.Buff(1, 10) >= 5 {
								p.Flag(r, r.violationAfterTicks(p.ClientTick(), 600), map[string]interface{}{
									"Distance": game.Round(dist, 4),
									"Type":     "Raw",
								})
							}
						} else {
							r.Buff(-0.05)
							r.violations = math.Max(r.violations-0.01, 0)
						}
					} else {
						r.awaitingTick = true
					}
				}
			}
		}
	case *packet.PlayerAuthInput:
		r.inputMode = pk.InputMode
		if r.awaitingTick {
			if t, ok := p.SearchEntity(r.attackedEntity); ok /*&& t.Player()*/ {
				e := p.Entity()
				cRot, lRot := e.Rotation(), e.LastRotation()
				cDv, lDv := game.DirectionVector(cRot.Z(), cRot.X()), game.DirectionVector(lRot.Z(), lRot.X())
				cPos, lPos := p.Entity().Position().Add(mgl64.Vec3{0, 1.62, 0}), r.attackPos
				if p.Sneaking() {
					cPos[1] -= 0.08
				}

				aabb := e.AABB().Translate(e.LastPosition())
				targetAABB := t.AABB().Grow(0.1).Translate(t.LastPosition())

				if !aabb.IntersectsWith(targetAABB) {
					dvDiff := cDv.Sub(lDv)
					posDiff := cPos.Sub(lPos)
					minDist, valid := 69000.0, false
					for i := 30; i != 0; i-- {
						uDv := lDv
						uPos := lPos
						if i != 0 {
							uDv = uDv.Add(dvDiff.Mul(float64(1 / i)))
							uPos = uPos.Add(posDiff.Mul(float64(1 / i)))
						}
						if ray, ok := trace.AABBIntercept(targetAABB, uPos, uPos.Add(uDv.Mul(14))); ok {
							minDist = math.Min(minDist, ray.Position().Sub(uPos).Len())
							valid = true
						}
					}
					if valid {
						if minDist >= 3.1 && math.Abs(minDist-game.AABBVectorDistance(targetAABB, r.attackPos)) < 0.4 {
							if r.Buff(1, 6) >= 3 {
								p.Flag(r, r.violationAfterTicks(p.ClientTick(), 600), map[string]interface{}{
									"Distance": game.Round(minDist, 2),
									"Type":     "Raycast",
								})
							}
						} else {
							r.Buff(-0.01)
							r.violations = math.Max(r.violations-0.0075, 0)
						}
					}
				}
			}
			r.awaitingTick = false
		}
	}
}
