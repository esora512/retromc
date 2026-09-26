package entities

import (
	"math"

	"github.com/leNicDev/retromc/constants"
)

type aabb struct {
	minX, minY, minZ float64
	maxX, maxY, maxZ float64
}

func (a aabb) union(o aabb) aabb {
	return aabb{
		minX: math.Min(a.minX, o.minX), minY: math.Min(a.minY, o.minY), minZ: math.Min(a.minZ, o.minZ),
		maxX: math.Max(a.maxX, o.maxX), maxY: math.Max(a.maxY, o.maxY), maxZ: math.Max(a.maxZ, o.maxZ),
	}
}

func (a aabb) offset(dx, dy, dz float64) aabb {
	a.minX += dx
	a.maxX += dx
	a.minY += dy
	a.maxY += dy
	a.minZ += dz
	a.maxZ += dz
	return a
}

type shapeKind uint8

const (
	shapeFull shapeKind = iota
	shapeNone
	shapeSlab
	shapeStairs
	shapeChest
	shapeFarmland
	shapeDoor
	shapeLadder
	shapeCactus
	shapeFence
	shapeCake
	shapeRepeater
	shapeTrapdoor
	shapeBed
	shapeSoulSand
	shapePistonHead
	shapeSnowLayer
)

type blockInfo struct {
	shape shapeKind
	// full, opaque cube (vanilla isBlockNormalCube); items inside one get pushed out
	normalCube bool
	// vanilla Material.isSolid(); only used to decide how water flows around a block
	solidMaterial bool
}

var blockTable [256]blockInfo

func init() {
	for i := range blockTable {
		blockTable[i] = blockInfo{shape: shapeFull, normalCube: false, solidMaterial: true}
	}

	set := func(shape shapeKind, blocks ...constants.Block) {
		for _, b := range blocks {
			blockTable[b.Value].shape = shape
		}
	}

	set(shapeNone,
		constants.Air, constants.Sapling,
		constants.WaterFlowing, constants.WaterStill, constants.LavaFlowing, constants.LavaStill,
		constants.PoweredRail, constants.DetectorRail, constants.Rail,
		constants.Cobweb, constants.Tallgrass, constants.Deadbush,
		constants.Dandelion, constants.Rose, constants.BrownMushroom, constants.RedMushroom,
		constants.Torch, constants.Fire, constants.RedstoneBlock, constants.Wheat,
		constants.SignGround, constants.SignWall, constants.Lever,
		constants.StonePressurePlate, constants.WoodenPressurePlate,
		constants.RedstoneTorchOff, constants.RedstoneTorchOn, constants.StoneButton,
		constants.Sugarcane, constants.NetherPortal)
	// id 36 is the moving piston block, which has no collision of its own
	blockTable[36].shape = shapeNone

	set(shapeSlab, constants.StoneSlab)
	set(shapeStairs, constants.WoodenStairs, constants.CobblestoneStairs)
	set(shapeChest, constants.Chest, constants.LockedChest)
	set(shapeFarmland, constants.Farmland)
	set(shapeDoor, constants.WoodenDoor, constants.IronDoor)
	set(shapeLadder, constants.Ladder)
	set(shapeCactus, constants.Cactus)
	set(shapeFence, constants.Fence)
	set(shapeCake, constants.Cake)
	set(shapeRepeater, constants.RedstoneRepeaterOff, constants.RedstoneRepeaterOn)
	set(shapeTrapdoor, constants.Trapdoor)
	set(shapeBed, constants.Bed)
	set(shapeSoulSand, constants.SoulSand)
	set(shapePistonHead, constants.PistonHead)
	set(shapeSnowLayer, constants.SnowLayer)

	// Full blocks
	for _, b := range []constants.Block{
		constants.Stone, constants.Grass, constants.Dirt, constants.Cobblestone, constants.Planks,
		constants.Bedrock, constants.Sand, constants.Gravel, constants.GoldOre, constants.IronOre,
		constants.CoalOre, constants.Log, constants.Sponge, constants.LapisLazuliOre,
		constants.LapisLazuliBlock, constants.Dispenser, constants.Sandstone, constants.Noteblock,
		constants.StickyPiston, constants.Piston, constants.Wool, constants.GoldBlock,
		constants.IronBlock, constants.DoubleStoneSlab, constants.Bricks, constants.TNT,
		constants.Bookshelf, constants.MossyCobblestone, constants.Obsidian, constants.DiamondOre,
		constants.DiamondBlock, constants.CraftingTable, constants.Furnace, constants.FurnaceLit,
		constants.RedstoneOreOff, constants.RedstoneOreOn, constants.SnowBlock, constants.Clay,
		constants.Jukebox, constants.Pumpkin, constants.Netherrack, constants.SoulSand,
		constants.Glowstone, constants.PumpkinLit,
	} {
		blockTable[b.Value].normalCube = true
	}

	// Non-solids
	for _, b := range []constants.Block{
		constants.Air, constants.Sapling,
		constants.WaterFlowing, constants.WaterStill, constants.LavaFlowing, constants.LavaStill,
		constants.PoweredRail, constants.DetectorRail, constants.Rail,
		constants.Tallgrass, constants.Deadbush,
		constants.Dandelion, constants.Rose, constants.BrownMushroom, constants.RedMushroom,
		constants.Torch, constants.Fire, constants.RedstoneBlock, constants.Wheat, constants.Ladder,
		constants.Lever, constants.RedstoneTorchOff, constants.RedstoneTorchOn, constants.StoneButton,
		constants.SnowLayer, constants.Sugarcane, constants.NetherPortal,
		constants.RedstoneRepeaterOff, constants.RedstoneRepeaterOn,
	} {
		blockTable[b.Value].solidMaterial = false
	}
}

func isNormalCube(b constants.WBlock) bool    { return blockTable[b.TypeId].normalCube }
func IsNormalCube(b constants.WBlock) bool    { return isNormalCube(b) }
func isSolidMaterial(b constants.WBlock) bool { return blockTable[b.TypeId].solidMaterial }

func slipperiness(b constants.WBlock) float32 {
	if b.TypeId == byte(constants.Ice.Value) {
		return 0.98
	}
	return 0.6
}

func box(minX, minY, minZ, maxX, maxY, maxZ float64) aabb {
	return aabb{minX, minY, minZ, maxX, maxY, maxZ}
}

var (
	fullBoxes       = []aabb{box(0, 0, 0, 1, 1, 1)}
	slabBoxes       = []aabb{box(0, 0, 0, 1, 0.5, 1)}
	chestBoxes      = []aabb{box(0.0625, 0, 0.0625, 0.9375, 0.875, 0.9375)}
	farmlandBoxes   = []aabb{box(0, 0, 0, 1, 0.9375, 1)}
	cactusBoxes     = []aabb{box(0.0625, 0, 0.0625, 0.9375, 0.9375, 0.9375)}
	fenceBoxes      = []aabb{box(0, 0, 0, 1, 1.5, 1)}
	repeaterBoxes   = []aabb{box(0, 0, 0, 1, 0.125, 1)}
	bedBoxes        = []aabb{box(0, 0, 0, 1, 0.5625, 1)}
	soulSandBoxes   = []aabb{box(0, 0, 0, 1, 0.875, 1)}
	snowHalfBoxes   = []aabb{box(0, 0, 0, 1, 0.5, 1)}
	noBoxes         = []aabb(nil)
	stairBoxes      [4][]aabb
	ladderBoxes     [6][]aabb
	doorBoxes       [4][]aabb
	trapdoorBoxes   [16][]aabb
	cakeBoxes       [8][]aabb
	pistonHeadBoxes [6][]aabb
)

func init() {
	stairBoxes = [4][]aabb{
		{box(0, 0, 0, 0.5, 0.5, 1), box(0.5, 0, 0, 1, 1, 1)},
		{box(0, 0, 0, 0.5, 1, 1), box(0.5, 0, 0, 1, 0.5, 1)},
		{box(0, 0, 0, 1, 0.5, 0.5), box(0, 0, 0.5, 1, 1, 1)},
		{box(0, 0, 0, 1, 1, 0.5), box(0, 0, 0.5, 1, 0.5, 1)},
	}

	const lt = 0.125
	ladderBoxes[2] = []aabb{box(0, 0, 1-lt, 1, 1, 1)}
	ladderBoxes[3] = []aabb{box(0, 0, 0, 1, 1, lt)}
	ladderBoxes[4] = []aabb{box(1-lt, 0, 0, 1, 1, 1)}
	ladderBoxes[5] = []aabb{box(0, 0, 0, lt, 1, 1)}

	const dt = 0.1875
	doorBoxes = [4][]aabb{
		{box(0, 0, 0, 1, 1, dt)},
		{box(1-dt, 0, 0, 1, 1, 1)},
		{box(0, 0, 1-dt, 1, 1, 1)},
		{box(0, 0, 0, dt, 1, 1)},
	}

	for meta := 0; meta < 16; meta++ {
		if meta&4 == 0 {
			trapdoorBoxes[meta] = []aabb{box(0, 0, 0, 1, dt, 1)}
			continue
		}
		switch meta & 3 {
		case 0:
			trapdoorBoxes[meta] = []aabb{box(0, 0, 1-dt, 1, 1, 1)}
		case 1:
			trapdoorBoxes[meta] = []aabb{box(0, 0, 0, 1, 1, dt)}
		case 2:
			trapdoorBoxes[meta] = []aabb{box(1-dt, 0, 0, 1, 1, 1)}
		case 3:
			trapdoorBoxes[meta] = []aabb{box(0, 0, 0, dt, 1, 1)}
		}
	}

	for meta := 0; meta < 8; meta++ {
		x0 := float64(1+meta*2) / 16.0
		cakeBoxes[meta] = []aabb{box(x0, 0, 0.0625, 0.9375, 0.4375, 0.9375)}
	}

	pistonHeadBoxes = [6][]aabb{
		{box(0, 0, 0, 1, 0.25, 1), box(0.375, 0.25, 0.375, 0.625, 1, 0.625)},
		{box(0, 0.75, 0, 1, 1, 1), box(0.375, 0, 0.375, 0.625, 0.75, 0.625)},
		{box(0, 0, 0, 1, 1, 0.25), box(0.25, 0.375, 0.25, 0.75, 0.625, 1)},
		{box(0, 0, 0.75, 1, 1, 1), box(0.25, 0.375, 0, 0.75, 0.625, 0.75)},
		{box(0, 0, 0, 0.25, 1, 1), box(0.375, 0.25, 0.25, 0.625, 0.75, 1)},
		{box(0.75, 0, 0, 1, 1, 1), box(0, 0.375, 0.25, 0.75, 0.625, 0.75)},
	}
}

// Collision boxes of a block in block-local coordinates (0..1 on each axis).
func blockBoxes(b constants.WBlock) []aabb {
	meta := int(b.Metadata)
	switch blockTable[b.TypeId].shape {
	case shapeNone:
		return noBoxes
	case shapeSlab:
		return slabBoxes
	case shapeStairs:
		return stairBoxes[meta&3]
	case shapeChest:
		return chestBoxes
	case shapeFarmland:
		return farmlandBoxes
	case shapeDoor:
		// bits 0-1 facing while closed, bit 2 open, bit 3 top half
		state := meta & 3
		if meta&4 == 0 {
			state = (meta - 1) & 3
		}
		return doorBoxes[state]
	case shapeLadder:
		if meta >= 2 && meta <= 5 {
			return ladderBoxes[meta]
		}
		return noBoxes
	case shapeCactus:
		return cactusBoxes
	case shapeFence:
		return fenceBoxes
	case shapeCake:
		return cakeBoxes[meta&7]
	case shapeRepeater:
		return repeaterBoxes
	case shapeTrapdoor:
		return trapdoorBoxes[meta&15]
	case shapeBed:
		return bedBoxes
	case shapeSoulSand:
		return soulSandBoxes
	case shapePistonHead:
		if meta&7 < 6 {
			return pistonHeadBoxes[meta&7]
		}
		return fullBoxes
	case shapeSnowLayer:
		if meta&7 >= 3 {
			return snowHalfBoxes
		}
		return noBoxes
	default:
		return fullBoxes
	}
}

func BlockIntersects(b constants.WBlock, x, y, z int32, minX, minY, minZ, maxX, maxY, maxZ float64) bool {
	for _, bb := range blockBoxes(b) {
		bb = bb.offset(float64(x), float64(y), float64(z))
		if bb.minX < maxX && bb.maxX > minX && bb.minY < maxY && bb.maxY > minY && bb.minZ < maxZ && bb.maxZ > minZ {
			return true
		}
	}
	return false
}
