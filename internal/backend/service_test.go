package backend

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestService(t *testing.T) *Service {
	t.Helper()

	cfg := Config{
		DBPath:     filepath.Join(t.TempDir(), "myduka-test.db"),
		Mode:       DeploymentModeStandalone,
		DeviceID:   "test-device-1",
		DeviceName: "test-device",
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("start service: %v", err)
	}
	t.Cleanup(func() {
		_ = svc.Close()
	})
	return svc
}

func TestBootstrapBusinessAndSettings(t *testing.T) {
	svc := setupTestService(t)

	err := svc.BootstrapBusiness(BootstrapInput{
		BusinessName: "MyDuka Test Shop",
		Location:     "Nairobi",
		Currency:     "KES",
		VATRate:      "16",
	})
	if err != nil {
		t.Fatalf("bootstrap business: %v", err)
	}

	settings, err := svc.ListSettings()
	if err != nil {
		t.Fatalf("list settings: %v", err)
	}
	if len(settings) < 4 {
		t.Fatalf("expected at least 4 settings, got %d", len(settings))
	}

	found := map[string]string{}
	for _, s := range settings {
		found[s.Key] = s.Value
	}
	if found["business_name"] != "MyDuka Test Shop" {
		t.Fatalf("business_name mismatch: %q", found["business_name"])
	}
	if found["currency"] != "KES" {
		t.Fatalf("currency mismatch: %q", found["currency"])
	}
}

func TestStaffPINVerification(t *testing.T) {
	svc := setupTestService(t)

	staff, err := svc.CreateStaff(CreateStaffInput{
		Name:     "Alice",
		Username: "alice",
		Role:     "cashier",
		Password: "1234",
	})
	if err != nil {
		t.Fatalf("create staff: %v", err)
	}

	ok, err := svc.VerifyStaffPIN(PINVerificationInput{
		StaffID: staff.ID,
		PIN:     "1234",
	})
	if err != nil {
		t.Fatalf("verify staff pin: %v", err)
	}
	if !ok {
		t.Fatalf("expected pin verification to succeed")
	}

	ok, err = svc.VerifyStaffPIN(PINVerificationInput{
		StaffID: staff.ID,
		PIN:     "9999",
	})
	if err != nil {
		t.Fatalf("verify wrong pin: %v", err)
	}
	if ok {
		t.Fatalf("expected wrong pin verification to fail")
	}
}

func TestAuthenticateStaff(t *testing.T) {
	svc := setupTestService(t)

	_, err := svc.CreateStaff(CreateStaffInput{
		Name:     "Auth User",
		Username: "authuser",
		Role:     "cashier",
		Password: "pass1234",
	})
	if err != nil {
		t.Fatalf("create staff: %v", err)
	}

	staff, err := svc.AuthenticateStaff(StaffLoginInput{
		Username: "authuser",
		Password: "pass1234",
	})
	if err != nil {
		t.Fatalf("authenticate staff: %v", err)
	}
	if staff.Username != "authuser" {
		t.Fatalf("expected username authuser, got %s", staff.Username)
	}

	_, err = svc.AuthenticateStaff(StaffLoginInput{
		Username: "authuser",
		Password: "wrong",
	})
	if err == nil {
		t.Fatalf("expected auth failure on wrong password")
	}
}

func TestListStaff(t *testing.T) {
	svc := setupTestService(t)

	_, err := svc.CreateStaff(CreateStaffInput{Name: "Owner", Username: "owner", Role: "admin", Password: "1234"})
	if err != nil {
		t.Fatalf("create admin staff: %v", err)
	}
	_, err = svc.CreateStaff(CreateStaffInput{Name: "Cashier", Username: "cashier", Role: "cashier", Password: "5678"})
	if err != nil {
		t.Fatalf("create cashier staff: %v", err)
	}

	staffList, err := svc.ListStaff()
	if err != nil {
		t.Fatalf("list staff: %v", err)
	}
	if len(staffList) != 2 {
		t.Fatalf("expected 2 staff records, got %d", len(staffList))
	}
}

func TestCreateSaleUpdatesStockAndTotals(t *testing.T) {
	svc := setupTestService(t)

	cashier, err := svc.CreateStaff(CreateStaffInput{
		Name:     "Bob",
		Username: "bob",
		Role:     "cashier",
		Password: "1234",
	})
	if err != nil {
		t.Fatalf("create staff: %v", err)
	}

	if err := svc.UpsertSetting(Setting{Key: "vat_rate", Value: "16"}); err != nil {
		t.Fatalf("upsert vat rate: %v", err)
	}

	category, err := svc.CreateCategory(CreateCategoryInput{Name: "Groceries", Emoji: "🛒"})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	product, err := svc.CreateProduct(CreateProductInput{
		Name:          "Rice",
		SKU:           "RICE-001",
		CategoryID:    category.ID,
		PriceCents:    1000,
		StartingStock: 10,
		ReorderLevel:  3,
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	sale, err := svc.CreateSale(CreateSaleInput{
		CashierStaffID: cashier.ID,
		PaymentMethod:  "cash",
		Items: []SaleItemInput{
			{ProductID: product.ID, Quantity: 2},
		},
	})
	if err != nil {
		t.Fatalf("create sale: %v", err)
	}

	if sale.SubtotalCents != 2000 {
		t.Fatalf("subtotal mismatch: got %d", sale.SubtotalCents)
	}
	if sale.VATCents != 320 {
		t.Fatalf("vat mismatch: got %d", sale.VATCents)
	}
	if sale.TotalCents != 2320 {
		t.Fatalf("total mismatch: got %d", sale.TotalCents)
	}

	products, err := svc.ListProducts()
	if err != nil {
		t.Fatalf("list products: %v", err)
	}
	if len(products) == 0 {
		t.Fatalf("expected products to be present")
	}

	var updated Product
	for _, p := range products {
		if p.ID == product.ID {
			updated = p
			break
		}
	}
	if updated.ID == "" {
		t.Fatalf("updated product not found")
	}
	if updated.StockQty != 8 {
		t.Fatalf("expected stock 8 after sale, got %d", updated.StockQty)
	}

	sales, err := svc.ListSales(10)
	if err != nil {
		t.Fatalf("list sales: %v", err)
	}
	if len(sales) != 1 {
		t.Fatalf("expected 1 sale, got %d", len(sales))
	}
}

func TestCreateSaleInsufficientStock(t *testing.T) {
	svc := setupTestService(t)

	cashier, err := svc.CreateStaff(CreateStaffInput{
		Name:     "Carol",
		Username: "carol",
		Role:     "cashier",
		Password: "1234",
	})
	if err != nil {
		t.Fatalf("create staff: %v", err)
	}

	product, err := svc.CreateProduct(CreateProductInput{
		Name:          "Soap",
		PriceCents:    250,
		StartingStock: 1,
		ReorderLevel:  1,
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	_, err = svc.CreateSale(CreateSaleInput{
		CashierStaffID: cashier.ID,
		PaymentMethod:  "cash",
		Items: []SaleItemInput{
			{ProductID: product.ID, Quantity: 2},
		},
	})
	if err == nil {
		t.Fatalf("expected insufficient stock error, got nil")
	}
	if !strings.Contains(err.Error(), "insufficient stock") {
		t.Fatalf("expected insufficient stock error, got %v", err)
	}
}

func TestAdjustStockLowStockAndDashboardSummary(t *testing.T) {
	svc := setupTestService(t)

	cashier, err := svc.CreateStaff(CreateStaffInput{
		Name:     "Dan",
		Username: "dan",
		Role:     "cashier",
		Password: "1234",
	})
	if err != nil {
		t.Fatalf("create staff: %v", err)
	}

	product1, err := svc.CreateProduct(CreateProductInput{
		Name:          "Milk",
		PriceCents:    300,
		StartingStock: 5,
		ReorderLevel:  3,
	})
	if err != nil {
		t.Fatalf("create product1: %v", err)
	}

	product2, err := svc.CreateProduct(CreateProductInput{
		Name:          "Bread",
		PriceCents:    500,
		StartingStock: 5,
		ReorderLevel:  1,
	})
	if err != nil {
		t.Fatalf("create product2: %v", err)
	}

	if err := svc.AdjustStock(StockAdjustmentInput{
		ProductID: product1.ID,
		QtyChange: -3,
		Reason:    "writeoff",
	}); err != nil {
		t.Fatalf("adjust stock #1: %v", err)
	}
	if err := svc.AdjustStock(StockAdjustmentInput{
		ProductID: product1.ID,
		QtyChange: -2,
		Reason:    "writeoff",
	}); err != nil {
		t.Fatalf("adjust stock #2: %v", err)
	}

	_, err = svc.CreateSale(CreateSaleInput{
		CashierStaffID: cashier.ID,
		PaymentMethod:  "cash",
		Items: []SaleItemInput{
			{ProductID: product2.ID, Quantity: 1},
		},
	})
	if err != nil {
		t.Fatalf("create sale: %v", err)
	}

	lowStock, err := svc.ListLowStockProducts()
	if err != nil {
		t.Fatalf("list low stock products: %v", err)
	}
	if len(lowStock) != 1 {
		t.Fatalf("expected 1 low-stock product, got %d", len(lowStock))
	}
	if lowStock[0].ID != product1.ID {
		t.Fatalf("expected low-stock product to be %s, got %s", product1.ID, lowStock[0].ID)
	}
	if lowStock[0].StockQty != 0 {
		t.Fatalf("expected stock 0 for low-stock product, got %d", lowStock[0].StockQty)
	}

	summary, err := svc.DashboardSummary()
	if err != nil {
		t.Fatalf("dashboard summary: %v", err)
	}
	if summary.RevenueTodayCents != 500 {
		t.Fatalf("expected revenue 500, got %d", summary.RevenueTodayCents)
	}
	if summary.TransactionsTodayCount != 1 {
		t.Fatalf("expected 1 transaction, got %d", summary.TransactionsTodayCount)
	}
	if summary.LowStockCount != 1 {
		t.Fatalf("expected 1 low stock item, got %d", summary.LowStockCount)
	}
	if summary.OutOfStockCount != 1 {
		t.Fatalf("expected 1 out-of-stock item, got %d", summary.OutOfStockCount)
	}
}

func TestSyncQueuePendingAndMarkSynced(t *testing.T) {
	svc := setupTestService(t)

	_, err := svc.CreateCategory(CreateCategoryInput{Name: "SyncCat", Emoji: "S"})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	_, err = svc.CreateProduct(CreateProductInput{
		Name:          "SyncProduct",
		PriceCents:    100,
		StartingStock: 3,
		ReorderLevel:  1,
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	pending, err := svc.ListPendingSyncRecords(50)
	if err != nil {
		t.Fatalf("list pending sync records: %v", err)
	}
	if len(pending) < 2 {
		t.Fatalf("expected at least 2 pending sync records, got %d", len(pending))
	}

	ids := make([]string, 0, len(pending))
	for _, rec := range pending {
		ids = append(ids, rec.ID)
	}
	if err := svc.MarkSyncRecordsSynced(ids); err != nil {
		t.Fatalf("mark sync records synced: %v", err)
	}

	pendingAfter, err := svc.ListPendingSyncRecords(50)
	if err != nil {
		t.Fatalf("list pending sync records after mark synced: %v", err)
	}
	if len(pendingAfter) != 0 {
		t.Fatalf("expected no pending sync records after mark synced, got %d", len(pendingAfter))
	}
}

func TestSeedDemoData(t *testing.T) {
	svc := setupTestService(t)

	result, err := svc.SeedDemoData()
	if err != nil {
		t.Fatalf("seed demo data: %v", err)
	}
	if result.BusinessName == "" {
		t.Fatalf("expected business name in seed result")
	}
	if len(result.Credentials) < 2 {
		t.Fatalf("expected demo credentials in seed result")
	}

	settings, err := svc.ListSettings()
	if err != nil {
		t.Fatalf("list settings: %v", err)
	}
	if len(settings) == 0 {
		t.Fatalf("expected seeded settings")
	}

	staff, err := svc.ListStaff()
	if err != nil {
		t.Fatalf("list staff: %v", err)
	}
	if len(staff) < 3 {
		t.Fatalf("expected at least 3 seeded staff, got %d", len(staff))
	}

	products, err := svc.ListProducts()
	if err != nil {
		t.Fatalf("list products: %v", err)
	}
	if len(products) < 8 {
		t.Fatalf("expected seeded products, got %d", len(products))
	}
	if result.SuppliersAdded == 0 {
		t.Fatalf("expected suppliers to be seeded")
	}
	if result.OrdersAdded == 0 {
		t.Fatalf("expected purchase orders to be seeded")
	}

	sales, err := svc.ListSales(50)
	if err != nil {
		t.Fatalf("list sales: %v", err)
	}
	if len(sales) == 0 {
		t.Fatalf("expected seeded sales")
	}
}

func TestSeedDemoDataIsIdempotent(t *testing.T) {
	svc := setupTestService(t)

	if _, err := svc.SeedDemoData(); err != nil {
		t.Fatalf("first seed run failed: %v", err)
	}
	productsBefore, err := svc.ListProducts()
	if err != nil {
		t.Fatalf("list products before second seed: %v", err)
	}
	staffBefore, err := svc.ListStaff()
	if err != nil {
		t.Fatalf("list staff before second seed: %v", err)
	}

	second, err := svc.SeedDemoData()
	if err != nil {
		t.Fatalf("second seed run failed: %v", err)
	}
	if second.ProductsAdded != 0 {
		t.Fatalf("expected second run to add 0 products, got %d", second.ProductsAdded)
	}
	if second.SuppliersAdded != 0 {
		t.Fatalf("expected second run to add 0 suppliers, got %d", second.SuppliersAdded)
	}
	if second.OrdersAdded != 0 {
		t.Fatalf("expected second run to add 0 purchase orders, got %d", second.OrdersAdded)
	}
	if second.SalesAdded != 0 {
		t.Fatalf("expected second run to add 0 sales, got %d", second.SalesAdded)
	}

	productsAfter, err := svc.ListProducts()
	if err != nil {
		t.Fatalf("list products after second seed: %v", err)
	}
	staffAfter, err := svc.ListStaff()
	if err != nil {
		t.Fatalf("list staff after second seed: %v", err)
	}

	if len(productsAfter) != len(productsBefore) {
		t.Fatalf("expected idempotent products count, before=%d after=%d", len(productsBefore), len(productsAfter))
	}
	if len(staffAfter) != len(staffBefore) {
		t.Fatalf("expected idempotent staff count, before=%d after=%d", len(staffBefore), len(staffAfter))
	}
}
