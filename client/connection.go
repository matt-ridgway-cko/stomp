package client

import (
	"github.com/jjeffery/stomp/message"
	"io"
	"log"
	"net"
	"time"
)

const (
	// Maximum number of pending frames allowed to a client.
	// before a disconnect occurs. If the client cannot keep
	// up with the server, we do not want the server to backlog
	// pending frames indefinitely.
	MaxPendingWrites = 16
)

// Connection with client
type Connection struct {
	conn           net.Conn
	requestChannel chan Request
	writeChannel   chan *message.Frame
	stateFunc      func(c *Connection, f *message.Frame) error
	readTimeout    time.Duration
	writeTimeout   time.Duration
}

func newConnection(conn net.Conn, channel chan Request) *Connection {
	c := new(Connection)
	c.conn = conn
	c.requestChannel = channel
	c.writeChannel = make(chan *message.Frame, 32)
	channel <- Request{Type: Create, Connection: c}
	go c.readLoop()
	go c.writeLoop()
	return c
}

// Write a frame to the connection.
func (c *Connection) Send(f *message.Frame) {
	// place the frame on the write channel, or
	// close the connection if the write channel is full,
	// as this means the client is not keeping up.
	select {
	case c.writeChannel <- f:
	default:
		// write channel is full
		c.Close()
	}
}

// TODO: should send other information, such as receipt-id
func (c *Connection) SendError(err error) {
	f := new(message.Frame)
	f.Command = message.ERROR
	f.Headers.Append(message.Message, err.Error())
	c.Send(f) // will close after successful send
}

func (c *Connection) readLoop() {
	reader := message.NewReader(c.conn)
	for {
		if readHeartbeat == Duration(0) {
			// infinite timeout
			c.conn.SetReadDeadline(time.Time(0))
		} else {
			c.conn.SetReadDeadline(time.Now() + readTimeout)
		}
		f, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				log.Println("connection closed:", c.conn.RemoteAddr())
			} else {
				log.Println("read failed:", err, c.conn.RemoteAddr())
			}
			c.Close()
			return
		}

		if f == nil {
			// if the frame is nil, then it is a heartbeat
			continue
		}

		err = c.stateFunc(c, f)
		if err != nil {
			c.SendError(err)
			c.Close()
		}
	}
}

func (c *Connection) writeLoop() {
	writer := message.NewWriter(c.conn)
	for {
		f := <-c.writeChannel
		err := writer.Write(f)
		if err != nil {
			c.conn.Close()
			c.requestChannel <- Request{Type: Disconnect, Connection: c}
			return
		}
		if f.Command == message.ERROR {
			// sent an ERROR frame, so disconnect
			c.Close()
			return
		}
	}
}

func (c *Connection) Close() {
	c.conn.Close()
	c.requestChannel <- Request{Type: Disconnect, Connection: c}
}

func (c *Connection) handleConnect(f *message.Frame) error {
	if heartbeat, ok := f.Headers.Contains(message.HeartBeat); ok {

	} else {

	}

	return nil
}

func connecting(c *Connection, f *message.Frame) error {
	switch f.Command {
	case message.CONNECT, message.STOMP:
		return c.handleConnect(f)
	}
	return errors.New("expecting CONNECT or STOMP command")
}
