const { app, BrowserWindow, ipcMain, Menu, Tray } = require("electron");
const path = require("path");
let tray = null;
let mainWindow = null;

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 900,
    height: 600,
    minWidth: 800,
    minHeight: 500,
    webPreferences: {
      nodeIntegration: true,
      contextIsolation: false,
    },
    backgroundColor: "#2c3e50",
    autoHideMenuBar: true,
    icon: path.join(__dirname, "icons", "app_icon.png"),
    frame: true,
    titleBarStyle: "hidden",
    show: false,
  });
  mainWindow.loadFile("index.html");

  mainWindow.once("ready-to-show", () => {
    mainWindow.show();
  });

  createTray();

  mainWindow.on("minimize", (event) => {
    if (process.platform === "darwin") {
      app.dock.hide();
    }
    mainWindow.hide();
    event.preventDefault();
  });

  mainWindow.on("close", (event) => {
    if (!app.isQuitting) {
      event.preventDefault();
      mainWindow.hide();
      if (process.platform === "darwin") {
        app.dock.hide();
      }
      return false;
    }
    return true;
  });
}

function createTray() {
  try {
    const iconPath = path.join(__dirname, "icons", "tray_icon.png");

    tray = new Tray(iconPath);
    const contextMenu = Menu.buildFromTemplate([
      {
        label: "Show neoRoom",
        click: () => {
          mainWindow.show();
          if (process.platform === "darwin") {
            app.dock.show();
          }
        },
      },
      {
        label: "Minimize to Tray",
        click: () => {
          mainWindow.hide();
        },
      },
      { type: "separator" },
      {
        label: "Quit",
        click: () => {
          app.isQuitting = true;
          app.quit();
        },
      },
    ]);

    tray.setToolTip("neoRoom Chat");
    tray.setContextMenu(contextMenu);

    tray.on("click", () => {
      mainWindow.show();
      if (process.platform === "darwin") {
        app.dock.show();
      }
    });
  } catch (error) {
    console.error("Tray creation error:", error);
  }
}

app.whenReady().then(() => {
  createWindow();

  app.on("activate", () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    } else {
      mainWindow.show();
      if (process.platform === "darwin") {
        app.dock.show();
      }
    }
  });

  setupIPC();
});

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") {
    app.quit();
  }
});

function setupIPC() {
  ipcMain.on("show-window", () => {
    mainWindow.show();
    if (process.platform === "darwin") {
      app.dock.show();
    }
  });

  ipcMain.on("hide-window", () => {
    mainWindow.hide();
    if (process.platform === "darwin") {
      app.dock.hide();
    }
  });

  ipcMain.on("app-quit", () => {
    app.isQuitting = true;
    app.quit();
  });

  ipcMain.handle("get-app-version", () => {
    return app.getVersion();
  });

  ipcMain.on("app-ready", (event) => {
    event.reply("main-response", "neoRoom is ready!");
  });
}

process.on("uncaughtException", (error) => {
  console.error("Uncaught Exception:", error);

  if (mainWindow) {
    mainWindow.webContents.send("error-occurred", error.message);
  }
});
