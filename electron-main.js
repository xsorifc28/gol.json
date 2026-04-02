import { app, BrowserWindow, session } from 'electron';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

function createWindow() {
  const win = new BrowserWindow({
    width: 1200,
    height: 800,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      webSecurity: false, // Required to bypass CORS for direct data fetching
    },
    backgroundColor: '#000000',
    title: 'gol.json',
  });

  // Handle headers to mimic official Sofascore app and avoid detection
  session.defaultSession.webRequest.onBeforeSendHeaders((details, callback) => {
    const { requestHeaders } = details;
    const url = new URL(details.url);

    if (url.hostname.includes('sofascore')) {
      requestHeaders['X-Requested-With'] = 'XMLHttpRequest';
      requestHeaders['User-Agent'] = 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';
    }

    callback({ requestHeaders });
  });

  if (process.env.VITE_DEV_SERVER_URL) {
    win.loadURL(process.env.VITE_DEV_SERVER_URL).catch(() => {
      setTimeout(() => {
        win.loadURL(process.env.VITE_DEV_SERVER_URL);
      }, 1000);
    });
  } else {
    win.loadFile(path.join(__dirname, 'docs/index.html'));
  }
}

app.whenReady().then(createWindow);

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('activate', () => {
  if (BrowserWindow.getAllWindows().length === 0) {
    createWindow();
  }
});
