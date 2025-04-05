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
		log.Printf("ERROR: couldn't read message from the client IP:Port %s: %s\n", safeRemoteAddress(connection), err.Error())
	}

	// the closing connection case (num == 0) should be handled by the caller
	return num
}

func safeWrite(message *string, connection *net.Conn) {
	_, err := (*connection).Write([]byte(*message + "\n"))
	if err != nil {
		log.Printf("ERROR: couldn't write message to the client IP:Port %s: %s\n", safeRemoteAddress(connection), err.Error())
	}
}

var messageQueue []message_t
var queueMutex sync.RWMutex

func handleReading(connection *net.Conn, name string, shutdown chan bool, clientDisconnected chan bool) {
	buffer := make([]byte, bufferLen)

	for {

		select {
		case <-clientDisconnected:
			return
		default:
			num := safeRead(buffer, connection)
			if num == 0 {
				// check if this is due to timeout or actual connection close
				// connection is closed
				log.Printf("INFO: client IP:Port %s closed connection\n", safeRemoteAddress(connection))
				close(clientDisconnected)
				return
			}

			message := string(buffer[:num])

			queueMutex.Lock()
			messageQueue = append(messageQueue, message_t{sender: name, msg: message})
			queueLatestIndex++
			queueMutex.Unlock()
		}
	}
}

func handleWriting(connection *net.Conn, userMessageIndex *int, shutdown chan bool, clientDisconnected chan bool) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
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

func handleConnection(connection net.Conn, serverShutdown chan bool) {

	// create a channel for this connection's shutdown signal
	clientShutdown := make(chan bool)

	// create a channel for client disconnection
	clientDisconnected := make(chan bool)

	defer func() {
	}()

	userMessageIndex := 0

	// read client's name first
	buffer := make([]byte, bufferLen)
	nameLength := safeRead(buffer, &connection)
	if nameLength == 0 {
		log.Printf("INFO: client IP:Port %s closed connection before sending name\n", safeRemoteAddress(&connection))
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
	go handleReading(&connection, name, clientShutdown, clientDisconnected)
	go handleWriting(&connection, &userMessageIndex, clientShutdown, clientDisconnected)

	for {
		select {
		case <-serverShutdown:
			{
				close(clientDisconnected) // signal the client handlers to exit
				goodbye := "SERVER: Server is shutting down. Goodbye!"
				safeWrite(&goodbye, &connection)

				time.Sleep(100 * time.Millisecond) // allow the message to be sent
				return
			}
		case <-clientDisconnected:
			{
				goodbye := "SERVER: Server is shutting down. Goodbye!"
				safeWrite(&goodbye, &connection)

				time.Sleep(100 * time.Millisecond) // allow the message to be sent
				return
			}
		}
	}
}

func main() {
	// create Ctrl+C (SIGINT) signal handler
	termSignal := make(chan os.Signal, 1)
	signal.Notify(termSignal, syscall.SIGINT)

	// channel to notify all routines when server shuts down
	serverShutdown := make(chan bool)

	listener, err := net.Listen("tcp", ":"+serverPort)
	if err != nil {
		log.Fatalf("ERROR: tcp connection creation failed on port %s: %s\n", serverPort, err.Error())
	}
	defer listener.Close()

	// initialize message queue
	messageQueue = make([]message_t, 0, initQueueLen)

	log.Printf("INFO: chat server started on port %s\n", serverPort)

	// connection acceptor goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// set accept deadline to prevent blocking forever
			listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))

			conn, err := listener.Accept()
			if err != nil {
				// check if this is due to timeout or an actual error
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// it's a timeout, check if we need to exit
					select {
					case <-serverShutdown:
						return
					default:
						continue
					}
				}

				// it's another error
				select {
				case <-serverShutdown:
					return
				default:
					log.Printf("ERROR: couldn't accept a connection: %s", err.Error())
					continue
				}
			}

			log.Printf("INFO: accepted connection from IP:Port -> %s\n", safeRemoteAddress(&conn))

			// handle each connection in a new goroutine
			go handleConnection(conn, serverShutdown)
		}
	}()

	// wait for termination signal
	<-termSignal
	close(termSignal)
	log.Printf("INFO: chat server received termination signal")

	// initiate graceful shutdown
	log.Printf("INFO: starting graceful shutdown")
	close(serverShutdown)

	closeAllConnections()

	// wait for acceptor goroutine to finish
	wg.Wait()

	log.Printf("INFO: chat server shutdown complete\n")
}
