package player

import (
	"log"
	"net"
	"sync"
	"time"
)

const writeDeadline = 15 * time.Second

const sendQueueSize = 8192

const maxCoalescedWrite = 64 * 1024

const socketSendBuffer = 64 * 1024

type asyncConn struct {
	net.Conn
	sendCh    chan []byte
	closed    chan struct{}
	closeOnce sync.Once
}

func NewAsyncConn(conn net.Conn) net.Conn {
	if b, ok := conn.(interface{ SetWriteBuffer(int) error }); ok {
		b.SetWriteBuffer(socketSendBuffer)
	}
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
			data = ac.coalesce(data)
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

func (ac *asyncConn) coalesce(first []byte) []byte {
	if len(ac.sendCh) == 0 || len(first) >= maxCoalescedWrite {
		return first
	}
	buf := make([]byte, 0, maxCoalescedWrite)
	buf = append(buf, first...)
	for len(buf) < maxCoalescedWrite {
		select {
		case next := <-ac.sendCh:
			buf = append(buf, next...)
		default:
			return buf
		}
	}
	return buf
}

func Backlog(conn net.Conn) int {
	if ac, ok := conn.(*asyncConn); ok {
		return len(ac.sendCh)
	}
	return 0
}

func (ac *asyncConn) Write(b []byte) (int, error) {
	data := make([]byte, len(b))
	copy(data, b)
	return ac.WriteOwned(data)
}

func (ac *asyncConn) WriteOwned(b []byte) (int, error) {
	select {
	case ac.sendCh <- b:
		return len(b), nil
	case <-ac.closed:
		return 0, net.ErrClosed
	default:
		log.Println("Client send queue full, dropping connection:", ac.Conn.RemoteAddr())
		ac.Close()
		return 0, net.ErrClosed
	}
}

func WriteOwned(conn net.Conn, b []byte) (int, error) {
	if ac, ok := conn.(*asyncConn); ok {
		return ac.WriteOwned(b)
	}
	return conn.Write(b)
}

func (ac *asyncConn) Close() error {
	ac.closeOnce.Do(func() {
		close(ac.closed)
	})
	return ac.Conn.Close()
}
