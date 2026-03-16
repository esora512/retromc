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

	RedstoneDust = Item{Value: 356, Meta: 0}
	Coal         = Item{Value: 263, Meta: 0}
	Shears       = Item{Value: 359, Meta: 0}
	Snowball     = Item{Value: 332, Meta: 0}
)
