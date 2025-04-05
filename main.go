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
const safeMode = true
const bufferLen = 512

func safeRemoteAddress(connection net.Conn) string {
	if safeMode {
		return "[REDACTED]"
	} else {
		return connection.RemoteAddr().String()
	}
}

func safeRead(buffer []byte, connection *net.Conn) int {
	num, err := (*connection).Read(buffer)
	if err != nil {
		log.Printf("ERROR: couldn't read message from the client IP:Port %s\n", safeRemoteAddress(*connection))
	}

	return num
}

func safeWrite(message *string, connection *net.Conn) {
	num, err := (*connection).Write([]byte(*message))
	if err != nil {
		log.Printf("ERROR: couldn't write message to the client IP:Port %s\n", safeRemoteAddress(*connection))
		return
	}
	if num < len(*message) { // err will be nil in this case
		log.Printf("ERROR: couldn't write the entire message to the client IP:Port %s; wrote only %s\n", safeRemoteAddress(*connection), (*message)[:num])
	}
}

func removeFeedNewline(message []byte) []byte {
	modified := make([]byte, bufferLen)
	for index, val := range message {
		if val == 13 {
			break
		}
		modified[index] = val
	}

	return modified
}

func handleConnection(connection net.Conn) {
	defer connection.Close()

	message := "Hi!\nPlease type in your name!\n"
	buffer := make([]byte, bufferLen)
	safeWrite(&message, &connection)
	safeRead(buffer, &connection)

	// buffer will contain \r\n at the end; so we remove it
	username := removeFeedNewline(buffer)
	message = fmt.Sprintf("Welcome %s to the chat room!\n", username)
	safeWrite(&message, &connection)

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

			log.Printf("INFO: accepted connection from IP:Port -> %s\n", safeRemoteAddress(conn))

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
