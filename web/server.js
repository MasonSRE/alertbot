import express from 'express';
import path from 'path';
import { fileURLToPath } from 'url';
import http from 'http';

const app = express();
const port = 3000;
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Simple proxy function
const proxyRequest = (target) => (req, res) => {
  const options = {
    hostname: 'alertbot',
    port: 8080,
    path: req.url,
    method: req.method,
    headers: req.headers
  };

  const proxyReq = http.request(options, (proxyRes) => {
    res.writeHead(proxyRes.statusCode, proxyRes.headers);
    proxyRes.pipe(res);
  });

  proxyReq.on('error', (err) => {
    console.error('Proxy error:', err);
    res.status(500).send('Proxy error');
  });

  if (req.method === 'POST' || req.method === 'PUT') {
    req.pipe(proxyReq);
  } else {
    proxyReq.end();
  }
};

// API routes - proxy to backend
app.use('/api', proxyRequest('http://alertbot:8080'));
app.use('/health', proxyRequest('http://alertbot:8080'));
app.use('/metrics', proxyRequest('http://alertbot:8080'));

// Serve static files from dist
app.use(express.static(path.join(__dirname, 'dist')));

// Handle React Router (SPA)
app.get('*', (req, res) => {
  res.sendFile(path.join(__dirname, 'dist', 'index.html'));
});

app.listen(port, '0.0.0.0', () => {
  console.log(`Frontend server running on http://0.0.0.0:${port}`);
});