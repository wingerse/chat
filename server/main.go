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
	r.Start()
}

func kickCommand(s *chat.Server, args []string) {
	fmt.Println("Test")
}
