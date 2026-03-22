# TODO

This file tracks remaining implementation work after the current integrated baseline.

## Already Implemented (Snapshot)

- Unified SQLite migrations and project-local DB default (`./myduka.sqlite`)
- Unified seeding (settings, staff, categories, suppliers, products, purchase orders, sales)
- Username/password staff auth
- POS cash checkout
- M-Pesa via Paystack:
  - initiate charge
  - verify charge
  - auto polling in POS
  - customer-already-paid flow (recent payments + manual reference)
  - one-time payment reference binding (`sale_payments`)
- Dashboard summary + chart cards
- Core backend tests for auth/sales/seed/sync/payments

## P0 - Must Finish Before Production

1. Payment completion hardening
- Add explicit persisted payment status timeline (`initiated`, `pending`, `verified`, `failed`, `timeout`, `manual_verified`).
- Store more provider metadata on `sale_payments` (masked phone, provider tx id, verified_at, raw gateway response hash).
- Add strict business validations:
  - currency must be `KES`
  - amount must equal sale total
  - stale payment rejection window (e.g. >15 min unless manager override)
- Acceptance:
  - no payment can finalize sale without deterministic verification path
  - failed/pending/timeout states are auditable

2. Card payment provider path
- Implement real card provider adapter interface and first provider integration.
- Add online/offline UX states and retry guidance.
- Acceptance:
  - card payments have parity with M-Pesa traceability

3. Sync auth + device trust
- Enforce sync auth (device token/HMAC) for `/sync/push` and `/sync/pull`.
- Enforce approved device list at sync boundary.
- Acceptance:
  - untrusted device cannot push/pull

4. Backup + restore operations
- Scheduled backup job (hourly + daily retention)
- Restore command + verification flow
- Startup DB integrity check and corruption handling
- Acceptance:
  - restore drill succeeds from latest backup

## P1 - Complete v1 Business Surface

1. Supplier and purchase order workflows
- Supplier CRUD UI + backend methods
- PO create/list/receive lifecycle
- Receiving PO writes stock transactions and updates derived stock

2. Reports + exports
- Daily, date-range, staff performance, stock valuation, slow movers
- CSV/PDF/Excel exports with test coverage

3. Sales controls
- Void/refund (admin authorized)
- Compensating payment + stock transactions
- Immutable audit trail

4. Settings and payment operations UI
- Paystack credential health check from settings
- Payment diagnostics screen (recent failures, verify errors, duplicates blocked)

## P2 - Reliability and Scale

1. Test expansion
- Concurrent cashier checkout on shared products
- Duplicate reference race tests across tills
- Sync conflict and tombstone propagation edge cases
- Payload fuzz tests for sync and payment parsing

2. Performance tuning
- Query plan based indexing for reports/payment lookups/sync pulls
- Benchmark sync apply and stock recompute under realistic load

3. Security hardening
- PIN/password retry lockout and cooldown policy
- Structured privileged action audit logs
- Optional at-rest encryption for sensitive settings

4. Migration policy
- Forward/backward compatibility checks
- Versioned migration release notes + rollback playbook

## P3 - Nice to Have

1. Worker lifecycle manager
- Unified scheduler for sync/backup/report jobs
- Suspend/resume safe handling

2. API ergonomics
- Generated typed API client checks in CI
- OpenAPI-like schema for sync HTTP endpoints

## Current Known Gaps (Short)

- Card gateway integration is pending.
- Sync device approval/auth is not enforced yet.
- Backup/restore automation is pending.
- Supplier/PO/reporting flows are partial.
- Payment metadata/audit depth can be improved for production compliance.
