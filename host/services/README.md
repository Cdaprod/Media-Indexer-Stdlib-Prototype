You’re building an extremely composable, modular Go-based host and camera services platform. Your current “services” directory is a full POC mesh of proxy, capture, device registry, pipeline, and multi-tier middleware. Here’s a summary of what you have, and how it all fits together at a system/DevOps/infra level. After, I’ll answer followups (or deep-dive) into any service you want.

⸻

# /host/services: Service Directory Architecture

## 1. Overview
You have a multi-layered architecture with clear boundaries:
• API Gateway: Handles all client HTTP(S) traffic; mounts backend API and static frontend, proxies APIs, applies middlewares (rate limiting, JWT, cache, etc).
• Camera Proxy: Host-level “device virtualization” proxy that exposes /stream/ endpoints, dynamically discovers camera devices (v4l2/USB), and injects real device info into all device/camera API calls for containerized backends—no mounts needed.
• Capture Daemon: Low-level device scanner/runner using FFmpeg to record/capture media from all discovered video devices, with a device registry and optional runner HTTP API for status/control.
• POC Pipeline Manager: Example for virtual video pipelines using named pipes + v4l2loopback for virtual device emulation and flexible frame routing/processing.
• Proxy: Another top-level HTTP(S) reverse proxy, mostly for “API only” with optional WebSocket pass-through for asset streaming.
• Shared: All reusable middlewares, types, utils for every layer. Each middleware is generic and can be composed into chains.

## 2. Service-by-Service
#### api-gateway/
• cmd/main.go: Main entrypoint. Flag-based config for all critical paths (static, media, DB), JWT secret, cache TTL, rate-limit, API upstream, etc.
• pkg/middleware/chain.go: Classic composable middleware chain builder. Each .Use() wraps the handler; last-added runs first.
• Core feature: Rich middleware stack—system resources, filesystem, logging, rate-limiting, API proxying, JWT auth, cache, CSP, static SPA, media streaming (/stream/), and WebSocket (/ws/).

#### camera-proxy/
• main.go: “Transparent device proxy.” Discovers /dev/video* and USB cameras; exposes JSON APIs for device discovery, creates on-demand MJPEG streams from real devices, proxies and injects device info in backend API/WebSocket responses.
• README.md: Explains deployment: systemd service on host, no changes required to containers, allows legacy apps to “see” camera devices even when running in Docker.
• Deployment: Built as a host daemon (/opt/camera-proxy), runs as its own user, binds to :8000, intercepts /stream/, /api/devices, /ws/, etc. Can be fronted by nginx (or your own proxy).

#### capture-daemon/
• main.go: Polls, scans for devices, runs FFmpeg capture runners for each, exposes registry API on :9000.
• registry/registry.go: Tracks all devices, starts/stops capture goroutines, HTTP API for current device state.
• runner/ffmpeg.go: Loops and records from device to disk, retries on error, timestamped output filenames.
• scanners/: Device scanner modules. csi_or_usb.go, system.go, and v4l2/v4l2.go implement detection for all standard and fallback devices.

#### overlay-hub/
• 
• 
• 
• 

#### shared/
• middleware/: Core chainable middlewares for backend (JWT, API proxy, logging, caching, rate limiting), frontend (static files, SPA, CSP, compression), host (system resource, filesystem, process limits), utils and shared types.

#### ../legacy/poc_video-pipeline-manager/
• main.go: Example "pipeline manager" that builds named pipes and virtual video devices using v4l2loopback. Useful for advanced multi-stage/AI processing, emulating "real" cameras for legacy systems or chaining inference stages.

#### ../legacy/proxy/
• main.go: A top-level HTTP(S) proxy--serves static frontend, compresses, applies API rate limits, routes /api/ and /ws/assets to backend (typically your Python API).

⸻

## How the Flow Works

### End-to-End Flow Example:
	1.	HTTP client (browser/app) → api-gateway:8080
• SPA is served statically.
• API requests (e.g., /api/video/) are proxied to backend containers or to the camera-proxy when appropriate.
• /stream/ or /ws/ handled directly for live video or WebSockets.
	2.	api-gateway proxies “camera”/device APIs or /stream/ routes to camera-proxy:8000.
	3.	camera-proxy
• Discovers real camera devices (v4l2, USB) on the host and exposes them to containers (no device mounts!).
• Streams video via /stream/:device, JSON device list at /api/devices, or upgrades WebSocket for device control.
	4.	capture-daemon
• Independently scans devices, runs FFmpeg capture loops to disk for each, exposes /devices API (on :9000), can be extended for media lifecycle or background ingestion.
	5.	poc_video-pipeline-manager
• Orchestrates named-pipe/virtual device graphs, for routing processed frames to apps that require /dev/videoN devices, for use with inference/ML or other “black box” apps.

⸻

### DevOps & Composability
• Each service can be run as a standalone systemd service or orchestrated in a minimal Compose/Podman/Docker Compose stack. You do not need to bind-mount devices into your containers.
• Infra upgrades, extensions, and migration: Drop-in additional pipeline processors, swap out middlewares, and swap from POC to production ready by extending only the parts you care about.
• Frontends can remain “dumb” and make fetch calls to the same endpoints (the API gateway abstracts everything).

⸻

### How to Extend or Productionize
• Security: Harden JWT, implement real validateJWT, remove dev-mode CORS and CheckOrigin: true.
• Monitoring: Add metrics middleware, Prometheus endpoints, journal logs, proper log rotation.
• Error handling: Ensure all device errors are surfaced through API/WS responses, not just logs.
• Config management: Use .env files, config maps, or secrets for all paths, keys, intervals.
• Add tests: System/integration/e2e tests per service.
• API Versioning: Add /v1/ routes for long-term contract stability.
• Modular runners: Each camera runner could push frames to a message bus (NATS, MQTT, etc.) for plug-and-play downstream processing (AI, cloud upload, etc).

⸻

### Next Steps / Followups
• Do you want a dev/prod docker-compose example that stitches all these together?
• Are you planning to move more device discovery logic into the pipeline manager?
• Interested in how to route “virtual camera” feeds back into containers for AI/inference?
• Want hardening/HA recommendations for the gateway or proxy?
• Want a full flowchart/diagram of how HTTP/camera/media travels through this system?

⸻

Ask for any deep-dive (infra, code, deployment, diagram) and I’ll break it down step by step, with concrete code and systemd/docker/compose as you prefer.