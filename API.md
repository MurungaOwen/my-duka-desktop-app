# MyDuka Backend API (Frontend Integration)

This document is the frontend integration contract for the desktop app backend.

It covers:
- Wails-bound methods exposed by `App` (frontend calls these directly)
- Data models and payloads
- Error behaviors
- LAN sync HTTP API used by sync workers (`/sync/push`, `/sync/pull`)

All money amounts are in **cents** (`int64`).
All IDs are strings (UUID/text).
All timestamps are UTC RFC3339 strings.

## 1. Runtime Modes

- `standalone`: local-only; no background LAN sync worker
- `lan_sync`: local DB + background sync engine

Env vars:
- `MYDUKA_MODE=standalone|lan_sync`
- `MYDUKA_DB_PATH=/absolute/path/to/myduka.db`
- `MYDUKA_SYNC_BASE_URL=http://myduka.local:8080`
- `MYDUKA_SYNC_INTERVAL_SECONDS=5`
- `MYDUKA_SYNC_BATCH_LIMIT=200`

## 2. Frontend Call Pattern (Wails)

In frontend, methods are generated under:
- `frontend/wailsjs/go/main/App.*`

Each method returns:
- success result, or
- rejected promise with error string

## 3. Core Status APIs

### `StartupStatus() string`

Returns:
- `"ok"` when backend started
- error string if startup failed

### `BackendHealth() HealthStatus`

`HealthStatus`:
```ts
type HealthStatus = {
  initialized: boolean
  mode: "standalone" | "lan_sync"
  syncEnabled: boolean
  dbPath: string
  deviceId: string
  deviceName: string
}
```

### `GetSyncStatus() AppSyncStatus`

For sync indicator UI.

```ts
type AppSyncStatus = {
  mode: string
  enabled: boolean
  running: boolean
  pendingCount: number
  lastPushed: number
  lastPulled: number
  consecutiveFailures: number
  lastSuccessAt: string
  lastError: string
}
```

### `SeedDemoData(): SeedResult`

Populates demo settings, staff, categories, suppliers, products, purchase orders, and sample sales.
Safe to run multiple times (idempotent).

```ts
type DemoCredentials = {
  role: string
  name: string
  username: string
  password: string
  notes: string
}

type SeedResult = {
  businessName: string
  staffAdded: number
  categoriesAdded: number
  suppliersAdded: number
  productsAdded: number
  ordersAdded: number
  salesAdded: number
  credentials: DemoCredentials[]
}
```

## 4. Setup and Settings APIs

### `BootstrapBusiness(input: BootstrapInput): void`

```ts
type BootstrapInput = {
  businessName: string
  location: string
  currency: string
  vatRate: string // e.g. "16"
}
```

### `UpsertSetting(setting: Setting): void`

```ts
type Setting = { key: string; value: string }
```

### `ListSettings(): Setting[]`

## 5. Staff APIs

### `CreateStaff(input: CreateStaffInput): Staff`

```ts
type CreateStaffInput = {
  name: string
  username: string
  role: "admin" | "cashier"
  password: string
}

type Staff = {
  id: string
  name: string
  username: string
  role: "admin" | "cashier"
  isActive: boolean
  createdAt: string
  updatedAt: string
}
```

### `ListStaff(): Staff[]`

### `AuthenticateStaff(input: StaffLoginInput): Staff`

```ts
type StaffLoginInput = {
  username: string
  password: string
}
```

### `VerifyStaffPIN(input: PINVerificationInput): boolean`

Legacy compatibility helper. Uses staff ID + secret string check.

```ts
type PINVerificationInput = {
  staffId: string
  pin: string
}
```

Returns:
- `true` if valid PIN
- `false` if invalid PIN
- throws on missing/inactive/nonexistent staff

## 6. Category APIs

### `CreateCategory(input: CreateCategoryInput): Category`

```ts
type CreateCategoryInput = {
  name: string
  emoji: string
  displayOrder: number
}

type Category = {
  id: string
  name: string
  emoji: string
  displayOrder: number
  createdAt: string
  updatedAt: string
}
```

### `ListCategories(): Category[]`

## 7. Product and Inventory APIs

### `CreateProduct(input: CreateProductInput): Product`

```ts
type CreateProductInput = {
  name: string
  sku: string
  barcode: string
  categoryId: string
  priceCents: number
  startingStock: number
  reorderLevel: number
}

type Product = {
  id: string
  name: string
  sku: string
  barcode: string
  categoryId: string
  priceCents: number
  startingStock: number
  stockQty: number
  reorderLevel: number
  isActive: boolean
  createdAt: string
  updatedAt: string
}
```

### `ListProducts(): Product[]`

### `AdjustStock(input: StockAdjustmentInput): void`

```ts
type StockAdjustmentInput = {
  productId: string
  qtyChange: number // cannot be 0; positive or negative
  reason: string
}
```

### `ListLowStockProducts(): ProductStockView[]`

```ts
type ProductStockView = {
  id: string
  name: string
  stockQty: number
  reorderLevel: number
}
```

## 8. Sales APIs

### `CreateSale(input: CreateSaleInput): SaleDetail`

```ts
type SaleItemInput = {
  productId: string
  quantity: number // > 0
}

type CreateSaleInput = {
  cashierStaffId: string
  paymentMethod: "cash" | "mpesa" | "card"
  paymentRef?: string // for verified external mpesa payments
  items: SaleItemInput[]
}

type Sale = {
  id: string
  cashierStaffId: string
  paymentMethod: "cash" | "mpesa" | "card"
  status: string // currently "completed"
  subtotalCents: number
  vatCents: number
  totalCents: number
  createdAt: string
}

type SaleItem = {
  id: string
  saleId: string
  productId: string
  quantity: number
  unitPriceCents: number
  lineTotalCents: number
}

type SaleDetail = Sale & {
  items: SaleItem[]
}
```

Behavior:
- validates cashier exists/active
- validates stock availability
- writes sale + line items + stock transactions atomically
- recomputes stock for affected products
- if `paymentMethod=mpesa` and `paymentRef` is present, backend binds the reference to the sale and rejects duplicate reuse.

### `StartMPesaCharge(input: StartMPesaChargeInput): MPesaChargeSession`

Creates a Paystack M-Pesa charge (`POST /charge`) for Kenya (`currency: KES`, provider `mpesa`).

```ts
type StartMPesaChargeInput = {
  phone: string
  amountCents: number
  email?: string
  reference?: string
}

type MPesaChargeSession = {
  reference: string
  status: string
  displayText: string
  message: string
}
```

### `VerifyMPesaCharge(reference: string): MPesaChargeStatus`

Checks charge state from Paystack (primary: `GET /transaction/verify/:reference`, fallback: `GET /charge/:reference`).

```ts
type MPesaChargeStatus = {
  reference: string
  status: string
  paid: boolean
  gatewayResponse: string
  displayText: string
  message: string
}
```

### `ListRecentMPesaPayments(input: ListRecentMPesaPaymentsInput): RecentMPesaPayment[]`

Use this for "customer already paid" flow.
Returns recent successful mobile money payments, filtered by time window, amount, and excluding references already used by previous sales.

```ts
type ListRecentMPesaPaymentsInput = {
  windowMinutes: number // default 15
  amountCents: number   // optional exact match if > 0
  limit: number         // default 30, max 100
}

type RecentMPesaPayment = {
  reference: string
  amountCents: number
  currency: string
  channel: string
  paidAt: string
  gatewayResponse: string
  customerEmail: string
  customerName: string
  authorizationKey: string
}
```

Configuration:
- `PAYSTACK_SECRET_KEY` env var (preferred), or setting key `paystack_secret_key`
- `PAYSTACK_POS_EMAIL` env var (required for POS charge requests), or setting key `paystack_pos_email`
- optional: `PAYSTACK_BASE_URL` (default `https://api.paystack.co`)

### `ListSales(limit: number): Sale[]`

Defaults:
- if `limit <= 0` or too large, backend caps to safe default (`100`)

### `GetSaleDetail(saleID: string): SaleDetail`

## 9. Dashboard API

### `DashboardSummary(): DashboardSummary`

```ts
type DashboardSummary = {
  revenueTodayCents: number
  transactionsTodayCount: number
  lowStockCount: number
  outOfStockCount: number
}
```

## 10. Local Sync Queue APIs (for diagnostics/admin)

### `ListPendingSyncRecords(limit: number): SyncRecord[]`

```ts
type SyncRecord = {
  id: string
  tableName: string
  recordId: string
  operation: string
  payload: string // raw JSON string
  createdAt: string
}
```

### `MarkSyncRecordsSynced(recordIDs: string[]): void`

## 11. Mutation Sync APIs (direct method form)

These are used internally by sync engine and can also be called manually if needed.

### `ApplyIncomingMutations(sourceDeviceID: string, mutations: SyncMutation[]): SyncPushResponse`

### `PullMutationsForDevice(deviceID: string, since: string, limit: number): SyncMutation[]`

```ts
type SyncMutation = {
  mutationId: string
  sourceDeviceId?: string
  tableName: string
  recordId: string
  operation: string
  payload: string
  createdAt: string
}

type SyncPushResponse = {
  applied: number
  skipped: number // idempotent duplicates
}
```

Supported mutation tables:
- `settings`
- `categories`
- `products`
- `stock_transactions`
- `staff`
- `sales`
- `sale_items`
- (`suppliers`, `purchase_orders` accepted as pass-through placeholders)

## 12. LAN Sync HTTP API (server side)

Used by sync workers over LAN.

### `GET /health`

Returns `HealthStatus`.

### `POST /sync/push`

Request:
```json
{
  "deviceId": "client-device-1",
  "mutations": [
    {
      "mutationId": "uuid",
      "sourceDeviceId": "client-device-1",
      "tableName": "products",
      "recordId": "product-id",
      "operation": "insert",
      "payload": "{\"id\":\"...\"}",
      "createdAt": "2026-01-01T10:00:00Z"
    }
  ]
}
```

Response:
```json
{
  "applied": 10,
  "skipped": 2
}
```

### `GET /sync/pull?device_id={id}&since={cursor}&limit={n}`

Response:
```json
{
  "mutations": [
    {
      "mutationId": "uuid",
      "sourceDeviceId": "other-device",
      "tableName": "stock_transactions",
      "recordId": "stx-id",
      "operation": "insert",
      "payload": "{\"id\":\"...\"}",
      "createdAt": "2026-01-01T10:00:03Z"
    }
  ]
}
```

Notes:
- Pull excludes records originating from requesting `device_id`.
- `sync_inbox` deduplicates by `mutationId`.

## 13. Sync Engine Behavior (lan_sync mode)

- Push pending local records in batches (`default 200`)
- On success, mark local pending as synced
- Pull remote mutations after cursor
- Apply incoming mutations locally and recompute stock from ledger
- Retry backoff on failure:
  - 1st failure: 5s
  - 2nd failure: 15s
  - 3rd failure: 30s
  - 4th+ failure: 60s

## 14. Error Semantics (Frontend Handling)

Display user-friendly fallback messages for these common backend errors:

- `"backend service unavailable"`:
  - app not initialized; prompt restart
- `"insufficient stock for <name>"`:
  - show out-of-stock/inadequate stock prompt
- `"staff not found"` / `"staff is inactive"`:
  - re-login staff
- `"payment method must be cash, mpesa, or card"`:
  - invalid frontend payload; bug in client
- `"device id is required"` (sync):
  - invalid sync config

For all unknown errors:
- log details (console + telemetry if enabled)
- show generic action prompt: `Something went wrong. Please retry.`

## 15. Minimal Frontend Integration Flow

1. On app load:
   - call `StartupStatus()`
   - call `BackendHealth()`
   - call `GetSyncStatus()`
2. On login:
   - call `AuthenticateStaff()`
3. For POS page:
   - call `ListCategories()`, `ListProducts()`
   - for M-Pesa: call `StartMPesaCharge()` then poll/call `VerifyMPesaCharge()`
   - call `CreateSale()`
4. For admin inventory:
   - call `ListProducts()`, `AdjustStock()`, `ListLowStockProducts()`
5. For dashboard:
   - call `DashboardSummary()`
6. For sync indicator (lan_sync):
   - poll `GetSyncStatus()` every few seconds
