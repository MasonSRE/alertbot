import { createServer } from 'http';
import { readFile } from 'fs/promises';
import { extname, join } from 'path';
import { fileURLToPath } from 'url';
import { dirname } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const port = 3000;

const mimeTypes = {
  '.html': 'text/html',
  '.js': 'application/javascript',
  '.css': 'text/css',
  '.json': 'application/json',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.svg': 'image/svg+xml',
  '.ico': 'image/x-icon'
};

const server = createServer(async (req, res) => {
  let filePath = join(__dirname, 'dist', req.url === '/' ? 'index.html' : req.url);
  
  // For SPA routing, serve index.html for all non-file requests
  if (!extname(filePath) && req.url !== '/') {
    filePath = join(__dirname, 'dist', 'index.html');
  }

  try {
    const data = await readFile(filePath);
    const ext = extname(filePath);
    const contentType = mimeTypes[ext] || 'application/octet-stream';
    
    res.writeHead(200, { 'Content-Type': contentType });
    res.end(data);
  } catch (err) {
    if (err.code === 'ENOENT') {
      // For 404, serve index.html (SPA routing)
      try {
        const indexPath = join(__dirname, 'dist', 'index.html');
        const indexData = await readFile(indexPath);
        res.writeHead(200, { 'Content-Type': 'text/html' });
        res.end(indexData);
      } catch (indexErr) {
        res.writeHead(404, { 'Content-Type': 'text/plain' });
        res.end('File not found');
      }
    } else {
      res.writeHead(500, { 'Content-Type': 'text/plain' });
      res.end('Server error');
    }
  }
});

server.listen(port, '0.0.0.0', () => {
  console.log(`Frontend server running on http://0.0.0.0:${port}`);
});