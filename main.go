package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const serverPort = "6969"
const safeMode = true
const bufferLen = 512
const initQueueLen = 1

var queueLatestIndex int = 0

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
	if err != nil && num > 0 {
		log.Printf("info: couldn't read message from the client IP:Port %s: %s\n", safeRemoteAddress(connection), err.Error())
	}

	// the closing connection case (num == 0) should be handled by the caller
	return num
}

func safeWrite(message *string, connection *net.Conn) {
	_, err := (*connection).Write([]byte(*message + "\n"))
	if err != nil {
		log.Printf("info: couldn't write message to the client IP:Port %s: %s\n", safeRemoteAddress(connection), err.Error())
	}
}

var messageQueue []message_t
var queueMutex sync.RWMutex

func handleReading(connection *net.Conn, name string, shutdown <-chan bool, clientDisconnected chan<- bool) {
	buffer := make([]byte, bufferLen)

	for {
		select {
		case <-shutdown:
			return
		default:
			num := safeRead(buffer, connection)

			if num == 0 {
				// connection is closed
				log.Printf("info: client IP:Port %s closed connection\n", safeRemoteAddress(connection))

				// safely notify about client disconnection
				select {
				case clientDisconnected <- true:
					// message sent successfully
				default:
					// channel might be closed or full, don't panic
				}
				return
			}

			if num > 0 { // only process if we actually read data
				message := string(buffer[:num])

				queueMutex.Lock()
				messageQueue = append(messageQueue, message_t{sender: name, msg: message})
				queueLatestIndex++
				queueMutex.Unlock()
			}
		}
	}
}

func handleWriting(connection *net.Conn, userMessageIndex *int, shutdown <-chan bool, clientDisconnected <-chan bool) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-shutdown:
			return
		case <-clientDisconnected:
			return
		case <-ticker.C:
			queueMutex.RLock()
			currentQueueIndex := queueLatestIndex

			// send all messages that have been received since the last send
			for ; *userMessageIndex < currentQueueIndex; *userMessageIndex++ {
				if *userMessageIndex < len(messageQueue) {
					str := messageQueue[*userMessageIndex].getMessageString()
					safeWrite(&str, connection)
				}
			}
			queueMutex.RUnlock()
		}
	}
}

func handleConnection(connection net.Conn, serverShutdown <-chan bool) {
	// create a buffered channel for client disconnection
	clientDisconnected := make(chan bool, 2) // buffered to prevent blocking

	defer func() {
		log.Printf("info: closing connection to the client IP:Port %s\n", safeRemoteAddress(&connection))
		connection.Close()
	}()

	userMessageIndex := 0

	// read client's name first
	buffer := make([]byte, bufferLen)
	nameLength := safeRead(buffer, &connection)
	if nameLength == 0 {
		log.Printf("info: client IP:Port %s closed connection before sending name\n", safeRemoteAddress(&connection))
		return
	}
	name := string(buffer[:nameLength])

	// let everyone know a new user joined
	joinMessage := message_t{sender: "SERVER", msg: name + " has joined the chat"}

	queueMutex.Lock()
	messageQueue = append(messageQueue, joinMessage)
	queueLatestIndex++
	queueMutex.Unlock()

	// start reading and writing in separate goroutines
	go handleReading(&connection, name, serverShutdown, clientDisconnected)
	go handleWriting(&connection, &userMessageIndex, serverShutdown, clientDisconnected)

	// wait for server shutdown or client disconnection
	select {
	case <-serverShutdown:
		goodbye := "SERVER: Server is shutting down. Goodbye!"
		safeWrite(&goodbye, &connection)

		// allow the message to be sent
		time.Sleep(100 * time.Millisecond)
	case <-clientDisconnected:
		// client disconnected, add a leave message
		leaveMessage := message_t{sender: "SERVER", msg: name + " has left the chat"}

		queueMutex.Lock()
		messageQueue = append(messageQueue, leaveMessage)
		queueLatestIndex++
		queueMutex.Unlock()
	}

	// Give goroutines time to exit naturally before the connection closes
	time.Sleep(200 * time.Millisecond)
}

func main() {
	// create Ctrl+C (SIGINT) signal handler
	termSignal := make(chan os.Signal, 1)
	signal.Notify(termSignal, syscall.SIGINT)

	// channel to notify all routines when server shuts down
	serverShutdown := make(chan bool)

	listener, err := net.Listen("tcp", ":"+serverPort)
	if err != nil {
		log.Fatalf("error: tcp connection creation failed on port %s: %s\n", serverPort, err.Error())
	}
	defer listener.Close()

	// initialize message queue
	messageQueue = make([]message_t, 0, initQueueLen)

	log.Printf("info: chat server started on port %s\n", serverPort)

	// connection acceptor goroutine
	go func() {
		for {
			// non-blocking accept with timeout to check for shutdown
			listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
			conn, err := listener.Accept()

			if err != nil {
				if os.IsTimeout(err) {
					// check if server is shutting down
					select {
					case <-serverShutdown:
						return
					default:
						continue
					}
				}

				log.Printf("info: couldn't accept a connection: %s", err.Error())
				continue
			}

			log.Printf("info: accepted connection from IP:Port -> %s\n", safeRemoteAddress(&conn))

			// handle each connection in a new goroutine
			go handleConnection(conn, serverShutdown)
		}
	}()

	// wait for termination signal
	<-termSignal
	log.Printf("info: chat server received termination signal")

	// initiate graceful shutdown
	log.Printf("info: starting graceful shutdown")
	close(serverShutdown)

	// give connections time to close gracefully
	time.Sleep(1 * time.Second)

	log.Printf("info: chat server shutdown complete\n")
}
