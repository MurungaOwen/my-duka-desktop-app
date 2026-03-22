# MyDuka Risk Register (v1)

This register tracks the primary delivery and operational risks for v1.

Scale:
- Likelihood: Low / Medium / High
- Impact: Low / Medium / High / Critical

## Active Risks

| ID | Risk | Likelihood | Impact | Owner | Trigger / Early Signal | Mitigation (Preventive) | Contingency (If it happens) |
|---|---|---|---|---|---|---|---|
| R-01 | Server device unavailable in LAN mode (powered off, moved, damaged) | Medium | High | Product + Backend | Client sync status red for >10 min | Explicit setup step to select always-on server device; server health monitor on dashboard | Operate clients offline, restore server from backup, replay pending sync logs |
| R-02 | Data loss from disk failure or app crash | Medium | Critical | Backend | SQLite corruption, failed startup migrations | WAL mode, atomic writes, automatic scheduled backups, startup integrity check | Restore latest backup, replay local unsynced records, run reconciliation report |
| R-03 | Duplicate sales/stock updates due to sync retries | Medium | High | Backend | Same logical sale appears twice after reconnect | Idempotency key per mutation, unique constraints, transactional apply on server | Run dedupe job by idempotency key, flag and reverse duplicates via admin workflow |
| R-04 | Metadata overwrite from last-write-wins conflicts | Medium | Medium | Backend + Product | Product fields unexpectedly changed after sync | Add `row_version` checks for sensitive fields (price, reorder level), conflict audit log | Show conflict review queue to admin, allow explicit keep-local or keep-server choice |
| R-05 | M-Pesa API secrets exposed on client devices | Medium | High | Backend + Security | Secret appears in client logs/build artifacts | Use server-mediated M-Pesa integration, encrypt config at rest, rotate keys quarterly | Revoke/rotate credentials immediately, switch to cash/manual reconciliation mode temporarily |
| R-06 | Manual M-Pesa confirmation fraud/mistakes | Medium | High | Product + Ops | High rate of manual confirms on one cashier | Require supervisor PIN for manual confirm, structured validation, immutable audit trail | Daily exception report, admin review and void/correct transactions |
| R-07 | SQLite lock contention under heavy POS + sync load | Medium | Medium | Backend | `database is locked` errors, checkout slowdown | Short transactions, bounded sync batches, retry with jitter, index tuning | Pause background sync temporarily, prioritize POS writes, resume incremental sync |
| R-08 | mDNS discovery fails on shop routers | High | Medium | Backend | New devices cannot auto-discover server | Manual IP fallback, join-code flow, diagnostics in setup wizard | Fallback to standalone mode for temporary continuity, complete manual pairing later |
| R-09 | Unauthorized access from lost/stolen approved device | Medium | High | Security + Product | Unknown device activity, suspicious login location/time | Device approval list, revoke control, forced re-auth, PIN retry lockout | Immediate revoke and token invalidation, audit transaction window, reset staff PINs |
| R-10 | Scope overload delays stable release | High | High | Product | Missed phase milestones, rising unresolved defects | Strict phase gates, freeze non-core features, weekly risk review | Cut lower-priority modules from v1, release with hardened core POS + inventory |
| R-11 | Inadequate cashier UX under error states | Medium | High | Product + Frontend | Abandoned transactions, high cashier support calls | Action-oriented error text, no technical codes in UI, non-blocking sync alerts | Hotfix top 5 checkout blockers, temporary SOP card for staff until patch lands |
| R-12 | Backup restore path not tested in production-like environment | Medium | Critical | Ops + Backend | Backup files exist but restore fails during drill | Monthly restore drill checklist, versioned backup format, restore command in admin tools | Freeze writes, restore known-good snapshot, reconcile via sales and stock reports |
| R-13 | Card payments fail during internet outages | High | Medium | Product + Payments | Spike in card failures when ISP is unstable | Detect connectivity before card flow; show clear card-unavailable state and fallback options | Route checkout to cash, or pause and retry card once connectivity returns |

## Mode-Specific Notes

- Standalone mode (single laptop/PC):
  - Removes LAN sync and server-discovery risks (`R-01`, `R-03`, `R-08`) during normal operations.
  - Increases importance of local backup and device security (`R-02`, `R-09`, `R-12`).
- LAN sync mode (multiple devices):
  - Adds coordination and conflict risks but improves multi-register throughput.

## Review Cadence

- Weekly during active development.
- Bi-weekly after v1 go-live.
- Any `High` or `Critical` incident triggers immediate review and mitigation update.
