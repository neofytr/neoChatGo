import socket

serverAddr = "127.0.0.1"
serverPort = 6969
bufferSize = 1024

clientSocket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
clientSocket.connect((serverAddr, serverPort))

initMessage = clientSocket.recv(bufferSize)
if initMessage != b"connected":
    print("ERROR: couldn't connect to the chat room, retry later")
    exit(1)
    
else:
    print("Welcome to the chat room!")


# the trailing newline in the input is stripped
name = input("Please enter your name: ")
msg = "quit"
clientSocket.sendall(msg.encode(encoding="utf-8"))

# it is guaranteed that any message we receive from the server does not have any line feed or newline characters at its ends
while True:
    message = clientSocket.recv(1024)
    print(message.decode(encoding="utf-8"), end="")
    
    reply = input()
    clientSocket.sendall(reply.encode(encoding="utf-8"))