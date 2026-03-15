package crafting

import "github.com/leNicDev/retromc/constants"

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

func Craft(grid [9]int16) Result {
	itemId, same := sameItem(grid)
	if !same {
		return Result{-1, 0, 0}
	}

	if grid[4] == -1 && itemId == constants.Planks.Value {
		return Result{constants.Chest.Value, 0, 1}
	}

	return Result{-1, 0, 0}
}
