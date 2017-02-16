package chat

import "net"
import "bufio"
import "strconv"
import "sync"
import "strings"
import "log"
import "os"

type message struct {
	message string
	client  *client
}

type nameCheck struct {
	name    string
	present bool
}

func (m message) getFormatted() string {
	return m.client.name + ": " + m.message + "\n"
}

type client struct {
	conn net.Conn
	name string
}

type Server struct {
	l             net.Listener
	clients       map[*client]struct{}
	messageChan   chan message
	addedChan     chan *client
	removedChan   chan *client
	nameCheckChan chan nameCheck
	port          uint16
	name          string
	m             sync.RWMutex
}

// NewServer starts listening, and returns an initialized server. If an error occured while listening started, (nil, error) is returned.
func NewServer(name string, port uint16) (*Server, error) {
	p := strconv.FormatUint(uint64(port), 10)
	l, e := net.Listen("tcp", "localhost:"+p)
	if e != nil {
		return nil, e
	}
	return &Server{l: l,
			clients:       make(map[*client]struct{}),
			messageChan:   make(chan message, 2),
			addedChan:     make(chan *client, 2),
			removedChan:   make(chan *client, 2),
			nameCheckChan: make(chan nameCheck),
			port:          port,
			name:          name},
		nil
}

// Start starts accepting clients + managing messages.
func (r *Server) Start() {
	log.SetOutput(os.Stdout)
	log.Println("starting server at "+r.l.Addr().String())
	go r.handleMessages()
	for {
		conn, e := r.l.Accept()
		if e != nil {
			continue
		}
		go r.handleClient(conn)
	}
}

func (r *Server) handleClient(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return
	}
	name := strings.TrimSpace(scanner.Text())

	r.nameCheckChan <- nameCheck{name: name}
	if n := <-r.nameCheckChan; n.present {
		conn.Write([]byte("A user with that name is already online. Choose another name\n"))
		return
	}

	c := &client{conn, name}
	r.addClient(c)
	defer r.removeClient(c)

	for {
		if !scanner.Scan() {
			if scanner.Err() == nil {
				return
			}
			if t, ok := scanner.Err().(net.Error); ok {
				if t.Timeout() {
					return
				}
			}
			continue
		}
		r.messageChan <- message{scanner.Text(), c}
	}
}

func (r *Server) handleMessages() {
	for {
		outerselect:
		select {
		case n := <-r.nameCheckChan:
			for k := range r.clients {
				if k.name == n.name {
					r.nameCheckChan <- nameCheck{n.name, true}
					break outerselect
				}
			}
			r.nameCheckChan <- nameCheck{n.name, false}
		case c := <-r.addedChan:
			r.clients[c] = struct{}{}
			r.publishMessage(c.name + " has connected to the server\n")
			log.Println(c.name + "(" + c.conn.RemoteAddr().String() + ")" + " has connected to the server")
		case m := <-r.messageChan:
			for k := range r.clients {
				if k != m.client {
					go k.conn.Write([]byte(m.getFormatted()))
				}
			}
			log.Print(m.getFormatted())
		case c := <-r.removedChan:
			delete(r.clients, c)
			r.publishMessage(c.name + " has disconnected from the server\n")
			log.Println(c.name + "(" + c.conn.RemoteAddr().String() + ")" + " has disconnected from the server")
		}
	}
}

func (r *Server) publishMessage(msg string) {
	for k := range r.clients {
		go k.conn.Write([]byte(msg))
	}
}

func (r *Server) addClient(c *client) {
	r.addedChan <- c
}

func (r *Server) removeClient(c *client) {
	r.removedChan <- c
}
