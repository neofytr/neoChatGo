const net = require("net");

const nameInput = document.getElementById("name-input");
const connectBtn = document.getElementById("connect-btn");
const messageInput = document.getElementById("message-input");
const sendBtn = document.getElementById("send-btn");
const messagesContainer = document.getElementById("messages");
const statusIndicator = document.querySelector(".status-indicator");
const statusText = document.getElementById("status-text");

const serverAddr = "127.0.0.1";
const serverPort = 6969;
let socket = null;
let connected = false;

connectBtn.addEventListener("click", () => {
  const name = nameInput.value.trim();

  if (name === "") {
    showStatus("Empty name is not allowed!", "error");
    return;
  } else if (name === "SERVER") {
    showStatus("The name SERVER is not allowed!", "error");
    return;
  }

  try {
    socket = new net.Socket();

    socket.connect(serverPort, serverAddr, () => {
      socket.write(name);

      connected = true;
      updateConnectionStatus(true, `Connected as ${name}`);
      nameInput.disabled = true;
      connectBtn.disabled = true;
      messageInput.disabled = false;
      sendBtn.disabled = false;
      messageInput.focus();

      addMessage("system", `You've joined the chat as ${name}!`);
    });

    socket.on("data", (data) => {
      const message = data.toString("utf-8");
      addMessage("received", message);
    });

    socket.on("close", () => {
      if (connected) {
        connected = false;
        updateConnectionStatus(false, "Disconnected from server");
        addMessage("system", "Connection closed");
        disableChat();
      }
    });

    socket.on("error", (err) => {
      showStatus(`Connection error: ${err.message}`, "error");
      connected = false;
      socket = null;
      disableChat();
    });
  } catch (error) {
    showStatus(`Failed to connect: ${error.message}`, "error");
  }
});

function sendMessage() {
  if (!connected) return;

  const message = messageInput.value.trim();
  if (message) {
    try {
      socket.write(message);
      messageInput.value = "";
      messageInput.focus();
    } catch (error) {
      showStatus(`Failed to send message: ${error.message}`, "error");
    }
  }
}

sendBtn.addEventListener("click", sendMessage);

messageInput.addEventListener("keypress", (e) => {
  if (e.key === "Enter") {
    sendMessage();
  }
});

function generateColorFromName(name) {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }

  const h = Math.abs(hash) % 360;
  return `hsl(${h}, 70%, 45%)`;
}

function addMessage(type, content) {
  const messageElement = document.createElement("div");
  messageElement.classList.add("message", type);

  const colonIndex = content.indexOf(":");

  if (colonIndex > 0) {
    const username = content.substring(0, colonIndex).trim();
    const messageText = content.substring(colonIndex + 1).trim();

    const usernameElement = document.createElement("span");
    usernameElement.classList.add("username-badge");
    usernameElement.textContent = username;

    usernameElement.style.backgroundColor = generateColorFromName(username);

    const textElement = document.createElement("span");
    textElement.classList.add("message-text");
    textElement.textContent = messageText;

    messageElement.textContent = "";
    messageElement.appendChild(usernameElement);
    messageElement.appendChild(textElement);
  } else {
    messageElement.classList.add("system");
    messageElement.textContent = content;
  }

  messagesContainer.appendChild(messageElement);
  messagesContainer.scrollTop = messagesContainer.scrollHeight;
}

function updateConnectionStatus(isConnected, message) {
  if (isConnected) {
    statusIndicator.classList.remove("offline");
    statusIndicator.classList.add("online");
  } else {
    statusIndicator.classList.remove("online");
    statusIndicator.classList.add("offline");
  }

  statusText.textContent = message;
}

function showStatus(message, type = "info") {
  addMessage("system", message);
  console.log(message);
}

function disableChat() {
  nameInput.disabled = false;
  connectBtn.disabled = false;
  messageInput.disabled = true;
  sendBtn.disabled = true;
  updateConnectionStatus(false, "Disconnected");
}
