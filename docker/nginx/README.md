# Nginx Gateway (That DAM Toolbox)

Below is a drop-in, "works-everywhere" gateway package that serves as the front door for the entire That DAM Toolbox stack.

⸻

## What it does

- **Reverse proxy** – Routes requests to video-api (port 8080) and video-web (port 3000)
- **Static serving** – Falls back to Next.js static export if present in `/usr/share/nginx/html`
- **WebSocket support** – Handles `/ws/` upgrades for real-time features
- **Stream optimization** – Unbuffered proxying for `/stream/` endpoints
- **Dual-mode networking** – Works with both `host` and `bridge` network modes
- **Template-driven** – Runtime configuration via environment variables

⸻

## Folder layout

```
docker/nginx/
├── Dockerfile        # nginx:alpine + envsubst + entrypoint
├── nginx.tmpl        # template rendered at startup
├── entrypoint.sh     # renders template + starts nginx
└── README.md         # you are here
```

⸻

## 1 ↦ nginx.tmpl

The main configuration template that gets rendered with environment variables:

```nginx
# Nginx front door for That DAM Toolbox
# Rendered at container start with envsubst → /etc/nginx/nginx.conf

user  nginx;
worker_processes  auto;
error_log  /var/log/nginx/error.log warn;
pid        /var/run/nginx.pid;

events { worker_connections 1024; }

http {
    ##
    ## sensible defaults
    ##
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;
    sendfile      on;
    tcp_nopush    on;
    tcp_nodelay   on;
    keepalive_timeout 65;

    ##
    ## gzip for text assets
    ##
    gzip on; gzip_vary on; gzip_min_length 1000; gzip_comp_level 5;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/rss+xml text/javascript;

    ##
    ## upstream back-ends
    ##   – If the gateway runs with `network_mode: host` these resolve to 127.0.0.1
    ##   – If the gateway runs on a user bridge they resolve to Docker DNS names.
    ##
    upstream video_api { server ${API_HOST:-video-api}:${API_PORT:-8080}; }
    upstream video_web { server ${WEB_HOST:-video-web}:${WEB_PORT:-3000}; }

    server {
        listen 80 default_server;
        server_name _;   # catch-all

        # ------------------ health ------------------------------------------
        location = /health {
            proxy_pass http://video_api/health;
            proxy_set_header Host $host;
        }

        # ------------------ REST / websocket / streams ----------------------
        location /api/       { include proxy_defaults.conf; proxy_pass http://video_api; }
        location /ws/        { include proxy_ws.conf;       proxy_pass http://video_api; }
        location /stream/    { include proxy_nobuf.conf;    proxy_pass http://video_api; }

        # ------------------ static frontend -------------------------------
        location / {
            root /usr/share/nginx/html;              # static export (if present)
            try_files $uri $uri/ @video_web;
        }

        # fall-through → Next.js dev server / SSR
        location @video_web { include proxy_defaults.conf; proxy_pass http://video_web; }

        client_max_body_size 100m;
    }
}
```

### Companion snippets

The entrypoint creates these reusable configuration snippets:

**proxy_defaults.conf**

```nginx
proxy_http_version 1.1;
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection $http_connection;
proxy_set_header Host $host;
proxy_cache_bypass $http_upgrade;
```

**proxy_ws.conf**

```nginx
proxy_http_version 1.1;
proxy_set_header Upgrade $http_upgrade;
proxy_Set_header Connection "Upgrade";
proxy_Set_header Host $host;
```

**proxy_nobuf.conf**

```nginx
proxy_pass_request_headers on;
proxy_buffering off;
proxy_cache off;
```

⸻

## 2 ↦ Dockerfile

```dockerfile
FROM nginx:1.27-alpine

# tiny entrypoint that renders the template with env-vars
RUN apk add --no-cache bash gettext

COPY nginx.tmpl /etc/nginx/nginx.tmpl
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
```

⸻

## 3 ↦ entrypoint.sh

```bash
#!/usr/bin/env bash
set -e

# write snippet files once
cat >/etc/nginx/proxy_defaults.conf <<'EOF'
proxy_http_version 1.1;
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection $http_connection;
proxy_set_header Host $host;
proxy_cache_bypass $http_upgrade;
EOF

cat >/etc/nginx/proxy_ws.conf <<'EOF'
proxy_http_version 1.1;
proxy_set_header Upgrade $http_upgrade;
proxy_Set_header Connection "Upgrade";
proxy_Set_header Host $host;
EOF

cat >/etc/nginx/proxy_nobuf.conf <<'EOF'
proxy_pass_request_headers on;
proxy_buffering off;
proxy_cache off;
EOF

# render the template
envsubst '${API_HOST} ${API_PORT} ${WEB_HOST} ${WEB_PORT}' \
  < /etc/nginx/nginx.tmpl > /etc/nginx/nginx.conf

exec nginx -g 'daemon off;'
```

⸻

## 4 ↦ Docker Compose integration

### Host networking mode (recommended for Pi)

```yaml
services:
  gw:
    build:
      context: docker/nginx        # Dockerfile sits here
    container_name: thatdam-gateway
    network_mode: host             # owns :80 / :443 on the Pi
    restart: unless-stopped
    depends_on:
      video-api: {condition: service_healthy}
      video-web: {condition: service_started}

    environment:
      # host-net → talk to services via 127.0.0.1
      API_HOST: "127.0.0.1"
      API_PORT: "8080"
      WEB_HOST: "127.0.0.1"
      WEB_PORT: "3000"

    volumes:
      # allow you to override nginx.tmpl without rebuilding
      - ./docker/nginx/nginx.tmpl:/etc/nginx/nginx.tmpl:ro
      - ./docker/nginx/html:/usr/share/nginx/html:ro  # optional static export
    
    logging:
      driver: json-file
      options: { max-size: "10m", max-file: "3" }
```

### Bridge networking mode

```yaml
services:
  gw:
    build:
      context: docker/nginx
    container_name: thatdam-gateway
    networks: [damnet]
    ports: ["80:80"]
    restart: unless-stopped
    depends_on:
      video-api: {condition: service_healthy}
      video-web: {condition: service_started}

    # No environment overrides needed - defaults work with Docker DNS
    # API_HOST defaults to "video-api"
    # WEB_HOST defaults to "video-web"

    volumes:
      - ./docker/nginx/nginx.tmpl:/etc/nginx/nginx.tmpl:ro
      - ./docker/nginx/html:/usr/share/nginx/html:ro
```

⸻

## 5 ↦ Environment variables

|Variable  |Default    |Description                             |
|----------|-----------|----------------------------------------|
|`API_HOST`|`video-api`|Hostname/IP for the backend API service |
|`API_PORT`|`8080`     |Port for the backend API service        |
|`WEB_HOST`|`video-web`|Hostname/IP for the frontend web service|
|`WEB_PORT`|`3000`     |Port for the frontend web service       |

**Host networking**: Set hosts to `127.0.0.1` to reach services on localhost  
**Bridge networking**: Use default service names for Docker DNS resolution

⸻

## 6 ↦ Request routing

The gateway routes requests as follows:

- **`/health`** → `video_api/health` (health checks)
- **`/api/*`** → `video_api` (REST API endpoints)
- **`/ws/*`** → `video_api` (WebSocket connections)
- **`/stream/*`** → `video_api` (unbuffered streaming)
- **`/*`** → Static files first, then fallback to `video_web`

### Static + SSR hybrid

1. Try to serve from `/usr/share/nginx/html` (Next.js static export)
1. If not found, proxy to `video_web` (Next.js dev server or SSR)

This allows you to serve a static build for production while falling back to the dev server for dynamic routes or development.

⸻

## 7 ↦ Usage

### Build and start

```bash
docker compose up -d gw
```

### Access the application

- **Local**: http://localhost
- **Network**: http://your-pi.local (via mDNS)
- **Hotspot**: http://thatdamtoolbox.local or http://192.168.42.1

### Update configuration without rebuild

```bash
# Edit the template
vim docker/nginx/nginx.tmpl

# Restart to apply changes
docker compose restart gw
```

### Add static export

```bash
# Build your Next.js app
npm run build

# Copy to nginx html directory
cp -r out/* docker/nginx/html/

# Restart gateway
docker compose restart gw
```

⸻

## What you gained

- **One config, two modes** – host-net or bridge without edits
- **Template driven** – tweak with docker cp + restart, no rebuild
- **Snippets** keep the main file readable and avoid repetition
- **Fail-proof** – missing env-vars fall back to service names
- **Hybrid serving** – static files + SSR fallback
- **WebSocket ready** – proper upgrade handling for real-time features
- **Stream optimized** – unbuffered proxying for video/media

Drop those files in `docker/nginx/`, rebuild the gateway, and you’re ready:

```bash
docker compose up -d gw
xdg-open http://<pi-host>.local        # mDNS or hotspot DNS
```

Happy proxying 🚀