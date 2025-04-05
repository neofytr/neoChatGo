package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Configuration constants
const (
	serverPort   = "6969"
	safeMode     = true // Redacts IP addresses in logs when true
	bufferLen    = 512  // Size of read buffer
	initQueueLen = 100  // Initial capacity of message queue
	readTimeout  = 5 * time.Minute
	writeTimeout = 10 * time.Second
)

// Message represents a chat message with sender and content
type Message struct {
	Sender  string
	Content string
}

// String returns the formatted message string
func (m Message) String() string {
	return fmt.Sprintf("%s: %s", m.Sender, m.Content)
}

// Server state
var (
	messageQueue    []Message
	queueLatestIdx  int
	queueMutex      sync.RWMutex
	serverShutdown  = make(chan bool)
	shutdownTimeout = 3 * time.Second
)

// safeRemoteAddress returns the connection's remote address, redacted if safeMode is enabled
func safeRemoteAddress(conn net.Conn) string {
	if safeMode {
		return "[REDACTED]"
	}
	return conn.RemoteAddr().String()
}

// readMessage reads a message from the connection with timeout
func readMessage(conn net.Conn) (string, int) {
	buffer := make([]byte, bufferLen)
	err := conn.SetReadDeadline(time.Now().Add(readTimeout))
	if err != nil {
		log.Printf("error: setting read deadline: %s", err.Error())
		return "", -1
	}

	n, err := conn.Read(buffer)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Printf("error: read timeout: %s", err.Error())
			return "", -1
		}

		if n > 0 {
			log.Printf("error: couldn't read from client IP:Port %s: %s\n", safeRemoteAddress(conn), err)
			return "", -1
		} else {
			return "", 0
		}

	}

	// Reset the deadline
	err = conn.SetReadDeadline(time.Time{})
	if err != nil {
		log.Printf("warning: couldn't reset read deadline: %v", err)
	}

	return string(buffer[:n]), n
}

// writeMessage writes a message to the connection with timeout
func writeMessage(conn net.Conn, message string) error {
	err := conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err != nil {
		return fmt.Errorf("setting write deadline: %w", err)
	}

	_, err = conn.Write([]byte(message + "\n"))
	if err != nil {
		return fmt.Errorf("writing message: %w", err)
	}

	// Reset the deadline
	err = conn.SetWriteDeadline(time.Time{})
	if err != nil {
		log.Printf("warning: couldn't reset write deadline: %v", err)
	}

	return nil
}

// broadcastMessage adds a message to the queue for all clients
func broadcastMessage(message Message) {
	queueMutex.Lock()
	messageQueue = append(messageQueue, message)
	queueLatestIdx++
	queueMutex.Unlock() // Fixed: Moved unlock outside to ensure index is updated before readers check

	// Log the broadcast for debugging
	log.Printf("debug: broadcasted message from %s (queue index: %d)", message.Sender, queueLatestIdx-1)
}

// serverMessage creates and broadcasts a server message
func serverMessage(content string) {
	log.Printf("server: %s", content) // Log server messages
	broadcastMessage(Message{Sender: "SERVER", Content: content})
}

// handleReading continuously reads messages from a client connection
func handleReading(conn net.Conn, name string, disconnected chan<- struct{}) {
	defer func() {
		// Signal that client disconnected
		select {
		case <-serverShutdown:
			// Server is already shutting down
		default:
			log.Printf("info: client with IP:Port %s closed connection", safeRemoteAddress(conn))
			close(disconnected)
		}
	}()

	for {
		message, num := readMessage(conn)
		if num == -1 {
			continue
		} else if num == 0 {
			return
		}

		// Trim any trailing newlines or whitespace
		message = strings.TrimSpace(message)
		if message == "" {
			continue // Skip empty messages
		}

		log.Printf("debug: received message from %s: %s", name, message)
		broadcastMessage(Message{Sender: name, Content: message})
	}
}

// handleWriting continuously sends new messages to a client connection
func handleWriting(conn net.Conn, userMsgIdx *int, disconnected <-chan struct{}) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-disconnected:
			return
		case <-serverShutdown:
			return
		case <-ticker.C:
			queueMutex.RLock()
			currentIdx := queueLatestIdx

			// Debug information
			if currentIdx > *userMsgIdx {
				log.Printf("debug: sending messages %d to %d to client", *userMsgIdx, currentIdx-1)
			}

			// Send all new messages
			for ; *userMsgIdx < currentIdx; *userMsgIdx++ {
				if *userMsgIdx < len(messageQueue) {
					msg := messageQueue[*userMsgIdx].String()
					err := writeMessage(conn, msg)
					if err != nil {
						log.Printf("error: sending to client (%s): %v", safeRemoteAddress(conn), err)
						queueMutex.RUnlock()
						return
					}
				}
			}
			queueMutex.RUnlock()
		}
	}
}

// handleConnection manages a single client connection
func handleConnection(conn net.Conn) {
	clientAddr := safeRemoteAddress(conn)
	log.Printf("info: new connection from %s", clientAddr)

	// Create channel to track if client disconnects
	disconnected := make(chan struct{})

	defer func() {
		log.Printf("info: closing connection to %s", clientAddr)
		conn.Close()
	}()

	// Get client's name
	name, num := readMessage(conn)
	if num == -1 {
		return
	} else if num == 0 {
		log.Printf("info: client IP:Port %s closed connection before sending name", safeRemoteAddress(conn))
		return
	}
	name = strings.TrimSpace(name)

	// Validate name
	if name == "" {
		err := writeMessage(conn, "SERVER: Name cannot be empty")
		if err != nil {
			log.Printf("error: couldn't send name validation error: %v", err)
		}
		return
	}

	if name == "SERVER" {
		err := writeMessage(conn, "SERVER: Name 'SERVER' is reserved")
		if err != nil {
			log.Printf("error: couldn't send name validation error: %v", err)
		}
		return
	}

	// Welcome message directly to this client
	welcomeMsg := fmt.Sprintf("SERVER: Welcome to the chat, %s!", name)
	err := writeMessage(conn, welcomeMsg)
	if err != nil {
		log.Printf("error: couldn't send welcome message: %v", err)
		return
	}

	// Announce new user
	log.Printf("info: client %s identified as '%s'", clientAddr, name)
	serverMessage(fmt.Sprintf("%s has joined the chat", name))

	// Track which messages this user has seen
	// Important: Get the current queue index AFTER broadcasting the join message
	queueMutex.RLock()
	userMsgIdx := queueLatestIdx
	queueMutex.RUnlock()

	// Start reading and writing goroutines
	go handleReading(conn, name, disconnected)
	go handleWriting(conn, &userMsgIdx, disconnected)

	// Wait for server shutdown or client disconnection
	select {
	case <-serverShutdown:
		// Send goodbye message
		err := writeMessage(conn, "SERVER: Server is shutting down. Goodbye!")
		if err != nil {
			log.Printf("error: couldn't send shutdown message to %s: %v", name, err)
		}
	case <-disconnected:
		// Client disconnected, announce departure
		serverMessage(fmt.Sprintf("%s has left the chat", name))
	}
}

// acceptConnections continuously accepts new connections
func acceptConnections(listener net.Listener) {
	for {
		// Set accept deadline so we can check for shutdown
		err := listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
		if err != nil {
			log.Printf("error: setting accept deadline: %v", err)
		}

		conn, err := listener.Accept()
		if err != nil {
			// Check if server is shutting down
			select {
			case <-serverShutdown:
				return
			default:
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// This is just the deadline we set
					continue
				}
				log.Printf("error: accepting connection: %v", err)
				continue
			}
		}

		// Handle connection in new goroutine
		go handleConnection(conn)
	}
}

func main() {
	// Set up logging with microseconds for debugging
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Set up signal handling for graceful shutdown
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Create TCP listener
	listener, err := net.Listen("tcp", ":"+serverPort)
	if err != nil {
		log.Fatalf("fatal: couldn't start server on port %s: %v", serverPort, err)
	}
	defer listener.Close()

	// Initialize message queue
	messageQueue = make([]Message, 0, initQueueLen)

	log.Printf("info: chat server started on port %s", serverPort)
	log.Printf("info: press Ctrl+C to shutdown")

	// Start accepting connections
	go acceptConnections(listener)

	// Wait for termination signal
	<-signals
	log.Printf("info: received shutdown signal")

	// Initiate graceful shutdown
	close(serverShutdown)
	log.Printf("info: waiting up to %v for clients to disconnect...", shutdownTimeout)

	// Give connections time to close gracefully
	time.Sleep(shutdownTimeout)

	log.Printf("info: server shutdown complete")
}
