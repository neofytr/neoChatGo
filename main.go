package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

const serverPort = "6969"

func handleConnection(connection net.Conn) {
	defer connection.Close()
}

func main() {
	// create Ctrl+C (SIGINT) signal handler to gracefully shut down the server upon termination
	termSignal := make(chan os.Signal, 1)
	signal.Notify(termSignal, syscall.SIGINT)

	// channel to notify the thread accepting connections that server has been shut down
	acceptNotify := make(chan bool)

	listener, err := net.Listen("tcp", ":"+serverPort)
	if err != nil {
		log.Fatalf("ERROR: TCP connection creation failed on port %s: %s\n", serverPort, err.Error())
	}

	defer func() {
		log.Printf("INFO: closing the chat server\n")
		err = listener.Close()
		if err != nil {
			log.Fatalf("ERROR: couldn't close the chat server: %s\n", err.Error())
		}
	}()

	log.Printf("INFO: Chat Server started on port %s\n", serverPort)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				switch {
				case <-acceptNotify:
					{
						log.Printf("INFO: server received SIGINT\n")
						return
					}
				default:
					{
						log.Printf("ERROR: couldn't accept a connection: %s", err.Error())
						continue
					}
				}

			}

			log.Printf("INFO: accepted connection from IP:Port -> %s\n", conn.RemoteAddr())

			// handle each connection in a new goroutine
			go handleConnection(conn)
		}
	}()

	<-termSignal // wait for the termination signal

	// termination signal is now received, signal the acceptor routine to exit
	close(acceptNotify) // a closed channel is always ready
}
