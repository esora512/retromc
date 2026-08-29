package constants

import (
	"math/rand"
)

type Block struct {
	Value int16
	Meta  uint16
}

var (
	Air              = Block{Value: 0, Meta: 0}
	Stone            = Block{Value: 1, Meta: 0}
	Grass            = Block{Value: 2, Meta: 0}
	Dirt             = Block{Value: 3, Meta: 0}
	Cobblestone      = Block{Value: 4, Meta: 0}
	Planks           = Block{Value: 5, Meta: 0}
	Sapling          = Block{Value: 6, Meta: 0}
	SpruceSapling    = Block{Value: 6, Meta: 1}
	BirchSapling     = Block{Value: 6, Meta: 2}
	Bedrock          = Block{Value: 7, Meta: 0}
	WaterFlowing     = Block{Value: 8, Meta: 0}
	WaterStill       = Block{Value: 9, Meta: 0}
	LavaFlowing      = Block{Value: 10, Meta: 0}
	LavaStill        = Block{Value: 11, Meta: 0}
	Sand             = Block{Value: 12, Meta: 0}
	Gravel           = Block{Value: 13, Meta: 0}
	GoldOre          = Block{Value: 14, Meta: 0}
	IronOre          = Block{Value: 15, Meta: 0}
	CoalOre          = Block{Value: 16, Meta: 0}
	Log              = Block{Value: 17, Meta: 0}
	Leaves           = Block{Value: 18, Meta: 0}
	Sponge           = Block{Value: 19, Meta: 0}
	Glass            = Block{Value: 20, Meta: 0}
	LapisLazuliOre   = Block{Value: 21, Meta: 0}
	LapisLazuliBlock = Block{Value: 22, Meta: 0}
	Dispenser        = Block{Value: 23, Meta: 0}
	Sandstone        = Block{Value: 24, Meta: 0}
	Noteblock        = Block{Value: 25, Meta: 0}
	Bed              = Block{Value: 26, Meta: 0}
	PoweredRail      = Block{Value: 27, Meta: 0}
	DetectorRail     = Block{Value: 28, Meta: 0}
	StickyPiston     = Block{Value: 29, Meta: 0}
	Cobweb           = Block{Value: 30, Meta: 0}
	Tallgrass        = Block{Value: 31, Meta: 0}
	Deadbush         = Block{Value: 32, Meta: 0}
	Piston           = Block{Value: 33, Meta: 0}
	PistonHead       = Block{Value: 34, Meta: 0}
	Wool             = Block{Value: 35, Meta: 0}
	OrangeWool       = Block{Value: 35, Meta: 1}
	MagentaWool      = Block{Value: 35, Meta: 2}
	LightBlueWool    = Block{Value: 35, Meta: 3}
	YellowWool       = Block{Value: 35, Meta: 4}
	LimeWool         = Block{Value: 35, Meta: 5}
	PinkWool         = Block{Value: 35, Meta: 6}
	GrayWool         = Block{Value: 35, Meta: 7}
	LightGrayWool    = Block{Value: 35, Meta: 8}
	CyanWool         = Block{Value: 35, Meta: 9}
	PurpleWool       = Block{Value: 35, Meta: 10}
	BlueWool         = Block{Value: 35, Meta: 11}
	BrownWool        = Block{Value: 35, Meta: 12}
	GreenWool        = Block{Value: 35, Meta: 13}
	RedWool          = Block{Value: 35, Meta: 14}
	BlackWool        = Block{Value: 35, Meta: 15}
	Dandelion        = Block{Value: 37, Meta: 0}
	Rose             = Block{Value: 38, Meta: 0}
	BrownMushroom    = Block{Value: 39, Meta: 0}
	RedMushroom      = Block{Value: 40, Meta: 0}
	GoldBlock        = Block{Value: 41, Meta: 0}
	IronBlock        = Block{Value: 42, Meta: 0}

	DoubleStoneSlab       = Block{Value: 43, Meta: 0}
	DoubleSandstoneSlab   = Block{Value: 43, Meta: 1}
	DoubleWoodenSlab      = Block{Value: 43, Meta: 2}
	DoubleCobblestoneSlab = Block{Value: 43, Meta: 4}

	StoneSlab       = Block{Value: 44, Meta: 0}
	SandstoneSlab   = Block{Value: 44, Meta: 1}
	WoodenSlab      = Block{Value: 44, Meta: 2}
	CobblestoneSlab = Block{Value: 44, Meta: 3}

	Bricks              = Block{Value: 45, Meta: 0}
	TNT                 = Block{Value: 46, Meta: 0}
	Bookshelf           = Block{Value: 47, Meta: 0}
	MossyCobblestone    = Block{Value: 48, Meta: 0}
	Obsidian            = Block{Value: 49, Meta: 0}
	Torch               = Block{Value: 50, Meta: 0}
	Fire                = Block{Value: 51, Meta: 0}
	MonsterSpawner      = Block{Value: 52, Meta: 0}
	WoodenStairs        = Block{Value: 53, Meta: 0}
	Chest               = Block{Value: 54, Meta: 0}
	RedstoneBlock       = Block{Value: 55, Meta: 0}
	DiamondOre          = Block{Value: 56, Meta: 0}
	DiamondBlock        = Block{Value: 57, Meta: 0}
	CraftingTable       = Block{Value: 58, Meta: 0}
	Wheat               = Block{Value: 59, Meta: 0}
	Farmland            = Block{Value: 60, Meta: 0}
	Furnace             = Block{Value: 61, Meta: 0}
	FurnaceLit          = Block{Value: 62, Meta: 0}
	SignGround          = Block{Value: 63, Meta: 0}
	WoodenDoor          = Block{Value: 64, Meta: 0}
	Ladder              = Block{Value: 65, Meta: 0}
	Rail                = Block{Value: 66, Meta: 0}
	CobblestoneStairs   = Block{Value: 67, Meta: 0}
	SignWall            = Block{Value: 68, Meta: 0}
	Lever               = Block{Value: 69, Meta: 0}
	StonePressurePlate  = Block{Value: 70, Meta: 0}
	IronDoor            = Block{Value: 71, Meta: 0}
	WoodenPressurePlate = Block{Value: 72, Meta: 0}
	RedstoneOreOff      = Block{Value: 73, Meta: 0}
	RedstoneOreOn       = Block{Value: 74, Meta: 0}
	RedstoneTorchOff    = Block{Value: 75, Meta: 0}
	RedstoneTorchOn     = Block{Value: 76, Meta: 0}
	StoneButton         = Block{Value: 77, Meta: 0}
	SnowLayer           = Block{Value: 78, Meta: 0}
	Ice                 = Block{Value: 79, Meta: 0}
	SnowBlock           = Block{Value: 80, Meta: 0}
	Cactus              = Block{Value: 81, Meta: 0}
	Clay                = Block{Value: 82, Meta: 0}
	Sugarcane           = Block{Value: 83, Meta: 0}
	Jukebox             = Block{Value: 84, Meta: 0}
	Fence               = Block{Value: 85, Meta: 0}
	Pumpkin             = Block{Value: 86, Meta: 0}
	Netherrack          = Block{Value: 87, Meta: 0}
	SoulSand            = Block{Value: 88, Meta: 0}
	Glowstone           = Block{Value: 89, Meta: 0}
	NetherPortal        = Block{Value: 90, Meta: 0}
	PumpkinLit          = Block{Value: 91, Meta: 0}
	Cake                = Block{Value: 92, Meta: 0}
	RedstoneRepeaterOff = Block{Value: 93, Meta: 0}
	RedstoneRepeaterOn  = Block{Value: 94, Meta: 0}
	LockedChest         = Block{Value: 95, Meta: 0}
	Trapdoor            = Block{Value: 96, Meta: 0}
)

var BlockCommandMap = map[string]Block{
	"stone":                   Stone,
	"dirt":                    Dirt,
	"furnace":                 Furnace,
	"planks":                  Planks,
	"cobblestone":             Cobblestone,
	"oak_sapling":             Sapling,
	"bedrock":                 Bedrock,
	"water":                   WaterStill,
	"lava":                    LavaStill,
	"sand":                    Sand,
	"gravel":                  Gravel,
	"gold_ore":                GoldOre,
	"iron_ore":                IronOre,
	"coal_ore":                CoalOre,
	"log":                     Log,
	"leaves":                  Leaves,
	"sponge":                  Sponge,
	"glass":                   Glass,
	"lapis_ore":               LapisLazuliOre,
	"lapis_block":             LapisLazuliBlock,
	"dispenser":               Dispenser,
	"sandstone":               Sandstone,
	"noteblock":               Noteblock,
	"rail":                    Rail,
	"powered_rail":            PoweredRail,
	"detector_rail":           DetectorRail,
	"sticky_piston":           StickyPiston,
	"web":                     Cobweb,
	"grass":                   Grass,
	"tallgrass":               Tallgrass,
	"deadbush":                Deadbush,
	"piston":                  Piston,
	"piston_head":             PistonHead,
	"wool":                    Wool,
	"orange_wool":             OrangeWool,
	"magenta_wool":            MagentaWool,
	"light_blue_wool":         LightBlueWool,
	"yellow_wool":             YellowWool,
	"lime_wool":               LimeWool,
	"pink_wool":               PinkWool,
	"gray_wool":               GrayWool,
	"light_gray_wool":         LightGrayWool,
	"cyan_wool":               CyanWool,
	"purple_wool":             PurpleWool,
	"blue_wool":               BlueWool,
	"brown_wool":              BrownWool,
	"green_wool":              GreenWool,
	"red_wool":                RedWool,
	"black_wool":              BlackWool,
	"yellow_flower":           Dandelion,
	"red_flower":              Rose,
	"brown_mushroom":          BrownMushroom,
	"red_mushroom":            RedMushroom,
	"gold_block":              GoldBlock,
	"iron_block":              IronBlock,
	"stone_double_slab":       DoubleStoneSlab,
	"stone_slab":              StoneSlab,
	"sandstone_double_slab":   DoubleSandstoneSlab,
	"sandstone_slab":          SandstoneSlab,
	"wooden_double_slab":      DoubleWoodenSlab,
	"wooden_slab":             WoodenSlab,
	"cobblestone_double_slab": DoubleCobblestoneSlab,
	"cobblestone_slab":        CobblestoneSlab,
	"bricks":                  Bricks,
	"tnt":                     TNT,
	"bookshelf":               Bookshelf,
	"mossy_cobblestone":       MossyCobblestone,
	"obsidian":                Obsidian,
	"torch":                   Torch,
	"fire":                    Fire,
	"mob_spawner":             MonsterSpawner,
	"oak_stairs":              WoodenStairs,
	"chest":                   Chest,
	"redstone_block":          RedstoneBlock,
	"redstone_ore":            RedstoneOreOff,
	"lit_redstone_ore":        RedstoneOreOn,
	"redstone_torch_off":      RedstoneTorchOff,
	"redstone_torch_on":       RedstoneTorchOn,
	"stone_button":            StoneButton,
	"snow_layer":              SnowLayer,
	"ice":                     Ice,
	"snow":                    SnowBlock,
	"cactus":                  Cactus,
	"sapling":                 Sapling,
	"spruce_sapling":          SpruceSapling,
	"birch_sapling":           BirchSapling,
	"clay":                    Clay,
	"reeds":                   Sugarcane,
	"jukebox":                 Jukebox,
	"fence":                   Fence,
	"pumpkin":                 Pumpkin,
	"netherrack":              Netherrack,
	"soul_sand":               SoulSand,
	"glowstone":               Glowstone,
	"portal":                  NetherPortal,
	"jack":                    PumpkinLit,
	"cake":                    Cake,
	"unlit_redstone_repeater": RedstoneRepeaterOff,
	"lit_redstone_repeater":   RedstoneRepeaterOn,
	"locked_chest":            LockedChest,
	"trapdoor":                Trapdoor,
	"lever":                   Lever,
}

func GetBlockByName(name string) Block {
	if block, ok := BlockCommandMap[name]; ok {
		return block
	}
	return Block{Value: -1, Meta: 0}
}

type WBlock struct {
	TypeId   byte
	Metadata byte
	Light    byte
	SkyLight byte
}

var directionalBlocks = map[int16]bool{
	WoodenStairs.Value:        true,
	CobblestoneStairs.Value:   true,
	Torch.Value:               true,
	Chest.Value:               true,
	Furnace.Value:             true,
	FurnaceLit.Value:          true,
	Sign.Value:                true,
	Ladder.Value:              true,
	Dispenser.Value:           true,
	StickyPiston.Value:        true,
	Piston.Value:              true,
	PistonHead.Value:          true,
	SignGround.Value:          true,
	SignWall.Value:            true,
	Lever.Value:               true,
	RedstoneTorchOn.Value:     true,
	RedstoneTorchOff.Value:    true,
	StoneButton.Value:         true,
	Pumpkin.Value:             true,
	PumpkinLit.Value:          true,
	RedstoneRepeater.Value:    true,
	RedstoneRepeaterOn.Value:  true,
	RedstoneRepeaterOff.Value: true,
	LockedChest.Value:         true,
	Trapdoor.Value:            true,
}

type BlockDirections struct {
	North byte
	South byte
	East  byte
	West  byte
}

func (b *WBlock) IsGrowable() bool {
	return b.TypeId == byte(Wheat.Value) || b.TypeId == byte(Sugarcane.Value) || b.TypeId == byte(Cactus.Value) || b.TypeId == byte(Sapling.Value) || b.TypeId == byte(Dirt.Value)
}

func (b *WBlock) IsTransparent() bool {
	return b.TypeId == byte(Air.Value) ||
		b.TypeId == byte(Dandelion.Value) ||
		b.TypeId == byte(Tallgrass.Value) ||
		b.TypeId == byte(Deadbush.Value) ||
		b.TypeId == byte(Grass.Value) ||
		b.TypeId == byte(Glass.Value) ||
		b.TypeId == byte(Sugarcane.Value) ||
		b.TypeId == byte(Rail.Value) ||
		b.TypeId == byte(PoweredRail.Value) ||
		b.TypeId == byte(DetectorRail.Value) ||
		b.TypeId == byte(Leaves.Value) ||
		b.TypeId == byte(SnowLayer.Value)
}

func (b *WBlock) IsFluidReplaceable() bool {
	return b.IsAir() ||
		b.IsSnowLayer() ||
		b.TypeId == byte(Fire.Value) ||
		b.TypeId == byte(Dandelion.Value) ||
		b.TypeId == byte(Rose.Value) ||
		b.TypeId == byte(BrownMushroom.Value) ||
		b.TypeId == byte(RedMushroom.Value) ||
		b.TypeId == byte(Tallgrass.Value)
}

func (b *WBlock) IsRail() bool {
	return b.TypeId == byte(Rail.Value) || b.TypeId == byte(PoweredRail.Value) || b.TypeId == byte(DetectorRail.Value)
}

func (b *WBlock) IsLiquid() bool {
	return b.TypeId == byte(WaterFlowing.Value) || b.TypeId == byte(WaterStill.Value) || b.TypeId == byte(LavaFlowing.Value) || b.TypeId == byte(LavaStill.Value)
}

func (b *WBlock) IsBedHead() bool {
	return b.TypeId == byte(Bed.Value) && b.Metadata&0x8 != 0
}

func (b *WBlock) IsBed() bool {
	return b.TypeId == byte(Bed.Value)
}

func (b *WBlock) IsSnowLayer() bool {
	return b.TypeId == byte(SnowLayer.Value)
}

func (b *WBlock) IsFlowing() bool {
	return b.TypeId == byte(WaterFlowing.Value) || b.TypeId == byte(LavaFlowing.Value)
}

func (b *WBlock) IsFlowingWater() bool {
	return b.TypeId == byte(WaterFlowing.Value)
}

func (b *WBlock) IsFlowingLava() bool {
	return b.TypeId == byte(LavaFlowing.Value)
}

func (b *WBlock) IsStillLava() bool {
	return b.TypeId == byte(LavaStill.Value)
}

func (b *WBlock) IsStillWater() bool {
	return b.TypeId == byte(WaterStill.Value)
}

func (b *WBlock) IsWater() bool {
	return b.TypeId == byte(WaterStill.Value) || b.TypeId == byte(WaterFlowing.Value)
}

func (b *WBlock) IsLava() bool {
	return b.TypeId == byte(LavaStill.Value) || b.TypeId == byte(LavaFlowing.Value)
}

func (b *WBlock) IsFluid() bool {
	return b.IsWater() || b.IsLava()
}

func (b *WBlock) NewBlock(meta byte) WBlock {
	return NewBlockById(int16(b.TypeId), meta)
}

func (b *WBlock) IsPoweredRail() bool {
	return b.TypeId == byte(PoweredRail.Value)
}

func (b *WBlock) GetDirections() BlockDirections {
	switch b.TypeId {
	case byte(Bed.Value):
		return BlockDirections{North: 3, South: 5, West: 2, East: 4}

	case byte(Ladder.Value):
		return BlockDirections{North: 4, South: 5, West: 3, East: 2}

	case byte(SignGround.Value):
		return BlockDirections{North: 4, South: 12, East: 8, West: 0}

	case byte(WoodenStairs.Value), byte(CobblestoneStairs.Value):
		return BlockDirections{North: 0, South: 1, East: 2, West: 3}

	case byte(Torch.Value),
		byte(RedstoneTorchOn.Value),
		byte(RedstoneTorchOff.Value),
		byte(Lever.Value),
		byte(StoneButton.Value):
		return BlockDirections{North: 2, South: 1, East: 4, West: 3}

	case byte(Furnace.Value),
		byte(FurnaceLit.Value),
		byte(Dispenser.Value),
		byte(Chest.Value),
		byte(Piston.Value),
		byte(PistonHead.Value):
		return BlockDirections{North: 4, South: 5, East: 2, West: 3}

	case byte(Pumpkin.Value), byte(PumpkinLit.Value):
		return BlockDirections{North: 1, South: 3, East: 2, West: 0}

	case byte(Trapdoor.Value):
		return BlockDirections{North: 2, South: 3, East: 4, West: 1}

	default:
		return BlockDirections{North: 0, South: 1, East: 2, West: 3}
	}
}

func NewRandomBlock() WBlock {
	val := rand.Intn(94)
	if val == int(Grass.Value) || val == int(Chest.Value) || val == int(Furnace.Value) || val == int(Dispenser.Value) || val == int(FurnaceLit.Value) || val == int(LavaFlowing.Value) || val == int(WaterFlowing.Value) {
		val = int(Stone.Value)
	}
	return NewBlockById(int16(val), 0)
}

func (b *WBlock) IsDirectional() bool {
	return directionalBlocks[int16(b.TypeId)]
}

func (b *WBlock) IsAir() bool {
	return b.TypeId == 0
}

func (b *WBlock) IsDoor() bool {
	return b.TypeId == byte(WoodenDoor.Value) || b.TypeId == byte(IronDoor.Value)
}

func (b *WBlock) IsLever() bool {
	return b.TypeId == byte(Lever.Value)
}

func (b *WBlock) IsButton() bool {
	return b.TypeId == byte(StoneButton.Value)
}

func (b *WBlock) IsTrapdoor() bool {
	return b.TypeId == byte(Trapdoor.Value)
}

func (b *WBlock) IsSolid() bool {
	return !b.IsAir() && !b.IsFluid() && !b.IsSnowLayer()
}

// ID 0 - Air
func NewAirBlock() WBlock {
	return WBlock{TypeId: 0x00, Metadata: 0x00, Light: 0x00, SkyLight: 0x0f}
}

// ID 1 - Stone
func NewStoneBlock() WBlock {
	return WBlock{TypeId: 0x01, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 2 - Grass
func NewGrassBlock() WBlock {
	return WBlock{TypeId: 0x02, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 3 - Dirt
func NewDirtBlock() WBlock {
	return WBlock{TypeId: 0x03, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 4 - Cobblestone
func NewCobblestoneBlock() WBlock {
	return WBlock{TypeId: 0x04, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 5 - Planks (metadata: wood type)
func NewPlanksBlock(woodType byte) WBlock {
	return WBlock{TypeId: 0x05, Metadata: woodType, Light: 0x00, SkyLight: 0x00}
}

// ID 6 - Sapling (metadata: wood type)
func NewSaplingBlock(woodType byte) WBlock {
	return WBlock{TypeId: 0x06, Metadata: woodType, Light: 0x00, SkyLight: 0x00}
}

// ID 7 - Bedrock
func NewBedrockBlock() WBlock {
	return WBlock{TypeId: 0x07, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 8 - Water (Flowing) (metadata: liquid height)
func NewFlowingWaterBlock(liquidHeight byte) WBlock {
	return WBlock{TypeId: 0x08, Metadata: liquidHeight, Light: 0x00, SkyLight: 0x00}
}

// ID 9 - Water (Still) (metadata: liquid height)
func NewStillWaterBlock(liquidHeight byte) WBlock {
	return WBlock{TypeId: 0x09, Metadata: liquidHeight, Light: 0x00, SkyLight: 0x00}
}

// ID 10 - Lava (Flowing) (metadata: liquid height)
func NewFlowingLavaBlock(liquidHeight byte) WBlock {
	return WBlock{TypeId: 0x0a, Metadata: liquidHeight, Light: 0x00, SkyLight: 0x00}
}

// ID 11 - Lava (Still) (metadata: liquid height)
func NewStillLavaBlock(liquidHeight byte) WBlock {
	return WBlock{TypeId: 0x0b, Metadata: liquidHeight, Light: 0x00, SkyLight: 0x00}
}

// ID 12 - Sand
func NewSandBlock() WBlock {
	return WBlock{TypeId: 0x0c, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 13 - Gravel
func NewGravelBlock() WBlock {
	return WBlock{TypeId: 0x0d, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 14 - Gold Ore
func NewGoldOreBlock() WBlock {
	return WBlock{TypeId: 0x0e, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 15 - Iron Ore
func NewIronOreBlock() WBlock {
	return WBlock{TypeId: 0x0f, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 16 - Coal Ore
func NewCoalOreBlock() WBlock {
	return WBlock{TypeId: 0x10, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 17 - Log (metadata: wood type)
func NewLogBlock(woodType byte) WBlock {
	return WBlock{TypeId: 0x11, Metadata: woodType, Light: 0x00, SkyLight: 0x00}
}

// ID 18 - Leaves (metadata: leaf type)
func NewLeavesBlock(leafType byte) WBlock {
	return WBlock{TypeId: 0x12, Metadata: leafType, Light: 0x00, SkyLight: 0x00}
}

// ID 19 - Sponge
func NewSpongeBlock() WBlock {
	return WBlock{TypeId: 0x13, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 20 - Glass
func NewGlassBlock() WBlock {
	return WBlock{TypeId: 0x14, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 21 - Lapis Lazuli Ore
func NewLapisLazuliOreBlock() WBlock {
	return WBlock{TypeId: 0x15, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 22 - Lapis Lazuli WBlock
func NewLapisLazuliBlock() WBlock {
	return WBlock{TypeId: 0x16, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 23 - Dispenser (metadata: direction)
func NewDispenserBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x17, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 24 - Sandstone
func NewSandstoneBlock() WBlock {
	return WBlock{TypeId: 0x18, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 25 - Noteblock
func NewNoteblock() WBlock {
	return WBlock{TypeId: 0x19, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 26 - Bed (metadata: top/bottom and direction)
func NewBedBlock(meta byte) WBlock {
	return WBlock{TypeId: 0x1a, Metadata: meta, Light: 0x00, SkyLight: 0x00}
}

// ID 27 - Powered Rail (metadata: direction)
func NewPoweredRailBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x1b, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 28 - Detector Rail (metadata: direction)
func NewDetectorRailBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x1c, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 29 - Sticky Piston (metadata: direction and state)
func NewStickyPistonBlock(meta byte) WBlock {
	return WBlock{TypeId: 0x1d, Metadata: meta, Light: 0x00, SkyLight: 0x00}
}

// ID 30 - Cobweb
func NewCobwebBlock() WBlock {
	return WBlock{TypeId: 0x1e, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 31 - Tallgrass (metadata: 0=shrub, 1=grass, 2=fern)
func NewTallgrassBlock(variant byte) WBlock {
	return WBlock{TypeId: 0x1f, Metadata: variant, Light: 0x00, SkyLight: 0x00}
}

// ID 32 - Deadbush
func NewDeadbushBlock() WBlock {
	return WBlock{TypeId: 0x20, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 33 - Piston (metadata: direction and state)
func NewPistonBlock(meta byte) WBlock {
	return WBlock{TypeId: 0x21, Metadata: meta, Light: 0x00, SkyLight: 0x00}
}

// ID 34 - Piston Head (metadata: direction)
func NewPistonHeadBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x22, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 35 - Wool (metadata: color)
func NewWoolBlock(color byte) WBlock {
	return WBlock{TypeId: 0x23, Metadata: color, Light: 0x00, SkyLight: 0x00}
}

// ID 37 - Dandelion
func NewDandelionBlock() WBlock {
	return WBlock{TypeId: 0x25, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 38 - Rose
func NewRoseBlock() WBlock {
	return WBlock{TypeId: 0x26, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 39 - Brown Mushroom
func NewBrownMushroomBlock() WBlock {
	return WBlock{TypeId: 0x27, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 40 - Red Mushroom
func NewRedMushroomBlock() WBlock {
	return WBlock{TypeId: 0x28, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 41 - Gold WBlock
func NewGoldBlock() WBlock {
	return WBlock{TypeId: 0x29, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 42 - Iron WBlock
func NewIronBlock() WBlock {
	return WBlock{TypeId: 0x2a, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 43 - Double Slab (metadata: 0=stone, 1=sandstone, 2=wood, 3=cobblestone)
func NewDoubleSlabBlock(slabType byte) WBlock {
	return WBlock{TypeId: 0x2b, Metadata: slabType, Light: 0x00, SkyLight: 0x00}
}

// ID 44 - Slab (metadata: 0=stone, 1=sandstone, 2=wood, 3=cobblestone)
func NewSlabBlock(slabType byte) WBlock {
	return WBlock{TypeId: 0x2c, Metadata: slabType, Light: 0x00, SkyLight: 0x00}
}

// ID 45 - Bricks
func NewBricksBlock() WBlock {
	return WBlock{TypeId: 0x2d, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 46 - TNT
func NewTNTBlock() WBlock {
	return WBlock{TypeId: 0x2e, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 47 - Bookshelf
func NewBookshelfBlock() WBlock {
	return WBlock{TypeId: 0x2f, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 48 - Mossy Cobblestone
func NewMossyCobblestoneBlock() WBlock {
	return WBlock{TypeId: 0x30, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 49 - Obsidian
func NewObsidianBlock() WBlock {
	return WBlock{TypeId: 0x31, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 50 - Torch (metadata: direction)
func NewTorchBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x32, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 51 - Fire
func NewFireBlock() WBlock {
	return WBlock{TypeId: 0x33, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 52 - Monster Spawner
func NewMonsterSpawnerBlock() WBlock {
	return WBlock{TypeId: 0x34, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 53 - Wooden Stairs (metadata: direction)
func NewWoodenStairsBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x35, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 54 - Chest
func NewChestBlock() WBlock {
	return WBlock{TypeId: 0x36, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 55 - Redstone (metadata: power level)
func NewRedstoneBlock(powerLevel byte) WBlock {
	return WBlock{TypeId: 0x37, Metadata: powerLevel, Light: 0x00, SkyLight: 0x00}
}

// ID 56 - Diamond Ore
func NewDiamondOreBlock() WBlock {
	return WBlock{TypeId: 0x38, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 57 - Diamond WBlock
func NewDiamondBlock() WBlock {
	return WBlock{TypeId: 0x39, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 58 - Crafting Table
func NewCraftingTable() WBlock {
	return WBlock{TypeId: 0x3a, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 59 - Wheat (metadata: growth stage 0-7)
func NewWheatBlock(growthStage byte) WBlock {
	return WBlock{TypeId: 0x3b, Metadata: growthStage, Light: 0x00, SkyLight: 0x00}
}

// ID 60 - Farmland (metadata: >0 if wet)
func NewFarmlandBlock(wetness byte) WBlock {
	return WBlock{TypeId: 0x3c, Metadata: wetness, Light: 0x00, SkyLight: 0x00}
}

// ID 61 - Furnace (metadata: direction)
func NewFurnaceBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x3d, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 62 - Furnace (Lit) (metadata: direction)
func NewLitFurnaceBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x3e, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 63 - Sign (Ground) (metadata: direction)
func NewGroundSignBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x3f, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 64 - Wooden Door
func NewWoodenDoorBlock() WBlock {
	return WBlock{TypeId: 0x40, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 65 - Ladder (metadata: direction)
func NewLadderBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x41, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 66 - Rail (metadata: direction)
func NewRailBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x42, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 67 - Cobblestone Stairs (metadata: direction)
func NewCobblestoneStairsBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x43, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 68 - Sign (Wall) (metadata: direction)
func NewWallSignBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x44, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 69 - Lever (metadata: toggled & direction)
func NewLeverBlock(meta byte) WBlock {
	return WBlock{TypeId: 0x45, Metadata: meta, Light: 0x00, SkyLight: 0x00}
}

// ID 70 - Stone Pressure Plate (metadata: toggled)
func NewStonePressurePlateBlock(toggled byte) WBlock {
	return WBlock{TypeId: 0x46, Metadata: toggled, Light: 0x00, SkyLight: 0x00}
}

// ID 71 - Iron Door
func NewIronDoorBlock() WBlock {
	return WBlock{TypeId: 0x47, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 72 - Wooden Pressure Plate (metadata: toggled)
func NewWoodenPressurePlateBlock(toggled byte) WBlock {
	return WBlock{TypeId: 0x48, Metadata: toggled, Light: 0x00, SkyLight: 0x00}
}

// ID 73 - Redstone Ore (Off)
func NewRedstoneOreBlock() WBlock {
	return WBlock{TypeId: 0x49, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 74 - Redstone Ore (On)
func NewLitRedstoneOreBlock() WBlock {
	return WBlock{TypeId: 0x4a, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 75 - Redstone Torch (Off) (metadata: direction)
func NewRedstoneTorchOffBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x4b, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 76 - Redstone Torch (On) (metadata: direction)
func NewRedstoneTorchOnBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x4c, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 77 - Stone Button (metadata: toggled & direction)
func NewStoneButtonBlock(meta byte) WBlock {
	return WBlock{TypeId: 0x4d, Metadata: meta, Light: 0x00, SkyLight: 0x00}
}

// ID 78 - Snow Layer
func NewSnowLayerBlock() WBlock {
	return WBlock{TypeId: 0x4e, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 79 - Ice
func NewIceBlock() WBlock {
	return WBlock{TypeId: 0x4f, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 80 - Snow WBlock
func NewSnowBlock() WBlock {
	return WBlock{TypeId: 0x50, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 81 - Cactus
func NewCactusBlock(Meta byte) WBlock {
	return WBlock{TypeId: 0x51, Metadata: Meta, Light: 0x00, SkyLight: 0x00}
}

// ID 82 - Clay
func NewClayBlock() WBlock {
	return WBlock{TypeId: 0x52, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 83 - Sugarcane
func NewSugarcaneBlock(Meta byte) WBlock {
	return WBlock{TypeId: 0x53, Metadata: Meta, Light: 0x00, SkyLight: 0x00}
}

// ID 84 - Jukebox
func NewJukeboxBlock() WBlock {
	return WBlock{TypeId: 0x54, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 85 - Fence
func NewFenceBlock() WBlock {
	return WBlock{TypeId: 0x55, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 86 - Pumpkin (metadata: direction)
func NewPumpkinBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x56, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 87 - Netherrack
func NewNetherrackBlock() WBlock {
	return WBlock{TypeId: 0x57, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 88 - Soul Sand
func NewSoulSandBlock() WBlock {
	return WBlock{TypeId: 0x58, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 89 - Glowstone
func NewGlowstoneBlock() WBlock {
	return WBlock{TypeId: 0x59, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 90 - Nether Portal
func NewNetherPortalBlock() WBlock {
	return WBlock{TypeId: 0x5a, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 91 - Pumpkin (Lit) (metadata: direction)
func NewLitPumpkinBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x5b, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 92 - Cake
func NewCakeBlock() WBlock {
	return WBlock{TypeId: 0x5c, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 93 - Redstone Repeater (Off) (metadata: direction)
func NewRedstoneRepeaterOffBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x5d, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 94 - Redstone Repeater (On) (metadata: direction)
func NewRedstoneRepeaterOnBlock(direction byte) WBlock {
	return WBlock{TypeId: 0x5e, Metadata: direction, Light: 0x00, SkyLight: 0x00}
}

// ID 95 - Locked Chest
func NewLockedChestBlock() WBlock {
	return WBlock{TypeId: 0x5f, Metadata: 0x00, Light: 0x00, SkyLight: 0x00}
}

// ID 96 - Trapdoor (metadata: toggled & direction)
func NewTrapdoorBlock(meta byte) WBlock {
	return WBlock{TypeId: 0x60, Metadata: meta, Light: 0x00, SkyLight: 0x00}
}

func NewBlockById(id int16, Meta byte) WBlock {
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
	case Sign.Value:
		return NewGroundSignBlock(Meta)
	default:
		return NewAirBlock()
	}
}
