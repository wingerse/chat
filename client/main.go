package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
)

func main() {

	name := getName()
	conn, e := connect()
	handleError(e)
	fmt.Fprintln(conn, name)
	go receiveMessages(conn)
	b := bufio.NewReader(os.Stdin)
	for {
		text, e := b.ReadString('\n')
		handleError(e)
		_, e = conn.Write([]byte(text))
		handleError(e)
	}
}

func connect() (*net.TCPConn, error) {
	addr, e := net.ResolveTCPAddr("tcp", "localhost:5000")
	if e != nil {
		return nil, e
	}
	return net.DialTCP("tcp", nil, addr)
}

func receiveMessages(conn *net.TCPConn) {
	for {
		io.Copy(os.Stdout, conn)
	}
}

func getName() string {
	if len(os.Args) != 2 {
		fmt.Println("No name given")
		os.Exit(1)
	}
	return os.Args[1]
}

func handleError(e error) {
	if e == nil {
		return
	}

	fmt.Println(e)
	os.Exit(1)
}
