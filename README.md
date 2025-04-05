# neoRoom

To run the server on a machine, do

```bash
go build
./neoChatGo
```

To run the client, first make sure that the serverAddr and serverPort variables in client/main.js are correct
Then do the following (make sure you have npm installed) 

```bash
npm init -y
npm install electron net
npm start
```