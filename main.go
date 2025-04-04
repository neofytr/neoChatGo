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

	message := "hi!\n"
	num, err := connection.Write([]byte(message))
	if err != nil {
		log.Printf("ERROR: error writing message to the client IP:Port %s\n", connection.RemoteAddr())
		if num < len(message) && num > 0 {
			log.Printf("ERROR: couldn't write the entire message to the client IP:Port %s; wrote only %s\n", connection.RemoteAddr(), message[:num])
		}
		return
	}
}

func main() {
	// Create Ctrl+C (SIGINT) signal handler
	termSignal := make(chan os.Signal, 1)
	signal.Notify(termSignal, syscall.SIGINT)

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

	// channel to stop accepting new connections
	stop := make(chan struct{})

	go func() {
		for {
			select {
			case <-stop:
				log.Println("INFO: Server shutting down, stopping accept loop")
				return
			default:
				conn, err := listener.Accept()
				if err != nil {
					log.Printf("ERROR: couldn't accept a connection: %s", err.Error())
					continue
				}

				log.Printf("INFO: accepted connection from IP:Port -> %s\n", conn.RemoteAddr())

				go handleConnection(conn)
			}
		}
	}()

	<-termSignal // wait for termination signal

	// signal the acceptor routine to stop
	close(stop)

	log.Println("INFO: Server shutting down gracefully")
}
