package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
)

var endChan = make(chan struct{})

func main() {
	name := getName()
	serverIP := getServerIP()

	conn, e := connect(serverIP)
	handleError(e)
	defer conn.Close()
	fmt.Fprintln(conn, name)
	go receiveMessages(conn)
	go sendMessages(conn)
	<-endChan
	fmt.Println("Connection closed")
}

func connect(serverIP string) (*net.TCPConn, error) {
	addr, e := net.ResolveTCPAddr("tcp", serverIP)
	if e != nil {
		return nil, e
	}
	return net.DialTCP("tcp", nil, addr)
}

func sendMessages(conn *net.TCPConn) {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		_, e := conn.Write([]byte(s.Text() + "\n"))
		if e != nil {
			fmt.Println(e)
		}
	}
}

func receiveMessages(conn *net.TCPConn) {
	for {
		_, e := io.Copy(os.Stdout, conn)
		if e == nil {
			endChan <- struct{}{}
			return
		}
	}
}

func getName() string {
	if len(os.Args) < 2 {
		fmt.Println("No name given")
		os.Exit(1)
	}
	return os.Args[1]
}

func getServerIP() string {
	if len(os.Args) < 3 {
		return "localhost:5000"
	}
	return os.Args[2]
}

func handleError(e error) {
	if e == nil {
		return
	}

	fmt.Println(e)
	os.Exit(1)
}
