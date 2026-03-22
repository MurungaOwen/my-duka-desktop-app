# MyDuka Architecture

## 1. Purpose

MyDuka is a local-first POS and inventory desktop application for Kenyan retail shops with shared in-store operations.

This document defines:
- System boundaries
- Core domain model
- Sync and conflict behavior
- Payment and offline behavior
- Delivery phases and risk controls

## 2. Product Context

### Primary users

- Cashier: high-speed checkout only
- Admin/Owner: operations visibility and control

### Core problems solved

- Unknown stock levels and shrinkage
- Weak daily reconciliation of sales/money
- Multi-cashier stock inconsistency
- Fragmented M-Pesa vs cash records
- Lack of historical reporting

## 3. Non-Goals (v1)

- Multi-branch cloud sync
- Browser/web-only admin panel
- Loyalty programs and advanced CRM
- Accounting platform integrations
- Restaurant/delivery-specific workflows

## 4. High-Level System

MyDuka supports two deployment modes in v1.

### Mode A: Standalone (single device)

- One laptop/PC runs the app only.
- One local SQLite database is authoritative.
- No LAN sync required.
- Works fully for POS, inventory, and reports without internet.
- Internet is required for live M-Pesa STK calls and card authorization.

### Mode B: LAN Sync (multi-device)

- One shop device is configured as server.
- Other devices run as clients.
- Server device:
  - Runs MyDuka app
  - Runs background MyDuka server process
  - Hosts authoritative SQLite database
- Client device:
  - Runs MyDuka app
  - Holds local SQLite cache + unsynced writes
  - Pushes/Pulls changes to/from server every 5s when reachable

If server is unreachable, clients keep selling offline and sync later.

## 5. Deployment Topology

- Standalone topology:
  - Single Windows laptop/PC in the shop
  - No dependency on LAN connectivity
- Multi-device topology:
  - Local WiFi LAN required for sync
  - Internet required for M-Pesa and card authorization
- Service discovery:
  - Primary: mDNS (`myduka.local:8080`)
  - Fallback: manual server address entry
  - Onboarding: join code (6 chars, 10-minute TTL) and admin approval
- Platform target:
  - Primary: Windows
  - Secondary: macOS/Linux

## 6. Architecture Principles

1. Local-first: POS must operate when internet is down.
2. Event-based inventory: stock is derived from transactions, not mutable counters.
3. Safe merge model: UUID identity and deterministic conflict rules.
4. Operational clarity: sync state visible but non-blocking to cashier flow.
5. Simple setup: owner should complete first-run setup within 30 minutes.

## 7. Core Components

### Desktop app (Wails)

- UI shell, route/role handling, local state
- POS workflows, admin workflows, reporting
- Local persistence and sync scheduler

### Local data layer (SQLite)

- Domain tables (products, sales, stock transactions, staff, etc.)
- Sync metadata (`device_id`, `synced_at`, tombstones)
- WAL mode enabled for reliability and read concurrency

### LAN server process (Go net/http)

- Device registration and approval endpoints
- Sync push ingestion and pull query endpoints
- Authoritative write path and conflict resolution

### Integrations

- Paystack API (`/charge` + verify polling)
- Export engines (CSV/PDF/Excel)
- Receipt printing (OS print path in v1; thermal-specific tuning later)

## 8. Data Model

All syncable records include:
- `id` (UUID text)
- `created_at`
- `updated_at`
- `deleted_at` (soft delete tombstone)
- `device_id`
- `synced_at` (null when local-only/not yet pushed)

### Tables

- `device_info`: local device identity + sync cursor
- `settings`: business/system configuration
- `categories`: product taxonomy
- `products`: catalog + cached computed `stock_qty`
- `stock_transactions`: immutable stock movement ledger (source of truth)
- `suppliers`: supplier records
- `purchase_orders`: ordering lifecycle
- `staff`: role + PIN hash
- `sales`: transaction header
- `sale_items`: line items
- `sync_log`: outbound pending change queue

## 9. Inventory Consistency Model

`products.stock_qty` is derived, not authoritative.

Authoritative stock formula:
- `current_stock = starting_stock + SUM(stock_transactions.qty_change)`

Why:
- Supports concurrent/offline writes without lost updates
- Avoids last-write-wins corruption on mutable counters

Recompute trigger:
- After applying pulled or local `stock_transactions`, affected product stocks are recomputed.

## 10. Sync Protocol

### Push cycle (every 5s when online)

1. Read pending changes from `sync_log` in small batches (e.g., 200).
2. POST payload to server (`device_id`, records).
3. On success, mark sent records `synced_at`.
4. On failure, backoff: 5s -> 15s -> 30s -> 60s (then hold/error state).

### Pull cycle

1. After successful push, call `GET /sync/pull?since=<last_sync_at>&device_id=<id>`.
2. Apply records from other devices.
3. Update `last_sync_at`.

### Standalone behavior

- `sync_log` is still used as a local append-only audit queue.
- Network push/pull jobs are disabled.
- All reads and writes are local and immediate.

### Conflict rules

- `stock_qty`: never merged directly; always recomputed from ledger.
- `sales`/`sale_items`: append-only by UUID identity.
- Metadata entities (`products`, `categories`, `staff`): last `updated_at` wins.
- `settings`: server version wins.
- Soft delete tombstone beats any non-delete update.

## 11. Payments and M-Pesa

### Supported payment methods

- Cash
- M-Pesa (Paystack charge + polling)
- Card (internet-required terminal authorization flow)

### M-Pesa flow (v1)

1. Cashier selects M-Pesa and enters customer phone.
2. App sends Paystack charge request (mobile money provider `mpesa`).
3. App polls verification every ~3s up to timeout (~2 minutes).
4. On success, sale finalizes and receipt is issued.
5. Payment reference is bound to the sale and cannot be reused.

### Customer-already-paid flow (v1)

1. Cashier opens "Verify existing payment".
2. App lists recent successful payments in a short time window.
3. Cashier selects one payment or enters reference manually.
4. Backend verifies payment and enforces exact amount match.
5. Sale commits only if reference is unused and verification passes.

### Offline and degraded cases

- Internet unavailable at start:
  - M-Pesa shown as unavailable
  - Card shown as unavailable
  - Cash remains available
  - Offer manual paybill flow with code entry verification
- Mid-transaction disconnect:
  - Allow manual verification path by reference/recent payments
  - Enforce amount match and one-time reference usage
  - Card transaction remains pending/failed until network is restored and must be retried

## 12. Security Model

- Role-based access:
  - Cashier: POS-only operations
  - Admin: full management and reports
- Payment actions require staff PIN
- Store PIN as bcrypt hash only
- New devices require admin approval
- Soft-delete and immutable stock ledger preserve auditability

## 13. UX and Reliability Requirements

- Cashier happy path should process simple sales in under ~10 seconds.
- Errors must be action-oriented (what to do next).
- Sync indicator states:
  - Green: synced
  - Orange: pending
  - Red: offline/error
- Sync failures should not interrupt active checkout unless data safety is at risk.

## 14. Delivery Plan

### Phase 1: Single-device core

- POS cash flow, admin core modules, local DB schema, basic reports

### Phase 2: Payments

- M-Pesa STK + polling, manual fallback, card confirmation, receipt flow

### Phase 3: Multi-device LAN sync

- Server process, discovery, join/approval, offline queueing, conflict apply

### Phase 4: Operational polish

- Report exports, suppliers/POs, CSV imports, print path hardening

## 15. Major Risks and Mitigations

- Server laptop leaves premises:
  - Mitigation: setup flow must explicitly choose an always-on server device.
- mDNS blocked on some routers:
  - Mitigation: manual server entry + join code onboarding.
- M-Pesa connectivity instability:
  - Mitigation: clear degraded-mode UX and manual confirmation audit trail.
- SQLite write contention under burst load:
  - Mitigation: WAL mode, short transactions, batched sync writes.

## 16. Implementation Note (Current Repo)

Core implementation is active, not just scaffold. Current repo includes:
- Unified SQLite migrations
- Project-root DB default (`./myduka.sqlite`)
- Unified demo seeding (settings, staff, categories, suppliers, products, purchase orders, sales)
- Paystack-backed M-Pesa flow with polling + existing-payment verification path
- POS/admin UI with dashboard charts and staff management
