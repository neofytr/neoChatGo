package main

import (
	"log"
	"net"
)

const serverPort = "6969"

func handleConnection(conn net.Conn) {
	message := "hi!\n"
	num, err := conn.Write([]byte("hi!\n"))
	if err != nil || num < len(message) {
		log.Printf("ERROR: could not write message to %s\n", conn.RemoteAddr())
	}
	conn.Close()
	log.Printf("INFO: closed connection from: %s\n", conn.RemoteAddr())
}

func main() {
	handler, err := net.Listen("tcp", ":"+serverPort)
	if err != nil {
		log.Fatalf("ERROR: could not listen to the port %s: %s\n", serverPort, err)
	}

	log.Printf("INFO: started server on port: %s\n", serverPort)

	for {
		conn, err := handler.Accept()
		if err != nil {
			log.Println("ERROR: couldn't accept a connection:", err)
			continue
		}
		log.Println("INFO: accepted connection from:", conn.RemoteAddr())
		go handleConnection(conn)
	}
}
