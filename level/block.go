package level

import (
	"math/rand"

	"github.com/leNicDev/retromc/constants"
)

type Block struct {
	TypeId   byte
	Metadata byte
	Light    byte
	SkyLight byte
}

var directionalBlocks = map[int16]bool{
	constants.WoodenStairs.Value:        true,
	constants.CobblestoneStairs.Value:   true,
	constants.Torch.Value:               true,
	constants.Chest.Value:               true,
	constants.Furnace.Value:             true,
	constants.FurnaceLit.Value:          true,
	constants.Sign.Value:                true,
	constants.Ladder.Value:              true,
	constants.Dispenser.Value:           true,
	constants.StickyPiston.Value:        true,
	constants.Piston.Value:              true,
	constants.PistonHead.Value:          true,
	constants.SignGround.Value:          true,
	constants.SignWall.Value:            true,
	constants.Lever.Value:               true,
	constants.RedstoneTorchOn.Value:     true,
	constants.RedstoneTorchOff.Value:    true,
	constants.StoneButton.Value:         true,
	constants.Pumpkin.Value:             true,
	constants.PumpkinLit.Value:          true,
	constants.RedstoneRepeater.Value:    true,
	constants.RedstoneRepeaterOn.Value:  true,
	constants.RedstoneRepeaterOff.Value: true,
	constants.LockedChest.Value:         true,
	constants.Trapdoor.Value:            true,
}

type BlockDirections struct {
	North byte
	South byte
	East  byte
	West  byte
}

func (b *Block) IsGrowable() bool {
	return b.TypeId == byte(constants.Wheat.Value) || b.TypeId == byte(constants.Sugarcane.Value) || b.TypeId == byte(constants.Cactus.Value) || b.TypeId == byte(constants.Sapling.Value) || b.TypeId == byte(constants.Dirt.Value)
}

func (b *Block) IsTransparent() bool {
	return b.TypeId == byte(constants.Air.Value) ||
		b.TypeId == byte(constants.Dandelion.Value) ||
		b.TypeId == byte(constants.Tallgrass.Value) ||
		b.TypeId == byte(constants.Deadbush.Value) ||
		b.TypeId == byte(constants.Grass.Value) ||
		b.TypeId == byte(constants.Glass.Value) ||
		b.TypeId == byte(constants.Sugarcane.Value) ||
		b.TypeId == byte(constants.Rail.Value) ||
		b.TypeId == byte(constants.PoweredRail.Value) ||
		b.TypeId == byte(constants.DetectorRail.Value) ||
		b.TypeId == byte(constants.Leaves.Value) ||
		b.TypeId == byte(constants.SnowLayer.Value)
}

func (b *Block) IsRail() bool {
	return b.TypeId == byte(constants.Rail.Value) || b.TypeId == byte(constants.PoweredRail.Value) || b.TypeId == byte(constants.DetectorRail.Value)
}

func (b *Block) IsLiquid() bool {
	return b.TypeId == byte(constants.WaterFlowing.Value) || b.TypeId == byte(constants.WaterStill.Value) || b.TypeId == byte(constants.LavaFlowing.Value) || b.TypeId == byte(constants.LavaStill.Value)
}

func (b *Block) IsBedHead() bool {
	return b.TypeId == byte(constants.Bed.Value) && b.Metadata&0x8 != 0
}

func (b *Block) IsBed() bool {
	return b.TypeId == byte(constants.Bed.Value)
}

func (b *Block) IsSnowLayer() bool {
	return b.TypeId == byte(constants.SnowLayer.Value)
}

func (b *Block) IsFlowing() bool {
	return b.TypeId == byte(constants.WaterFlowing.Value) || b.TypeId == byte(constants.LavaFlowing.Value)
}

func (b *Block) IsFlowingWater() bool {
	return b.TypeId == byte(constants.WaterFlowing.Value)
}

func (b *Block) IsFlowingLava() bool {
	return b.TypeId == byte(constants.LavaFlowing.Value)
}

func (b *Block) IsStillLava() bool {
	return b.TypeId == byte(constants.LavaStill.Value)
}

func (b *Block) IsStillWater() bool {
	return b.TypeId == byte(constants.WaterStill.Value)
}

func (b *Block) IsWater() bool {
	return b.TypeId == byte(constants.WaterStill.Value) || b.TypeId == byte(constants.WaterFlowing.Value)
}

func (b *Block) IsLava() bool {
	return b.TypeId == byte(constants.LavaStill.Value) || b.TypeId == byte(constants.LavaFlowing.Value)
}

func (b *Block) IsFluid() bool {
	return b.IsWater() || b.IsLava()
}

func (b *Block) NewBlock(meta byte) Block {
	return NewBlockById(int16(b.TypeId), meta)
}

func (b *Block) IsPoweredRail() bool {
	return b.TypeId == byte(constants.PoweredRail.Value)
}

func (b *Block) GetDirections() BlockDirections {
	if b.TypeId == byte(constants.Bed.Value) {
		return BlockDirections{
			North: 3,
			South: 5,
			West:  2,
			East:  4,
		}
	}

	if b.TypeId == byte(constants.Ladder.Value) {
		return BlockDirections{
			North: 4,
			South: 5,
			West:  3,
			East:  2,
		}
	}

	if b.TypeId == byte(constants.SignGround.Value) {
		return BlockDirections{
			North: 4,
			South: 12,
			East:  8,
			West:  0,
		}
	}

	if b.TypeId == byte(constants.WoodenStairs.Value) || b.TypeId == byte(constants.CobblestoneStairs.Value) {
		return BlockDirections{
			North: 0,
			South: 1,
			East:  2,
			West:  3,
		}
	}

	if b.TypeId == byte(constants.Torch.Value) || b.TypeId == byte(constants.RedstoneTorchOn.Value) || b.TypeId == byte(constants.RedstoneTorchOff.Value) || b.TypeId == byte(constants.Lever.Value) {
		return BlockDirections{
			North: 2,
			South: 1,
			East:  4,
			West:  3,
		}
	}

	if b.TypeId == byte(constants.Furnace.Value) || b.TypeId == byte(constants.FurnaceLit.Value) || b.TypeId == byte(constants.Dispenser.Value) || b.TypeId == byte(constants.Chest.Value) {
		return BlockDirections{
			North: 4,
			South: 5,
			East:  2,
			West:  3,
		}
	}

	if b.TypeId == byte(constants.Piston.Value) || b.TypeId == byte(constants.PistonHead.Value) {
		return BlockDirections{
			North: 4,
			South: 5,
			East:  2,
			West:  3,
		}
	}

	if b.TypeId == byte(constants.Pumpkin.Value) || b.TypeId == byte(constants.PumpkinLit.Value) {
		return BlockDirections{
			North: 1,
			South: 3,
			East:  2,
			West:  0,
		}
	}

	if b.TypeId == byte(constants.StoneButton.Value) {
		return BlockDirections{
			North: 2,
			South: 1,
			East:  4,
			West:  3,
		}
	}

	if b.TypeId == byte(constants.Trapdoor.Value) {
		return BlockDirections{
			North: 2,
			South: 3,
			East:  4,
			West:  1,
		}
	}

	return BlockDirections{
		North: 0,
		South: 1,
		East:  2,
		West:  3,
	}
}

func NewRandomBlock() Block {
	val := rand.Intn(94)
	if val == int(constants.Grass.Value) || val == int(constants.Chest.Value) || val == int(constants.Furnace.Value) || val == int(constants.Dispenser.Value) || val == int(constants.FurnaceLit.Value) || val == int(constants.LavaFlowing.Value) || val == int(constants.WaterFlowing.Value) {
		val = int(constants.Stone.Value)
	}
	return NewBlockById(int16(val), 0)
}

func (b *Block) IsDirectional() bool {
	return directionalBlocks[int16(b.TypeId)]
}

func (b *Block) IsAir() bool {
	return b.TypeId == 0
}

func (b *Block) IsSolid () bool {
	return !b.IsAir() && !b.IsFluid() && !b.IsSnowLayer()
}

// ID 0 - Air
func NewAirBlock() Block {
	return Block{TypeId: 0x00, Metadata: 0x00, Light: 0x00, SkyLight: 0x0f}
}

// ID 1 - Stone
func NewStoneBlock() Block {
	return Block{TypeId: 0x01, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 2 - Grass
func NewGrassBlock() Block {
	return Block{TypeId: 0x02, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 3 - Dirt
func NewDirtBlock() Block {
	return Block{TypeId: 0x03, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 4 - Cobblestone
func NewCobblestoneBlock() Block {
	return Block{TypeId: 0x04, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 5 - Planks (metadata: wood type)
func NewPlanksBlock(woodType byte) Block {
	return Block{TypeId: 0x05, Metadata: woodType, Light: 0x00, SkyLight: 0x00}
}

// ID 6 - Sapling (metadata: wood type)
func NewSaplingBlock(woodType byte) Block {
	return Block{TypeId: 0x06, Metadata: woodType, Light: 0x00, SkyLight: 0x00}
}

// ID 7 - Bedrock
func NewBedrockBlock() Block {
	return Block{TypeId: 0x07, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 8 - Water (Flowing) (metadata: liquid height)
func NewFlowingWaterBlock(liquidHeight byte) Block {
	return Block{TypeId: 0x08, Metadata: liquidHeight, Light: 0x00, SkyLight: 0x00}
}

// ID 9 - Water (Still) (metadata: liquid height)
func NewStillWaterBlock(liquidHeight byte) Block {
	return Block{TypeId: 0x09, Metadata: liquidHeight, Light: 0x00, SkyLight: 0x00}
}

// ID 10 - Lava (Flowing) (metadata: liquid height)
func NewFlowingLavaBlock(liquidHeight byte) Block {
	return Block{TypeId: 0x0a, Metadata: liquidHeight, Light: 0x00, SkyLight: 0x00}
}

// ID 11 - Lava (Still) (metadata: liquid height)
func NewStillLavaBlock(liquidHeight byte) Block {
	return Block{TypeId: 0x0b, Metadata: liquidHeight, Light: 0x00, SkyLight: 0x00}
}

// ID 12 - Sand
func NewSandBlock() Block {
	return Block{TypeId: 0x0c, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 13 - Gravel
func NewGravelBlock() Block {
	return Block{TypeId: 0x0d, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 14 - Gold Ore
func NewGoldOreBlock() Block {
	return Block{TypeId: 0x0e, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 15 - Iron Ore
func NewIronOreBlock() Block {
	return Block{TypeId: 0x0f, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 16 - Coal Ore
func NewCoalOreBlock() Block {
	return Block{TypeId: 0x10, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 17 - Log (metadata: wood type)
func NewLogBlock(woodType byte) Block {
	return Block{TypeId: 0x11, Metadata: woodType, Light: 0x00, SkyLight: 0x00}
}

// ID 18 - Leaves (metadata: leaf type)
func NewLeavesBlock(leafType byte) Block {
	return Block{TypeId: 0x12, Metadata: leafType, Light: 0x00, SkyLight: 0x00}
}

// ID 19 - Sponge
func NewSpongeBlock() Block {
	return Block{TypeId: 0x13, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 20 - Glass
func NewGlassBlock() Block {
	return Block{TypeId: 0x14, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 21 - Lapis Lazuli Ore
func NewLapisLazuliOreBlock() Block {
	return Block{TypeId: 0x15, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 22 - Lapis Lazuli Block
func NewLapisLazuliBlock() Block {
	return Block{TypeId: 0x16, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 23 - Dispenser (metadata: direction)
func NewDispenserBlock(direction byte) Block {
	return Block{TypeId: 0x17, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 24 - Sandstone
func NewSandstoneBlock() Block {
	return Block{TypeId: 0x18, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 25 - Noteblock
func NewNoteblock() Block {
	return Block{TypeId: 0x19, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 26 - Bed (metadata: top/bottom and direction)
func NewBedBlock(meta byte) Block {
	return Block{TypeId: 0x1a, Metadata: meta, Light: 0x00, SkyLight: 0x00}
}

// ID 27 - Powered Rail (metadata: direction)
func NewPoweredRailBlock(direction byte) Block {
	return Block{TypeId: 0x1b, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 28 - Detector Rail (metadata: direction)
func NewDetectorRailBlock(direction byte) Block {
	return Block{TypeId: 0x1c, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 29 - Sticky Piston (metadata: direction and state)
func NewStickyPistonBlock(meta byte) Block {
	return Block{TypeId: 0x1d, Metadata: meta, Light: 0x00, SkyLight: 0x00}
}

// ID 30 - Cobweb
func NewCobwebBlock() Block {
	return Block{TypeId: 0x1e, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 31 - Tallgrass (metadata: 0=shrub, 1=grass, 2=fern)
func NewTallgrassBlock(variant byte) Block {
	return Block{TypeId: 0x1f, Metadata: variant, Light: 0x00, SkyLight: 0x00}
}

// ID 32 - Deadbush
func NewDeadbushBlock() Block {
	return Block{TypeId: 0x20, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 33 - Piston (metadata: direction and state)
func NewPistonBlock(meta byte) Block {
	return Block{TypeId: 0x21, Metadata: meta, Light: 0x00, SkyLight: 0x00}
}

// ID 34 - Piston Head (metadata: direction)
func NewPistonHeadBlock(direction byte) Block {
	return Block{TypeId: 0x22, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 35 - Wool (metadata: color)
func NewWoolBlock(color byte) Block {
	return Block{TypeId: 0x23, Metadata: color, Light: 0x00, SkyLight: 0x00}
}

// ID 37 - Dandelion
func NewDandelionBlock() Block {
	return Block{TypeId: 0x25, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 38 - Rose
func NewRoseBlock() Block {
	return Block{TypeId: 0x26, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 39 - Brown Mushroom
func NewBrownMushroomBlock() Block {
	return Block{TypeId: 0x27, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 40 - Red Mushroom
func NewRedMushroomBlock() Block {
	return Block{TypeId: 0x28, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 41 - Gold Block
func NewGoldBlock() Block {
	return Block{TypeId: 0x29, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 42 - Iron Block
func NewIronBlock() Block {
	return Block{TypeId: 0x2a, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 43 - Double Slab (metadata: 0=stone, 1=sandstone, 2=wood, 3=cobblestone)
func NewDoubleSlabBlock(slabType byte) Block {
	return Block{TypeId: 0x2b, Metadata: slabType, Light: 0x00, SkyLight: 0x00}
}

// ID 44 - Slab (metadata: 0=stone, 1=sandstone, 2=wood, 3=cobblestone)
func NewSlabBlock(slabType byte) Block {
	return Block{TypeId: 0x2c, Metadata: slabType, Light: 0x00, SkyLight: 0x00}
}

// ID 45 - Bricks
func NewBricksBlock() Block {
	return Block{TypeId: 0x2d, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 46 - TNT
func NewTNTBlock() Block {
	return Block{TypeId: 0x2e, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 47 - Bookshelf
func NewBookshelfBlock() Block {
	return Block{TypeId: 0x2f, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 48 - Mossy Cobblestone
func NewMossyCobblestoneBlock() Block {
	return Block{TypeId: 0x30, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 49 - Obsidian
func NewObsidianBlock() Block {
	return Block{TypeId: 0x31, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 50 - Torch (metadata: direction)
func NewTorchBlock(direction byte) Block {
	return Block{TypeId: 0x32, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 51 - Fire
func NewFireBlock() Block {
	return Block{TypeId: 0x33, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 52 - Monster Spawner
func NewMonsterSpawnerBlock() Block {
	return Block{TypeId: 0x34, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 53 - Wooden Stairs (metadata: direction)
func NewWoodenStairsBlock(direction byte) Block {
	return Block{TypeId: 0x35, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 54 - Chest
func NewChestBlock() Block {
	return Block{TypeId: 0x36, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 55 - Redstone (metadata: power level)
func NewRedstoneBlock(powerLevel byte) Block {
	return Block{TypeId: 0x37, Metadata: powerLevel, Light: 0x00, SkyLight: 0x00}
}

// ID 56 - Diamond Ore
func NewDiamondOreBlock() Block {
	return Block{TypeId: 0x38, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 57 - Diamond Block
func NewDiamondBlock() Block {
	return Block{TypeId: 0x39, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 58 - Crafting Table
func NewCraftingTable() Block {
	return Block{TypeId: 0x3a, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 59 - Wheat (metadata: growth stage 0-7)
func NewWheatBlock(growthStage byte) Block {
	return Block{TypeId: 0x3b, Metadata: growthStage, Light: 0x00, SkyLight: 0x00}
}

// ID 60 - Farmland (metadata: >0 if wet)
func NewFarmlandBlock(wetness byte) Block {
	return Block{TypeId: 0x3c, Metadata: wetness, Light: 0x00, SkyLight: 0x00}
}

// ID 61 - Furnace (metadata: direction)
func NewFurnaceBlock(direction byte) Block {
	return Block{TypeId: 0x3d, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 62 - Furnace (Lit) (metadata: direction)
func NewLitFurnaceBlock(direction byte) Block {
	return Block{TypeId: 0x3e, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 63 - Sign (Ground) (metadata: direction)
func NewGroundSignBlock(direction byte) Block {
	return Block{TypeId: 0x3f, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 64 - Wooden Door
func NewWoodenDoorBlock() Block {
	return Block{TypeId: 0x40, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 65 - Ladder (metadata: direction)
func NewLadderBlock(direction byte) Block {
	return Block{TypeId: 0x41, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 66 - Rail (metadata: direction)
func NewRailBlock(direction byte) Block {
	return Block{TypeId: 0x42, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 67 - Cobblestone Stairs (metadata: direction)
func NewCobblestoneStairsBlock(direction byte) Block {
	return Block{TypeId: 0x43, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 68 - Sign (Wall) (metadata: direction)
func NewWallSignBlock(direction byte) Block {
	return Block{TypeId: 0x44, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 69 - Lever (metadata: toggled & direction)
func NewLeverBlock(meta byte) Block {
	return Block{TypeId: 0x45, Metadata: meta, Light: 0x00, SkyLight: 0x00}
}

// ID 70 - Stone Pressure Plate (metadata: toggled)
func NewStonePressurePlateBlock(toggled byte) Block {
	return Block{TypeId: 0x46, Metadata: toggled, Light: 0x00, SkyLight: 0x00}
}

// ID 71 - Iron Door
func NewIronDoorBlock() Block {
	return Block{TypeId: 0x47, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 72 - Wooden Pressure Plate (metadata: toggled)
func NewWoodenPressurePlateBlock(toggled byte) Block {
	return Block{TypeId: 0x48, Metadata: toggled, Light: 0x00, SkyLight: 0x00}
}

// ID 73 - Redstone Ore (Off)
func NewRedstoneOreBlock() Block {
	return Block{TypeId: 0x49, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 74 - Redstone Ore (On)
func NewLitRedstoneOreBlock() Block {
	return Block{TypeId: 0x4a, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 75 - Redstone Torch (Off) (metadata: direction)
func NewRedstoneTorchOffBlock(direction byte) Block {
	return Block{TypeId: 0x4b, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 76 - Redstone Torch (On) (metadata: direction)
func NewRedstoneTorchOnBlock(direction byte) Block {
	return Block{TypeId: 0x4c, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 77 - Stone Button (metadata: toggled & direction)
func NewStoneButtonBlock(meta byte) Block {
	return Block{TypeId: 0x4d, Metadata: meta, Light: 0x00, SkyLight: 0x00}
}

// ID 78 - Snow Layer
func NewSnowLayerBlock() Block {
	return Block{TypeId: 0x4e, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 79 - Ice
func NewIceBlock() Block {
	return Block{TypeId: 0x4f, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 80 - Snow Block
func NewSnowBlock() Block {
	return Block{TypeId: 0x50, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 81 - Cactus
func NewCactusBlock(Meta byte) Block {
	return Block{TypeId: 0x51, Metadata: Meta, Light: 0x00, SkyLight: 0x00}
}

// ID 82 - Clay
func NewClayBlock() Block {
	return Block{TypeId: 0x52, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 83 - Sugarcane
func NewSugarcaneBlock(Meta byte) Block {
	return Block{TypeId: 0x53, Metadata: Meta, Light: 0x00, SkyLight: 0x00}
}

// ID 84 - Jukebox
func NewJukeboxBlock() Block {
	return Block{TypeId: 0x54, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 85 - Fence
func NewFenceBlock() Block {
	return Block{TypeId: 0x55, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 86 - Pumpkin (metadata: direction)
func NewPumpkinBlock(direction byte) Block {
	return Block{TypeId: 0x56, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 87 - Netherrack
func NewNetherrackBlock() Block {
	return Block{TypeId: 0x57, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 88 - Soul Sand
func NewSoulSandBlock() Block {
	return Block{TypeId: 0x58, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 89 - Glowstone
func NewGlowstoneBlock() Block {
	return Block{TypeId: 0x59, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 90 - Nether Portal
func NewNetherPortalBlock() Block {
	return Block{TypeId: 0x5a, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 91 - Pumpkin (Lit) (metadata: direction)
func NewLitPumpkinBlock(direction byte) Block {
	return Block{TypeId: 0x5b, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 92 - Cake
func NewCakeBlock() Block {
	return Block{TypeId: 0x5c, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 93 - Redstone Repeater (Off) (metadata: direction)
func NewRedstoneRepeaterOffBlock(direction byte) Block {
	return Block{TypeId: 0x5d, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 94 - Redstone Repeater (On) (metadata: direction)
func NewRedstoneRepeaterOnBlock(direction byte) Block {
	return Block{TypeId: 0x5e, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 95 - Locked Chest
func NewLockedChestBlock() Block {
	return Block{TypeId: 0x5f, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 96 - Trapdoor (metadata: toggled & direction)
func NewTrapdoorBlock(meta byte) Block {
	return Block{TypeId: 0x60, Metadata: meta, Light: 0x00, SkyLight: 0x00}
}

func NewBlockById(id int16, Meta byte) Block {
	switch id {
	case 0x00:
		return NewAirBlock()
	case 0x01:
		return NewStoneBlock()
	case 0x02:
		return NewGrassBlock()
	case 0x03:
		return NewDirtBlock()
	case 0x04:
		return NewCobblestoneBlock()
	case 0x05:
		return NewPlanksBlock(Meta)
	case 0x06:
		return NewSaplingBlock(Meta)
	case 0x07:
		return NewBedrockBlock()
	case 0x08:
		return NewFlowingWaterBlock(Meta)
	case 0x09:
		return NewStillWaterBlock(Meta)
	case 0x0a:
		return NewFlowingLavaBlock(Meta)
	case 0x0b:
		return NewStillLavaBlock(Meta)
	case 0x0c:
		return NewSandBlock()
	case 0x0d:
		return NewGravelBlock()
	case 0x0e:
		return NewGoldOreBlock()
	case 0x0f:
		return NewIronOreBlock()
	case 0x10:
		return NewCoalOreBlock()
	case 0x11:
		return NewLogBlock(Meta)
	case 0x12:
		return NewLeavesBlock(Meta)
	case 0x13:
		return NewSpongeBlock()
	case 0x14:
		return NewGlassBlock()
	case 0x15:
		return NewLapisLazuliOreBlock()
	case 0x16:
		return NewLapisLazuliBlock()
	case 0x17:
		return NewDispenserBlock(Meta)
	case 0x18:
		return NewSandstoneBlock()
	case 0x19:
		return NewNoteblock()
	case 0x1a:
		return NewBedBlock(Meta)
	case 0x1b:
		return NewPoweredRailBlock(Meta)
	case 0x1c:
		return NewDetectorRailBlock(Meta)
	case 0x1d:
		return NewStickyPistonBlock(Meta)
	case 0x1e:
		return NewCobwebBlock()
	case 0x1f:
		return NewTallgrassBlock(Meta)
	case 0x20:
		return NewDeadbushBlock()
	case 0x21:
		return NewPistonBlock(Meta)
	case 0x22:
		return NewPistonHeadBlock(Meta)
	case 0x23:
		return NewWoolBlock(Meta)
	case 0x25:
		return NewDandelionBlock()
	case 0x26:
		return NewRoseBlock()
	case 0x27:
		return NewBrownMushroomBlock()
	case 0x28:
		return NewRedMushroomBlock()
	case 0x29:
		return NewGoldBlock()
	case 0x2a:
		return NewIronBlock()
	case 0x2b:
		return NewDoubleSlabBlock(Meta)
	case 0x2c:
		return NewSlabBlock(Meta)
	case 0x2d:
		return NewBricksBlock()
	case 0x2e:
		return NewTNTBlock()
	case 0x2f:
		return NewBookshelfBlock()
	case 0x30:
		return NewMossyCobblestoneBlock()
	case 0x31:
		return NewObsidianBlock()
	case 0x32:
		return NewTorchBlock(Meta)
	case 0x33:
		return NewFireBlock()
	case 0x34:
		return NewMonsterSpawnerBlock()
	case 0x35:
		return NewWoodenStairsBlock(Meta)
	case 0x36:
		return NewChestBlock()
	case 0x37:
		return NewRedstoneBlock(Meta)
	case 0x38:
		return NewDiamondOreBlock()
	case 0x39:
		return NewDiamondBlock()
	case 0x3a:
		return NewCraftingTable()
	case 0x3b:
		return NewWheatBlock(Meta)
	case 0x3c:
		return NewFarmlandBlock(Meta)
	case 0x3d:
		return NewFurnaceBlock(Meta)
	case 0x3e:
		return NewLitFurnaceBlock(Meta)
	case 0x3f:
		return NewGroundSignBlock(Meta)
	case 0x40:
		return NewWoodenDoorBlock()
	case 0x41:
		return NewLadderBlock(Meta)
	case 0x42:
		return NewRailBlock(Meta)
	case 0x43:
		return NewCobblestoneStairsBlock(Meta)
	case 0x44:
		return NewWallSignBlock(Meta)
	case 0x45:
		return NewLeverBlock(Meta)
	case 0x46:
		return NewStonePressurePlateBlock(Meta)
	case 0x47:
		return NewIronDoorBlock()
	case 0x48:
		return NewWoodenPressurePlateBlock(Meta)
	case 0x49:
		return NewRedstoneOreBlock()
	case 0x4a:
		return NewLitRedstoneOreBlock()
	case 0x4b:
		return NewRedstoneTorchOffBlock(Meta)
	case 0x4c:
		return NewRedstoneTorchOnBlock(Meta)
	case 0x4d:
		return NewStoneButtonBlock(Meta)
	case 0x4e:
		return NewSnowLayerBlock()
	case 0x4f:
		return NewIceBlock()
	case 0x50:
		return NewSnowBlock()
	case 0x51:
		return NewCactusBlock(Meta)
	case 0x52:
		return NewClayBlock()
	case 0x53:
		return NewSugarcaneBlock(Meta)
	case 0x54:
		return NewJukeboxBlock()
	case 0x55:
		return NewFenceBlock()
	case 0x56:
		return NewPumpkinBlock(Meta)
	case 0x57:
		return NewNetherrackBlock()
	case 0x58:
		return NewSoulSandBlock()
	case 0x59:
		return NewGlowstoneBlock()
	case 0x5a:
		return NewNetherPortalBlock()
	case 0x5b:
		return NewLitPumpkinBlock(Meta)
	case 0x5c:
		return NewCakeBlock()
	case 0x5d:
		return NewRedstoneRepeaterOffBlock(Meta)
	case 0x5e:
		return NewRedstoneRepeaterOnBlock(Meta)
	case 0x5f:
		return NewLockedChestBlock()
	case 0x60:
		return NewTrapdoorBlock(Meta)
	case constants.Sign.Value:
		return NewGroundSignBlock(Meta)
	default:
		return NewAirBlock()
	}
}
