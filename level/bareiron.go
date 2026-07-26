package level

// IMPORTANT: This is AI generated code, based on: https://github.com/p2r3/bareiron/blob/main/src/worldgen.c

import "encoding/binary"

// ---- Tunable generator constants (see header comment above) ----------------
const (
	biIslandsMiniSize    = 8  // ASSUMED: minichunk size (must divide 16)
	biIslandsBiomeSize   = 32 // ASSUMED: chunks-per-biome-cell (in minichunks)
	biIslandsBiomeRadius = 12 // ASSUMED: island radius, in minichunks
	biIslandsBaseHeight  = 64 // ASSUMED: TERRAIN_BASE_HEIGHT
	biIslandsCaveDepth   = 32 // ASSUMED: CAVE_BASE_DEPTH
)

// ---- Biomes ------------------------------------------------------------
type betaIslandBiome uint8

const (
	biPlains betaIslandBiome = iota
	biDesert
	biSwamp // stand-in for the original "mangrove swamp"
	biSnowyPlains
	biBeach // not seed-selected; the ring around/outside every island
)

// ---- Beta 1.7.3 block IDs (only the ones this generator needs) --------------
const (
	idAir        = 0
	idStone      = 1
	idGrass      = 2
	idDirt       = 3
	idCobble     = 4
	idBedrock    = 7
	idWater      = 9  // stationary water
	idLava       = 11 // stationary lava
	idSand       = 12
	idGoldOre    = 14
	idIronOre    = 15
	idCoalOre    = 16
	idLog        = 17
	idLeaves     = 18
	idLapisOre   = 21 // stand-in for B_copper_ore (Beta has no copper)
	idSandstone  = 24
	idTallGrass  = 31 // metadata 1 = "grass" style
	idDeadBush   = 32
	idCactus     = 81
	idDiamondOre = 56
	idRedstoneOr = 73
	idSnowLayer  = 78
	idIce        = 79
	idLilyPad    = 111
)

// splitmix64Beta mirrors the C helper `splitmix64` used by getChunkHash.
func splitmix64Beta(x uint64) uint64 {
	x += 0x9E3779B97F4A7C15
	z := x
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	z = z ^ (z >> 31)
	return z
}

// divFloorBeta / modAbsBeta mirror the C helpers div_floor / mod_abs:
// floor division, and a mod that's always non-negative.
func divFloorBeta(a, b int) int {
	q := a / b
	if a%b != 0 && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

func modAbsBeta(a, b int) int {
	m := a % b
	if m < 0 {
		m += b
	}
	return m
}

// betaChunkHash mirrors C's getChunkHash(short x, short z): packs the
// minichunk coordinates and world seed into 8 bytes (matching the original
// memcpy layout) and runs them through splitmix64.
func betaChunkHash(mx, mz int32, seed uint32) uint32 {
	var buf [8]byte
	binary.LittleEndian.PutUint16(buf[0:2], uint16(int16(mx)))
	binary.LittleEndian.PutUint16(buf[2:4], uint16(int16(mz)))
	binary.LittleEndian.PutUint32(buf[4:8], seed)
	v := binary.LittleEndian.Uint64(buf[:])
	return uint32(splitmix64Beta(v))
}

// betaChunkBiome mirrors C's getChunkBiome: circular islands laid out on a
// repeating grid, with the 4 land biomes read 2 bits at a time from the seed.
func betaChunkBiome(mx, mz int32, seed uint32) betaIslandBiome {
	x := int(mx) + biIslandsBiomeRadius
	z := int(mz) + biIslandsBiomeRadius

	dx := biIslandsBiomeRadius - modAbsBeta(x, biIslandsBiomeSize)
	dz := biIslandsBiomeRadius - modAbsBeta(z, biIslandsBiomeSize)

	if dx*dx+dz*dz > biIslandsBiomeRadius*biIslandsBiomeRadius {
		return biBeach
	}

	biomeX := int32(divFloorBeta(x, biIslandsBiomeSize))
	biomeZ := int32(divFloorBeta(z, biIslandsBiomeSize))

	h := betaIslandBiomeHash(biomeX, biomeZ, seed)

	return betaIslandBiome(h % 4)
}

func betaIslandBiomeHash(biomeX, biomeZ int32, seed uint32) uint32 {
	var buf [12]byte
	binary.LittleEndian.PutUint32(buf[0:4], uint32(biomeX))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(biomeZ))
	binary.LittleEndian.PutUint32(buf[8:12], seed)
	v := uint64(uint32(biomeX))<<32 | uint64(uint32(biomeZ))
	v ^= uint64(seed) * 0x9E3779B97F4A7C15
	return uint32(splitmix64Beta(v))
}

// func betaIslandBiomeHash(biomeX, biomeZ int32, seed uint32) uint32 {
// 	v := uint64(uint32(biomeX))<<32 | uint64(uint32(biomeZ))
// 	v ^= uint64(seed) * 0x9E3779B97F4A7C15
// 	return uint32(splitmix64Beta(v))
// }

// betaCornerHeight mirrors C's getCornerHeight. Deliberately uses byte
// (uint8) arithmetic throughout so it wraps exactly like the original
// uint8_t math (including the mangrove/swamp underflow-toward-water quirk).
func betaCornerHeight(hash uint32, biome betaIslandBiome) byte {
	height := byte(biIslandsBaseHeight)

	switch biome {
	case biSwamp:
		height += byte((hash % 3) + ((hash >> 4) % 3) + ((hash >> 8) % 3) + ((hash >> 12) % 3))
		if height < 64 {
			height -= byte((hash >> 24) & 3)
		}
	case biPlains:
		height += byte((hash & 3) + (hash >> 4 & 3) + (hash >> 8 & 3) + (hash >> 12 & 3))
	case biDesert:
		height += 4 + byte((hash&3)+(hash>>4&3))
	case biBeach:
		height = 62 - byte((hash&3)+(hash>>4&3)+(hash>>8&3))
	case biSnowyPlains:
		height += byte((hash & 7) + (hash >> 4 & 7))
	}

	return height
}

// betaInterpolate mirrors C's interpolate(): bilinear interpolation across a
// minichunk given the 4 corner heights and a local position in [0, miniSize).
func betaInterpolate(a, b, c, d byte, x, z int) byte {
	size := biIslandsMiniSize
	top := uint16(a)*uint16(size-x) + uint16(b)*uint16(x)
	bottom := uint16(c)*uint16(size-x) + uint16(d)*uint16(x)
	return byte((top*uint16(size-z) + bottom*uint16(z)) / uint16(size*size))
}

// betaAnchor mirrors C's ChunkAnchor: a precomputed minichunk hash + biome.
type betaAnchor struct {
	x, z  int32
	hash  uint32
	biome betaIslandBiome
}

func makeBetaAnchor(mx, mz int32, seed uint32) betaAnchor {
	return betaAnchor{
		x:     mx,
		z:     mz,
		hash:  betaChunkHash(mx, mz, seed),
		biome: betaChunkBiome(mx, mz, seed),
	}
}

// betaHeightFromAnchors mirrors C's getHeightAtFromAnchors: given a pointer
// (here: index + row stride) into a grid of anchors, interpolate height at a
// local (rx, rz) position within that minichunk.
func betaHeightFromAnchors(rx, rz int, idx, stride int, anchors []betaAnchor) byte {
	if rx == 0 && rz == 0 {
		h := int(betaCornerHeight(anchors[idx].hash, anchors[idx].biome))
		if h > 67 {
			return byte(h - 1)
		}
	}
	return betaInterpolate(
		betaCornerHeight(anchors[idx].hash, anchors[idx].biome),
		betaCornerHeight(anchors[idx+1].hash, anchors[idx+1].biome),
		betaCornerHeight(anchors[idx+stride].hash, anchors[idx+stride].biome),
		betaCornerHeight(anchors[idx+stride+1].hash, anchors[idx+stride+1].biome),
		rx, rz,
	)
}

// betaHeightFromHash mirrors C's getHeightAtFromHash: same as above, but
// computes the 3 neighboring corners on demand instead of from a grid.
func betaHeightFromHash(rx, rz int, mx, mz int32, hash uint32, biome betaIslandBiome, seed uint32) byte {
	if rx == 0 && rz == 0 {
		h := int(betaCornerHeight(hash, biome))
		if h > 67 {
			return byte(h - 1)
		}
	}
	a1 := makeBetaAnchor(mx+1, mz, seed)
	a2 := makeBetaAnchor(mx, mz+1, seed)
	a3 := makeBetaAnchor(mx+1, mz+1, seed)
	return betaInterpolate(
		betaCornerHeight(hash, biome),
		betaCornerHeight(a1.hash, a1.biome),
		betaCornerHeight(a2.hash, a2.biome),
		betaCornerHeight(a3.hash, a3.biome),
		rx, rz,
	)
}

// betaHeightAt mirrors C's getHeightAt: terrain height at a world (x, z).
func betaHeightAt(x, z int, seed uint32) byte {
	mx := divFloorBeta(x, biIslandsMiniSize)
	mz := divFloorBeta(z, biIslandsMiniSize)
	rx := modAbsBeta(x, biIslandsMiniSize)
	rz := modAbsBeta(z, biIslandsMiniSize)
	hash := betaChunkHash(int32(mx), int32(mz), seed)
	biome := betaChunkBiome(int32(mx), int32(mz), seed)
	return betaHeightFromHash(rx, rz, int32(mx), int32(mz), hash, biome, seed)
}

// betaFeature mirrors C's ChunkFeature: one candidate tree/cactus/etc. slot
// per minichunk. Y == 255 means "no feature here".
type betaFeature struct {
	x, z    int
	y       int // 255 (well, we use -1 here) means "skipped"
	variant byte
}

const betaFeatureSkip = -1

// betaFeatureFromAnchor mirrors C's getFeatureFromAnchor.
func betaFeatureFromAnchor(anchor betaAnchor, seed uint32) betaFeature {
	size := biIslandsMiniSize
	pos := int(anchor.hash) % (size * size)
	fx := pos % size
	fz := pos / size

	skip := false
	if anchor.biome != biSwamp {
		if fx < 3 || fx > size-3 {
			skip = true
		} else if fz < 3 || fz > size-3 {
			skip = true
		}
	}

	var f betaFeature
	if skip {
		f.y = betaFeatureSkip
		return f
	}

	f.x = fx + int(anchor.x)*size
	f.z = fz + int(anchor.z)*size
	h := betaHeightFromHash(modAbsBeta(f.x, size), modAbsBeta(f.z, size), anchor.x, anchor.z, anchor.hash, anchor.biome, seed)
	f.y = int(h) + 1
	f.variant = byte((anchor.hash >> uint((f.x+f.z)%32)) & 1)
	return f
}

// betaTerrainBlock mirrors C's getTerrainAtFromCache. Returns the block id
// and metadata for a single world-space block, given precomputed per-column
// context (anchor, feature, terrain height).
func betaTerrainBlock(x, y, z int, rx, rz int, anchor betaAnchor, feature betaFeature, height byte) (id byte, meta byte) {

	if y >= 64 && y >= int(height) && feature.y != betaFeatureSkip {
		switch anchor.biome {

		case biPlains: // trees
			if feature.y >= 64 {
				if x == feature.x && z == feature.z {
					if y == feature.y-1 {
						return idDirt, 0
					}
					if y >= feature.y && y < feature.y-int(feature.variant)+6 {
						return idLog, 0
					}
				}

				dx := absIntBeta(x - feature.x)
				dz := absIntBeta(z - feature.z)

				if dx < 3 && dz < 3 && y > feature.y-int(feature.variant)+2 && y < feature.y-int(feature.variant)+5 {
					if !(y == feature.y-int(feature.variant)+4 && dx == 2 && dz == 2) {
						return idLeaves, 0
					}
				}
				if dx < 2 && dz < 2 && y >= feature.y-int(feature.variant)+5 && y <= feature.y-int(feature.variant)+6 {
					if !(y == feature.y-int(feature.variant)+6 && dx == 1 && dz == 1) {
						return idLeaves, 0
					}
				}
			}
			if y == int(height) {
				return idGrass, 0
			}
			return idAir, 0

		case biDesert: // dead bushes / cacti
			if x == feature.x && z == feature.z {
				if feature.variant == 0 {
					if y == int(height)+1 {
						return idDeadBush, 0
					}
				} else if y > int(height) {
					if int(height)&1 == 1 && y <= int(height)+3 {
						return idCactus, 0
					}
					if y <= int(height)+2 {
						return idCactus, 0
					}
				}
			}

		case biSwamp: // lily pads + ground cover (stand-in for moss carpet)
			if x == feature.x && z == feature.z && y == 64 && int(height) < 63 {
				return idLilyPad, 0
			}
			if y == int(height)+1 {
				dx := absIntBeta(x - feature.x)
				dz := absIntBeta(z - feature.z)
				if dx+dz < 4 {
					return idTallGrass, 1
				}
			}

		case biSnowyPlains: // grass tufts
			if x == feature.x && z == feature.z && y == int(height)+1 && int(height) >= 64 {
				return idTallGrass, 1
			}
		}
	}

	// Surface-level terrain (topmost blocks)
	if height >= 63 {
		if y == int(height) {
			switch anchor.biome {
			case biSwamp:
				return idDirt, 0 // stand-in for mud
			case biSnowyPlains:
				return idGrass, 0 // snowy-grass-block stand-in; snow layer added below
			case biDesert, biBeach:
				return idSand, 0
			default:
				return idGrass, 0
			}
		}
		if anchor.biome == biSnowyPlains && y == int(height)+1 {
			return idSnowLayer, 0
		}
	}

	// Caves + ores, starting 4 blocks below the surface
	if y <= int(height)-4 {
		gap := int(height) - biIslandsBaseHeight
		if y < biIslandsCaveDepth+gap && y > biIslandsCaveDepth-gap {
			return idAir, 0
		}

		oreY := byte(((rx & 15) << 4) + (rz & 15))
		oreY ^= oreY << 4
		oreY ^= oreY >> 5
		oreY ^= oreY << 1
		oreY &= 63

		if y == int(oreY) {
			oreProbability := byte((anchor.hash >> uint(int(oreY)%24)) & 255)
			if y < 15 {
				if oreProbability < 10 {
					return idDiamondOre, 0
				}
				if oreProbability < 12 {
					return idGoldOre, 0
				}
				if oreProbability < 15 {
					return idRedstoneOr, 0
				}
			}
			if y < 30 {
				if oreProbability < 3 {
					return idGoldOre, 0
				}
				if oreProbability < 8 {
					return idRedstoneOr, 0
				}
			}
			if y < 54 {
				if oreProbability < 30 {
					return idIronOre, 0
				}
				if oreProbability < 40 {
					return idLapisOre, 0 // stand-in for copper ore
				}
			}
			if oreProbability < 60 {
				return idCoalOre, 0
			}
			if y < 5 {
				return idLava, 0
			}
			return idCobble, 0
		}

		return idStone, 0
	}

	// Between stone and grass
	if y <= int(height) {
		switch anchor.biome {
		case biDesert:
			return idSandstone, 0
		case biSwamp:
			return idDirt, 0 // stand-in for mud
		case biBeach:
			if height > 64 {
				return idSandstone, 0
			}
		}
		return idDirt, 0
	}

	// Below sea level and nothing else claimed this block
	if y == 63 && anchor.biome == biSnowyPlains {
		return idIce, 0
	}
	if y < 64 {
		return idWater, 0
	}

	return idAir, 0
}

func absIntBeta(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (c *Chunk) GenerateIslandBiomes(seed uint32, cx, cz int32) {
	blocksAmount := CHUNK_SIZE_X * CHUNK_SIZE_Y * CHUNK_SIZE_Z
	nibbleCount := blocksAmount / 2

	blockTypes := make([]byte, blocksAmount)
	blockMetadata := make([]byte, nibbleCount)
	blockLight := make([]byte, nibbleCount)
	blockSkyLight := make([]byte, nibbleCount)

	worldX := int(cx) * CHUNK_SIZE_X
	worldZ := int(cz) * CHUNK_SIZE_Z

	miniPerChunk := CHUNK_SIZE_X / biIslandsMiniSize // minichunks across one axis of this chunk
	stride := miniPerChunk + 1                       // anchor grid width (needs +1 for the far edge)

	baseMx := int32(divFloorBeta(worldX, biIslandsMiniSize))
	baseMz := int32(divFloorBeta(worldZ, biIslandsMiniSize))

	// Precompute anchors (hash+biome) for every minichunk corner touching
	// this chunk, plus one extra row/col so every minichunk has all 4 corners.
	anchors := make([]betaAnchor, stride*stride)
	for gz := 0; gz < stride; gz++ {
		for gx := 0; gx < stride; gx++ {
			anchors[gz*stride+gx] = makeBetaAnchor(baseMx+int32(gx), baseMz+int32(gz), seed)
		}
	}

	// Precompute one feature (tree/cactus/etc. candidate) per minichunk.
	features := make([]betaFeature, miniPerChunk*miniPerChunk)
	for gz := 0; gz < miniPerChunk; gz++ {
		for gx := 0; gx < miniPerChunk; gx++ {
			features[gz*miniPerChunk+gx] = betaFeatureFromAnchor(anchors[gz*stride+gx], seed)
		}
	}

	// Precompute interpolated terrain height for every column in the chunk.
	heights := make([][]byte, CHUNK_SIZE_X)
	for lx := 0; lx < CHUNK_SIZE_X; lx++ {
		heights[lx] = make([]byte, CHUNK_SIZE_Z)
		for lz := 0; lz < CHUNK_SIZE_Z; lz++ {
			gx := lx / biIslandsMiniSize
			gz := lz / biIslandsMiniSize
			idx := gz*stride + gx
			rx := lx % biIslandsMiniSize
			rz := lz % biIslandsMiniSize
			heights[lx][lz] = betaHeightFromAnchors(rx, rz, idx, stride, anchors)
		}
	}

	// Fill blocks.
	for lx := 0; lx < CHUNK_SIZE_X; lx++ {
		for lz := 0; lz < CHUNK_SIZE_Z; lz++ {
			gx := lx / biIslandsMiniSize
			gz := lz / biIslandsMiniSize
			anchor := anchors[gz*stride+gx]
			feature := features[gz*miniPerChunk+gx]
			height := heights[lx][lz]
			rx := lx % biIslandsMiniSize
			rz := lz % biIslandsMiniSize

			for ly := 0; ly < CHUNK_SIZE_Y; ly++ {
				id, meta := betaTerrainBlock(worldX+lx, ly, worldZ+lz, rx, rz, anchor, feature, height)

				i := lx*CHUNK_SIZE_Z*CHUNK_SIZE_Y + lz*CHUNK_SIZE_Y + ly
				blockTypes[i] = id

				ni := i / 2
				if i%2 == 0 {
					blockMetadata[ni] = meta & 0x0f
				} else {
					blockMetadata[ni] |= (meta & 0x0f) << 4
				}
			}
		}
	}

	c.Data = blockTypes
	c.Data = append(c.Data, blockMetadata...)
	c.Data = append(c.Data, blockLight...)
	c.Data = append(c.Data, blockSkyLight...)
	c.relightAll()
}
