# PVE Pilot

A lightweight Proxmox VE management dashboard with a Go backend and Next.js frontend.

## Features

- **Dashboard** — Node health, resource gauges, cluster overview
- **VM Management** — List, start, stop, reboot QEMU virtual machines
- **Container Management** — List, start, stop, reboot LXC containers
- **Template Cloning** — Clone VM and LXC templates to new instances
- **Storage Overview** — Usage per storage pool
- **Real-time Polling** — Auto-refreshing metrics every 5 seconds
- **Dark Theme** — Terminal-inspired UI

## Quick Start

### 1. Create a Proxmox API Token

In your Proxmox web UI:
1. Go to **Datacenter → Permissions → API Tokens**
2. Click **Add**
3. User: `root@pam` (or your admin user)
4. Token ID: `pilot`
5. Uncheck **Privilege Separation**
6. Copy the token secret (shown only once)

### 2. Configure

```bash
cp .env.example .env
```

Edit `.env` with your Proxmox connection details:
```env
PROXMOX_URL=https://192.168.1.100:8006
PROXMOX_TOKEN_ID=root@pam!pilot
PROXMOX_TOKEN_SECRET=your-token-secret-uuid
```

### 3. Run

```bash
# With Docker (recommended)
make dev

# Or run separately
make backend-dev   # Go backend on :8080
make frontend-dev  # Next.js on :3000
```

Open `http://localhost:3000`

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Browser   │────▶│  Next.js    │────▶│  Go API     │────▶ Proxmox API
│             │◀────│  :3000      │◀────│  :8080      │◀──── :8006
└─────────────┘     └─────────────┘     └─────────────┘
```

- **Go Backend** — Thin proxy to Proxmox REST API with aggregation
- **Next.js Frontend** — Dark-themed dashboard with polling
- **No Database** — Stateless, all data from Proxmox in real-time

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/health` | Health check + Proxmox connectivity |
| GET | `/api/cluster/summary` | Aggregated cluster stats |
| GET | `/api/cluster/resources` | All cluster resources |
| GET | `/api/nodes` | List nodes |
| GET | `/api/nodes/:node/status` | Node details |
| GET | `/api/nodes/:node/vms` | List VMs on node |
| POST | `/api/nodes/:node/vms/:vmid/start` | Start VM |
| POST | `/api/nodes/:node/vms/:vmid/stop` | Stop VM |
| POST | `/api/nodes/:node/vms/:vmid/clone` | Clone VM |
| GET | `/api/nodes/:node/containers` | List containers |
| POST | `/api/nodes/:node/containers/:vmid/start` | Start container |
| POST | `/api/nodes/:node/containers/:vmid/stop` | Stop container |
| POST | `/api/nodes/:node/containers/:vmid/clone` | Clone container |
| GET | `/api/nodes/:node/storage` | Storage pools |
| GET | `/api/templates` | List all templates |

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PROXMOX_URL` | Yes | — | Proxmox API URL |
| `PROXMOX_TOKEN_ID` | Yes | — | API token ID (user@realm!tokenid) |
| `PROXMOX_TOKEN_SECRET` | Yes | — | API token secret (UUID) |
| `INSECURE_TLS` | No | `true` | Skip TLS verification |
| `PORT` | No | `8080` | Backend port |
| `FRONTEND_URL` | No | `http://localhost:3000` | Frontend URL (CORS) |
| `NEXT_PUBLIC_API_URL` | No | `http://localhost:8080` | API URL for frontend |

## Development

```bash
# Backend
cd backend && go run .

# Frontend
cd frontend && npm run dev
```

## License

MIT
