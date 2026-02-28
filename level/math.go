package level

// Expected behavior: 1111 0000 + 1111 0000 = 1111 1111
func mergeNibbles(a byte, b byte) byte {
	return (a | (b >> 4))
}

// WorldToChunkCoord converts a world block coordinate to its chunk coordinate.
// Uses arithmetic right-shift so negative coords floor correctly (e.g. -1 → -1, not 0).
func WorldToChunkCoord(world int32) int32 {
	return world >> 4
}

// WorldToLocalCoord converts a world block coordinate to a 0-15 local coordinate within its chunk.
func WorldToLocalCoord(world int32) int {
	return int(world & 15)
}

func divEvenOrRoundUp(x int, y int) int {
	if x%y > 0 {
		return int(x/y) + 1
	} else {
		return x / y
	}
}
