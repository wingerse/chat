package chat

import "net"
import "bufio"
import "strconv"
import "sync"
import "io"
import "strings"
import "fmt"

type message struct {
	message string
	client  *client
}

type client struct {
	conn   net.Conn
	reader *bufio.Reader
	name   string
}

type Room struct {
	l           net.Listener
	clients     map[*client]struct{}
	messageChan chan message
	port        uint16
	name        string
	m           sync.RWMutex
}

func NewRoom(name string, port uint16) (*Room, error) {
	p := strconv.FormatUint(uint64(port), 10)
	l, e := net.Listen("tcp", "localhost:"+p)
	if e != nil {
		return nil, e
	}
	return &Room{l, make(map[*client]struct{}), make(chan message), port, name, sync.RWMutex{}}, nil
}

func (r *Room) Start() {
	go r.sendMessages()
	for {
		conn, e := r.l.Accept()
		if e != nil {
			continue
		}
		go r.handleConn(conn)
	}
}

func (r *Room) handleConn(conn net.Conn) {
	reader := bufio.NewReader(conn)
	name, e := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if e != nil {
		conn.Close()
		return
	}

	c := &client{conn, reader, name}
	r.addClient(c)
	r.publishMessage(c.name + " has joined the server\n")
	r.handleClient(c)
}

func (r *Room) handleClient(c *client) {
	for {
		msg, e := c.reader.ReadString('\n')
		if e == io.EOF {
			c.conn.Close()
			r.deleteClient(c)
			r.publishMessage(c.name + " has left the server\n")
			return
		} else if e != nil {
			continue
		}
		r.messageChan <- message{c.name + ": " + msg, c}
	}
}

func (r *Room) sendMessages() {
	for {
		m := <-r.messageChan
		r.m.RLock()
		for k := range r.clients {
			if k != m.client {
				go k.conn.Write([]byte(m.message))
			}
		}
		r.m.RUnlock()
		fmt.Print(m.message)
	}
}

func (r *Room) publishMessage(msg string) {
	r.m.RLock()
	for k := range r.clients {
		go k.conn.Write([]byte(msg))
	}
	r.m.RUnlock()
	fmt.Print(msg)
}

func (r *Room) addClient(c *client) {
	r.m.Lock()
	r.clients[c] = struct{}{}
	r.m.Unlock()
}

func (r *Room) deleteClient(c *client) {
	r.m.Lock()
	delete(r.clients, c)
	r.m.Unlock()
}
