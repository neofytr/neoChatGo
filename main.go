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

	num, err := conn.Write([]byte("hi!\n"))
	if err != nil {
		log.Printf("ERROR: could not write message to %s\n", conn.RemoteAddr())
	}

	if num <= len(message) {
		log.Printf("ERROR: could not write the entire message '%s' to %s\n", message, conn.RemoteAddr())
	}
	log.Printf("INFO: closed connection from: %s\n", conn.RemoteAddr())
}

func main() {
	handler, err := net.Listen("tcp", ":"+serverPort)
	if err != nil {
		log.Fatalf("ERROR: could not listen to the port %s: %s\n", serverPort, err)
	}

	defer func() {
		fmt.Println("Shutting down the TCP server on port:", serverPort)
		handler.Close()
	}()

	log.Printf("INFO: started server on port: %s\n", serverPort)

	// setup a signal handler for termination signal (Ctrl+C)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		for {
			conn, err := handler.Accept()
			if err != nil {
				log.Println("ERROR: couldn't accept a connection:", err)
				continue
			}
			log.Println("INFO: accepted connection from:", conn.RemoteAddr())
			go handleConnection(conn)
		}
	}()

	<-stop // wait for termination signal
}
