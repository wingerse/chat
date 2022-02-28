package chat

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

type message struct {
	message string
	client  *Client
}

func (m *message) String() string {
	return m.client.Name + ": " + m.message
}

type nameCheck struct {
	name    string
	present bool
}

type Client struct {
	Conn     net.Conn
	Name     string
	SendChan chan string
}

// Server represents a chat server
type Server struct {
	L                net.Listener
	Clients          map[string]*Client
	messageChan      chan message
	addClientChan    chan *Client
	removeClientChan chan *Client
	nameCheckChan    chan nameCheck
	cmdChan          chan Command
	printChan        chan string
	Port             uint16
	Name             string
	commands         map[string]Command
}

// NewServer starts listening, and returns an initialized server. If an error occured while listening started, (nil, error) is returned.
func NewServer(name string, port uint16) (*Server, error) {
	p := strconv.FormatUint(uint64(port), 10)
	l, e := net.Listen("tcp", "localhost:"+p)
	if e != nil {
		return nil, e
	}
	return &Server{
			L:                l,
			Clients:          make(map[string]*Client),
			messageChan:      make(chan message, 2),
			addClientChan:    make(chan *Client),
			removeClientChan: make(chan *Client),
			nameCheckChan:    make(chan nameCheck),
			cmdChan:          make(chan Command),
			printChan:        make(chan string, 10),
			Port:             port,
			Name:             name,
			commands:         make(map[string]Command)},
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

func (r *Server) handleClient(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	// first line is the name
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

	c := &Client{conn, name, make(chan string)}
	r.addClientChan <- c

	go r.handleRecv(c, scanner)
	go r.handleSend(c)
}

func (s *Server) handleRecv(c *Client, scanner *bufio.Scanner) {
	for {
		if !scanner.Scan() {
			s.removeClientChan <- c
			return
		}
		s.messageChan <- message{scanner.Text(), c}
	}
}

func (s *Server) handleSend(c *Client) {
	for {
		message, ok := <-c.SendChan
		if !ok {
			c.Conn.Close()
			return
		}
		_, e := c.Conn.Write([]byte(message + "\n"))
		if e != nil {
			s.removeClientChan <- c
			return
		}
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
			r.BroadcastMessage("[Server Admin]: " + line)
			continue
		}
		line = line[1:]
		args := strings.Split(line, " ")
		if len(args) == 0 {
			r.printChan <- invalidFormat
			continue
		}
		r.executeCommand(args[0], args[1:])
	}
}

func (r *Server) handleMessages() {
	for {
		select {
		case n := <-r.nameCheckChan:
			if n.name == "[Server Admin]" {
				n.present = true
			} else {
				_, present := r.Clients[n.name]
				n.present = present
			}
			r.nameCheckChan <- n
		case c := <-r.addClientChan:
			r.AddClient(c)
		case m := <-r.messageChan:
			for _, c := range r.Clients {
				if c != m.client {
					c.SendChan <- m.String()
				}
			}
			log.Println(m.String())
		case c := <-r.removeClientChan:
			r.RemoveClient(c)
		case s := <-r.printChan:
			fmt.Print(s)
		case c := <-r.cmdChan:
			c.handler(r, c.args)
		}
	}
}

func (r *Server) executeCommand(name string, args []string) {
	// this function ^ should be called from a different goroutine than handleMessages because the command should be able to do everything without blocking others
	if c, ok := r.commands[name]; ok {
		c.args = args
		r.cmdChan <- c
	} else {
		r.printChan <- "No such command exists\n"
	}
}

func (r *Server) BroadcastMessage(msg string) {
	for _, c := range r.Clients {
		c.SendChan <- msg
	}
}

func (r *Server) AddClient(c *Client) {
	r.Clients[c.Name] = c
	r.BroadcastMessage(c.Name + " has connected to the server")
	log.Println(c.Name + "(" + c.Conn.RemoteAddr().String() + ")" + " has connected to the server")
}

// RemoveClient removes the specified client from the server. It disconnects the client, as well as remove it from the server's list.
func (r *Server) RemoveClient(c *Client) {
	if _, ok := r.Clients[c.Name]; !ok {
		return
	}
	delete(r.Clients, c.Name)
	close(c.SendChan)
	r.BroadcastMessage(c.Name + " has disconnected from the server")
	log.Println(c.Name + "(" + c.Conn.RemoteAddr().String() + ")" + " has disconnected from the server")
}

// Command is a data struct holding information about a command.
type Command struct {
	name    string
	args    []string
	handler func(s *Server, args []string)
}
