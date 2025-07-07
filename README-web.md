# Postie Web Version

This README covers the web version of Postie, which compiles the frontend as a static web app and serves it via HTTP instead of the desktop GUI.

## Quick Start with Docker

### Using Docker Compose

1. **Clone the repository:**

   ```bash
   git clone https://github.com/javi11/postie.git
   cd postie
   ```

2. **Start the web version:**

   ```bash
   docker-compose -f docker-compose.web.yml up -d
   ```

3. **Access the web interface:**
   Open your browser and navigate to `http://localhost:8080`

### Using Docker directly

```bash
# Build the image
docker build -f docker/Dockerfile.web -t postie-web .

# Run the container
docker run -d \
  --name postie-web \
  -p 8080:8080 \
  -v ./config:/config \
  -v ./watch:/watch \
  -v ./output:/output \
  postie-web
```

## Building from Source

### Prerequisites

- Go 1.24+
- Node.js 20+
- Bun (for frontend dependencies)

### Build Steps

1. **Install frontend dependencies:**

   ```bash
   cd frontend
   bun install
   ```

2. **Build the frontend:**

   ```bash
   bun run build
   ```

3. **Enable the embed directive:**

   ```bash
   sed -i 's|// //go:embed all:frontend/build|//go:embed all:frontend/build|' cmd/web/main.go
   ```

4. **Build the web server:**

   ```bash
   go mod download
   CGO_ENABLED=1 go build -o postie-web cmd/web/main.go
   ```

5. **Run the web server:**
   ```bash
   ./postie-web
   ```

## Configuration

### Environment Variables

- `PORT`: Server port (default: 8080)
- `HOST`: Server host (default: 0.0.0.0)
- `PUID`: User ID for file permissions (default: 1000)
- `PGID`: Group ID for file permissions (default: 1000)

### Volume Mounts

- `/config`: Configuration files
- `/watch`: Directory to watch for new files
- `/output`: Directory for processed output files

### Configuration File

Create a `config.yaml` file in the `/config` volume:

```yaml
# Example configuration
servers:
  - name: "Main Server"
    host: "news.example.com"
    port: 119
    username: "your-username"
    password: "your-password"
    ssl: false
    connections: 10

posting:
  from: "your-email@example.com"
  groups: ["alt.binaries.test"]
  subject_prefix: "[Postie]"

par2:
  enabled: true
  redundancy: 10
  block_size: 2000

compression:
  enabled: true
  level: 6
```

## API Endpoints

The web version exposes a REST API with the following endpoints:

### Status and Configuration

- `GET /api/status` - Get application status
- `GET /api/config` - Get current configuration
- `POST /api/config` - Update configuration

### Queue Management

- `GET /api/queue` - Get queue items
- `POST /api/queue/{id}/retry` - Retry a failed job
- `DELETE /api/queue/{id}/cancel` - Cancel a job

### Processing

- `GET /api/processor/status` - Get processor status
- `GET /api/running-jobs` - Get currently running jobs

### Logs and Upload

- `GET /api/logs?limit=100&offset=0` - Get paginated logs
- `POST /api/upload` - Upload files for processing
- `WS /api/ws` - WebSocket for real-time updates

## Differences from Desktop Version

### Architecture Changes

- **Frontend**: Compiled as static web app instead of embedded in desktop app
- **Backend**: HTTP server instead of Wails desktop runtime
- **Communication**: REST API + WebSocket instead of Wails bindings
- **Deployment**: Docker containers instead of desktop executables

### Feature Parity

The web version maintains full feature parity with the desktop version:

- ✅ File upload and processing
- ✅ Queue management
- ✅ Configuration management
- ✅ Real-time updates
- ✅ Log viewing
- ✅ Progress tracking

### Benefits

- **Accessibility**: Access from any device with a web browser
- **Deployment**: Easy deployment with Docker
- **Scalability**: Can be deployed on servers/cloud platforms
- **Multi-user**: Multiple users can access the same instance
- **Mobile**: Works on mobile devices

## Security Considerations

### Production Deployment

For production use, consider:

1. **Reverse Proxy**: Use nginx or similar with SSL termination
2. **Authentication**: Add authentication layer (OAuth, basic auth, etc.)
3. **Firewall**: Restrict access to trusted networks
4. **HTTPS**: Always use HTTPS in production
5. **File Permissions**: Ensure proper file system permissions

### Example nginx Configuration

```nginx
server {
    listen 443 ssl;
    server_name postie.yourdomain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /api/ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## Troubleshooting

### Common Issues

1. **Frontend not loading**: Ensure the frontend was built and embedded properly
2. **WebSocket connection failed**: Check firewall and proxy configuration
3. **Permission denied**: Verify PUID/PGID settings match your system
4. **Config not loading**: Check volume mounts and file permissions

### Debug Mode

Set environment variable for debugging:

```bash
export LOG_LEVEL=debug
```

### Health Check

The container includes a health check that can be monitored:

```bash
docker inspect --format='{{.State.Health.Status}}' postie-web
```

## Contributing

See the main [CONTRIBUTING.md](CONTRIBUTING.md) for general contribution guidelines.

### Web-specific Development

1. **Frontend development**: Use `bun run dev` in the frontend directory
2. **Backend development**: Run `go run cmd/web/main.go` after building frontend
3. **Docker development**: Use `docker-compose -f docker-compose.web.yml up --build`

## License

See [LICENSE](LICENSE) for details.
