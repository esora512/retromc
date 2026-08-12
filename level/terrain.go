package level

// IMPORTANT: This is AI-generated code; should be used as working placeholder  unitl world gen is better understood

const (
	MAZE_CELL_SIZE = 2 // 1 path block + 1 wall block per cell
	MAZE_CELLS_X   = CHUNK_SIZE_X / MAZE_CELL_SIZE
	MAZE_CELLS_Z   = CHUNK_SIZE_Z / MAZE_CELL_SIZE
	MAZE_FLOOR_Y   = GROUND_LEVEL // reuse the existing "ground level" as the walking surface
	MAZE_HEIGHT    = 4            // corridor/wall height in blocks above the floor
)

const (
	wallNorth byte = 1 << iota // -Z
	wallSouth                  // +Z
	wallEast                   // +X
	wallWest                   // -X
	allWalls  = wallNorth | wallSouth | wallEast | wallWest
)

type mazeCell struct {
	walls byte
}

var mazeDirs = []struct {
	dx, dz          int
	wallBit, oppBit byte
}{
	{0, -1, wallNorth, wallSouth},
	{0, 1, wallSouth, wallNorth},
	{1, 0, wallEast, wallWest},
	{-1, 0, wallWest, wallEast},
}

func generateMazeGrid(seed int64, cx, cz int32) [][]mazeCell {
	r := chunkRand(seed, cx, cz)

	grid := make([][]mazeCell, MAZE_CELLS_X)
	visited := make([][]bool, MAZE_CELLS_X)
	for x := range grid {
		grid[x] = make([]mazeCell, MAZE_CELLS_Z)
		visited[x] = make([]bool, MAZE_CELLS_Z)
		for z := range grid[x] {
			grid[x][z].walls = allWalls
		}
	}

	type pos struct{ x, z int }
	stack := []pos{{0, 0}}
	visited[0][0] = true

	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		carved := false

		for _, di := range r.Perm(4) {
			d := mazeDirs[di]
			nx, nz := cur.x+d.dx, cur.z+d.dz
			if nx < 0 || nz < 0 || nx >= MAZE_CELLS_X || nz >= MAZE_CELLS_Z || visited[nx][nz] {
				continue
			}
			grid[cur.x][cur.z].walls &^= d.wallBit
			grid[nx][nz].walls &^= d.oppBit
			visited[nx][nz] = true
			stack = append(stack, pos{nx, nz})
			carved = true
			break
		}

		if !carved {
			stack = stack[:len(stack)-1] // backtrack
		}
	}

	return grid
}

func doorIndex(seed int64, cx, cz int32, axis int64, count int) int {
	r := chunkRand(seed+axis*1_000_003, cx, cz)
	return r.Intn(count)
}

func applyMazeBorders(seed int64, cx, cz int32, grid [][]mazeCell) {
	doorZ := doorIndex(seed, cx, cz, 0, MAZE_CELLS_Z)
	grid[MAZE_CELLS_X-1][doorZ].walls &^= wallEast

	doorX := doorIndex(seed, cx, cz, 1, MAZE_CELLS_X)
	grid[doorX][MAZE_CELLS_Z-1].walls &^= wallSouth
}

func (c *Chunk) GenerateMaze(seed int64, cx, cz int32) {
	blocksAmount := CHUNK_SIZE_X * CHUNK_SIZE_Y * CHUNK_SIZE_Z
	nibbleCount := blocksAmount / 2

	blockTypes := make([]byte, blocksAmount)
	blockMetadata := make([]byte, nibbleCount)
	blockLight := make([]byte, nibbleCount)
	blockSkyLight := make([]byte, nibbleCount)

	grid := generateMazeGrid(seed, cx, cz)
	applyMazeBorders(seed, cx, cz, grid)

	// open[x][z] marks which columns are corridor (true) vs solid wall/post (false).
	open := make([][]bool, CHUNK_SIZE_X)
	for x := range open {
		open[x] = make([]bool, CHUNK_SIZE_Z)
	}
	for cxi := 0; cxi < MAZE_CELLS_X; cxi++ {
		for czi := 0; czi < MAZE_CELLS_Z; czi++ {
			cell := grid[cxi][czi]
			bx, bz := cxi*MAZE_CELL_SIZE, czi*MAZE_CELL_SIZE

			open[bx][bz] = true // the cell's own floor tile is always open
			if cell.walls&wallEast == 0 {
				open[bx+1][bz] = true
			}
			if cell.walls&wallSouth == 0 {
				open[bx][bz+1] = true
			}
			// (bx+1, bz+1) is a wall post shared by up to 4 cells - stays solid
		}
	}

	setBlock := func(lx, ly, lz int, block Block) {
		i := lx*CHUNK_SIZE_Z*CHUNK_SIZE_Y + lz*CHUNK_SIZE_Y + ly
		blockTypes[i] = block.TypeId
		ni := i / 2
		if i%2 == 0 {
			blockMetadata[ni] = block.Metadata & 0x0f
			blockLight[ni] = block.Light & 0x0f
			blockSkyLight[ni] = block.SkyLight & 0x0f
		} else {
			blockMetadata[ni] |= (block.Metadata & 0x0f) << 4
			blockLight[ni] |= (block.Light & 0x0f) << 4
			blockSkyLight[ni] |= (block.SkyLight & 0x0f) << 4
		}
	}

	for lx := 0; lx < CHUNK_SIZE_X; lx++ {
		for lz := 0; lz < CHUNK_SIZE_Z; lz++ {
			for ly := 0; ly < MAZE_FLOOR_Y; ly++ {
				setBlock(lx, ly, lz, NewBedrockBlock())
			}

			corridor := open[lx][lz]
			for h := 0; h < MAZE_HEIGHT; h++ {
				ly := MAZE_FLOOR_Y + h
				var block Block
				if corridor {
					block = NewAirBlock()
					block.SkyLight = 0x0f
				} else {
					block = NewBedrockBlock()
				}
				setBlock(lx, ly, lz, block)
			}
		}
	}

	c.Data = blockTypes
	c.Data = append(c.Data, blockMetadata...)
	c.Data = append(c.Data, blockLight...)
	c.Data = append(c.Data, blockSkyLight...)
	c.relightAll()
}
