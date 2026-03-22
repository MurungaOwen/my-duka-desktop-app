export type HealthStatus = {
  initialized: boolean;
  mode: "standalone" | "lan_sync";
  syncEnabled: boolean;
  dbPath: string;
  deviceId: string;
  deviceName: string;
};

export type AppSyncStatus = {
  mode: string;
  enabled: boolean;
  running: boolean;
  pendingCount: number;
  lastPushed: number;
  lastPulled: number;
  consecutiveFailures: number;
  lastSuccessAt: string;
  lastError: string;
};

export type BootstrapInput = {
  businessName: string;
  location: string;
  currency: string;
  vatRate: string; // e.g. "16"
};

export type Setting = {
  key: string;
  value: string;
};

export type StaffRole = "admin" | "cashier";

export type CreateStaffInput = {
  name: string;
  username: string;
  role: StaffRole;
  password: string;
};

export type Staff = {
  id: string;
  name: string;
  username: string;
  role: StaffRole;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
};

export type StaffLoginInput = {
  username: string;
  password: string;
};

export type PINVerificationInput = {
  staffId: string;
  pin: string;
};

export type CreateCategoryInput = {
  name: string;
  emoji: string;
  displayOrder: number;
};

export type Category = {
  id: string;
  name: string;
  emoji: string;
  displayOrder: number;
  createdAt: string;
  updatedAt: string;
};

export type CreateProductInput = {
  name: string;
  sku: string;
  barcode: string;
  categoryId: string;
  priceCents: number;
  startingStock: number;
  reorderLevel: number;
};

export type Product = {
  id: string;
  name: string;
  sku: string;
  barcode: string;
  categoryId: string;
  priceCents: number;
  startingStock: number;
  stockQty: number;
  reorderLevel: number;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
};

export type StockAdjustmentInput = {
  productId: string;
  qtyChange: number; // positive or negative
  reason: string;
};

export type ProductStockView = {
  id: string;
  name: string;
  stockQty: number;
  reorderLevel: number;
};

export type SaleItemInput = {
  productId: string;
  quantity: number; // > 0
};

export type PaymentMethod = "cash" | "mpesa" | "card";

export type CreateSaleInput = {
  cashierStaffId: string;
  paymentMethod: PaymentMethod;
  items: SaleItemInput[];
};

export type Sale = {
  id: string;
  cashierStaffId: string;
  paymentMethod: PaymentMethod;
  status: string;
  subtotalCents: number;
  vatCents: number;
  totalCents: number;
  createdAt: string;
};

export type SaleItem = {
  id: string;
  saleId: string;
  productId: string;
  quantity: number;
  unitPriceCents: number;
  lineTotalCents: number;
};

export type SaleDetail = Sale & {
  items: SaleItem[];
};

export type DashboardSummary = {
  revenueTodayCents: number;
  transactionsTodayCount: number;
  lowStockCount: number;
  outOfStockCount: number;
};

export type SyncRecord = {
  id: string;
  tableName: string;
  recordId: string;
  operation: string;
  payload: string; // raw JSON string
  createdAt: string;
};

export type SyncMutation = {
  mutationId: string;
  sourceDeviceId?: string;
  tableName: string;
  recordId: string;
  operation: string;
  payload: string;
  createdAt: string;
};

export type SyncPushResponse = {
  applied: number;
  skipped: number;
};
