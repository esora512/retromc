package crafting

import (
	"github.com/leNicDev/retromc/constants"
)

type Result struct {
	TypeId   int16
	Metadata byte
	Count    byte
}

type Recipe struct {
	Pattern [4]int16
	Result  Result
}

type Recipe3x3 struct {
	Pattern [9]int16
	Result  Result
}

type Crafter2x2 struct {
	Recipes []Recipe
}

type Crafter3x3 struct {
	Recipes []Recipe3x3
}

func New2x2Crafter() *Crafter2x2 {
	return &Crafter2x2{
		Recipes: []Recipe{
			// crafting table
			{Pattern: [4]int16{5, 5, 5, 5}, Result: Result{58, 0, 1}},
		},
	}
}

func (c *Crafter2x2) Craft(grid [4]int16) Result {
	for _, r := range c.Recipes {
		if patternMatches4(r.Pattern, grid) {
			return r.Result
		}
	}
	return Result{-1, 0, 0}
}

func patternMatches4(pattern, grid [4]int16) bool {
	for i := range pattern {
		if pattern[i] != grid[i] {
			return false
		}
	}
	return true
}

func getGridData(grid [9]int16) (int16, int16, int16, int16, bool) {
	var itemId = int16(-1)
	var location int16 = -1
	var count int16 = 0
	var itemCount int16 = 0
	var same bool = true
	for i := range grid {
		if grid[i] == -1 {
			continue
		}
		if itemId == -1 && grid[i] != -1 {
			itemId = grid[i]
			location = int16(i)
		}
		if grid[i] == itemId {
			itemCount++
		}
		if grid[i] != itemId {
			same = false
		}

		count++
	}
	return itemId, location, count, itemCount, same
}

func getToolForType(itemId int16, toolName string) int16 {
	toolMap := map[string]map[int16]int16{
		"axe": {
			constants.Planks.Value:      constants.WoodenAxe.Value,
			constants.Cobblestone.Value: constants.StoneAxe.Value,
			constants.Iron.Value:        constants.IronAxe.Value,
			constants.Gold.Value:        constants.GoldAxe.Value,
			constants.Diamond.Value:     constants.DiamondAxe.Value,
		},
		"pickaxe": {
			constants.Planks.Value:      constants.WoodenPickaxe.Value,
			constants.Cobblestone.Value: constants.StonePickaxe.Value,
			constants.Iron.Value:        constants.IronPickaxe.Value,
			constants.Gold.Value:        constants.GoldPickaxe.Value,
			constants.Diamond.Value:     constants.DiamondPickaxe.Value,
		},
		"shovel": {
			constants.Planks.Value:      constants.WoodenShovel.Value,
			constants.Cobblestone.Value: constants.StoneShovel.Value,
			constants.Iron.Value:        constants.IronShovel.Value,
			constants.Gold.Value:        constants.GoldShovel.Value,
			constants.Diamond.Value:     constants.DiamondShovel.Value,
		},
		"hoe": {
			constants.Planks.Value:      constants.WoodenHoe.Value,
			constants.Cobblestone.Value: constants.StoneHoe.Value,
			constants.Iron.Value:        constants.IronHoe.Value,
			constants.Gold.Value:        constants.GoldHoe.Value,
			constants.Diamond.Value:     constants.DiamondHoe.Value,
		},
		"sword": {
			constants.Planks.Value:      constants.WoodenSword.Value,
			constants.Cobblestone.Value: constants.StoneSword.Value,
			constants.Iron.Value:        constants.IronSword.Value,
			constants.Gold.Value:        constants.GoldSword.Value,
			constants.Diamond.Value:     constants.DiamondSword.Value,
		},
	}

	if materials, ok := toolMap[toolName]; ok {
		if tool, ok := materials[itemId]; ok {
			return tool
		}
	}
	return -1
}

func getSlab(itemId int16) (int16, byte) {
	switch itemId {
	case constants.Planks.Value:
		return constants.WoodenSlab.Value, constants.WoodenSlab.Meta
	case constants.Cobblestone.Value:
		return constants.CobblestoneSlab.Value, constants.CobblestoneSlab.Meta
	case constants.Stone.Value:
		return constants.StoneSlab.Value, constants.StoneSlab.Meta
	case constants.Sandstone.Value:
		return constants.SandstoneSlab.Value, constants.SandstoneSlab.Meta
	case constants.Wheat.Value:
		return constants.Bread.Value, byte(0)
	default:
		return -1, byte(0)
	}
}

func getArmorForType(itemId int16, armorPiece string) int16 {
	if armorPiece == "helmet" {
		switch itemId {
		case constants.Leather.Value:
			return constants.LeatherCap.Value
		case constants.Iron.Value:
			return constants.IronHelmet.Value
		case constants.Diamond.Value:
			return constants.DiamondHelmet.Value
		case constants.Gold.Value:
			return constants.GoldHelmet.Value
		}
	}
	if armorPiece == "chestplate" {
		switch itemId {
		case constants.Leather.Value:
			return constants.LeatherTunic.Value
		case constants.Iron.Value:
			return constants.IronChestplate.Value
		case constants.Diamond.Value:
			return constants.DiamondChestplate.Value
		case constants.Gold.Value:
			return constants.GoldChestplate.Value
		}
	}
	if armorPiece == "leggings" {
		switch itemId {
		case constants.Leather.Value:
			return constants.LeatherPants.Value
		case constants.Iron.Value:
			return constants.IronLeggings.Value
		case constants.Diamond.Value:
			return constants.DiamondLeggings.Value
		case constants.Gold.Value:
			return constants.GoldLeggings.Value
		}
	}
	if armorPiece == "boots" {
		switch itemId {
		case constants.Leather.Value:
			return constants.LeatherBoots.Value
		case constants.Iron.Value:
			return constants.IronBoots.Value
		case constants.Diamond.Value:
			return constants.DiamondBoots.Value
		case constants.Gold.Value:
			return constants.GoldBoots.Value
		}
	}
	return 0
}

func getStairsForType(itemId int16) int16 {
	switch itemId {
	case constants.Planks.Value:
		return constants.WoodenStairs.Value
	case constants.Cobblestone.Value:
		return constants.CobblestoneStairs.Value
	default:
		return -1
	}
}

// Based on bareiron by p2r3: https://github.com/p2r3/bareiron/blob/main/src/crafting.c
// more "fun" and perhaps more efficient

// TODO: Add cake (special recipe as it leaves items behind!)
func Craft(grid [9]int16) Result {
	// first item id, location, total count and if all items are first item
	itemId, location, count, itemCount, same := getGridData(grid)
	if itemId == -1 {
		return Result{-1, 0, 0}
	}
	firstRow := location / 3    // If 0, item in first row
	firstColumn := location % 3 // If 0, item in first column
	switch count {
	case 1:
		switch itemId {
		case constants.Log.Value:
			return Result{constants.Planks.Value, 0, 4}
		case constants.IronBlock.Value:
			return Result{constants.Iron.Value, 0, 9}
		case constants.GoldBlock.Value:
			return Result{constants.Gold.Value, 0, 9}
		case constants.DiamondBlock.Value:
			return Result{constants.Diamond.Value, 0, 9}
		case constants.RedstoneBlock.Value:
			return Result{constants.Redstone.Value, 0, 9}
		case constants.SugarcaneItem.Value:
			return Result{constants.Sugar.Value, 0, 3}
		}

	case 2:
		switch itemId {
		case constants.Slime.Value:
			if firstColumn != 2 && grid[location+1] == constants.Piston.Value {
				return Result{constants.StickyPiston.Value, 0, 1}
			}

		case constants.Planks.Value:
			if firstColumn != 2 && grid[location+1] == itemId {
				return Result{constants.WoodenPressurePlate.Value, 0, 1}
			} else if firstRow != 2 && grid[location+3] == itemId {
				return Result{constants.Stick.Value, 0, 4}
			}
		case constants.Coal.Value:
			if firstRow != 2 && grid[location+3] == constants.Stick.Value {
				return Result{constants.Torch.Value, 0, 4}
			}
		case constants.Redstone.Value:
			if firstRow != 2 && grid[location+3] == constants.Stick.Value {
				return Result{constants.RedstoneTorchOn.Value, 0, 1}
			}
		case constants.Iron.Value:
			if ((firstRow != 2 && firstColumn != 2) && grid[location+4] == itemId) || ((firstRow != 2 && firstColumn != 0) && grid[location+2] == itemId) {
				return Result{constants.Shears.Value, 0, 1}
			}
			if grid[location] == itemId && grid[location+4] == constants.Flint.Value {
				return Result{constants.FlintAndSteel.Value, 0, 1}
			}
		case constants.Stone.Value:
			if grid[location] == itemId && grid[location+3] == itemId {
				return Result{constants.StoneButton.Value, 0, 1}
			}
			if firstColumn == 0 && grid[location] == itemId && grid[location+1] == itemId {
				return Result{constants.StonePressurePlate.Value, 0, 1}
			}
		case constants.Minecart.Value:
			if location > 2 && grid[location-3] != -1 {
				if grid[location-3] == constants.Chest.Value {
					return Result{constants.ChestMinecart.Value, 0, 1}
				}
				if grid[location-3] == constants.Furnace.Value {
					return Result{constants.FurnaceMinecart.Value, 0, 1}
				}
			}
		case constants.Stick.Value:
			if firstRow != 2 && grid[location+3] == constants.Cobblestone.Value {
				return Result{constants.Lever.Value, 0, 1}
			}
		case constants.Pumpkin.Value:
			if firstRow != 2 && grid[location+3] == constants.Torch.Value {
				return Result{constants.PumpkinLit.Value, 0, 1}
			}
		}
	case 3:
		if same {
			switch itemId {
			case constants.Stone.Value, constants.Sandstone.Value, constants.Planks.Value, constants.Cobblestone.Value, constants.Wheat.Value:
				// slab
				if firstColumn == 0 && grid[location] == itemId && grid[location+1] == itemId && grid[location+2] == itemId {
					slabId, meta := getSlab(itemId)
					return Result{slabId, meta, 3}
				}
				if firstColumn == 0 && grid[location+4] == itemId && grid[location+2] == itemId && itemId == constants.Planks.Value {
					return Result{constants.Bowl.Value, 0, 4}
				}

			case constants.SugarcaneItem.Value:
				if firstColumn == 0 && grid[location] == itemId && grid[location+1] == itemId && grid[location+2] == itemId {
					return Result{constants.Paper.Value, 0, 3}
				}
			case constants.Paper.Value:
				if firstRow == 0 && grid[location] == itemId && grid[location+3] == itemId && grid[location+6] == itemId {
					return Result{constants.Book.Value, 0, 1}
				}
			case constants.Iron.Value:
				if firstColumn == 0 && grid[location+4] == itemId && grid[location+2] == itemId {
					return Result{constants.Bucket.Value, 0, 1}
				}
			}
		} else {
			switch itemId {
			case constants.Planks.Value, constants.Cobblestone.Value, constants.Iron.Value, constants.Gold.Value, constants.Diamond.Value:
				if firstRow == 0 &&
					grid[location+3] == constants.Stick.Value &&
					grid[location+6] == constants.Stick.Value {
					return Result{getToolForType(itemId, "shovel"), 0, 1}
				}
				if firstRow == 0 &&
					grid[location+3] == itemId &&
					grid[location+6] == constants.Stick.Value {
					return Result{getToolForType(itemId, "sword"), 0, 1}
				}
			case constants.Flint.Value:
				if grid[0] == itemId && grid[4] == constants.Stick.Value && grid[8] == constants.Feather.Value {
					return Result{constants.Arrow.Value, 0, 4}
				}
			}
		}

	case 4:
		switch itemId {
		case constants.Planks.Value, constants.Snowball.Value, constants.ClayItem.Value, constants.Brick.Value, constants.String.Value, constants.GlowstoneDust.Value:
			if firstColumn != 2 && firstRow != 2 && same {
				if itemId == constants.Planks.Value {
					return Result{constants.CraftingTable.Value, 0, 1}
				}
				if itemId == constants.Snowball.Value {
					return Result{constants.SnowBlock.Value, 0, 1}
				}
				if itemId == constants.ClayItem.Value {
					return Result{constants.Clay.Value, 0, 1}
				}
				if itemId == constants.Brick.Value {
					return Result{constants.Bricks.Value, 0, 1}
				}
				if itemId == constants.String.Value {
					return Result{constants.Wool.Value, 0, 1}
				}
				if itemId == constants.GlowstoneDust.Value {
					return Result{constants.Glowstone.Value, 0, 1}
				}
			}
		case constants.Leather.Value, constants.Iron.Value, constants.Gold.Value, constants.Diamond.Value:
			if firstColumn == 0 && firstRow < 2 && grid[location+2] == itemId && grid[location+3] == itemId && grid[location+5] == itemId {
				return Result{getArmorForType(itemId, "boots"), 0, 1}

			}
		}
	case 5:
		switch itemId {
		case constants.Stick.Value:
			if grid[2] == itemId && grid[4] == itemId && grid[6] == itemId && grid[5] == constants.String.Value && grid[8] == constants.String.Value {
				return Result{constants.FishingRod.Value, 0, 1}
			}

		case constants.Planks.Value:
			if same && firstColumn == 0 && grid[location+3] == itemId && grid[location+2] == itemId && grid[location+4] == itemId && grid[location+5] == itemId {
				return Result{constants.Boat.Value, 0, 1}
			}
		case constants.Iron.Value, constants.Gold.Value, constants.Diamond.Value:
			if same && itemId == constants.Iron.Value && firstColumn == 0 && grid[location+3] == itemId && grid[location+2] == itemId && grid[location+4] == itemId && grid[location+5] == itemId {
				return Result{constants.Minecart.Value, 0, 1}
			}
			if itemCount == 4 && grid[4] == constants.Redstone.Value {
				if grid[3] == itemId && grid[7] == itemId && grid[1] == itemId && grid[5] == itemId {
					if itemId == constants.Iron.Value {
						return Result{constants.Compass.Value, 0, 1}
					}
					if itemId == constants.Gold.Value {
						return Result{constants.Clock.Value, 0, 1}
					}
				}
			}

			if location == 0 &&
				grid[location+1] == itemId &&
				grid[location+2] == itemId &&
				grid[location+4] == constants.Stick.Value &&
				grid[location+7] == constants.Stick.Value {
				return Result{getToolForType(itemId, "pickaxe"), 0, 1}
			}
			if location < 2 &&
				grid[location+1] == itemId &&
				(grid[location+3] == itemId &&
					grid[location+4] == constants.Stick.Value &&
					grid[location+7] == constants.Stick.Value) ||
				(grid[location+4] == itemId &&
					grid[location+3] == constants.Stick.Value &&
					grid[location+6] == constants.Stick.Value) {
				return Result{getToolForType(itemId, "axe"), 0, 1}
			}

			if firstColumn == 0 && firstRow < 2 &&
				grid[location+1] == itemId &&
				grid[location+2] == itemId &&
				grid[location+3] == itemId &&
				grid[location+5] == itemId {
				return Result{getArmorForType(itemId, "helmet"), 0, 1}
			}
		case constants.Leather.Value:
			if firstColumn == 0 && firstRow < 2 &&
				grid[location+1] == itemId &&
				grid[location+2] == itemId &&
				grid[location+3] == itemId &&
				grid[location+5] == itemId {
				return Result{getArmorForType(itemId, "helmet"), 0, 1}
			}
		}
	case 6:
		switch itemId {
		case constants.Planks.Value, constants.Cobblestone.Value, constants.Stick.Value, constants.Iron.Value:
			if same && grid[1] == -1 && grid[2] == -1 && grid[5] == -1 && (itemId == constants.Planks.Value || itemId == constants.Cobblestone.Value) {
				return Result{getStairsForType(itemId), 0, 4}
			}
			if same && firstColumn == 0 && grid[location+1] == itemId && grid[location+2] == itemId && grid[location+4] == itemId && grid[location+5] == itemId {
				if itemId == constants.Planks.Value {
					return Result{constants.Trapdoor.Value, 0, 2}
				}
				if itemId == constants.Stick.Value {
					return Result{constants.Fence.Value, 0, 2}
				}
			}
			if same && firstColumn != 2 && grid[location+1] == itemId && grid[location+3] == itemId && grid[location+4] == itemId && grid[location+6] == itemId && grid[location+7] == itemId {
				if itemId == constants.Planks.Value {
					return Result{constants.WoodenDoor.Value, 0, 1}
				}
				if itemId == constants.Iron.Value {
					return Result{constants.IronDoor.Value, 0, 1}
				}
			}
			if grid[1] == constants.Stick.Value && grid[3] == constants.Stick.Value && grid[7] == constants.Stick.Value {
				if grid[2] == constants.String.Value && grid[5] == constants.String.Value && grid[8] == constants.String.Value {
					return Result{constants.Bow.Value, 0, 1}
				}
			}
		case constants.Wool.Value:
			if firstColumn == 0 && grid[location+1] == itemId && grid[location+2] == itemId && grid[location+3] == constants.Planks.Value && grid[location+4] == itemId && grid[location+5] == itemId {
				return Result{constants.Bed.Value, 0, 1}
			}
		case constants.RedstoneTorchOn.Value:
			if firstColumn == 0 && grid[location+2] == itemId && grid[location+1] == constants.Redstone.Value {
				if grid[location+3] == constants.Stone.Value && grid[location+4] == constants.Stone.Value && grid[location+5] == constants.Stone.Value {
					return Result{constants.RedstoneRepeaterOff.Value, 0, 1}
				}
			}
		}
	case 7:
		if same && grid[4] == -1 && grid[7] == -1 {
			switch itemId {
			case constants.Leather.Value, constants.Iron.Value, constants.Gold.Value, constants.Diamond.Value:
				return Result{getArmorForType(itemId, "leggings"), 0, 1}
			}
		}

		if same && grid[1] == -1 && grid[7] == -1 && itemId == constants.Stick.Value {
			return Result{constants.Ladder.Value, 0, 1}
		}

		if itemCount == 6 && itemId == constants.Planks.Value && grid[6] == -1 && grid[8] == -1 && grid[7] == constants.Stick.Value {
			return Result{constants.Sign.Value, 0, 1}
		}
		if itemCount == 6 && grid[1] == -1 && grid[4] == constants.Stick.Value {
			if itemId == constants.Iron.Value {
				if grid[7] == -1 {
					return Result{constants.Rail.Value, 0, 16}
				}
				if grid[7] == constants.Redstone.Value {
					return Result{constants.DetectorRail.Value, 0, 6}
				}
			}
			if itemId == constants.Gold.Value && grid[7] == constants.Redstone.Value {
				return Result{constants.PoweredRail.Value, 0, 6}
			}
		}

	case 8:
		if same {
			if grid[4] == -1 {
				switch itemId {
				case constants.Planks.Value:
					return Result{constants.Chest.Value, 0, 1}
				case constants.Cobblestone.Value:
					return Result{constants.Furnace.Value, 0, 1}
				}
			}
			if grid[1] == -1 {
				switch itemId {
				case constants.Leather.Value, constants.Iron.Value, constants.Gold.Value, constants.Diamond.Value:
					return Result{getArmorForType(itemId, "chestplate"), 0, 1}
				}
			}
		}
	case 9:

		if itemCount == 8 && itemId == constants.Stick.Value && grid[4] == constants.Wool.Value {
			return Result{constants.Painting.Value, 0, 1}
		}

		if itemCount == 8 && itemId == constants.Gold.Value && grid[4] == constants.Apple.Value {
			return Result{constants.GoldenApple.Value, 0, 1}
		}

		if itemCount == 8 && itemId == constants.Paper.Value && grid[4] == constants.Compass.Value {
			return Result{constants.Map.Value, 0, 1}
		}

		if itemCount == 3 && itemId == constants.Planks.Value {
			if grid[4] == constants.Iron.Value && grid[7] == constants.Redstone.Value {
				if grid[3] == constants.Cobblestone.Value && grid[5] == constants.Cobblestone.Value && grid[6] == constants.Cobblestone.Value && grid[8] == constants.Cobblestone.Value {
					return Result{constants.Piston.Value, 0, 1}
				}
			}
		}

		if itemCount == 7 && itemId == constants.Cobblestone.Value {
			if grid[4] == constants.Bow.Value && grid[7] == constants.Redstone.Value {
				return Result{constants.Dispenser.Value, 0, 1}
			}
		}

		if same {
			switch itemId {
			case constants.Iron.Value:
				return Result{constants.IronBlock.Value, 0, 1}
			case constants.Gold.Value:
				return Result{constants.GoldBlock.Value, 0, 1}
			case constants.Diamond.Value:
				return Result{constants.DiamondBlock.Value, 0, 1}
			case constants.Redstone.Value:
				return Result{constants.RedstoneBlock.Value, 0, 1}
			}
		}

		if itemCount == 8 && itemId == constants.Planks.Value && grid[4] == constants.Diamond.Value {
			return Result{constants.Jukebox.Value, 0, 1}
		}

		if itemCount == 8 && itemId == constants.Planks.Value && grid[4] == constants.Redstone.Value {
			return Result{constants.Noteblock.Value, 0, 1}
		}
		if itemCount == 6 && itemId == constants.Planks.Value && grid[3] == constants.Book.Value && grid[4] == constants.Book.Value && grid[5] == constants.Book.Value {
			return Result{constants.Bookshelf.Value, 0, 1}
		}
		if itemCount == 5 && itemId == constants.Gunpowder.Value && grid[1] == constants.Sand.Value && grid[3] == constants.Sand.Value && grid[5] == constants.Sand.Value && grid[7] == constants.Sand.Value {
			return Result{constants.TNT.Value, 0, 1}
		}

	}
	return Result{-1, 0, 0}
}
