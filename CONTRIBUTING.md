# Contributing to PVE Pilot

Thanks for your interest in contributing! This guide covers the development workflow, code style, and how to submit changes.

## Getting Started

1. **Fork** the repository
2. **Clone** your fork:
   ```bash
   git clone https://github.com/<your-username>/pve-pilot.git
   cd pve-pilot
   ```
3. **Set up** the development environment:
   ```bash
   cp .env.example .env
   # Edit .env with your Proxmox connection details
   docker compose build
   docker compose up -d
   ```
4. **Create a branch** for your feature:
   ```bash
   git checkout -b feat/your-feature-name
   ```

## Project Structure

```
pve-pilot/
├── backend/                    # Go REST API + NATS worker
│   ├── config/                # Environment config
│   ├── handlers/              # HTTP handlers (Gin)
│   ├── jobs/                  # NATS job store, types, worker
│   ├── middleware/            # CORS
│   ├── proxmox/               # Proxmox API client
│   └── main.go
├── frontend/                  # Next.js 16 app
│   └── src/
│       ├── app/              # Pages (App Router)
│       ├── components/       # React components
│       ├── contexts/         # React contexts (JobsContext)
│       ├── hooks/            # Custom hooks (usePolling)
│       └── lib/              # API client, types, utilities
└── docker-compose.yml
```

## Development Workflow

### Backend (Go)

```bash
cd backend
go run .          # Run locally (needs NATS and .env)
go vet ./...      # Lint
go build ./...    # Build check
```

- Handlers go in `handlers/` — one file per domain (vms.go, containers.go, backup.go)
- Proxmox API methods go in `proxmox/` — one file per domain (vms.go, containers.go, backup.go, storage.go)
- Types go in `proxmox/types.go`
- Routes are registered in `main.go`

### Frontend (Next.js / TypeScript)

```bash
cd frontend
npm run dev       # Dev server on :3000
npx tsc --noEmit  # Type check
```

- Pages use the App Router (`src/app/`)
- Shared components live in `src/components/shared/`
- Type definitions in `src/lib/types.ts`
- API helpers in `src/lib/api.ts`

### Docker

```bash
docker compose build        # Build all images
docker compose up -d        # Start NATS + backend + frontend
docker compose logs backend # Check logs
docker compose ps           # Verify health
```

## Code Style

### Go
- Standard Go formatting (`gofmt`)
- Use `go vet ./...` before committing
- Error messages: lowercase, no trailing period
- Handler pattern: parse params → validate → call Proxmox client → return JSON
- Async Proxmox operations return UPIDs — always use `WaitForTask()` to track completion

### TypeScript / React
- Functional components only
- `"use client"` directive on interactive components
- Tailwind CSS for styling (dark theme: `#0a0a0a` bg, `#161616` cards, `#00ff88` accent, `#222222` borders)
- Types in `lib/types.ts`, not inline
- Modal pattern: confirm phase &rarr; working phase &rarr; done/failed phase (see `ScaleModal.tsx` for reference)

### Commits
- Use descriptive commit messages (imperative mood):
  - `Add backup and restore features`
  - `Fix SSH key encoding for cloud-init`
  - `Update container detail page layout`
- One logical change per commit
- Reference issues if applicable

## Submitting Changes

1. **Ensure your code compiles**:
   ```bash
   cd backend && go vet ./...
   cd frontend && npx tsc --noEmit
   ```

2. **Build Docker images** to verify:
   ```bash
   docker compose build
   ```

3. **Push** your branch and open a **Pull Request** against `main`

4. **PR requirements**:
   - Backend builds without errors (`go vet ./...`)
   - Frontend type-checks (`npx tsc --noEmit`)
   - Docker images build successfully
   - Clear description of what changed and why

## Adding a New Feature

### New Proxmox API endpoint
1. Add the client method in `backend/proxmox/<domain>.go`
2. Add any new types to `backend/proxmox/types.go`
3. Add the handler in `backend/handlers/<domain>.go`
4. Register the route in `backend/main.go`
5. Add the frontend type in `frontend/src/lib/types.ts`

### New modal / UI component
1. Create the component in `frontend/src/components/shared/`
2. Follow the modal pattern: props with `isOpen`, `onClose`, `onSuccess`
3. Use phases: `confirm` &rarr; `working` &rarr; `done` / `failed`
4. Wire into the detail page with a state variable and button

### New LXC template
1. Create install script at `ansible/roles/proxmox_lxc_templates/files/install_scripts/<name>.sh`
2. Add entry to `ansible/roles/proxmox_lxc_templates/defaults/main.yml`
3. If Docker-based: add compose file at `ansible/roles/proxmox_lxc_templates/files/compose/<name>.yml`
4. If custom systemd: add unit at `ansible/roles/proxmox_lxc_templates/templates/systemd/<name>.service.j2`
5. Test by building the single template:
   ```bash
   cd ansible
   ansible-playbook playbooks/proxmox.yml --tags lxc-templates \
     --extra-vars '{"proxmox_lxc_templates": [<your template entry>]}'
   ```

## Key Patterns to Know

- **Proxmox UPIDs**: Clone, start, stop, backup all return async task IDs. Always `WaitForTask(upid, timeout)`.
- **LXC vs QEMU**: Fundamentally different code paths. LXC uses `pct exec` via SSH; QEMU uses guest agent. LXC hot-scales; VMs need restart.
- **Storage filtering**: Use `rootdir` content type for LXC, `images` for QEMU. Always exclude `local` storage from user selection.
- **SSH key encoding**: `url.QueryEscape` then replace `+` with `%20` for Proxmox cloud-init.
- **Guest agent exec**: JSON body only (`{"command": "bash", "input-data": "script\n"}`). Form-encoded returns 596.

## Questions?

Open an issue or start a discussion on GitHub.
