package main

import (
    "net"
    "fmt"
    "bufio"
	"os"
	"io"
)

func main() {

    name := getName()
    chatFormat := name + ": \"%s\"\n"
    conn, e := connect()
    handleError(e)
    go receiveMessages(conn)
    s := bufio.NewScanner(os.Stdin)
    for s.Scan() {
        text := s.Text()
        _, e := fmt.Fprintf(conn, chatFormat, text)
        handleError(e)
    }
}

func connect() (*net.TCPConn, error) {
    addr, e := net.ResolveTCPAddr("tcp", "localhost:20000")
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