package utils

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/entity/physics"
	"github.com/df-mc/dragonfly/server/world"
	"math"
)

// BlockClimbable returns whether the given block is climbable.
func BlockClimbable(b world.Block) bool {
	switch b.(type) {
	case block.Ladder:
		return true
		// TODO: Add vines here.
	}
	return false
}

// BlocksNearby returns a slice of blocks that are nearby the search position.
func BlocksNearby(aabb physics.AABB, w *world.World, solid bool) []world.Block {
	grown := aabb.Grow(1)
	min, max := grown.Min(), grown.Max()
	minX, minY, minZ := min[0], min[1], min[2]
	maxX, maxY, maxZ := max[0], max[1], max[2]

	var blocks []world.Block
	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			for z := minZ; z < maxZ; z++ {
				b := w.Block(cube.Pos{int(x), int(y), int(z)})
				if _, ok := b.Model().(model.Solid); !ok && solid {
					// The block isn't solid, move along and check the next one.
					continue
				}
				blocks = append(blocks, b)
			}
		}
	}
	return blocks
}

// CollidingBlocks returns all block AABBs that collide with the entity.
func CollidingBlocks(aabb physics.AABB, w *world.World) []physics.AABB {
	grown := aabb.Grow(1)
	min, max := grown.Min(), grown.Max()
	minX, minY, minZ := int(math.Floor(min[0])), int(math.Floor(min[1])), int(math.Floor(min[2]))
	maxX, maxY, maxZ := int(math.Floor(max[0])), int(math.Floor(max[1])), int(math.Floor(max[2]))

	var blockAABBs []physics.AABB
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			for z := minZ; z <= maxZ; z++ {
				blockPos := cube.Pos{x, y, z}
				for _, box := range w.Block(blockPos).Model().AABB(blockPos, w) {
					if box.Translate(blockPos.Vec3()).IntersectsWith(aabb) {
						blockAABBs = append(blockAABBs, box.Translate(blockPos.Vec3()))
					}
				}
			}
		}
	}
	return blockAABBs
}
