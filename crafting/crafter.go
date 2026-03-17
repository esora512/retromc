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

func New3x3Crafter() *Crafter3x3 {
	return &Crafter3x3{
		Recipes: []Recipe3x3{
			// furnace: cobblestone ring with empty center
			{Pattern: [9]int16{4, 4, 4, 4, -1, 4, 4, 4, 4}, Result: Result{constants.Furnace.Value, 0, 1}},
			// chest: wood planks ring with empty center
			{Pattern: [9]int16{5, 5, 5, 5, -1, 5, 5, 5, 5}, Result: Result{54, 0, 1}},

			{Pattern: [9]int16{-1, -1, -1, -1, -1, -1, 5, 5, 5}, Result: Result{44, 2, 3}},
			{Pattern: [9]int16{-1, -1, -1, 5, 5, 5, -1, -1, -1}, Result: Result{44, 2, 3}},
			{Pattern: [9]int16{5, 5, 5, -1, -1, -1, -1, -1, -1}, Result: Result{44, 2, 3}},
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

func (c *Crafter3x3) Craft(grid [9]int16) Result {
	for _, r := range c.Recipes {
		if patternMatches9(r.Pattern, grid) {
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

func patternMatches9(pattern, grid [9]int16) bool {
	for i := range pattern {
		if pattern[i] != grid[i] {
			return false
		}
	}
	return true
}

type ShapedRecipe struct {
	Shape  [9]rune
	Key    map[rune]int16
	Result Result
}

func NewShapedRecipe(shape [9]rune, key map[rune]int16, result Result) ShapedRecipe {
	return ShapedRecipe{Shape: shape, Key: key, Result: result}
}

func (r ShapedRecipe) ToPattern() [9]int16 {
	var pattern [9]int16
	for i, char := range r.Shape {
		if char == ' ' {
			pattern[i] = -1
		} else {
			pattern[i] = r.Key[char]
		}
	}
	return pattern
}

func (r ShapedRecipe) ToRecipe3x3() Recipe3x3 {
	return Recipe3x3{r.ToPattern(), r.Result}
}

func AnyRow(item int16, result Result) []Recipe3x3 {
	return []Recipe3x3{
		{Pattern: [9]int16{item, item, item, -1, -1, -1, -1, -1, -1}, Result: result},
		{Pattern: [9]int16{-1, -1, -1, item, item, item, -1, -1, -1}, Result: result},
		{Pattern: [9]int16{-1, -1, -1, -1, -1, -1, item, item, item}, Result: result},
	}
}

func AnyColumn(item int16, result Result) []Recipe3x3 {
	return []Recipe3x3{
		{Pattern: [9]int16{item, -1, -1, item, -1, -1, item, -1, -1}, Result: result},
		{Pattern: [9]int16{-1, item, -1, -1, item, -1, -1, item, -1}, Result: result},
		{Pattern: [9]int16{-1, -1, item, -1, -1, item, -1, -1, item}, Result: result},
	}
}

func New3x3CrafterV2() *Crafter3x3 {
	recipes := []Recipe3x3{}

	recipes = append(recipes, NewShapedRecipe(
		[9]rune{
			'C', 'C', 'C',
			'C', ' ', 'C',
			'C', 'C', 'C',
		},
		map[rune]int16{'C': constants.Cobblestone.Value},
		Result{constants.Furnace.Value, 0, 1},
	).ToRecipe3x3())

	recipes = append(recipes, NewShapedRecipe(
		[9]rune{
			'P', 'P', 'P',
			'P', ' ', 'P',
			'P', 'P', 'P',
		},
		map[rune]int16{'P': constants.Planks.Value},
		Result{constants.Chest.Value, 0, 1},
	).ToRecipe3x3())

	recipes = append(recipes, AnyRow(constants.Planks.Value, Result{constants.WoodenSlab.Value, constants.WoodenSlab.Meta, 3})...)
	recipes = append(recipes, AnyRow(constants.Cobblestone.Value, Result{constants.CobblestoneSlab.Value, constants.CobblestoneSlab.Meta, 3})...)
	recipes = append(recipes, AnyRow(constants.Stone.Value, Result{constants.StoneSlab.Value, constants.StoneSlab.Meta, 3})...)

	recipes = append(recipes, NewShapedRecipe(
		[9]rune{
			'C', ' ', ' ',
			'C', 'C', ' ',
			'C', 'C', 'C',
		},
		map[rune]int16{'C': constants.Cobblestone.Value},
		Result{constants.CobblestoneStairs.Value, 0, 4},
	).ToRecipe3x3())

	recipes = append(recipes, NewShapedRecipe(
		[9]rune{
			'C', ' ', ' ',
			'C', 'C', ' ',
			'C', 'C', 'C',
		},
		map[rune]int16{'C': constants.Planks.Value},
		Result{constants.WoodenStairs.Value, 0, 4},
	).ToRecipe3x3())

	recipes = append(recipes, NewShapedRecipe(
		[9]rune{
			' ', 'C', 'C',
			' ', 'B', 'C',
			' ', 'B', ' ',
		},
		map[rune]int16{'C': constants.Cobblestone.Value, 'B': constants.Stick.Value},
		Result{constants.StoneAxe.Value, 0, 1},
	).ToRecipe3x3())

	recipes = append(recipes, NewShapedRecipe(
		[9]rune{
			' ', ' ', 'C',
			' ', ' ', 'C',
			' ', ' ', ' ',
		},
		map[rune]int16{'C': constants.Planks.Value},
		Result{constants.Stick.Value, 0, 4},
	).ToRecipe3x3())
	return &Crafter3x3{Recipes: recipes}
}

func sameItem(grid [9]int16) (int16, bool) {
	var itemId = int16(-1)
	for i := range grid {
		if grid[i] == -1 {
			continue
		}
		if itemId == -1 {
			itemId = grid[i]
		}
		if grid[i] != itemId {
			return -1, false
		}
	}
	return itemId, true
}

func firstItem(grid [9]int16) (int16, int16) {
	for i, item := range grid {
		if item != -1 {
			return item, int16(i)
		}
	}
	return -1, -1
}

func countItem(grid [9]int16) int16 {
	var count int16
	for _, item := range grid {
		if item != -1 {
			count++
		}
	}
	return count
}

func getGridData(grid [9]int16) (int16, int16, int16, bool) {
	var itemId = int16(-1)
	var location int16 = -1
	var count int16 = 0
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

		}
		if grid[i] != itemId {
			same = false
		}

		count++
	}
	return itemId, location, count, same
}

func getToolForType(itemId int16, toolName string) int16 {
	if toolName == "axe" && itemId == constants.Planks.Value {
		return constants.WoodenAxe.Value
	}
	if toolName == "axe" && itemId == constants.Cobblestone.Value {
		return constants.StoneAxe.Value
	}
	if toolName == "axe" && itemId == constants.Iron.Value {
		return constants.IronAxe.Value
	}
	if toolName == "axe" && itemId == constants.Gold.Value {
		return constants.GoldAxe.Value
	}
	if toolName == "axe" && itemId == constants.Diamond.Value {
		return constants.DiamondAxe.Value
	}
	if toolName == "pickaxe" && itemId == constants.Planks.Value {
		return constants.WoodenPickaxe.Value
	}
	if toolName == "pickaxe" && itemId == constants.Cobblestone.Value {
		return constants.StonePickaxe.Value
	}
	if toolName == "pickaxe" && itemId == constants.Iron.Value {
		return constants.IronPickaxe.Value
	}
	if toolName == "pickaxe" && itemId == constants.Gold.Value {
		return constants.GoldPickaxe.Value
	}
	if toolName == "pickaxe" && itemId == constants.Diamond.Value {
		return constants.DiamondPickaxe.Value
	}
	if toolName == "shovel" && itemId == constants.Planks.Value {
		return constants.WoodenShovel.Value
	}
	if toolName == "shovel" && itemId == constants.Cobblestone.Value {
		return constants.StoneShovel.Value
	}
	if toolName == "shovel" && itemId == constants.Iron.Value {
		return constants.IronShovel.Value
	}
	if toolName == "shovel" && itemId == constants.Gold.Value {
		return constants.GoldShovel.Value
	}
	if toolName == "shovel" && itemId == constants.Diamond.Value {
		return constants.DiamondShovel.Value
	}
	if toolName == "hoe" && itemId == constants.Planks.Value {
		return constants.WoodenHoe.Value
	}
	if toolName == "hoe" && itemId == constants.Cobblestone.Value {
		return constants.StoneHoe.Value
	}
	if toolName == "hoe" && itemId == constants.Iron.Value {
		return constants.IronHoe.Value
	}
	if toolName == "hoe" && itemId == constants.Gold.Value {
		return constants.GoldHoe.Value
	}
	if toolName == "hoe" && itemId == constants.Diamond.Value {
		return constants.DiamondHoe.Value
	}
	if toolName == "sword" && itemId == constants.Planks.Value {
		return constants.WoodenSword.Value
	}
	if toolName == "sword" && itemId == constants.Cobblestone.Value {
		return constants.StoneSword.Value
	}
	if toolName == "sword" && itemId == constants.Iron.Value {
		return constants.IronSword.Value
	}
	if toolName == "sword" && itemId == constants.Gold.Value {
		return constants.GoldSword.Value
	}
	if toolName == "sword" && itemId == constants.Diamond.Value {
		return constants.DiamondSword.Value
	}
	return -1
}

func getSlab(itemId int16) int16 {
	switch itemId {
	case constants.Planks.Value:
		return constants.WoodenSlab.Value
	case constants.Cobblestone.Value:
		return constants.CobblestoneSlab.Value
	case constants.Stone.Value:
		return constants.StoneSlab.Value
	case constants.Sandstone.Value:
		return constants.SandstoneSlab.Value
	default:
		return -1
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

// Based on bareiron by p2r3: https://github.com/p2r3/bareiron/blob/main/src/crafting.c
// more "fun" and perhaps more efficient
func Craft(grid [9]int16) Result {
	// first item id, location, total count and if all items are first item
	itemId, location, count, same := getGridData(grid)
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
		case constants.Redstone.Value:
			return Result{constants.RedstoneDust.Value, 0, 9}
		}

	case 2:
		switch itemId {
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
		case constants.Iron.Value:
			if ((firstRow != 2 && firstColumn != 2) && grid[location+4] == itemId) || ((firstRow != 2 && firstColumn != 0) && grid[location+2] == itemId) {
				return Result{constants.Shears.Value, 0, 1}
			}
		}
	case 3:
		switch itemId {
		case constants.Stone.Value, constants.Sandstone.Value:
			// slab
			if firstRow == 0 && grid[location+1] == itemId && grid[location+2] == itemId {
				return Result{getSlab(itemId), 0, 3}
			}

		case constants.Planks.Value, constants.Cobblestone.Value:
			// slab and tools
			if firstRow == 0 && grid[location+1] == itemId && grid[location+2] == itemId {
				return Result{getSlab(itemId), 0, 3}
			}
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

		case constants.Iron.Value, constants.Gold.Value, constants.Diamond.Value:
			// sword and shovel
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
		}
	case 4:
		switch itemId {
		case constants.Planks.Value, constants.Snowball.Value:
			if firstColumn != 2 && firstRow != 2 && same {
				if itemId == constants.Planks.Value {
					return Result{constants.CraftingTable.Value, 0, 1}
				}
				if itemId == constants.Snowball.Value {
					return Result{constants.SnowBlock.Value, 0, 1}
				}
			}
		case constants.Leather.Value, constants.Iron.Value, constants.Gold.Value, constants.Diamond.Value:
			if firstColumn == 0 && firstRow < 2 && grid[location+2] == itemId && grid[location+3] == itemId && grid[location+5] == itemId {
				return Result{getArmorForType(itemId, "boots"), 0, 1}

			}
		}
	}
	return Result{-1, 0, 0}
}
