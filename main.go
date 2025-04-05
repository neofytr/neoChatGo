package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

const serverPort = "6969"
const safeMode = true

func safeRemoteAddress(connection net.Conn) string {
	if safeMode {
		return "[REDACTED]"
	} else {
		return safeRemoteAddress(connection).String()
	}
}

func handleConnection(connection net.Conn) {
	defer connection.Close()

	message := "hi!\n"

	num, err := connection.Write([]byte(message))
	if err != nil {
		log.Printf("ERROR: error writing message to the client IP:Port %s\n", safeRemoteAddress(connection))
		return
	}

	if num < len(message) && num > 0 { // err will be nil in this case
		log.Printf("ERROR: couldn't write the entire message to the client IP:Port %s; wrote only %s\n", safeRemoteAddress(connection), message[:num])
	}

}

func main() {
	// create Ctrl+C (SIGINT) signal handler to gracefully shut down the server upon termination
	termSignal := make(chan os.Signal, 1)
	signal.Notify(termSignal, syscall.SIGINT)

	// channel to notify the thread accepting connections that server has been shut down
	acceptNotify := make(chan bool)

	listener, err := net.Listen("tcp", ":"+serverPort)
	if err != nil {
		log.Fatalf("ERROR: tcp connection creation failed on port %s: %s\n", serverPort, err.Error())
	}

	defer func() {

	}()

	log.Printf("INFO: chat server started on port %s\n", serverPort)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-acceptNotify:
					{
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
	close(termSignal)

	log.Printf("INFO: chat server received termination signal")
	log.Printf("INFO: closing the chat server\n")

	// termination signal is now received, signal the acceptor routine to exit when server is shutdown
	close(acceptNotify)    // a closed channel is always ready
	err = listener.Close() // this will cause the listener.Accept() to return an error
	// and since the acceptNotify channel is now ready, it will cause the acceptor goroutine to exit
	if err != nil {
		log.Fatalf("ERROR: couldn't close the chat server: %s\n", err.Error())
	}
}
