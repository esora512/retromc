package mcregion

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const sectorSize = 4096

// regionCoord converts chunk coords to region coords (floor division via
// arithmetic shift, which is correct for negative int32 in Go).
func regionCoord(c int32) int32 { return c >> 5 }

// localCoord gives the 0-31 position of a chunk within its region.
func localCoord(c int32) int32 { return c & 31 }

// A zlib writer allocates ~1 MB of compressor state, so reuse them instead of creating one per chunk.
var zlibWriterPool = sync.Pool{New: func() any { return zlib.NewWriter(nil) }}

// WriteRegion writes one .mcr file containing the given chunks. chunks maps
// local-in-region (lx, lz), each 0-31, to that chunk's already-built root
// NBT compound (the "" compound whose only child is "Level").
// Chunks are streamed to disk one by one; the header is filled in last.
func WriteRegion(path string, chunks map[[2]int32]*Compound, rawChunks map[[2]int32]RawChunk) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	// Write to a temp file and rename so an interrupted save can't leave a truncated region.
	tmpPath := path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	fail := func(err error) error {
		f.Close()
		os.Remove(tmpPath)
		return err
	}

	// Sectors 0-1 hold the locations and timestamps tables.
	header := make([]byte, 2*sectorSize)
	if _, err := f.Seek(int64(len(header)), io.SeekStart); err != nil {
		return fail(err)
	}
	bw := bufio.NewWriterSize(f, 64*1024)
	nextSector := int32(2)
	var padding [sectorSize]byte

	writeEntry := func(lx, lz int32, compressionType byte, payload []byte) error {
		total := 5 + len(payload)
		sectors := (total + sectorSize - 1) / sectorSize
		if sectors > 255 {
			return fmt.Errorf("chunk (%d,%d) too large: %d sectors", lx, lz, sectors)
		}

		var lenBuf [5]byte
		binary.BigEndian.PutUint32(lenBuf[:4], uint32(len(payload)+1))
		lenBuf[4] = compressionType
		bw.Write(lenBuf[:])
		bw.Write(payload)
		if _, err := bw.Write(padding[:sectors*sectorSize-total]); err != nil {
			return err
		}

		locEntryOff := 4 * (lx + lz*32)
		header[locEntryOff] = byte(nextSector >> 16)
		header[locEntryOff+1] = byte(nextSector >> 8)
		header[locEntryOff+2] = byte(nextSector)
		header[locEntryOff+3] = byte(sectors)

		nextSector += int32(sectors)
		return nil
	}

	// Freshly built chunks (loaded in memory, possibly modified this session).
	zw := zlibWriterPool.Get().(*zlib.Writer)
	defer zlibWriterPool.Put(zw)
	var compressed bytes.Buffer
	for pos, comp := range chunks {
		compressed.Reset()
		zw.Reset(&compressed)
		if err := comp.WriteRoot(zw); err != nil {
			return fail(err)
		}
		if err := zw.Close(); err != nil {
			return fail(err)
		}
		if err := writeEntry(pos[0], pos[1], 2, compressed.Bytes()); err != nil {
			return fail(err)
		}
	}

	// Leftover chunks from disk that weren't loaded this session, copied
	// through byte-for-byte, untouched.
	for pos, raw := range rawChunks {
		if _, alreadyWritten := chunks[pos]; alreadyWritten {
			continue // in-memory version takes priority
		}
		if err := writeEntry(pos[0], pos[1], raw.CompressionType, raw.Payload); err != nil {
			return fail(err)
		}
	}

	if err := bw.Flush(); err != nil {
		return fail(err)
	}
	if _, err := f.WriteAt(header, 0); err != nil {
		return fail(err)
	}
	if err := f.Sync(); err != nil {
		return fail(err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

// RegionFileName returns e.g. "r.8.20.mcr" for the region containing the
// given chunk coordinates.
func RegionFileName(chunkX, chunkZ int32) string {
	return fmt.Sprintf("r.%d.%d.mcr", regionCoord(chunkX), regionCoord(chunkZ))
}

// ReadChunk reads one chunk's Level tag from a .mcr file at local-in-region
// position (lx, lz), each 0-31
func ReadChunk(f *os.File, lx, lz int32) (*Tag, error) {
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
		return nil, fmt.Errorf("chunk (%d,%d) unsupported compression %d", lx, lz, compressionType)
	}

	payload := make([]byte, length-1)
	if _, err := f.ReadAt(payload, byteOffset+5); err != nil {
		return nil, err
	}

	var zr io.ReadCloser
	var err error
	if pooled := zlibReaderPool.Get(); pooled != nil {
		zr = pooled.(io.ReadCloser)
		err = zr.(zlib.Resetter).Reset(bytes.NewReader(payload), nil)
	} else {
		zr, err = zlib.NewReader(bytes.NewReader(payload))
	}
	if err != nil {
		return nil, fmt.Errorf("chunk (%d,%d) zlib: %w", lx, lz, err)
	}
	defer zlibReaderPool.Put(zr)
	// A chunk's NBT is a little over 80 KB; size the buffer up front instead of growing it repeatedly.
	raw := bytes.NewBuffer(make([]byte, 0, 96*1024))
	if _, err := raw.ReadFrom(zr); err != nil {
		return nil, fmt.Errorf("chunk (%d,%d) inflate: %w", lx, lz, err)
	}

	root, err := ParseRoot(raw.Bytes())
	if err != nil {
		return nil, err
	}
	return root.Get("Level"), nil
}

var zlibReaderPool sync.Pool

type RawChunk struct {
	CompressionType byte
	Payload         []byte // same as on disk
}

func ReadRegionRaw(path string) (map[[2]int32]RawChunk, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	header := make([]byte, sectorSize)
	if _, err := io.ReadFull(f, header); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			// Truncated/empty region file, e.g. left behind by a save that
			// was interrupted mid-write. Treat it like a missing file
			// instead of failing the whole save, the fresh in-memory
			// chunks will overwrite it.
			return nil, nil
		}
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() < 2*sectorSize {
		return nil, nil // truncated, see above
	}
	body := make([]byte, info.Size()-2*sectorSize)
	if _, err := f.ReadAt(body, 2*sectorSize); err != nil { // skip timestamps
		return nil, err
	}

	result := make(map[[2]int32]RawChunk)
	for lz := int32(0); lz < 32; lz++ {
		for lx := int32(0); lx < 32; lx++ {
			off := 4 * (lx + lz*32)
			sectorOffset := int32(header[off])<<16 | int32(header[off+1])<<8 | int32(header[off+2])
			sectorCount := header[off+3]
			if sectorOffset == 0 && sectorCount == 0 {
				continue // not generated
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

			start, end := byteOffset+5, byteOffset+5+(length-1)
			if int(end) > len(body) {
				return nil, fmt.Errorf("region %s: chunk (%d,%d) payload out of bounds", path, lx, lz)
			}

			result[[2]int32{lx, lz}] = RawChunk{
				CompressionType: compressionType,
				Payload:         body[start:end:end], // shares the region body, no copy
			}
		}
	}
	return result, nil
}
