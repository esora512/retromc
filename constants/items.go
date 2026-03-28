package constants


type Item struct {
	Value int16
	Meta  byte
}

var (
	Stick       = Item{Value: 280, Meta: 0}
	IronShovel  = Item{Value: 256, Meta: 0}
	IronPickaxe = Item{Value: 257, Meta: 0}
	IronAxe     = Item{Value: 258, Meta: 0}
	IronSword   = Item{Value: 267, Meta: 0}

	WoodenSword    = Item{Value: 268, Meta: 0}
	WoodenShovel   = Item{Value: 269, Meta: 0}
	WoodenPickaxe  = Item{Value: 270, Meta: 0}
	WoodenAxe      = Item{Value: 271, Meta: 0}
	StoneSword     = Item{Value: 272, Meta: 0}
	StoneShovel    = Item{Value: 273, Meta: 0}
	StonePickaxe   = Item{Value: 274, Meta: 0}
	StoneAxe       = Item{Value: 275, Meta: 0}
	DiamondSword   = Item{Value: 276, Meta: 0}
	DiamondShovel  = Item{Value: 277, Meta: 0}
	DiamondPickaxe = Item{Value: 278, Meta: 0}
	DiamondAxe     = Item{Value: 279, Meta: 0}
	GoldSword      = Item{Value: 283, Meta: 0}
	GoldShovel     = Item{Value: 284, Meta: 0}
	GoldPickaxe    = Item{Value: 285, Meta: 0}
	GoldAxe        = Item{Value: 286, Meta: 0}

	WoodenHoe  = Item{Value: 290, Meta: 0}
	StoneHoe   = Item{Value: 291, Meta: 0}
	IronHoe    = Item{Value: 292, Meta: 0}
	DiamondHoe = Item{Value: 293, Meta: 0}
	GoldHoe    = Item{Value: 294, Meta: 0}

	Iron    = Item{Value: 265, Meta: 0}
	Gold    = Item{Value: 266, Meta: 0}
	Diamond = Item{Value: 264, Meta: 0}

	Redstone = Item{Value: 331, Meta: 0}
	Coal     = Item{Value: 263, Meta: 0}
	Shears   = Item{Value: 359, Meta: 0}
	Snowball = Item{Value: 332, Meta: 0}

	FlintAndSteel = Item{Value: 259, Meta: 0}
	Apple         = Item{Value: 260, Meta: 0}
	Bow           = Item{Value: 261, Meta: 0}
	Arrow         = Item{Value: 262, Meta: 0}

	Bowl         = Item{Value: 281, Meta: 0}
	MushroomStew = Item{Value: 282, Meta: 0}

	String    = Item{Value: 287, Meta: 0}
	Feather   = Item{Value: 288, Meta: 0}
	Gunpowder = Item{Value: 289, Meta: 0}

	Seeds     = Item{Value: 295, Meta: 0}
	WheatItem = Item{Value: 296, Meta: 0}
	Bread     = Item{Value: 297, Meta: 0}

	LeatherCap          = Item{Value: 298, Meta: 0}
	LeatherTunic        = Item{Value: 299, Meta: 0}
	LeatherPants        = Item{Value: 300, Meta: 0}
	LeatherBoots        = Item{Value: 301, Meta: 0}
	ChainmailHelmet     = Item{Value: 302, Meta: 0}
	ChainmailChestplate = Item{Value: 303, Meta: 0}
	ChainmailLeggings   = Item{Value: 304, Meta: 0}
	ChainmailBoots      = Item{Value: 305, Meta: 0}
	IronHelmet          = Item{Value: 306, Meta: 0}
	IronChestplate      = Item{Value: 307, Meta: 0}
	IronLeggings        = Item{Value: 308, Meta: 0}
	IronBoots           = Item{Value: 309, Meta: 0}
	DiamondHelmet       = Item{Value: 310, Meta: 0}
	DiamondChestplate   = Item{Value: 311, Meta: 0}
	DiamondLeggings     = Item{Value: 312, Meta: 0}
	DiamondBoots        = Item{Value: 313, Meta: 0}
	GoldHelmet          = Item{Value: 314, Meta: 0}
	GoldChestplate      = Item{Value: 315, Meta: 0}
	GoldLeggings        = Item{Value: 316, Meta: 0}
	GoldBoots           = Item{Value: 317, Meta: 0}

	Flint           = Item{Value: 318, Meta: 0}
	Porkchop        = Item{Value: 319, Meta: 0}
	CookedPorkchop  = Item{Value: 320, Meta: 0}
	Painting        = Item{Value: 321, Meta: 0}
	GoldenApple     = Item{Value: 322, Meta: 0}
	Sign            = Item{Value: 323, Meta: 0}
	WoodenDoorItem  = Item{Value: 324, Meta: 0}
	Bucket          = Item{Value: 325, Meta: 0}
	WaterBucket     = Item{Value: 326, Meta: 0}
	LavaBucket      = Item{Value: 327, Meta: 0}
	Minecart        = Item{Value: 328, Meta: 0}
	Saddle          = Item{Value: 329, Meta: 0}
	IronDoorItem    = Item{Value: 330, Meta: 0}
	Boat            = Item{Value: 333, Meta: 0}
	Leather         = Item{Value: 334, Meta: 0}
	MilkBucket      = Item{Value: 335, Meta: 0}
	Brick           = Item{Value: 336, Meta: 0}
	ClayItem        = Item{Value: 337, Meta: 0}
	SugarcaneItem   = Item{Value: 338, Meta: 0}
	Paper           = Item{Value: 339, Meta: 0}
	Book            = Item{Value: 340, Meta: 0}
	Slime           = Item{Value: 341, Meta: 0}
	ChestMinecart   = Item{Value: 342, Meta: 0}
	FurnaceMinecart = Item{Value: 343, Meta: 0}
	Egg             = Item{Value: 344, Meta: 0}
	Compass         = Item{Value: 345, Meta: 0}
	FishingRod      = Item{Value: 346, Meta: 0}
	Clock           = Item{Value: 347, Meta: 0}
	GlowstoneDust   = Item{Value: 348, Meta: 0}
	Fish            = Item{Value: 349, Meta: 0}
	CookedFish      = Item{Value: 350, Meta: 0}

	Dye = Item{Value: 351, Meta: 0}

	Bone             = Item{Value: 352, Meta: 0}
	Sugar            = Item{Value: 353, Meta: 0}
	CakeItem         = Item{Value: 354, Meta: 0}
	BedItem          = Item{Value: 355, Meta: 0}
	RedstoneRepeater = Item{Value: 356, Meta: 0}
	Cookie           = Item{Value: 357, Meta: 0}
	Map              = Item{Value: 358, Meta: 0}

	Record13  = Item{Value: 2256, Meta: 0}
	RecordCat = Item{Value: 2257, Meta: 0}
)

var itemMap = map[string]Item{
	"stick":       Stick,
	"iron_shovel": IronShovel,
	"iron_pickaxe": IronPickaxe,
	"iron_axe":     IronAxe,
	"iron_sword":   IronSword,
	"wooden_sword": WoodenSword,
	"wooden_shovel": WoodenShovel,
	"wooden_pickaxe": WoodenPickaxe,
	"wooden_axe":     WoodenAxe,
	"stone_sword":   StoneSword,
	"stone_shovel":  StoneShovel,
	"stone_pickaxe": StonePickaxe,
	"stone_axe":     StoneAxe,
	"diamond_sword":   DiamondSword,
	"diamond_shovel":  DiamondShovel,
	"diamond_pickaxe": DiamondPickaxe,
	"diamond_axe":     DiamondAxe,
	"gold_sword":   GoldSword,
	"gold_shovel":  GoldShovel,
	"gold_pickaxe": GoldPickaxe,
	"gold_axe":     GoldAxe,
	"iron_hoe":     IronHoe,
	"wooden_hoe":   WoodenHoe,
	"stone_hoe":    StoneHoe,
	"diamond_hoe":  DiamondHoe,
	"gold_hoe":     GoldHoe,
	"iron":    Iron,
	"gold":    Gold,
	"diamond": Diamond,
	"redstone": Redstone,
	"coal":     Coal,
	"shears":   Shears,
	"snowball": Snowball,
	"flint_and_steel": FlintAndSteel,
	"apple":         Apple,
	"bow":           Bow,
	"arrow":         Arrow,
	"bowl":         Bowl,
	"mushroom_stew": MushroomStew,
	"string":    String,
	"feather":   Feather,
	"gunpowder": Gunpowder,
	"seeds":     Seeds,
	"wheat":     WheatItem,
	"bread":     Bread,
	"leather_cap":          LeatherCap,
	"leather_tunic":        LeatherTunic,
	"leather_pants":        LeatherPants,
	"leather_boots":        LeatherBoots,
	"chainmail_helmet":     ChainmailHelmet,
	"chainmail_chestplate": ChainmailChestplate,
	"chainmail_leggings":   ChainmailLeggings,
	"chainmail_boots":      ChainmailBoots,
	"iron_helmet":          IronHelmet,
	"iron_chestplate":      IronChestplate,
	"iron_leggings":        IronLeggings,
	"iron_boots":           IronBoots,
	"diamond_helmet":       DiamondHelmet,
	"diamond_chestplate":   DiamondChestplate,
	"diamond_leggings":     DiamondLeggings,
	"diamond_boots":        DiamondBoots,
	"gold_helmet":          GoldHelmet,
	"gold_chestplate":      GoldChestplate,
	"gold_leggings":        GoldLeggings,
	"gold_boots":           GoldBoots,
	"flint":           Flint,
	"porkchop":        Porkchop,
	"cooked_porkchop": CookedPorkchop,
	"painting":        Painting,
	"golden_apple":     GoldenApple,
	"sign":            Sign,
	"wooden_door":     WoodenDoorItem,
	"bucket":          Bucket,
	"water_bucket":     WaterBucket,
	"lava_bucket":      LavaBucket,
	"minecart":        Minecart,
	"saddle":          Saddle,
	"iron_door":    IronDoorItem,
	"boat":            Boat,
	"leather":         Leather,
	"milk_bucket":      MilkBucket,
	"brick":           Brick,
	"clay":            ClayItem,
	"sugarcane":       SugarcaneItem,
	"paper":           Paper,
	"book":            Book,
	"slime":           Slime,
	"chest_minecart":   ChestMinecart,
	"furnace_minecart": FurnaceMinecart,
	"egg":             Egg,
	"compass":         Compass,
	"fishing_rod":      FishingRod,
	"clock":           Clock,
	"glowstone_dust":   GlowstoneDust,
	"fish":             Fish,
	"cooked_fish":      CookedFish,
	"dye":              Dye,
	"bone":             Bone,
	"sugar":            Sugar,
	"cake":             CakeItem,
	"bed":              BedItem,
	"redstone_repeater": RedstoneRepeater,
	"cookie":             Cookie,
	"map":                Map,
	"record_13":          Record13,
	"record_cat":         RecordCat,
}

func GetItemByName(name string) Item {
	if item, ok := itemMap[name]; ok {
		return item
	}
	return Item{Value: -1, Meta: 0}
}
