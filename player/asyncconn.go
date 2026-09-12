package player

import (
	"log"
	"net"
	"sync"
	"time"
)

const writeDeadline = 15 * time.Second

const sendQueueSize = 512

type asyncConn struct {
	net.Conn
	sendCh    chan []byte
	closed    chan struct{}
	closeOnce sync.Once
}

func NewAsyncConn(conn net.Conn) net.Conn {
	ac := &asyncConn{
		Conn:   conn,
		sendCh: make(chan []byte, sendQueueSize),
		closed: make(chan struct{}),
	}
	go ac.writeLoop()
	return ac
}

func (ac *asyncConn) writeLoop() {
	for {
		select {
		case data := <-ac.sendCh:
			ac.Conn.SetWriteDeadline(time.Now().Add(writeDeadline))
			if _, err := ac.Conn.Write(data); err != nil {
				ac.Close()
				return
			}
		case <-ac.closed:
			for {
				select {
				case data := <-ac.sendCh:
					ac.Conn.SetWriteDeadline(time.Now().Add(writeDeadline))
					ac.Conn.Write(data)
				default:
					return
				}
			}
		}
	}
}

func (ac *asyncConn) Write(b []byte) (int, error) {
	data := make([]byte, len(b))
	copy(data, b)

	select {
	case ac.sendCh <- data:
		return len(b), nil
	case <-ac.closed:
		return 0, net.ErrClosed
	default:
		log.Println("Client send queue full, dropping connection:", ac.Conn.RemoteAddr())
		ac.Close()
		return 0, net.ErrClosed
	}
}

func (ac *asyncConn) Close() error {
	ac.closeOnce.Do(func() {
		close(ac.closed)
	})
	return ac.Conn.Close()
}
