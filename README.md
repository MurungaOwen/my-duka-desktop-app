# MyDuka (Inventory Desktop)

MyDuka is a local-first desktop POS and inventory system for Kenyan shops.

It is designed for small retail businesses with 1-5 active cashiers, where owners need:
- Clear stock visibility
- Daily money reconciliation
- Fast checkout without downtime

## Current Status

This repository currently contains a Wails + React starter scaffold.

The product architecture and scope are defined in [ARCHITECTURE.md](/home/hood/Desktop/inventory-desktop/ARCHITECTURE.md). Implementation is in early stage.

## Product Goals

- Work reliably with unstable internet
- Support multi-device shop-floor operations over local WiFi
- Reconcile cash and M-Pesa payments in one system
- Keep cashier workflows fast and simple
- Give owners immediate daily/weekly operational insight

## In Scope (v1)

- Desktop app (Windows-first) for cashier and admin roles
- Local SQLite storage on every device
- Standalone mode for one laptop/PC (no LAN sync dependency)
- Single in-shop server device with LAN sync for all clients
- POS checkout: cash, M-Pesa (internet), card (internet)
- Inventory tracking using stock transactions (event-based)
- Reports, CSV/PDF/Excel exports, supplier and purchase order basics

## Out of Scope (v1)

- Cloud multi-branch sync
- Web dashboard
- Loyalty, payroll, accounting integrations
- E-commerce integrations

## Proposed Stack

- Desktop shell: Wails v2
- Backend: Go
- Frontend: React + TypeScript + Tailwind
- Database: SQLite (WAL mode)
- Sync model: local-first with periodic push/pull

Full technical decisions are documented in [ARCHITECTURE.md](/home/hood/Desktop/inventory-desktop/ARCHITECTURE.md).

## Deployment Modes

- `Standalone (1 device)`: one laptop/PC runs everything locally; no LAN setup required.
- `LAN Sync (multi-device)`: one server device plus one or more cashier/admin clients on shop WiFi.

## Development

### Prerequisites

- Go 1.23+
- Node.js 18+
- Wails CLI v2

### Run in development

```bash
wails dev
```

### Build desktop package

```bash
wails build
```

### Run backend tests

```bash
go test ./...
```

### Seed demo data

```bash
go run ./cmd/seed
```

Optional:

```bash
go run ./cmd/seed -db /absolute/path/to/myduka.db -mode standalone
```

### Backend runtime config (optional)

- `MYDUKA_MODE=standalone|lan_sync` (default: `standalone`)
- `MYDUKA_DB_PATH=/absolute/path/to/myduka.db` (default: OS user config dir)
- `MYDUKA_SYNC_BASE_URL=http://myduka.local:8080` (used in `lan_sync`)
- `MYDUKA_SYNC_INTERVAL_SECONDS=5` (used in `lan_sync`)
- `MYDUKA_SYNC_BATCH_LIMIT=200` (used in `lan_sync`)

## Repository Layout

- `main.go`, `app.go`: Wails app bootstrap and Go bindings
- `internal/backend/facade.go`: stable backend API exposed to app layer
- `internal/backend/store/`: SQLite service, migrations, business operations, sync core
- `internal/backend/syncapi/`: HTTP handlers for `/sync/push` and `/sync/pull`
- `internal/backend/syncengine/`: background push/pull worker with retry backoff
- `internal/backend/*_test.go`: backend integration and feature tests
- `frontend/`: React application
- `build/`: packaging assets and platform-specific build config
- `ARCHITECTURE.md`: system design and implementation plan

## Next Milestones

1. Implement Phase 1 single-device POS and admin workflows.
2. Add M-Pesa/card payment flows with robust fallback handling.
3. Add LAN server process, discovery, and offline sync reconciliation.
