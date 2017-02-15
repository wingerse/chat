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

type Server struct {
	l           net.Listener
	clients     map[*client]struct{}
	messageChan chan message
	port        uint16
	name        string
	m           sync.RWMutex
}

// NewServer starts listening, and returns an initialized server. If an error occured while listening started, (nil, error) is returned.
func NewServer(name string, port uint16) (*Server, error) {
	p := strconv.FormatUint(uint64(port), 10)
	l, e := net.Listen("tcp", "localhost:"+p)
	if e != nil {
		return nil, e
	}
	return &Server{l, make(map[*client]struct{}), make(chan message), port, name, sync.RWMutex{}}, nil
}

// Start starts accepting clients + managing messages.
func (r *Server) Start() {
	go r.sendMessages()
	for {
		conn, e := r.l.Accept()
		if e != nil {
			continue
		}
		go r.handleConn(conn)
	}
}

func (r *Server) handleConn(conn net.Conn) {
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

func (r *Server) handleClient(c *client) {
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

func (r *Server) sendMessages() {
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

func (r *Server) publishMessage(msg string) {
	r.m.RLock()
	for k := range r.clients {
		go k.conn.Write([]byte(msg))
	}
	r.m.RUnlock()
	fmt.Print(msg)
}

func (r *Server) addClient(c *client) {
	r.m.Lock()
	r.clients[c] = struct{}{}
	r.m.Unlock()
}

func (r *Server) deleteClient(c *client) {
	r.m.Lock()
	delete(r.clients, c)
	r.m.Unlock()
}
