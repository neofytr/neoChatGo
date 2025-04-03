package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

const serverPort = "6969"

func handleConnection(conn net.Conn) {
	defer conn.Close()
	message := "hi!\n"

	num, err := conn.Write([]byte(message))
	if err != nil {
		log.Printf("ERROR: could not write message to %s\n", conn.RemoteAddr())
	}

	if num < len(message) {
		log.Printf("ERROR: could not write the entire message '%s' to %s, wrote %s\n", message, conn.RemoteAddr(), message[:num])
	}
	log.Printf("INFO: closed connection from: %s\n", conn.RemoteAddr())
}

func main() {
	// setup a signal handler for termination signal (Ctrl+C)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	closeServer := make(chan struct{})

	handler, err := net.Listen("tcp", ":"+serverPort)
	if err != nil {
		log.Fatalf("ERROR: could not listen to the port %s: %s\n", serverPort, err)
	}

	log.Printf("INFO: started server on port: %s\n", serverPort)

	go func() {
		for {
			conn, err := handler.Accept()
			if err != nil {
				select {
				case <-closeServer:
					// server is shutting down
					return
				default:
					log.Println("ERROR: couldn't accept a connection:", err)
					continue
				}
			}
			log.Println("INFO: accepted connection from:", conn.RemoteAddr())
			go handleConnection(conn)
		}
	}()

	<-stop // wait for termination signal

	// signal to the goroutine that we're shutting down
	close(closeServer) // a channel will always be ready if it's closed

	fmt.Println("Shutting down the TCP server on port:", serverPort)
	handler.Close()
}
