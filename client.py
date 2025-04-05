import socket

serverAddr = "127.0.0.1"
serverPort = "6969"

clientSocket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
clientSocket.connect((serverAddr, serverPort))