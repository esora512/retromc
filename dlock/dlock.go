package dlock

import (
	"log"
	"runtime"
	"strings"
	"sync"
	"time"
)

// maxStackDepth caps how many frames we walk when building a caller chain.
const maxStackDepth = 32

type DebugRWMutex struct {
	mu sync.RWMutex

	name string

	stateMu sync.Mutex

	writer *lockHolder

	readerCount int
	readSince   time.Time
}

type lockHolder struct {
	caller string
	since  time.Time
}

func NewDebugRWMutex(name string) *DebugRWMutex {
	return &DebugRWMutex{
		name: name,
	}
}

func (m *DebugRWMutex) Lock() {
	waitStart := time.Now()

	m.mu.Lock()

	waited := time.Since(waitStart)

	holder := &lockHolder{
		caller: callStack(2),
		since:  time.Now(),
	}

	m.stateMu.Lock()
	m.writer = holder
	m.stateMu.Unlock()

	log.Printf(
		"[LOCK] %s WRITE acquired | waited=%s | stack=%s",
		m.name,
		waited,
		holder.caller,
	)
}

func (m *DebugRWMutex) Unlock() {
	m.stateMu.Lock()

	holder := m.writer
	m.writer = nil

	m.stateMu.Unlock()

	held := time.Since(holder.since)

	log.Printf(
		"[UNLOCK] %s WRITE released | held=%s | stack=%s",
		m.name,
		held,
		holder.caller,
	)

	m.mu.Unlock()
}

func (m *DebugRWMutex) RLock() {
	waitStart := time.Now()

	m.mu.RLock()

	waited := time.Since(waitStart)

	stack := callStack(2)

	m.stateMu.Lock()

	if m.readerCount == 0 {
		m.readSince = time.Now()
	}

	m.readerCount++

	readerCount := m.readerCount

	m.stateMu.Unlock()

	log.Printf(
		"[RLOCK] %s READ acquired | waited=%s | readers=%d | stack=%s",
		m.name,
		waited,
		readerCount,
		stack,
	)
}

func (m *DebugRWMutex) RUnlock() {
	stack := callStack(2)

	m.stateMu.Lock()

	m.readerCount--
	readerCount := m.readerCount

	held := time.Since(m.readSince)

	if m.readerCount == 0 {
		m.readSince = time.Time{}
	}

	m.stateMu.Unlock()

	log.Printf(
		"[RUNLOCK] %s READ released | held=%s | readers=%d | stack=%s",
		m.name,
		held,
		readerCount,
		stack,
	)

	m.mu.RUnlock()
}

// callStack walks the goroutine's call stack starting `skip` frames above
// callStack itself, and returns a human-readable chain like:
//
//	gameLoop -> fallingBlocksPhysics -> SnapshotEntities
//
// The chain is ordered outermost-caller-first, innermost-caller-last, which
// mirrors how you'd narrate "this was called from X, which was called from Y".
func callStack(skip int) string {
	pcs := make([]uintptr, maxStackDepth)
	// +2 to skip runtime.Callers itself and this callStack frame.
	n := runtime.Callers(skip+1, pcs)
	if n == 0 {
		return "unknown"
	}

	frames := runtime.CallersFrames(pcs[:n])

	var names []string
	for {
		frame, more := frames.Next()

		if frame.Function == "" {
			break
		}

		// Stop once we hit goroutine bootstrap / runtime internals —
		// anything below that is noise (runtime.goexit, etc).
		if strings.HasPrefix(frame.Function, "runtime.") {
			break
		}

		names = append(names, shortFuncName(frame.Function))

		if !more {
			break
		}
	}

	// Reverse so the outermost caller (e.g. gameLoop) reads first.
	for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
		names[i], names[j] = names[j], names[i]
	}

	return strings.Join(names, " -> ")
}

// shortFuncName strips the package path and receiver noise from a fully
// qualified function name, e.g.
//
//	github.com/leNicDev/retromc/level.(*World).SnapshotEntities
//
// becomes
//
//	SnapshotEntities
func shortFuncName(full string) string {
	if idx := strings.LastIndex(full, "/"); idx >= 0 {
		full = full[idx+1:]
	}
	parts := strings.Split(full, ".")
	return parts[len(parts)-1]
}
