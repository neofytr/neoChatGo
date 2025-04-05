const net = require("net");

// DOM Elements
const nameInput = document.getElementById("name-input");
const connectBtn = document.getElementById("connect-btn");
const messageInput = document.getElementById("message-input");
const sendBtn = document.getElementById("send-btn");
const messagesContainer = document.getElementById("messages");
const statusIndicator = document.querySelector(".status-indicator");
const statusText = document.getElementById("status-text");

// Connection settings
const serverAddr = "127.0.0.1";
const serverPort = 6969;
let socket = null;
let connected = false;

// Connect to the server
connectBtn.addEventListener("click", () => {
  const name = nameInput.value.trim();

  // Name validation
  if (name === "") {
    showStatus("Empty name is not allowed!", "error");
    return;
  } else if (name === "SERVER") {
    showStatus("The name SERVER is not allowed!", "error");
    return;
  }

  // Create socket connection
  try {
    socket = new net.Socket();

    socket.connect(serverPort, serverAddr, () => {
      // Send name to server
      socket.write(name);

      // Update UI
      connected = true;
      updateConnectionStatus(true, `Connected as ${name}`);
      nameInput.disabled = true;
      connectBtn.disabled = true;
      messageInput.disabled = false;
      sendBtn.disabled = false;
      messageInput.focus();

      // Add welcome message
      addMessage("system", `You've joined the chat as ${name}!`);
    });

    // Handle incoming messages
    socket.on("data", (data) => {
      const message = data.toString("utf-8");
      addMessage("received", message);
    });

    // Handle connection close
    socket.on("close", () => {
      if (connected) {
        connected = false;
        updateConnectionStatus(false, "Disconnected from server");
        addMessage("system", "Connection closed");
        disableChat();
      }
    });

    // Handle errors
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

// Send message
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

// Send message on button click
sendBtn.addEventListener("click", sendMessage);

// Send message on Enter key
messageInput.addEventListener("keypress", (e) => {
  if (e.key === "Enter") {
    sendMessage();
  }
});

// Add message to the messages container
function generateColorFromName(name) {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }

  // Generate HSL color with fixed saturation and lightness for readability
  const h = Math.abs(hash) % 360;
  return `hsl(${h}, 70%, 45%)`;
}

// Updated addMessage function
function addMessage(type, content) {
  const messageElement = document.createElement("div");
  messageElement.classList.add("message", type);

  // Check if the message contains a username (format: "name: message")
  const colonIndex = content.indexOf(":");

  if (colonIndex > 0) {
    const username = content.substring(0, colonIndex).trim();
    const messageText = content.substring(colonIndex + 1).trim();

    // Create username element
    const usernameElement = document.createElement("span");
    usernameElement.classList.add("username-badge");
    usernameElement.textContent = username;

    // Set background color based on username
    usernameElement.style.backgroundColor = generateColorFromName(username);

    // Create message text element
    const textElement = document.createElement("span");
    textElement.classList.add("message-text");
    textElement.textContent = messageText;

    // Clear and append new elements
    messageElement.textContent = "";
    messageElement.appendChild(usernameElement);
    messageElement.appendChild(textElement);
  } else {
    // For system messages without username format
    messageElement.classList.add("system");
    messageElement.textContent = content;
  }

  messagesContainer.appendChild(messageElement);
  messagesContainer.scrollTop = messagesContainer.scrollHeight;
}

// Update connection status
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

// Show status message
function showStatus(message, type = "info") {
  addMessage("system", message);
  console.log(message);
}

// Disable chat interface
function disableChat() {
  nameInput.disabled = false;
  connectBtn.disabled = false;
  messageInput.disabled = true;
  sendBtn.disabled = true;
  updateConnectionStatus(false, "Disconnected");
}
