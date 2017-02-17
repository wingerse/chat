package main

import (
	"fmt"
	"github.com/wsendon/chat/server/chat"
)

func main() {
	r, e := chat.NewServer("test", 5000)
	if e != nil {
		fmt.Println(e)
	}

	r.RegisterCommand("kick", kickCommand)
	r.RegisterCommand("list", listCommand)
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
		s.RemoveClient(c)
	} else {
		fmt.Println("kick: there is no user online named "+name)
	}
}
