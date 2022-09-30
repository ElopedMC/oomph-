package check

import (
	"math"

	"github.com/df-mc/dragonfly/server/block/cube/trace"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/oomph-ac/oomph/game"
	"github.com/oomph-ac/oomph/utils"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const interpolationInterations float64 = 8

type ReachA struct {
	attackData                *protocol.UseItemOnEntityTransactionData
	currentEntPos, lastEntPos mgl64.Vec3
	cancelNext                bool
	secondaryBuffer           float64
	basic
}

func NewReachA() *ReachA {
	return &ReachA{}
}

func (*ReachA) Name() (string, string) {
	return "Reach", "A"
}

func (*ReachA) Description() string {
	return "This checks if a player's combat range is invalid."
}

func (*ReachA) MaxViolations() float64 {
	return 15
}

func (r *ReachA) Process(p Processor, pk packet.Packet) bool {
	if p.CombatMode() != utils.ModeSemiAuthoritative {
		return false
	}

	if t, ok := pk.(*packet.InventoryTransaction); ok {
		d, ok := t.TransactionData.(*protocol.UseItemOnEntityTransactionData)
		if !ok {
			return false
		}

		if d.ActionType != protocol.UseItemOnEntityActionAttack {
			return false
		}

		if r.cancelNext {
			r.cancelNext = false
			return true
		}

		e, ok := p.SearchEntity(d.TargetEntityRuntimeID)
		if !ok {
			return false
		}

		r.lastEntPos = e.LastPosition()
		r.currentEntPos = e.Position()

		r.attackData = d
	} else if i, ok := pk.(*packet.PlayerAuthInput); ok && r.attackData != nil {
		defer func() {
			r.attackData = nil
		}()

		attackPos := game.Vec32To64(r.attackData.Position)

		// We're checking this again because within the span of the attack being sent and the
		// client sending an input packet, the entity could have been removed.
		e, ok := p.SearchEntity(r.attackData.TargetEntityRuntimeID)
		if !ok {
			return false
		}

		bbDist, entPos, dPos := 6969.0, r.lastEntPos, r.currentEntPos.Sub(r.lastEntPos).Mul(1.0/interpolationInterations)
		for i := 0.0; i < interpolationInterations; i++ {
			if i != 0 {
				entPos = entPos.Add(dPos)
			}

			bb := e.AABB().Translate(entPos).Grow(0.1)
			dist := game.AABBVectorDistance(bb, attackPos)
			if bbDist > dist {
				bbDist = dist
			}
		}

		if bbDist > 3.15 {
			p.Flag(r, 1, map[string]any{
				"dist": game.Round(bbDist, 4),
				"type": "bb-dist",
			})
			r.cancelNext = true
			return false
		}
		r.violations = math.Max(0, r.violations-0.001)

		if i.InputMode == packet.InputModeTouch {
			return false
		}

		minDist, valid := 6969.0, false
		distAvg, totalHits := 0.0, 0.0
		rot := game.DirectionVector(p.Entity().LastRotation().Z(), p.Entity().LastRotation().X())
		dRot := game.DirectionVector(p.Entity().Rotation().Z(), p.Entity().Rotation().X()).Sub(rot).Mul(1.0 / interpolationInterations)
		entPos = r.lastEntPos

		for i := 0.0; i < interpolationInterations; i++ {
			if i != 0 {
				entPos = entPos.Add(dPos)
			}

			for x := 0.0; x < interpolationInterations; x++ {
				if x != 0 {
					rot = rot.Add(dRot)
				}
				bb := e.AABB().Translate(entPos).Grow(0.1)
				result, ok := trace.BBoxIntercept(bb, attackPos, attackPos.Add(rot.Mul(7.0)))
				if !ok {
					continue
				}

				valid = true
				dist := result.Position().Sub(attackPos).Len()
				distAvg += dist
				totalHits++
				if minDist > dist {
					minDist = dist
				}
			}
		}

		if !valid {
			if r.Buff(1, 5) >= 4.5 {
				p.Flag(r, 1, map[string]any{
					"type": "hitbox",
				})
				r.cancelNext = true
			}

			return false
		}
		r.Buff(-0.025, 5)

		distAvg /= totalHits
		if distAvg <= 3.0001 {
			r.secondaryBuffer = math.Max(0, r.secondaryBuffer-0.0075)
			r.violations = math.Max(0, r.violations-0.001)
			return false
		}

		r.secondaryBuffer++
		r.secondaryBuffer = math.Min(r.secondaryBuffer, 5)

		if r.secondaryBuffer > 1 {
			p.Flag(r, 1, map[string]any{
				"dist": game.Round(distAvg, 4),
				"type": "raycast",
			})
		}

		r.cancelNext = true
	}

	return false
}
