package main

import (
	"bufio"
	"net"
)

func main() {
	listener, _ := net.Listen("tcp", ":20000")
	clients := make([]net.Conn, 0, 3)
	for {
		conn, _ := listener.Accept()
		clients = append(clients, conn)
		go func() {
			r := bufio.NewReader(conn)
			for {
				message, _ := r.ReadBytes('\n')
				for _, c := range clients {
					if c != conn {
						c.Write(message)
					}
				}
			}
		}()
	}
}
