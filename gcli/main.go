package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func main() {
	socketDir := flag.String("socketDir", "/tmp/gocore", "the folder where gcli will look for unix domain sockets")
	packageName := flag.String("packageName", "", "the name of the unix domain socket.  This must be specified if there is more than 1 gocore process running")
	keepAlive := flag.Bool("keepAlive", false, "keep the socket open (useful for trace)")
	flag.Parse()

	// flag.Args() returns all non-flag arguments.  However, it doesn't understand multi-word quoted arguments
	var args []string
	i := 0
	for {
		arg := flag.Arg(i)
		if arg == "" {
			break
		}

		if strings.Contains(arg, " ") {
			arg = fmt.Sprintf("%q", arg)
		}
		args = append(args, arg)
		i++
	}

	var addr string
	if *packageName == "" {
		files, err := filepath.Glob(filepath.Join(*socketDir, "*.sock"))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if len(files) == 1 {
			addr = files[0]
		} else if len(files) == 0 {
			fmt.Println("No gocore processes are running.")
			os.Exit(1)
		} else {
			fmt.Printf("There are %d sockets and no packageName specified.\n", len(files))
			os.Exit(1)
		}
	} else {

		addr = fmt.Sprintf("%s/%s.sock", *socketDir, *packageName)
	}

	conn, err := net.Dial("unix", addr)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	tcpconn, ok := conn.(*net.UnixConn)
	if !ok {
		fmt.Printf("Failed to cast %v to net.UnixConn\n", conn)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func(command string) {
		tcpconn.Write([]byte(command + "\n"))
		if *keepAlive {
			io.Copy(tcpconn, os.Stdin)
			tcpconn.CloseWrite()
		} else {
			tcpconn.Write([]byte("quit\n"))
			tcpconn.CloseWrite()
		}
		wg.Done()
	}(strings.Join(args, " "))

	go func() {
		io.Copy(os.Stdout, tcpconn)
		tcpconn.CloseRead()
		wg.Done()
	}()

	wg.Wait()
	return
}
