import socket
import threading
import tkinter as tk
from tkinter import scrolledtext, ttk
import sys

class ChatClientGUI:
    def __init__(self, root):
        self.root = root
        self.root.title("Chat Room")
        self.root.geometry("600x500")
        self.root.configure(bg="#f0f0f0")
        
        # Server connection details
        self.serverAddr = "127.0.0.1"
        self.serverPort = 6969
        self.bufferSize = 1024
        self.clientSocket = None
        self.connected = False
        
        # Create GUI elements
        self.create_widgets()
        
    def create_widgets(self):
        # Main frame
        main_frame = ttk.Frame(self.root)
        main_frame.pack(padx=10, pady=10, fill=tk.BOTH, expand=True)
        
        # Style configuration
        style = ttk.Style()
        style.configure('TFrame', background='#f0f0f0')
        style.configure('TButton', background='#4a7abc', foreground='white', font=('Arial', 10))
        style.configure('TLabel', background='#f0f0f0', font=('Arial', 11))
        
        # Name entry section
        name_frame = ttk.Frame(main_frame)
        name_frame.pack(fill=tk.X, pady=(0, 10))
        
        ttk.Label(name_frame, text="Your Name:").pack(side=tk.LEFT, padx=(0, 5))
        
        self.name_entry = ttk.Entry(name_frame, width=30)
        self.name_entry.pack(side=tk.LEFT, padx=(0, 5))
        self.name_entry.focus()
        
        self.connect_button = ttk.Button(name_frame, text="Connect", command=self.connect_to_server)
        self.connect_button.pack(side=tk.LEFT)
        
        # Chat display area
        chat_frame = ttk.Frame(main_frame)
        chat_frame.pack(fill=tk.BOTH, expand=True, pady=(0, 10))
        
        # Message display area
        self.message_display = scrolledtext.ScrolledText(chat_frame, wrap=tk.WORD, height=15, bg="#ffffff", font=('Arial', 10))
        self.message_display.pack(fill=tk.BOTH, expand=True)
        self.message_display.config(state=tk.DISABLED)
        
        # Message input area
        input_frame = ttk.Frame(main_frame)
        input_frame.pack(fill=tk.X)
        
        self.message_entry = ttk.Entry(input_frame, width=50)
        self.message_entry.pack(side=tk.LEFT, padx=(0, 5), fill=tk.X, expand=True)
        self.message_entry.bind('<Return>', self.send_message)
        self.message_entry.config(state=tk.DISABLED)
        
        self.send_button = ttk.Button(input_frame, text="Send", command=self.send_message)
        self.send_button.pack(side=tk.RIGHT)
        self.send_button.config(state=tk.DISABLED)
        
        # Status bar
        self.status_var = tk.StringVar()
        self.status_var.set("Not connected")
        status_bar = ttk.Label(self.root, textvariable=self.status_var, relief=tk.SUNKEN, anchor=tk.W)
        status_bar.pack(side=tk.BOTTOM, fill=tk.X)
    
    def connect_to_server(self):
        name = self.name_entry.get().strip()
        
        # Name validation
        if name == "":
            self.update_status("Empty name is not allowed!")
            return
        elif name == "SERVER":
            self.update_status("The name SERVER is not allowed!")
            return
        
        try:
            # Create socket and connect
            self.clientSocket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            self.clientSocket.connect((self.serverAddr, self.serverPort))
            
            # Send name to server
            self.clientSocket.sendall(name.encode(encoding="utf-8"))
            
            # Update UI
            self.connected = True
            self.update_status(f"Connected as {name}")
            self.name_entry.config(state=tk.DISABLED)
            self.connect_button.config(state=tk.DISABLED)
            self.message_entry.config(state=tk.NORMAL)
            self.send_button.config(state=tk.NORMAL)
            
            # Start receiving messages
            receive_thread = threading.Thread(target=self.receive_messages)
            receive_thread.daemon = True
            receive_thread.start()
            
        except Exception as e:
            self.update_status(f"Connection error: {e}")
    
    def receive_messages(self):
        try:
            while self.connected:
                message = self.clientSocket.recv(self.bufferSize)
                if message:
                    decoded_message = message.decode(encoding="utf-8")
                    self.display_message(decoded_message)
        except Exception as e:
            if self.connected:
                self.update_status(f"Connection error: {e}")
                self.disconnect()
    
    def send_message(self, event=None):
        if not self.connected:
            return
        
        message = self.message_entry.get().strip()
        if message:
            try:
                self.clientSocket.sendall(message.encode(encoding="utf-8"))
                self.message_entry.delete(0, tk.END)
            except Exception as e:
                self.update_status(f"Failed to send message: {e}")
    
    def display_message(self, message):
        self.message_display.config(state=tk.NORMAL)
        self.message_display.insert(tk.END, message)
        self.message_display.see(tk.END)  # Auto-scroll to the bottom
        self.message_display.config(state=tk.DISABLED)
    
    def update_status(self, status_message):
        self.status_var.set(status_message)
        print(status_message)  # Also print to console for debugging
    
    def disconnect(self):
        if self.connected:
            self.connected = False
            if self.clientSocket:
                try:
                    self.clientSocket.close()
                except:
                    pass
            self.update_status("Disconnected from server")
            self.message_entry.config(state=tk.DISABLED)
            self.send_button.config(state=tk.DISABLED)
    
    def on_closing(self):
        self.disconnect()
        self.root.destroy()
        sys.exit(0)

if __name__ == "__main__":
    root = tk.Tk()
    app = ChatClientGUI(root)
    root.protocol("WM_DELETE_WINDOW", app.on_closing)
    root.mainloop()