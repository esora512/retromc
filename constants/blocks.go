package constants

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
	"bed":                     Bed,
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
	"jack":             PumpkinLit,
	"cake":                    Cake,
	"unlit_redstone_repeater": RedstoneRepeaterOff,
	"lit_redstone_repeater":   RedstoneRepeaterOn,
	"locked_chest":            LockedChest,
	"trapdoor":                Trapdoor,
}

func GetBlockByName(name string) Block {
	if block, ok := BlockCommandMap[name]; ok {
		return block
	}
	return Block{Value: -1, Meta: 0}
}
