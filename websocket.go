package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type wsConn struct {
	ws     *websocket.Conn
	reader io.Reader
	mu     sync.Mutex // guards writes; gorilla only allows one writer at a time
}

func newWSConn(ws *websocket.Conn) *wsConn {
	return &wsConn{ws: ws}
}

func (c *wsConn) Read(p []byte) (int, error) {
	for {
		if c.reader == nil {
			_, r, err := c.ws.NextReader()
			if err != nil {
				return 0, err
			}
			c.reader = r
		}
		n, err := c.reader.Read(p)
		if err == io.EOF {
			c.reader = nil
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

func (c *wsConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ws.WriteMessage(websocket.BinaryMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (c *wsConn) Close() error         { return c.ws.Close() }
func (c *wsConn) LocalAddr() net.Addr  { return c.ws.LocalAddr() }
func (c *wsConn) RemoteAddr() net.Addr { return c.ws.RemoteAddr() }
func (c *wsConn) SetDeadline(t time.Time) error {
	_ = c.ws.SetReadDeadline(t)
	return c.ws.SetWriteDeadline(t)
}
func (c *wsConn) SetReadDeadline(t time.Time) error  { return c.ws.SetReadDeadline(t) }
func (c *wsConn) SetWriteDeadline(t time.Time) error { return c.ws.SetWriteDeadline(t) }

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true }, // MC clients won't send Origin at all
}

// runOnRender starts the server in Render mode and never returns when the
// RENDER env var (set automatically by Render) is present; otherwise it
// returns immediately so the caller can fall through to the plain TCP
// listener. In Render mode the world is restored from/backed up to B2 (if
// configured) and game traffic is tunneled over a WebSocket on $PORT, since
// Render only exposes HTTP. Use bridge.py locally to connect a client.
func runOnRender(s *Server) {
	if _, ok := os.LookupEnv("RENDER"); !ok {
		return
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000" // Render's default
	}

	b2 := newB2Client()
	if b2 != nil {
		restoreWorldFromB2(b2, s.World.WorldDir)
		s.World.SetTriggerManualBackup(func() {
			go backupWorldToB2(b2, s.World)
		})
	} else {
		log.Println("B2 credentials not set (KEY_ID/APP_KEY/B2_BUCKET); world backups disabled")
	}

	s.Run()

	if b2 != nil {
		startBackupLoop(b2, s.World)
	}

	mux := http.NewServeMux()

	// Satisfies Render's health check (TCP probe or HTTP GET, either way).
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("retromc is up"))
	})

	// Real game traffic tunnels through here.
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WS upgrade failed:", err)
			return
		}
		go handleConnection(newWSConn(conn), s.World, s.Tracker)
	})

	log.Printf("Render detected: HTTP/WebSocket bridge listening on :%s (game traffic on /ws, PID: %d)", port, os.Getpid())
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
