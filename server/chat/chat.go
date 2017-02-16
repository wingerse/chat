package chat

import "net"
import "bufio"
import "strconv"
import "strings"
import "fmt"
import "log"
import "os"

type message struct {
	message string
	client  *client
}

func (m message) getFormatted() string {
	return m.client.name + ": " + m.message + "\n"
}

type nameCheck struct {
	name    string
	present bool
}

type client struct {
	conn net.Conn
	name string
}

type Server struct {
	L             net.Listener
	Clients       map[*client]struct{}
	messageChan   chan message
	addedChan     chan *client
	removedChan   chan *client
	nameCheckChan chan nameCheck
	cmdChan       chan command
	printChan     chan string
	Port          uint16
	Name          string
	commands      map[string]command
}

// NewServer starts listening, and returns an initialized server. If an error occured while listening started, (nil, error) is returned.
func NewServer(name string, port uint16) (*Server, error) {
	p := strconv.FormatUint(uint64(port), 10)
	l, e := net.Listen("tcp", "localhost:"+p)
	if e != nil {
		return nil, e
	}
	return &Server{
			L:             l,
			Clients:       make(map[*client]struct{}),
			messageChan:   make(chan message, 2),
			addedChan:     make(chan *client, 2),
			removedChan:   make(chan *client, 2),
			nameCheckChan: make(chan nameCheck),
			cmdChan:       make(chan command),
			printChan:     make(chan string),
			Port:          port,
			Name:          name,
			commands:      make(map[string]command)},
		nil
}

func (r *Server) RegisterCommand(name string, handler func(s *Server, args []string)) {
	r.commands[name] = command{name, nil, handler}
}

// Start starts accepting clients + managing messages.
func (r *Server) Start() {
	log.SetOutput(os.Stdout)
	log.Println("starting server at " + r.L.Addr().String())
	go r.handleCommands()
	go r.handleMessages()
	for {
		conn, e := r.L.Accept()
		if e != nil {
			continue
		}
		go r.handleClient(conn)
	}
}

func (r *Server) handleCommands() {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		const invalidFormat = "Invalid command. Format /<command> args\n"
		line := s.Text()
		if !strings.HasPrefix(line, "/") {
			r.printChan <- invalidFormat
			continue
		}
		line = line[1:]
		args := strings.Split(line, " ")
		if len(args) == 0 {
			r.printChan <- invalidFormat
			continue
		}
		//dispatch to correct command
		if c, ok := r.commands[args[0]]; ok {
			c.args = args[1:]
			r.cmdChan <- c
		} else {
			r.printChan <- "No such comand exists\n"
		}
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
	defer r.RemoveClient(c)

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
		select {
		case n := <-r.nameCheckChan:
			present := false
			for k := range r.Clients {
				if k.name == n.name {
					present = true
				}
			}
			r.nameCheckChan <- nameCheck{n.name, present}
		case c := <-r.addedChan:
			r.Clients[c] = struct{}{}
			r.publishMessage(c.name + " has connected to the server\n")
			log.Println(c.name + "(" + c.conn.RemoteAddr().String() + ")" + " has connected to the server")
		case m := <-r.messageChan:
			for k := range r.Clients {
				if k != m.client {
					go k.conn.Write([]byte(m.getFormatted()))
				}
			}
			log.Print(m.getFormatted())
		case c := <-r.removedChan:
			_, existed := r.Clients[c]
			delete(r.Clients, c)
			if existed {
				r.publishMessage(c.name + " has disconnected from the server\n")
				log.Println(c.name + "(" + c.conn.RemoteAddr().String() + ")" + " has disconnected from the server")
			}
		case s := <-r.printChan:
			fmt.Print(s)
		case c := <-r.cmdChan:
			c.handler(r, c.args)
		}

	}
}

func (r *Server) publishMessage(msg string) {
	for k := range r.Clients {
		go k.conn.Write([]byte(msg))
	}
}

func (r *Server) addClient(c *client) {
	r.addedChan <- c
}

func (r *Server) RemoveClient(c *client) {
	r.removedChan <- c
}

type command struct {
	name    string
	args    []string
	handler func(s *Server, args []string)
}
