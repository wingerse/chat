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
	client  *Client
}

func (m message) getFormatted() string {
	return m.client.Name + ": " + m.message + "\n"
}

type nameCheck struct {
	name    string
	present bool
}

type Client struct {
	Conn net.Conn
	Name string
}

// Server represents a chat server
type Server struct {
	L             net.Listener
	Clients       map[string]*Client
	messageChan   chan message
	addedChan     chan *Client
	removedChan   chan *Client
	nameCheckChan chan nameCheck
	cmdChan       chan Command
	printChan     chan string
	Port          uint16
	Name          string
	commands      map[string]Command
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
			Clients:       make(map[string]*Client),
			messageChan:   make(chan message, 2),
			addedChan:     make(chan *Client),
			removedChan:   make(chan *Client),
			nameCheckChan: make(chan nameCheck),
			cmdChan:       make(chan Command),
			printChan:     make(chan string, 10),
			Port:          port,
			Name:          name,
			commands:      make(map[string]Command)},
		nil
}

// RegisterCommand registers a command which will then be executed when /name is written.
func (r *Server) RegisterCommand(name string, handler func(s *Server, args []string)) {
	r.commands[name] = Command{name, nil, handler}
}

// Start starts accepting clients + managing messages.
func (r *Server) Start() {
	log.SetOutput(os.Stdout)
	r.RegisterCommand("help", helpCommand)
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

func helpCommand(s *Server, args []string) {
	fmt.Println("List of commands:")
	for _, command := range s.commands {
		fmt.Printf("%s,", command.name)
	}
	fmt.Println()
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
		r.ExecuteCommand(args[0], args[1:])
	}
}

func (r *Server) handleClient(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return
	}
	name := strings.TrimSpace(scanner.Text())

	r.nameCheckChan <- nameCheck{name: name}
	if n := <-r.nameCheckChan; n.present {
		conn.Write([]byte("A user with that name is already online. Choose another name\n"))
		conn.Close()
		return
	}

	c := &Client{conn, name}
	r.addClientSafe(c)
	

	for {
		if !scanner.Scan() {
			if scanner.Err() == nil {
				r.RemoveClient(c)
				return
			}
			if t, ok := scanner.Err().(net.Error); ok {
				if t.Timeout() {
					r.RemoveClient(c)
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
			_, present := r.Clients[n.name]
			r.nameCheckChan <- nameCheck{n.name, present}
		case c := <-r.addedChan:
			r.AddClient(c)
		case m := <-r.messageChan:
			for _, c := range r.Clients {
				if c != m.client {
					go c.Conn.Write([]byte(m.getFormatted()))
				}
			}
			log.Print(m.getFormatted())
		case c := <-r.removedChan:
			r.RemoveClient(c)
		case s := <-r.printChan:
			fmt.Print(s)
		case c := <-r.cmdChan:
			c.handler(r, c.args)
		}

	}
}

// ExecuteCommand executes a command given its name and args.
func (r *Server) ExecuteCommand(name string, args []string) {
	// this function ^ should be called from a different goroutine than handleMessages because the command should be able to do everything without blocking others
	if c, ok := r.commands[name]; ok {
		c.args = args
		r.cmdChan <- c
	} else {
		r.printChan <- "No such command exists\n"
	}
}

func (r *Server) PublishMessage(msg string) {
	for _, c := range r.Clients {
		go c.Conn.Write([]byte(msg))
	}
}

func (r *Server) AddClient(c *Client) {
	r.Clients[c.Name] = c
	r.PublishMessage(c.Name + " has connected to the server\n")
	log.Println(c.Name + "(" + c.Conn.RemoteAddr().String() + ")" + " has connected to the server")
}

func (r *Server) addClientSafe(c *Client) {
	r.addedChan <- c
}

func (r *Server) removeClientSafe(c *Client) {
	r.removedChan <- c
}

// RemoveClient removes the specified client from the server. It disconnects the client, as well as remove it from the server's list.
func (r *Server) RemoveClient(c *Client) {
	for n, cl := range r.Clients {
		if cl == c {
			delete(r.Clients, n)
			cl.Conn.Close()
			r.PublishMessage(c.Name + " has disconnected from the server\n")
			log.Println(c.Name + "(" + c.Conn.RemoteAddr().String() + ")" + " has disconnected from the server")
		}
	}
}

// Command is a data struct holding information about a command.
type Command struct {
	name    string
	args    []string
	handler func(s *Server, args []string)
}
