# Backend TODO

This file tracks what is still pending on backend after current implementation.

## P0 - Must Complete Before Production

1. Payments integration (real)
- Implement M-Pesa Daraja service (STK push, status polling, manual confirmation audit).
- Implement card authorization integration interface (internet-required path).
- Persist payment reference fields on `sales` (transaction code, provider status, manual-confirm metadata).
- Acceptance:
  - successful/failed/timeout payment states are recorded and test-covered
  - cashier fallback path is deterministic for no-internet scenarios

2. Sync protocol hardening
- Add auth/signing for `/sync/push` and `/sync/pull` (device token or HMAC).
- Add strict payload validation and size limits.
- Add per-table conflict rules for `updated_at`/soft-delete precedence in mutation apply.
- Acceptance:
  - unauthorized sync requests are rejected
  - malformed mutation batches are rejected safely
  - conflict behavior is deterministic and documented

3. Device registration and approval
- Implement join-code generation/expiry + admin approval workflow.
- Persist approved device registry and revoke flow.
- Acceptance:
  - unapproved device cannot sync
  - revoke immediately blocks push/pull

4. Data safety and backup
- Add scheduled local backups (e.g., hourly + daily rotation).
- Add restore command and restore verification.
- Add DB integrity check on startup and backup job.
- Acceptance:
  - restore drill from backup succeeds on clean machine
  - documented RTO/RPO target is met

## P1 - Needed For Full v1 Feature Scope

1. Supplier + purchase order domain logic
- CRUD endpoints/methods for suppliers.
- Purchase order create/list/receive flow.
- Stock updates on PO receive.
- Acceptance:
  - receiving PO writes stock transactions and recomputes stock correctly

2. Reporting and exports
- Implement report services:
  - daily summary
  - date-range sales
  - staff performance
  - stock valuation
  - slow movers
- Implement CSV/PDF/Excel generation services and backend methods.
- Acceptance:
  - exports open correctly and values match DB totals

3. Sales controls
- Add void/refund workflow (admin-authorized).
- Write compensating stock/payment transactions and audit trail.
- Acceptance:
  - void/refund preserves immutable history and consistent stock totals

4. Better sync observability
- Add sync metrics table (last run, latency, error counters).
- Add diagnostics endpoint/method for admin support screen.
- Acceptance:
  - can identify why a device is behind from backend data alone

## P2 - Hardening and Scale

1. Test coverage expansion
- Add tests for:
  - concurrent sales on same product
  - mutation conflict edge cases
  - soft-delete propagation
  - large batch sync
- Add fuzz tests for sync payload decode/validation.

2. Performance tuning
- Add indexes based on query plans (`EXPLAIN`) for reports and sync pulls.
- Benchmark batch apply and recompute under realistic volume.

3. Security hardening
- PIN retry lockout policy and cooldown.
- Optional at-rest encryption for sensitive settings.
- Structured audit log for privileged actions.

4. Migration/versioning policy
- Add formal migration version compatibility checks.
- Add rollback guidance for failed migrations.

## P3 - Nice to Have

1. Background workers cleanup
- Separate worker manager for sync/backup/report jobs.
- Graceful worker lifecycle hooks for app suspend/resume.

2. Internal API ergonomics
- Generate typed frontend SDK from backend type contracts.
- Add OpenAPI-like schema for sync HTTP endpoints.

## Current Known Gaps (Short)

- Real payment providers are not yet wired.
- Device auth/approval is not yet enforced in sync endpoints.
- Supplier/PO/report services are still pending.
- Backup/restore automation is not yet implemented.
- Sync conflict handling is partially implemented and needs full policy enforcement.
