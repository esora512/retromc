package level

type SetBlock func(world *World, x, y, z int32, block *Block)

func InfiniteWaterSource(world *World, cfg FluidConfig, setBlock SetBlock) {
	for key := range cfg.Flowing {
		x, y, z := key.X, key.Y, key.Z
		sourceCount := 0

		for _, n := range []struct{ dx, dz int32 }{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, nz := x+n.dx, z+n.dz
			nKey := BlockKey{X: nx, Y: y, Z: nz}
			if _, isSource := cfg.Sources[nKey]; isSource {
				sourceCount++
			}
		}

		if sourceCount >= 2 {
			b := NewStillWaterBlock(0)
			setBlock(world, x, int32(y), z, &b)
		}
	}
}

func RefreshSourceBlocks(world *World, cfg FluidConfig, setBlock SetBlock) {
	for key := range cfg.Sources {
		b := NewStillWaterBlock(0)
		setBlock(world, key.X, int32(key.Y), key.Z, &b)
	}
}

func CheckLavaHarden(world *World, setBlock SetBlock) {
	neighbors := []struct{ dx, dy, dz int32 }{
		{0, 0, -1}, {0, 0, 1}, {-1, 0, 0}, {1, 0, 0}, {0, 1, 0},
	}

	touchesWater := func(x, y, z int32) bool {
		for _, n := range neighbors {
			b := world.GetBlock(x+n.dx, byte(y+n.dy), z+n.dz)
			if b.IsWater() {
				return true
			}
		}
		return false
	}

	chunks := world.LoadChunks()
	for _, chunk := range chunks {
		logic := chunk.Logic
		for key, height := range logic.FlowingLava {
			x, y, z := key.X, int32(key.Y), key.Z
			if !touchesWater(x, y, z) {
				continue
			}
			if height <= 4 {
				cobble := NewCobblestoneBlock()
				setBlock(world, x, y, z, &cobble)
			}
		}

		for key := range logic.LavaSources {
			x, y, z := key.X, int32(key.Y), key.Z
			// Make sure it wasn't already replaced this tick
			b := world.GetBlock(x, byte(y), z)
			if !b.IsLava() {
				delete(logic.LavaSources, key)
				continue
			}
			if !touchesWater(x, y, z) {
				continue
			}
			obsidian := NewObsidianBlock()
			setBlock(world, x, y, z, &obsidian)
		}
	}
}

type FluidConfig struct {
	IsFluid         func(b Block) bool
	NewBlock        func(liquidHeight byte) Block
	Sources         map[BlockKey]byte
	Flowing         map[BlockKey]byte
	MaxSpreadHeight byte
}

func NewWaterConfig(world *World) FluidConfig {
	loadedSources := make(map[BlockKey]byte)
	loadedFlowing := make(map[BlockKey]byte)
	chunks := world.LoadChunks()
	for _, chunk := range chunks {
		logic := chunk.Logic
		for key, height := range logic.WaterSources {
			loadedSources[key] = height
		}
		for key, height := range logic.FlowingWater {
			loadedFlowing[key] = height
		}
	}
	return FluidConfig{
		IsFluid: func(b Block) bool { return b.IsWater() },
		NewBlock: func(h byte) Block {
			block := NewFlowingWaterBlock(h)
			return block
		},
		Sources:         loadedSources,
		Flowing:         loadedFlowing,
		MaxSpreadHeight: 7,
	}
}

func NewLavaConfig(world *World) FluidConfig {
	loadedSources := make(map[BlockKey]byte)
	loadedFlowing := make(map[BlockKey]byte)
	chunks := world.LoadChunks()
	for _, chunk := range chunks {
		logic := chunk.Logic
		for key, height := range logic.LavaSources {
			loadedSources[key] = height
		}
		for key, height := range logic.FlowingLava {
			loadedFlowing[key] = height
		}
	}
	return FluidConfig{
		IsFluid: func(b Block) bool { return b.IsLava() },
		NewBlock: func(h byte) Block {
			block := NewFlowingLavaBlock(h)
			return block
		},
		Sources:         loadedSources,
		Flowing:         loadedFlowing,
		MaxSpreadHeight: 7,
	}
}

func setFlowingFluid(world *World, x, y, z int32, liquidHeight byte, cfg FluidConfig, setBlock SetBlock) {
	block := cfg.NewBlock(liquidHeight)
	setBlock(world, x, y, z, &block)
}

func FluidDecay(world *World, cfg FluidConfig, setBlock SetBlock) {
	visited := make(map[BlockKey]bool)
	queue := []BlockKey{}

	for key := range cfg.Sources {
		if _, isFlowing := cfg.Flowing[key]; !isFlowing {
			visited[key] = true
			queue = append(queue, key)
		}
	}

	spreadNeighbors := []struct{ dx, dy, dz int32 }{
		{1, 0, 0}, {-1, 0, 0}, {0, 0, 1}, {0, 0, -1}, {0, -1, 0},
	}
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		x, y, z := key.X, key.Y, key.Z

		for _, n := range spreadNeighbors {
			nx := x + n.dx
			ny := int32(y) + n.dy
			nz := z + n.dz
			if ny < 0 || ny > 255 {
				continue
			}
			nKey := BlockKey{X: nx, Y: byte(ny), Z: nz}
			if visited[nKey] {
				continue
			}
			if cfg.IsFluid(world.GetBlock(nx, byte(ny), nz)) {
				visited[nKey] = true
				queue = append(queue, nKey)
			}
		}
	}

	toRemove := []BlockKey{}
	for key := range cfg.Flowing {
		if !visited[key] {
			toRemove = append(toRemove, key)
		}
	}
	for _, key := range toRemove {
		air := NewAirBlock()
		setBlock(world, key.X, int32(key.Y), key.Z, &air)
	}
}

func abs(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

func findHoleNearSource(world *World, sourceKey BlockKey, cfg FluidConfig) (BlockKey, bool) {
	const maxDist = 4
	x, y, z := sourceKey.X, sourceKey.Y, sourceKey.Z

	visited := make(map[BlockKey]bool)
	queue := []BlockKey{{X: x, Y: y, Z: z}}
	visited[sourceKey] = true

	// BFS to find the nearest reachable hole
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		b := world.GetBlock(cur.X, cur.Y-1, cur.Z)
		// a hole must be either air or flowing water
		// latter allows us to keep the same state even if the hole is filled with flowing water
		if cur.Y > 0 && (b.IsAir() || b.IsFlowing()) {
			return cur, true
		}

		if abs(cur.X-x) >= int32(maxDist) || abs(cur.Z-z) >= int32(maxDist) {
			continue
		}

		for _, n := range []struct{ dx, dz int32 }{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, nz := cur.X+n.dx, cur.Z+n.dz
			nKey := BlockKey{X: nx, Y: y, Z: nz}
			if visited[nKey] {
				continue
			}
			visited[nKey] = true
			if b := world.GetBlock(nx, y, nz); b.IsAir() || cfg.IsFluid(b) {
				queue = append(queue, nKey)
			}
		}
	}
	return BlockKey{}, false
}

func FluidSpreading(world *World, cfg FluidConfig, setBlock SetBlock) {
	type fluidEntry struct {
		key    BlockKey
		height byte
	}

	flowingSnapshot := make(map[BlockKey]byte, len(cfg.Flowing))
	for k, v := range cfg.Flowing {
		flowingSnapshot[k] = v
	}

	var allFluids []fluidEntry
	for key, h := range cfg.Sources {
		allFluids = append(allFluids, fluidEntry{key, h})
	}
	for key, h := range flowingSnapshot {
		allFluids = append(allFluids, fluidEntry{key, h})
	}

	for _, entry := range allFluids {
		x, y, z := entry.key.X, entry.key.Y, entry.key.Z
		height := entry.height

		b := world.GetBlock(x, y-1, z)
		if y > 0 && b.IsAir() {
			setFlowingFluid(world, x, int32(y)-1, z, 0, cfg, setBlock)
			continue
		}

		if height >= cfg.MaxSpreadHeight {
			continue
		}
		nextHeight := height + 1

		type offset struct{ dx, dz int32 }
		neighbors := []offset{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
		holeTarget, holeFound := findHoleNearSource(world, entry.key, cfg)
		if holeFound {
			var biased []offset
			for _, n := range neighbors {
				nx, nz := x+n.dx, z+n.dz
				curDist := abs(x-holeTarget.X) + abs(z-holeTarget.Z)
				newDist := abs(nx-holeTarget.X) + abs(nz-holeTarget.Z)
				if newDist < curDist {
					biased = append(biased, n)
				}
			}
			neighbors = biased
			if len(neighbors) == 0 {
				continue
			}
		}

		for _, n := range neighbors {
			nx, nz := x+n.dx, z+n.dz
			b := world.GetBlock(nx, y, nz)
			if !b.IsAir() && !cfg.IsFluid(b) {
				continue
			}
			nKey := BlockKey{X: nx, Y: y, Z: nz}
			if _, isSource := cfg.Sources[nKey]; isSource {
				continue
			}
			if existing, exists := flowingSnapshot[nKey]; exists && existing <= nextHeight {
				continue
			}
			setFlowingFluid(world, nx, int32(y), nz, nextHeight, cfg, setBlock)
		}
	}
}
