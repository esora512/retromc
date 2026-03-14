package crafting

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
			{Pattern: [9]int16{4, 4, 4, 4, -1, 4, 4, 4, 4}, Result: Result{61, 0, 1}},
			// chest: wood planks ring with empty center
			{Pattern: [9]int16{5, 5, 5, 5, -1, 5, 5, 5, 5}, Result: Result{54, 0, 1}},

			{Pattern: [9]int16{-1, -1, -1, -1, -1, -1, 5, 5, 5}, Result: Result{44, 2, 3}},
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
