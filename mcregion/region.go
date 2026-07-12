package mcregion

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
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

// ReadChunk reads one chunk's Level tag from a .mcr file at local-in-region
// position (lx, lz), each 0-31
func ReadChunk(path string, lx, lz int32) (*Tag, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var header [4]byte
	if _, err := f.ReadAt(header[:], 4*(int64(lx)+int64(lz)*32)); err != nil {
		return nil, err
	}
	sectorOffset := int32(header[0])<<16 | int32(header[1])<<8 | int32(header[2])
	sectorCount := header[3]
	if sectorOffset == 0 && sectorCount == 0 {
		return nil, nil // not generated
	}

	byteOffset := int64(sectorOffset) * sectorSize
	lenBuf := make([]byte, 5)
	if _, err := f.ReadAt(lenBuf, byteOffset); err != nil {
		return nil, err
	}
	length := int32(binary.BigEndian.Uint32(lenBuf[:4]))
	if length < 1 {
		return nil, nil
	}
	compressionType := lenBuf[4]
	if compressionType != 2 {
		return nil, fmt.Errorf("%s: chunk (%d,%d) unsupported compression %d", path, lx, lz, compressionType)
	}

	payload := make([]byte, length-1)
	if _, err := f.ReadAt(payload, byteOffset+5); err != nil {
		return nil, err
	}

	zr, err := zlib.NewReader(bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("%s: chunk (%d,%d) zlib: %w", path, lx, lz, err)
	}
	defer zr.Close()
	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("%s: chunk (%d,%d) inflate: %w", path, lx, lz, err)
	}

	root, err := ParseRoot(raw)
	if err != nil {
		return nil, err
	}
	return root.Get("Level"), nil
}

// ReadRegion reads every present chunk out of a .mcr file at once, keyed
// by local-in-region (lx, lz), can be used for bulk loading
func ReadRegion(path string) (map[[2]int32]*Tag, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	header := make([]byte, sectorSize)
	if _, err := io.ReadFull(f, header); err != nil {
		return nil, err
	}
	if _, err := f.Seek(sectorSize, io.SeekCurrent); err != nil { // skip timestamps
		return nil, err
	}
	body, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	result := make(map[[2]int32]*Tag)
	for lz := int32(0); lz < 32; lz++ {
		for lx := int32(0); lx < 32; lx++ {
			off := 4 * (lx + lz*32)
			sectorOffset := int32(header[off])<<16 | int32(header[off+1])<<8 | int32(header[off+2])
			sectorCount := header[off+3]
			if sectorOffset == 0 && sectorCount == 0 {
				continue
			}

			byteOffset := (sectorOffset - 2) * sectorSize
			if byteOffset < 0 || int(byteOffset)+5 > len(body) {
				return nil, fmt.Errorf("region %s: chunk (%d,%d) bad offset", path, lx, lz)
			}

			length := int32(binary.BigEndian.Uint32(body[byteOffset : byteOffset+4]))
			if length < 1 {
				continue
			}
			compressionType := body[byteOffset+4]
			if compressionType != 2 {
				return nil, fmt.Errorf("region %s: chunk (%d,%d) unsupported compression %d", path, lx, lz, compressionType)
			}

			start, end := byteOffset+5, byteOffset+5+(length-1)
			if int(end) > len(body) {
				return nil, fmt.Errorf("region %s: chunk (%d,%d) payload out of bounds", path, lx, lz)
			}

			zr, err := zlib.NewReader(bytes.NewReader(body[start:end]))
			if err != nil {
				return nil, fmt.Errorf("region %s: chunk (%d,%d) zlib: %w", path, lx, lz, err)
			}
			raw, err := io.ReadAll(zr)
			zr.Close()
			if err != nil {
				return nil, fmt.Errorf("region %s: chunk (%d,%d) inflate: %w", path, lx, lz, err)
			}

			tag, err := ParseRoot(raw)
			if err != nil {
				return nil, fmt.Errorf("region %s: chunk (%d,%d) nbt: %w", path, lx, lz, err)
			}
			result[[2]int32{lx, lz}] = tag
		}
	}
	return result, nil
}
