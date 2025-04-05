const net = require("net");

const nameInput = document.getElementById("name-input");
const connectBtn = document.getElementById("connect-btn");
const messageInput = document.getElementById("message-input");
const sendBtn = document.getElementById("send-btn");
const messagesContainer = document.getElementById("messages");
const statusIndicator = document.querySelector(".status-indicator");
const statusText = document.getElementById("status-text");
const themeToggle = document.getElementById("theme-toggle");
const reconnectBtn = document.getElementById("reconnect-btn");

const serverAddr = "127.0.0.1";
const serverPort = 6969;
let socket = null;
let connected = false;
let username = "";
let reconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 5;
const RECONNECT_DELAY = 3000; 

function initTheme() {
  const savedTheme = localStorage.getItem("theme") || "light";
  document.body.setAttribute("data-theme", savedTheme);
  if (themeToggle) {
    themeToggle.checked = savedTheme === "dark";
  }
}

if (themeToggle) {
  themeToggle.addEventListener("change", () => {
    const newTheme = themeToggle.checked ? "dark" : "light";
    document.body.setAttribute("data-theme", newTheme);
    localStorage.setItem("theme", newTheme);
  });
}

initTheme();

function attemptReconnect() {
  if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
    reconnectAttempts++;
    showStatus(
      `Reconnecting... Attempt ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS}`,
      "info"
    );

    setTimeout(() => {
      if (!connected && username) {
        connectToServer(username);
      }
    }, RECONNECT_DELAY);
  } else {
    showStatus(
      "Maximum reconnection attempts reached. Please try again manually.",
      "error"
    );
    reconnectBtn.disabled = false;
  }
}

function connectToServer(name) {
  try {
    socket = new net.Socket();

    updateConnectionStatus(false, "Connecting...");

    socket.connect(serverPort, serverAddr, () => {
      socket.write(name);

      connected = true;
      username = name;
      reconnectAttempts = 0;
      updateConnectionStatus(true, `Connected as ${name}`);
      nameInput.disabled = true;
      connectBtn.disabled = true;
      reconnectBtn.disabled = true;
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

        attemptReconnect();
      }
    });

    socket.on("error", (err) => {
      showStatus(`Connection error: ${err.message}`, "error");
      connected = false;
      socket = null;
      disableChat();

      reconnectBtn.disabled = false;
    });
  } catch (error) {
    showStatus(`Failed to connect: ${error.message}`, "error");
    disableChat();
    reconnectBtn.disabled = false;
  }
}

connectBtn.addEventListener("click", () => {
  const name = nameInput.value.trim();

  if (name === "") {
    showStatus("Empty name is not allowed!", "error");
    return;
  } else if (name === "SERVER") {
    showStatus("The name SERVER is not allowed!", "error");
    return;
  }

  username = name;
  connectToServer(name);
});

reconnectBtn.addEventListener("click", () => {
  if (username) {
    reconnectBtn.disabled = true;
    reconnectAttempts = 0;
    connectToServer(username);
  } else {
    showStatus("Please enter a username first", "error");
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

function getFormattedTime() {
  const now = new Date();
  const hours = String(now.getHours()).padStart(2, "0");
  const minutes = String(now.getMinutes()).padStart(2, "0");
  return `${hours}:${minutes}`;
}

function addMessage(type, content) {
  const messageElement = document.createElement("div");
  messageElement.classList.add("message", type);

  const timestampElement = document.createElement("span");
  timestampElement.classList.add("timestamp");
  timestampElement.textContent = getFormattedTime();

  const colonIndex = content.indexOf(":");

  if (type !== "system" && colonIndex > 0) {
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
    messageElement.appendChild(timestampElement);
  } else {
    messageElement.classList.add("system");

    const textElement = document.createElement("span");
    textElement.classList.add("message-text");
    textElement.textContent = content;

    messageElement.textContent = "";
    messageElement.appendChild(textElement);
    messageElement.appendChild(timestampElement);
  }

  messagesContainer.appendChild(messageElement);

  smoothScrollToBottom();
}

function smoothScrollToBottom() {
  const target = messagesContainer.scrollHeight;
  const duration = 300; 
  const start = messagesContainer.scrollTop;
  const distance = target - start;
  let startTime = null;

  function animation(currentTime) {
    if (startTime === null) startTime = currentTime;
    const elapsedTime = currentTime - startTime;
    const progress = Math.min(elapsedTime / duration, 1);
    const easeProgress = 0.5 - Math.cos(progress * Math.PI) / 2; 

    messagesContainer.scrollTop = start + distance * easeProgress;

    if (progress < 1) {
      window.requestAnimationFrame(animation);
    }
  }

  window.requestAnimationFrame(animation);
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

  if (type === "error") {
    const notification = document.createElement("div");
    notification.classList.add("notification", "error");
    notification.textContent = message;

    document.body.appendChild(notification);

    setTimeout(() => {
      notification.classList.add("fade-out");
      setTimeout(() => {
        notification.remove();
      }, 500);
    }, 5000);
  }

  console.log(`[${type.toUpperCase()}] ${message}`);
}

function disableChat() {
  nameInput.disabled = false;
  connectBtn.disabled = false;
  messageInput.disabled = true;
  sendBtn.disabled = true;
  updateConnectionStatus(false, "Disconnected");
}

window.addEventListener("focus", () => {
  if (!messageInput.disabled) {
    messageInput.focus();
  }
});

let unreadCount = 0;
let originalTitle = document.title;
let windowFocused = true;

window.addEventListener("focus", () => {
  windowFocused = true;
  unreadCount = 0;
  document.title = originalTitle;
});

window.addEventListener("blur", () => {
  windowFocused = false;
});

function updateUnreadCount() {
  if (!windowFocused) {
    unreadCount++;
    document.title = `(${unreadCount}) ${originalTitle}`;
  }
}

const originalOnData = socket && socket.on;
if (socket && originalOnData) {
  socket.on = function (event, callback) {
    if (event === "data") {
      return originalOnData.call(this, event, (data) => {
        updateUnreadCount();
        callback(data);
      });
    }
    return originalOnData.call(this, event, callback);
  };
}
