package level

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/leNicDev/retromc/mcregion"
)

// externalChunkGenTimeout bounds a single chunkgen invocation. 
const externalChunkGenTimeout = 10 * time.Second

// generateChunkExternal shells out to the external C++ chunkgen binary
func (w *World) generateChunkExternal(cx, cz, dim int32) (*Chunk, error) {
	if w.ExternalChunkGenBin == "" {
		return nil, fmt.Errorf("external chunkgen not configured")
	}
	binPath, err := filepath.Abs(w.ExternalChunkGenBin)
	if err != nil {
		return nil, fmt.Errorf("resolving external chunkgen path: %w", err)
	}
	if _, err := os.Stat(binPath); err != nil {
		return nil, fmt.Errorf("external chunkgen binary unavailable: %w", err)
	}

	outFile, err := os.CreateTemp("", fmt.Sprintf("retromc-chunk-%d-%d-*.nbt", cx, cz))
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	outPath := outFile.Name()
	outFile.Close()
	defer os.Remove(outPath)

	ctx, cancel := context.WithTimeout(context.Background(), externalChunkGenTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath,
		"--seed", strconv.FormatInt(w.Seed, 10),
		"--dimension", strconv.FormatInt(int64(dim), 10),
		"--x", strconv.FormatInt(int64(cx), 10),
		"--z", strconv.FormatInt(int64(cz), 10),
		"--out", outPath,
	)
	cmd.Dir = filepath.Dir(binPath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("chunkgen invocation failed: %w (stderr: %s)", err, stderr.String())
	}

	raw, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("reading chunkgen output: %w", err)
	}

	c, err := chunkFromNBTBytes(w, raw, cx, cz)
	if err != nil {
		return nil, err
	}
	c.HasChanged = false
	return c, nil
}

// chunkFromNBTBytes decodes a gzip-compressed standalone chunk NBT file 
func chunkFromNBTBytes(w *World, raw []byte, cx, cz int32) (*Chunk, error) {
	gr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("chunkgen output is not gzip: %w", err)
	}
	defer gr.Close()
	nbtData, err := io.ReadAll(gr)
	if err != nil {
		return nil, fmt.Errorf("decompressing chunkgen output: %w", err)
	}

	root, err := mcregion.ParseRoot(nbtData)
	if err != nil {
		return nil, fmt.Errorf("parsing chunkgen NBT: %w", err)
	}
	level := root.Get("Level")
	if level == nil {
		return nil, fmt.Errorf("chunkgen NBT missing Level compound")
	}

	c, err := w.readChunkFromNBT(level, cx, cz)
	if err != nil {
		return nil, fmt.Errorf("decoding chunkgen chunk: %w", err)
	}
	return c, nil
}
