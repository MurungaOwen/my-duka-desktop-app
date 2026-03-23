package store

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type seedStaff struct {
	Name     string
	Username string
	Role     string
	Password string
}

type seedCategory struct {
	Name  string
	Emoji string
}

type seedSupplier struct {
	Name  string
	Phone string
	Email string
	Notes string
}

type seedProduct struct {
	Name          string
	SKU           string
	Barcode       string
	CategoryName  string
	SupplierName  string
	PriceCents    int64
	StartingStock int64
	ReorderLevel  int64
}

type seedOptions struct {
	Settings       bool
	Users          bool
	Categories     bool
	Suppliers      bool
	Products       bool
	PurchaseOrders bool
	Sales          bool
}

func (s *Service) SeedDemoData() (SeedResult, error) {
	db, err := s.getDB()
	if err != nil {
		return SeedResult{}, err
	}
	opts := seedOptionsFromEnv()

	tx, err := db.Begin()
	if err != nil {
		return SeedResult{}, fmt.Errorf("begin seed transaction: %w", err)
	}

	result := SeedResult{BusinessName: "MyDuka Demo Shop"}

	settings := []Setting{
		{Key: "business_name", Value: result.BusinessName},
		{Key: "location", Value: "Nairobi"},
		{Key: "currency", Value: "KES"},
		{Key: "vat_rate", Value: "16"},
		{Key: "industry", Value: "general_retail"},
	}
	if opts.Settings {
		for _, kv := range settings {
			if err := s.upsertSettingTx(tx, kv); err != nil {
				_ = tx.Rollback()
				return SeedResult{}, err
			}
		}
	}

	staffSeeds := []seedStaff{
		{Name: "Owner", Username: "owner", Role: "admin", Password: "owner1234"},
		{Name: "Alice", Username: "alice", Role: "cashier", Password: "alice1234"},
		{Name: "Brian", Username: "brian", Role: "cashier", Password: "brian1234"},
	}
	if opts.Users {
		result.Credentials = []DemoCredentials{
			{Role: "admin", Name: "Owner", Username: "owner", Password: "owner1234", Notes: "Full access"},
			{Role: "cashier", Name: "Alice", Username: "alice", Password: "alice1234", Notes: "POS access"},
			{Role: "cashier", Name: "Brian", Username: "brian", Password: "brian1234", Notes: "POS access"},
		}
		for _, seed := range staffSeeds {
			added, err := s.seedStaffTx(tx, seed)
			if err != nil {
				_ = tx.Rollback()
				return SeedResult{}, err
			}
			if added {
				result.StaffAdded++
			}
		}
	}

	categorySeeds := []seedCategory{
		{Name: "Beverages", Emoji: "🥤"},
		{Name: "Snacks", Emoji: "🍪"},
		{Name: "Groceries", Emoji: "🧺"},
		{Name: "Personal Care", Emoji: "🧴"},
	}

	categoryIDs := make(map[string]string, len(categorySeeds))
	if opts.Categories {
		for idx, seed := range categorySeeds {
			catID, added, err := s.seedCategoryTx(tx, seed, int64(idx+1))
			if err != nil {
				_ = tx.Rollback()
				return SeedResult{}, err
			}
			categoryIDs[seed.Name] = catID
			if added {
				result.CategoriesAdded++
			}
		}
	} else {
		ids, err := s.resolveCategoryIDsByNameTx(tx, categorySeeds)
		if err != nil {
			_ = tx.Rollback()
			return SeedResult{}, err
		}
		for k, v := range ids {
			categoryIDs[k] = v
		}
	}

	supplierSeeds := []seedSupplier{
		{Name: "Nairobi Beverage Distributors", Phone: "0711000001", Email: "sales@nairobi-bev.example", Notes: "Beverages and chilled stock"},
		{Name: "Metro Grocers Wholesale", Phone: "0711000002", Email: "orders@metro-grocers.example", Notes: "Staples and groceries"},
		{Name: "Prime Care Supplies", Phone: "0711000003", Email: "hello@primecare.example", Notes: "Personal care and hygiene items"},
	}

	supplierIDs := make(map[string]string, len(supplierSeeds))
	if opts.Suppliers {
		for _, seed := range supplierSeeds {
			supplierID, added, err := s.seedSupplierTx(tx, seed)
			if err != nil {
				_ = tx.Rollback()
				return SeedResult{}, err
			}
			supplierIDs[seed.Name] = supplierID
			if added {
				result.SuppliersAdded++
			}
		}
	} else {
		ids, err := s.resolveSupplierIDsByNameTx(tx, supplierSeeds)
		if err != nil {
			_ = tx.Rollback()
			return SeedResult{}, err
		}
		for k, v := range ids {
			supplierIDs[k] = v
		}
	}

	productSeeds := []seedProduct{
		{Name: "Coca-Cola 500ml", SKU: "BEV-COKE-500", Barcode: "616110000101", CategoryName: "Beverages", SupplierName: "Nairobi Beverage Distributors", PriceCents: 12000, StartingStock: 48, ReorderLevel: 12},
		{Name: "Fanta Orange 500ml", SKU: "BEV-FANTA-500", Barcode: "616110000102", CategoryName: "Beverages", SupplierName: "Nairobi Beverage Distributors", PriceCents: 12000, StartingStock: 36, ReorderLevel: 10},
		{Name: "Mineral Water 1L", SKU: "BEV-WATER-1L", Barcode: "616110000103", CategoryName: "Beverages", SupplierName: "Nairobi Beverage Distributors", PriceCents: 7000, StartingStock: 60, ReorderLevel: 15},
		{Name: "Milk 500ml", SKU: "GRC-MILK-500", Barcode: "616110000201", CategoryName: "Groceries", SupplierName: "Metro Grocers Wholesale", PriceCents: 6500, StartingStock: 30, ReorderLevel: 8},
		{Name: "Bread 400g", SKU: "GRC-BREAD-400", Barcode: "616110000202", CategoryName: "Groceries", SupplierName: "Metro Grocers Wholesale", PriceCents: 5500, StartingStock: 24, ReorderLevel: 6},
		{Name: "Sugar 1kg", SKU: "GRC-SUGAR-1KG", Barcode: "616110000203", CategoryName: "Groceries", SupplierName: "Metro Grocers Wholesale", PriceCents: 18000, StartingStock: 20, ReorderLevel: 5},
		{Name: "Potato Crisps", SKU: "SNK-CRISPS-001", Barcode: "616110000301", CategoryName: "Snacks", SupplierName: "Metro Grocers Wholesale", PriceCents: 10000, StartingStock: 40, ReorderLevel: 10},
		{Name: "Biscuits Pack", SKU: "SNK-BISCUIT-001", Barcode: "616110000302", CategoryName: "Snacks", SupplierName: "Metro Grocers Wholesale", PriceCents: 9000, StartingStock: 35, ReorderLevel: 8},
		{Name: "Toothpaste 100ml", SKU: "PRC-TOOTH-100", Barcode: "616110000401", CategoryName: "Personal Care", SupplierName: "Prime Care Supplies", PriceCents: 16000, StartingStock: 18, ReorderLevel: 5},
		{Name: "Bath Soap", SKU: "PRC-SOAP-001", Barcode: "616110000402", CategoryName: "Personal Care", SupplierName: "Prime Care Supplies", PriceCents: 8500, StartingStock: 25, ReorderLevel: 7},
	}

	if opts.Products {
		for _, seed := range productSeeds {
			categoryID := categoryIDs[seed.CategoryName]
			if categoryID == "" {
				_ = tx.Rollback()
				return SeedResult{}, fmt.Errorf("missing category id for %s", seed.CategoryName)
			}
			supplierID := supplierIDs[seed.SupplierName]
			if supplierID == "" {
				_ = tx.Rollback()
				return SeedResult{}, fmt.Errorf("missing supplier id for %s", seed.SupplierName)
			}
			added, err := s.seedProductTx(tx, seed, categoryID, supplierID)
			if err != nil {
				_ = tx.Rollback()
				return SeedResult{}, err
			}
			if added {
				result.ProductsAdded++
			}
		}
	}

	if opts.PurchaseOrders {
		for supplierName, supplierID := range supplierIDs {
			if supplierID == "" {
				continue
			}
			added, err := s.seedPurchaseOrderTx(tx, supplierID, supplierName)
			if err != nil {
				_ = tx.Rollback()
				return SeedResult{}, err
			}
			if added {
				result.OrdersAdded++
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return SeedResult{}, fmt.Errorf("commit seed transaction: %w", err)
	}

	if opts.Sales {
		salesAdded, err := s.seedSalesData()
		if err != nil {
			return SeedResult{}, err
		}
		result.SalesAdded = salesAdded
	}

	return result, nil
}

func (s *Service) SeedUsersOnly() (SeedResult, error) {
	db, err := s.getDB()
	if err != nil {
		return SeedResult{}, err
	}

	tx, err := db.Begin()
	if err != nil {
		return SeedResult{}, fmt.Errorf("begin users-only seed transaction: %w", err)
	}

	result := SeedResult{
		BusinessName: "MyDuka Demo Shop",
		Credentials: []DemoCredentials{
			{Role: "admin", Name: "Owner", Username: "owner", Password: "owner1234", Notes: "Full access"},
			{Role: "cashier", Name: "Alice", Username: "alice", Password: "alice1234", Notes: "POS access"},
			{Role: "cashier", Name: "Brian", Username: "brian", Password: "brian1234", Notes: "POS access"},
		},
	}

	staffSeeds := []seedStaff{
		{Name: "Owner", Username: "owner", Role: "admin", Password: "owner1234"},
		{Name: "Alice", Username: "alice", Role: "cashier", Password: "alice1234"},
		{Name: "Brian", Username: "brian", Role: "cashier", Password: "brian1234"},
	}

	for _, seed := range staffSeeds {
		added, err := s.seedStaffTx(tx, seed)
		if err != nil {
			_ = tx.Rollback()
			return SeedResult{}, err
		}
		if added {
			result.StaffAdded++
		}
	}

	if err := tx.Commit(); err != nil {
		return SeedResult{}, fmt.Errorf("commit users-only seed transaction: %w", err)
	}

	return result, nil
}

func seedOptionsFromEnv() seedOptions {
	return seedOptions{
		Settings:       envBoolWithDefault("MYDUKA_SEED_SETTINGS", true),
		Users:          envBoolWithDefault("MYDUKA_SEED_USERS", true),
		Categories:     envBoolWithDefault("MYDUKA_SEED_CATEGORIES", true),
		Suppliers:      envBoolWithDefault("MYDUKA_SEED_SUPPLIERS", true),
		Products:       envBoolWithDefault("MYDUKA_SEED_PRODUCTS", true),
		PurchaseOrders: envBoolWithDefault("MYDUKA_SEED_PURCHASE_ORDERS", true),
		Sales:          envBoolWithDefault("MYDUKA_SEED_SALES", true),
	}
}

func envBoolWithDefault(key string, defaultValue bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func (s *Service) resolveCategoryIDsByNameTx(tx *sql.Tx, seeds []seedCategory) (map[string]string, error) {
	out := make(map[string]string, len(seeds))
	for _, seed := range seeds {
		var id string
		err := tx.QueryRow(
			`SELECT id FROM categories WHERE LOWER(name) = LOWER(?) AND deleted_at IS NULL LIMIT 1`,
			seed.Name,
		).Scan(&id)
		if err == nil {
			out[seed.Name] = id
			continue
		}
		if err != sql.ErrNoRows {
			return nil, fmt.Errorf("resolve category id for %s: %w", seed.Name, err)
		}
	}
	return out, nil
}

func (s *Service) resolveSupplierIDsByNameTx(tx *sql.Tx, seeds []seedSupplier) (map[string]string, error) {
	out := make(map[string]string, len(seeds))
	for _, seed := range seeds {
		var id string
		err := tx.QueryRow(
			`SELECT id FROM suppliers WHERE LOWER(name) = LOWER(?) AND deleted_at IS NULL LIMIT 1`,
			seed.Name,
		).Scan(&id)
		if err == nil {
			out[seed.Name] = id
			continue
		}
		if err != sql.ErrNoRows {
			return nil, fmt.Errorf("resolve supplier id for %s: %w", seed.Name, err)
		}
	}
	return out, nil
}

type seedSale struct {
	CashierUsername string
	PaymentMethod   string
	CreatedAt       time.Time
	Items           []seedSaleItem
}

type seedSaleItem struct {
	SKU      string
	Quantity int64
}

func (s *Service) seedSalesData() (int, error) {
	db, err := s.getDB()
	if err != nil {
		return 0, err
	}

	var existing int64
	if err := db.QueryRow(`SELECT COUNT(1) FROM sales WHERE deleted_at IS NULL`).Scan(&existing); err != nil {
		return 0, fmt.Errorf("query existing sales for seed: %w", err)
	}
	if existing > 0 {
		return 0, nil
	}

	cashierIDs, err := s.seedStaffIDsByUsername([]string{"alice", "brian"})
	if err != nil {
		return 0, err
	}
	productIDs, err := s.seedProductIDsBySKU([]string{
		"BEV-COKE-500",
		"BEV-WATER-1L",
		"GRC-BREAD-400",
		"GRC-MILK-500",
		"SNK-CRISPS-001",
		"PRC-SOAP-001",
	})
	if err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	seeds := []seedSale{
		{CashierUsername: "alice", PaymentMethod: "cash", CreatedAt: now.Add(-6 * 24 * time.Hour), Items: []seedSaleItem{{SKU: "BEV-COKE-500", Quantity: 2}, {SKU: "SNK-CRISPS-001", Quantity: 1}}},
		{CashierUsername: "brian", PaymentMethod: "mpesa", CreatedAt: now.Add(-6 * 24 * time.Hour), Items: []seedSaleItem{{SKU: "GRC-BREAD-400", Quantity: 2}}},
		{CashierUsername: "alice", PaymentMethod: "cash", CreatedAt: now.Add(-5 * 24 * time.Hour), Items: []seedSaleItem{{SKU: "BEV-WATER-1L", Quantity: 2}, {SKU: "PRC-SOAP-001", Quantity: 1}}},
		{CashierUsername: "brian", PaymentMethod: "card", CreatedAt: now.Add(-4 * 24 * time.Hour), Items: []seedSaleItem{{SKU: "GRC-MILK-500", Quantity: 2}}},
		{CashierUsername: "alice", PaymentMethod: "mpesa", CreatedAt: now.Add(-3 * 24 * time.Hour), Items: []seedSaleItem{{SKU: "BEV-COKE-500", Quantity: 1}, {SKU: "GRC-BREAD-400", Quantity: 1}}},
		{CashierUsername: "brian", PaymentMethod: "cash", CreatedAt: now.Add(-2 * 24 * time.Hour), Items: []seedSaleItem{{SKU: "SNK-CRISPS-001", Quantity: 2}}},
		{CashierUsername: "alice", PaymentMethod: "card", CreatedAt: now.Add(-1 * 24 * time.Hour), Items: []seedSaleItem{{SKU: "PRC-SOAP-001", Quantity: 1}, {SKU: "BEV-WATER-1L", Quantity: 1}}},
		{CashierUsername: "brian", PaymentMethod: "mpesa", CreatedAt: now, Items: []seedSaleItem{{SKU: "BEV-COKE-500", Quantity: 1}, {SKU: "GRC-MILK-500", Quantity: 1}}},
	}

	added := 0
	for _, seed := range seeds {
		cashierID, ok := cashierIDs[seed.CashierUsername]
		if !ok {
			return added, fmt.Errorf("missing seeded cashier %s", seed.CashierUsername)
		}

		items := make([]SaleItemInput, 0, len(seed.Items))
		for _, item := range seed.Items {
			productID, ok := productIDs[item.SKU]
			if !ok {
				return added, fmt.Errorf("missing seeded product sku %s", item.SKU)
			}
			items = append(items, SaleItemInput{
				ProductID: productID,
				Quantity:  item.Quantity,
			})
		}

		sale, err := s.CreateSale(CreateSaleInput{
			CashierStaffID: cashierID,
			PaymentMethod:  seed.PaymentMethod,
			Items:          items,
		})
		if err != nil {
			return added, fmt.Errorf("seed sale create (%s): %w", seed.PaymentMethod, err)
		}

		if err := s.backdateSale(sale.ID, seed.CreatedAt); err != nil {
			return added, err
		}
		added++
	}

	return added, nil
}

func (s *Service) backdateSale(saleID string, createdAt time.Time) error {
	db, err := s.getDB()
	if err != nil {
		return err
	}
	createdAtRaw := createdAt.UTC().Format(time.RFC3339Nano)
	_, err = db.Exec(
		`UPDATE sales SET created_at = ?, updated_at = ? WHERE id = ?`,
		createdAtRaw,
		createdAtRaw,
		saleID,
	)
	if err != nil {
		return fmt.Errorf("backdate sale %s: %w", saleID, err)
	}
	return nil
}

func (s *Service) seedStaffIDsByUsername(usernames []string) (map[string]string, error) {
	db, err := s.getDB()
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(usernames))
	for _, username := range usernames {
		var id string
		err := db.QueryRow(
			`SELECT id FROM staff WHERE username = ? AND deleted_at IS NULL LIMIT 1`,
			username,
		).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("resolve staff by username %s: %w", username, err)
		}
		out[username] = id
	}
	return out, nil
}

func (s *Service) seedProductIDsBySKU(skus []string) (map[string]string, error) {
	db, err := s.getDB()
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(skus))
	for _, sku := range skus {
		var id string
		err := db.QueryRow(
			`SELECT id FROM products WHERE sku = ? AND deleted_at IS NULL LIMIT 1`,
			sku,
		).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("resolve product by sku %s: %w", sku, err)
		}
		out[sku] = id
	}
	return out, nil
}

func (s *Service) seedStaffTx(tx *sql.Tx, seed seedStaff) (bool, error) {
	var existingID string
	err := tx.QueryRow(
		`SELECT id FROM staff WHERE username = ? AND deleted_at IS NULL LIMIT 1`,
		seed.Username,
	).Scan(&existingID)
	if err == nil {
		return false, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("query seed staff %s: %w", seed.Name, err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(seed.Password), bcrypt.DefaultCost)
	if err != nil {
		return false, fmt.Errorf("hash seed staff pin for %s: %w", seed.Name, err)
	}

	id := uuid.NewString()
	now := nowTS()
	_, err = tx.Exec(
		`INSERT INTO staff(id, name, username, role, pin_hash, is_active, created_at, updated_at, deleted_at, device_id, synced_at)
		 VALUES (?, ?, ?, ?, ?, 1, ?, ?, NULL, ?, NULL)`,
		id, seed.Name, seed.Username, seed.Role, string(hash), now, now, s.cfg.DeviceID,
	)
	if err != nil {
		return false, fmt.Errorf("insert seed staff %s: %w", seed.Name, err)
	}

	_ = s.appendSyncLogTx(tx, "staff", id, "insert", map[string]any{
		"id":        id,
		"name":      seed.Name,
		"username":  seed.Username,
		"role":      seed.Role,
		"isActive":  true,
		"createdAt": now,
		"updatedAt": now,
	})
	return true, nil
}

func (s *Service) seedCategoryTx(tx *sql.Tx, seed seedCategory, displayOrder int64) (string, bool, error) {
	var existingID string
	err := tx.QueryRow(
		`SELECT id FROM categories WHERE LOWER(name) = LOWER(?) AND deleted_at IS NULL LIMIT 1`,
		seed.Name,
	).Scan(&existingID)
	if err == nil {
		return existingID, false, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return "", false, fmt.Errorf("query seed category %s: %w", seed.Name, err)
	}

	id := uuid.NewString()
	now := nowTS()
	_, err = tx.Exec(
		`INSERT INTO categories(id, name, emoji, display_order, created_at, updated_at, deleted_at, device_id, synced_at)
		 VALUES (?, ?, ?, ?, ?, ?, NULL, ?, NULL)`,
		id, seed.Name, seed.Emoji, displayOrder, now, now, s.cfg.DeviceID,
	)
	if err != nil {
		return "", false, fmt.Errorf("insert seed category %s: %w", seed.Name, err)
	}
	_ = s.appendSyncLogTx(tx, "categories", id, "insert", map[string]any{
		"id":           id,
		"name":         seed.Name,
		"emoji":        seed.Emoji,
		"displayOrder": displayOrder,
		"createdAt":    now,
		"updatedAt":    now,
	})
	return id, true, nil
}

func (s *Service) seedSupplierTx(tx *sql.Tx, seed seedSupplier) (string, bool, error) {
	var existingID string
	err := tx.QueryRow(
		`SELECT id FROM suppliers WHERE LOWER(name) = LOWER(?) AND deleted_at IS NULL LIMIT 1`,
		seed.Name,
	).Scan(&existingID)
	if err == nil {
		return existingID, false, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return "", false, fmt.Errorf("query seed supplier %s: %w", seed.Name, err)
	}

	id := uuid.NewString()
	now := nowTS()
	_, err = tx.Exec(
		`INSERT INTO suppliers(id, name, phone, email, notes, created_at, updated_at, deleted_at, device_id, synced_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, NULL, ?, NULL)`,
		id, seed.Name, seed.Phone, seed.Email, seed.Notes, now, now, s.cfg.DeviceID,
	)
	if err != nil {
		return "", false, fmt.Errorf("insert seed supplier %s: %w", seed.Name, err)
	}
	_ = s.appendSyncLogTx(tx, "suppliers", id, "insert", map[string]any{
		"id":        id,
		"name":      seed.Name,
		"phone":     seed.Phone,
		"email":     seed.Email,
		"notes":     seed.Notes,
		"createdAt": now,
		"updatedAt": now,
	})
	return id, true, nil
}

func (s *Service) seedProductTx(tx *sql.Tx, seed seedProduct, categoryID, supplierID string) (bool, error) {
	var existingID string
	err := tx.QueryRow(
		`SELECT id FROM products WHERE sku = ? AND deleted_at IS NULL LIMIT 1`,
		seed.SKU,
	).Scan(&existingID)
	if err == nil {
		return false, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("query seed product %s: %w", seed.Name, err)
	}

	id := uuid.NewString()
	now := nowTS()
	_, err = tx.Exec(
		`INSERT INTO products(
			id, name, sku, barcode, category_id, supplier_id, price_cents, starting_stock, stock_qty, reorder_level, is_active,
			created_at, updated_at, deleted_at, device_id, synced_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, NULL, ?, NULL)`,
		id,
		seed.Name,
		seed.SKU,
		nullableString(seed.Barcode),
		categoryID,
		supplierID,
		seed.PriceCents,
		seed.StartingStock,
		seed.StartingStock,
		seed.ReorderLevel,
		now,
		now,
		s.cfg.DeviceID,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return false, nil
		}
		return false, fmt.Errorf("insert seed product %s: %w", seed.Name, err)
	}

	_ = s.appendSyncLogTx(tx, "products", id, "insert", map[string]any{
		"id":            id,
		"name":          seed.Name,
		"sku":           seed.SKU,
		"barcode":       seed.Barcode,
		"categoryId":    categoryID,
		"priceCents":    seed.PriceCents,
		"startingStock": seed.StartingStock,
		"stockQty":      seed.StartingStock,
		"reorderLevel":  seed.ReorderLevel,
		"isActive":      true,
		"createdAt":     now,
		"updatedAt":     now,
	})
	return true, nil
}

func (s *Service) seedPurchaseOrderTx(tx *sql.Tx, supplierID, supplierName string) (bool, error) {
	var existingID string
	err := tx.QueryRow(
		`SELECT id FROM purchase_orders WHERE supplier_id = ? AND status = 'raised' AND notes = 'Initial demo order' AND deleted_at IS NULL LIMIT 1`,
		supplierID,
	).Scan(&existingID)
	if err == nil {
		return false, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("query seed purchase order for %s: %w", supplierName, err)
	}

	id := uuid.NewString()
	now := nowTS()
	_, err = tx.Exec(
		`INSERT INTO purchase_orders(id, supplier_id, status, notes, expected_date, received_at, created_at, updated_at, deleted_at, device_id, synced_at)
		 VALUES (?, ?, 'raised', 'Initial demo order', NULL, NULL, ?, ?, NULL, ?, NULL)`,
		id, supplierID, now, now, s.cfg.DeviceID,
	)
	if err != nil {
		return false, fmt.Errorf("insert seed purchase order for %s: %w", supplierName, err)
	}
	_ = s.appendSyncLogTx(tx, "purchase_orders", id, "insert", map[string]any{
		"id":         id,
		"supplierId": supplierID,
		"status":     "raised",
		"notes":      "Initial demo order",
		"createdAt":  now,
		"updatedAt":  now,
	})
	return true, nil
}
