package level

import (
	"os"
	"path/filepath"
	"testing"
)

// externalChunkGenTestBin resolves the path to the repo's prebuilt chunkgen
// binary (bin/chunkgen, one level up from this package) and skips the test
// if it isn't present, so the suite still passes in a checkout that hasn't
// fetched/built the external tool.
func externalChunkGenTestBin(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "bin", "chunkgen"))
	if err != nil {
		t.Fatalf("resolving chunkgen path: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Skipf("chunkgen binary not available at %s: %v", path, err)
	}
	return path
}

func TestGenerateChunkExternal(t *testing.T) {
	binPath := externalChunkGenTestBin(t)

	w := NewWorld("test", 12345, Default)
	w.SetExternalChunkGenBin(binPath)

	c, err := w.generateChunkExternal(3, -2, 0)
	if err != nil {
		t.Fatalf("generateChunkExternal returned error: %v", err)
	}
	if c == nil {
		t.Fatal("generateChunkExternal returned a nil chunk with no error")
	}

	if c.X != 3*CHUNK_SIZE_X || c.Z != -2*CHUNK_SIZE_Z {
		t.Errorf("chunk coords = (%d, %d), want (%d, %d)", c.X, c.Z, 3*CHUNK_SIZE_X, -2*CHUNK_SIZE_Z)
	}

	wantLen := chunkBlocksAmount + 3*chunkNibbleCount
	if len(c.Data) != wantLen {
		t.Fatalf("len(c.Data) = %d, want %d (blocks+data+light+skylight)", len(c.Data), wantLen)
	}

	// Sanity-check the decoded chunk actually looks like generated terrain,
	// not an all-zero/garbage buffer: expect both solid ground near the
	// bottom and open air near the top of a 128-tall Beta chunk.
	airAtTop := 0
	stoneNearBottom := 0
	for lx := 0; lx < CHUNK_SIZE_X; lx++ {
		for lz := 0; lz < CHUNK_SIZE_Z; lz++ {
			if c.GetBlock(lx, 120, lz).TypeId == 0 {
				airAtTop++
			}
			if b := c.GetBlock(lx, 4, lz).TypeId; b != 0 {
				stoneNearBottom++
			}
		}
	}
	if airAtTop == 0 {
		t.Error("expected at least some air near the top of the chunk (y=120), found none")
	}
	if stoneNearBottom == 0 {
		t.Error("expected solid terrain near the bottom of the chunk (y=4), found none")
	}
}

func TestGenerateChunkExternalRelativePath(t *testing.T) {
	absBin := externalChunkGenTestBin(t)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting cwd: %v", err)
	}
	relBin, err := filepath.Rel(cwd, absBin)
	if err != nil {
		t.Fatalf("computing relative chunkgen path: %v", err)
	}
	if filepath.IsAbs(relBin) {
		t.Fatalf("expected a relative path, got %q", relBin)
	}

	w := NewWorld("test", 12345, Default)
	w.SetExternalChunkGenBin(relBin)

	c, err := w.generateChunkExternal(1, 1, 0)
	if err != nil {
		t.Fatalf("generateChunkExternal with relative path %q failed: %v", relBin, err)
	}
	if c == nil {
		t.Fatal("generateChunkExternal returned a nil chunk with no error")
	}
}

func TestGenerateChunkExternalDeterministic(t *testing.T) {
	binPath := externalChunkGenTestBin(t)

	w := NewWorld("test", 999, Default)
	w.SetExternalChunkGenBin(binPath)

	a, err := w.generateChunkExternal(0, 0, 0)
	if err != nil {
		t.Fatalf("first generateChunkExternal call failed: %v", err)
	}
	b, err := w.generateChunkExternal(0, 0, 0)
	if err != nil {
		t.Fatalf("second generateChunkExternal call failed: %v", err)
	}

	if len(a.Data) != len(b.Data) {
		t.Fatalf("regenerating the same chunk produced different-length data: %d vs %d", len(a.Data), len(b.Data))
	}
	for i := range a.Data {
		if a.Data[i] != b.Data[i] {
			t.Fatalf("regenerating chunk (0,0) with the same seed produced different block data at byte %d: %d vs %d", i, a.Data[i], b.Data[i])
		}
	}
}

// TestGenerateChunkFallback verifies that when the configured external
// binary is missing, generateChunk logs the failure and falls back to the
// Go generator instead of panicking or returning a broken chunk.
func TestGenerateChunkFallback(t *testing.T) {
	w := NewWorld("test", 12345, Default)
	w.SetExternalChunkGenBin(filepath.Join(t.TempDir(), "does-not-exist"))

	c := w.generateChunk(0, 0, Default, 0)
	if c == nil {
		t.Fatal("generateChunk returned nil after external chunkgen failure")
	}
	wantLen := chunkBlocksAmount + 3*chunkNibbleCount
	if len(c.Data) != wantLen {
		t.Fatalf("fallback chunk len(Data) = %d, want %d", len(c.Data), wantLen)
	}
}

// TestGenerateChunkDefaultUnaffected verifies that with no external binary
// configured (the default), chunk generation is unchanged.
func TestGenerateChunkDefaultUnaffected(t *testing.T) {
	w := NewWorld("test", 12345, Default)

	c := w.generateChunk(0, 0, Default, 0)
	if c == nil {
		t.Fatal("generateChunk returned nil")
	}
	wantLen := chunkBlocksAmount + 3*chunkNibbleCount
	if len(c.Data) != wantLen {
		t.Fatalf("len(c.Data) = %d, want %d", len(c.Data), wantLen)
	}
}
