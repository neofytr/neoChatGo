package main

import (
	"bytes"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const serverPort = "6969"
const safeMode = true
const bufferLen = 512
const initQueueLen = 1024

type message_t struct {
	sender string
	msg    string
}

func (message message_t) getMessageString() string {
	return message.sender + ": " + message.msg
}

func safeRemoteAddress(connection *net.Conn) string {
	if safeMode {
		return "[REDACTED]"
	} else {
		return (*connection).RemoteAddr().String()
	}
}

func safeRead(buffer []byte, connection *net.Conn) int {
	num, err := (*connection).Read(buffer)
	if err != nil && num != 0 {
		log.Printf("ERROR: couldn't read message from the client IP:Port %s: %s\n", safeRemoteAddress(connection), err.Error())
	}

	// it is an error if the client closes connection while we wait for reading
	// num is simply zero in that case
	// we handle the connection closing outside this function

	// it is guaranteed that all the messages we receive (even if typed into the terminal) from the client does not have any feed or newline characters at its ends
	return num
}

func safeWrite(message *string, connection *net.Conn) {
	num, err := (*connection).Write([]byte(*message))
	if err != nil {
		log.Printf("ERROR: couldn't write message to the client IP:Port %s: %s\n", safeRemoteAddress(connection), err.Error())
		return
	}
	if num < len(*message) { // err will be nil in this case
		log.Printf("ERROR: couldn't write the entire message to the client IP:Port %s; wrote only %s\n", safeRemoteAddress(connection), (*message)[:num])
	}
}

func isMessageEqual(firstMessage, secondMessage []byte) bool {
	return bytes.Equal(firstMessage, secondMessage)

	/*
	   if len(firstMessage) != len(secondMessage) {
	       return false
	   }

	   for index, val := range firstMessage {
	       if val != secondMessage[index] {
	           return false
	       }
	   }

	   return true
	*/
}

var queueMutex sync.RWMutex

func handleConnection(connection net.Conn, messageQueue []message_t) {
	defer func() {
		log.Printf("INFO: closing connection to the client IP:Port %s\n", safeRemoteAddress(&connection))
		connection.Close()
	}()

	buffer := make([]byte, bufferLen)
	nameLength := safeRead(buffer, &connection)
	if nameLength == 0 {
		log.Printf("INFO: client IP:Port %s closed connection\n", safeRemoteAddress(&connection))
		return
	}

	name := string(buffer[:nameLength])

	for {
		num := safeRead(buffer, &connection) // returns zero if the client closed connection; otherwise returns the number of bytes read
		if num == 0 {
			log.Printf("INFO: client IP:Port %s closed connection\n", safeRemoteAddress(&connection))
			return
		}

		queueMutex.Lock()
		messageQueue = append(messageQueue, message_t{sender: name, msg: string(buffer[:num])})
		queueMutex.Unlock()

		
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

	messageQueue := make([]message_t, initQueueLen)

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

			log.Printf("INFO: accepted connection from IP:Port -> %s\n", safeRemoteAddress(&conn))

			// handle each connection in a new goroutine
			go handleConnection(conn, messageQueue)
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
