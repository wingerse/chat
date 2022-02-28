package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/wingerse/chat/server/chat"
)

func parseArgs() int {
	switch len(os.Args) {
	case 1:
		return 5000
	case 2:
		i, e := strconv.Atoi(os.Args[1])
		if e != nil || i < 0 {
			fmt.Println("Invalid port")
			os.Exit(1)
		}
		return i
	default:
		fmt.Println("format: server [port]")
		os.Exit(1)
		return 0
	}
}

func main() {
	r, e := chat.NewServer("test", uint16(parseArgs()))
	if e != nil {
		fmt.Println(e)
	}

	r.RegisterCommand("kick", kickCommand)
	r.RegisterCommand("list", listCommand)
	r.RegisterCommand("kickall", allKickCommand)
	r.Start()
}

func listCommand(s *chat.Server, args []string) {
	fmt.Println("List of online people:")
	for n, c := range s.Clients {
		fmt.Printf("%v(%v),", n, c.Conn.RemoteAddr())
	}
	fmt.Println()
}

func kickCommand(s *chat.Server, args []string) {
	if len(args) != 1 {
		fmt.Println("kick: invalid format. use /kick <name>")
		return
	}
	name := args[0]
	if c, present := s.Clients[name]; present {
		s.BroadcastMessage(c.Name + " has been kicked from the server")
		s.RemoveClient(c)
	} else {
		fmt.Println("kick: there is no user online named " + name)
	}
}

func allKickCommand(s *chat.Server, args []string) {
	for _, c := range s.Clients {
		s.BroadcastMessage(c.Name + " has been kicked from the server")
		s.RemoveClient(c)
	}
}
