package crafting

import (
	"log"

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
			count++
		}
		if grid[i] != itemId {
			same = false
		}
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
	return -1
}

func Craft(grid [9]int16) Result {
	itemId, location, count, same := getGridData(grid)
	log.Printf("itemId %d, Location %d, Count %d, Same %v", itemId, location, count, same)
	if itemId == -1 {
		return Result{-1, 0, 0}
	}

	// Sticks OR Pressure Plate
	if count == 2 && same {
		if (location+3 < 9) && grid[location+3] == constants.Planks.Value {
			return Result{constants.Stick.Value, 0, 4}
		} else if (location+1 < 9) && grid[location+1] != -1 && (location+1)%3 != 0 {
			if itemId == constants.Planks.Value {
				return Result{constants.WoodenPressurePlate.Value, 0, 1}
			}
			if itemId == constants.Stone.Value {
				return Result{constants.StonePressurePlate.Value, 0, 1}
			}
		} else {
			return Result{-1, 0, 0}
		}
	}

	// Slabs
	if count == 3 && same {
		if (location+1 < 9) && (location+2 < 9) && ((location+1)%3 != 0) && ((location+2)%3 != 0) {
			if itemId == constants.Planks.Value {
				return Result{constants.WoodenSlab.Value, constants.WoodenSlab.Meta, 3}
			}
			if itemId == constants.Cobblestone.Value {
				return Result{constants.CobblestoneSlab.Value, constants.CobblestoneSlab.Meta, 3}
			}
			if itemId == constants.Stone.Value {
				return Result{constants.StoneSlab.Value, constants.StoneSlab.Meta, 3}
			}
		}
	}

	// Chest & Furnace
	if grid[4] == -1 && same && count == 8 {
		if itemId == constants.Planks.Value {
			return Result{constants.Chest.Value, 0, 1}
		}
		if itemId == constants.Cobblestone.Value {
			return Result{constants.Furnace.Value, 0, 1}
		}
	}

	// Stairs
	if grid[1] == -1 && grid[2] == -1 && grid[5] == -1 && same && count == 6 {
		if itemId == constants.Planks.Value {
			return Result{constants.WoodenStairs.Value, 0, 4}
		}
		if itemId == constants.Cobblestone.Value {
			return Result{constants.CobblestoneStairs.Value, 0, 4}
		}
	}

	// Tools
	if !same && grid[4] == constants.Stick.Value && grid[7] == constants.Stick.Value {
		if count == 1 && location == 1 {
			return Result{getToolForType(itemId, "shovel"), 0, 1}
		}
		if count == 2 && (location == 0 || location == 1) {
			if grid[location+1] == itemId && (location+1)%3 != 0 {
				return Result{getToolForType(itemId, "hoe"), 0, 1}
			}
		}
		if count == 3 {
			if location == 0 {
				// Pickaxe for sure
				if grid[1] == itemId && grid[2] == itemId {
					return Result{getToolForType(itemId, "pickaxe"), 0, 1}
					// Axe
				} else if grid[1] == itemId && grid[3] == itemId {
					return Result{getToolForType(itemId, "axe"), 0, 1}
				}
			}
			if location == 1 {
				// Axe for sure
				if grid[2] == itemId && grid[5] == itemId {
					return Result{getToolForType(itemId, "axe"), 0, 1}
				}
			}
		}
	}

	return Result{-1, 0, 0}
}
