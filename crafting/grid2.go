package crafting

type Recipe struct {
	Pattern [4]int16
	Result  int16
}

type Crafter2x2 struct {
	Recipes []Recipe
}

var globalCrafter *Crafter2x2

func Init() {
	globalCrafter = NewCrafter()
}

func GetCrafter2x2() *Crafter2x2 {
	if globalCrafter == nil {
		globalCrafter = NewCrafter()
	}
	return globalCrafter
}

func NewCrafter() *Crafter2x2 {
	return &Crafter2x2{
		Recipes: []Recipe{
			// crafting table
			{Pattern: [4]int16{5, 5, 5, 5}, Result: 58},
		},
	}
}

func (c *Crafter2x2) Craft(grid [4]int16) int16 {
	for _, r := range c.Recipes {
		if patternMatches(r.Pattern, grid) {
			return r.Result
		}
	}
	return -1
}

func patternMatches(pattern, grid [4]int16) bool {
	for i := range pattern {
		if pattern[i] != -1 && pattern[i] != grid[i] {
			return false
		}
	}
	return true
}
