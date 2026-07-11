package mcregion

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

const sectorSize = 4096

// regionCoord converts chunk coords to region coords (floor division via
// arithmetic shift, which is correct for negative int32 in Go).
func regionCoord(c int32) int32 { return c >> 5 }

// localCoord gives the 0-31 position of a chunk within its region.
func localCoord(c int32) int32 { return c & 31 }

// WriteRegion writes one .mcr file containing the given chunks. chunks maps
// local-in-region (lx, lz), each 0-31, to that chunk's already-built root
// NBT compound (the "" compound whose only child is "Level").
func WriteRegion(path string, chunks map[[2]int32]*Compound) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	locations := make([]byte, sectorSize)
	timestamps := make([]byte, sectorSize)

	var body bytes.Buffer
	nextSector := int32(2) // sectors 0-1 are the header

	for pos, comp := range chunks {
		lx, lz := pos[0], pos[1]

		var compressed bytes.Buffer
		zw := zlib.NewWriter(&compressed)
		if _, err := zw.Write(comp.Root()); err != nil {
			return err
		}
		if err := zw.Close(); err != nil {
			return err
		}

		// 4-byte length (covers compression-type byte + payload) + 1-byte
		// compression type (2 = zlib), then payload, padded to sector size.
		payloadLen := compressed.Len() + 1
		var chunkBuf bytes.Buffer
		binary.Write(&chunkBuf, binary.BigEndian, int32(payloadLen))
		chunkBuf.WriteByte(2)
		chunkBuf.Write(compressed.Bytes())

		sectors := (chunkBuf.Len() + sectorSize - 1) / sectorSize
		padding := sectors*sectorSize - chunkBuf.Len()
		chunkBuf.Write(make([]byte, padding))

		if sectors > 255 {
			return fmt.Errorf("chunk (%d,%d) too large: %d sectors", lx, lz, sectors)
		}

		locEntryOff := 4 * (lx + lz*32)
		locations[locEntryOff] = byte(nextSector >> 16)
		locations[locEntryOff+1] = byte(nextSector >> 8)
		locations[locEntryOff+2] = byte(nextSector)
		locations[locEntryOff+3] = byte(sectors)

		body.Write(chunkBuf.Bytes())
		nextSector += int32(sectors)
	}

	if _, err := f.Write(locations); err != nil {
		return err
	}
	if _, err := f.Write(timestamps); err != nil {
		return err
	}
	if _, err := f.Write(body.Bytes()); err != nil {
		return err
	}
	return nil
}

// RegionFileName returns e.g. "r.8.20.mcr" for the region containing the
// given chunk coordinates.
func RegionFileName(chunkX, chunkZ int32) string {
	return fmt.Sprintf("r.%d.%d.mcr", regionCoord(chunkX), regionCoord(chunkZ))
}