package main

import (
	"fmt"
	"github.com/wsendon/chat/server/chat"
)

func main() {
	r, e := chat.NewRoom("test", 5000)
	if e != nil {
		fmt.Println(e)
	}
	r.Start()
}
