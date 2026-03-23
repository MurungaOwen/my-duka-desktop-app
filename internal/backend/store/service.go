package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

var errNotStarted = errors.New("backend service not started")

type Service struct {
	cfg Config

	mu      sync.RWMutex
	started bool
	db      *sql.DB
}

func NewService(cfg Config) (*Service, error) {
	if !cfg.Mode.Valid() {
		return nil, fmt.Errorf("invalid deployment mode %q", cfg.Mode)
	}
	if strings.TrimSpace(cfg.DeviceID) == "" {
		cfg.DeviceID = uuid.NewString()
	}
	if strings.TrimSpace(cfg.DeviceName) == "" {
		cfg.DeviceName = "myduka-device"
	}
	if strings.TrimSpace(cfg.DBPath) == "" {
		return nil, errors.New("db path is required")
	}
	return &Service{cfg: cfg}, nil
}

func (s *Service) Start(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(s.cfg.DBPath), 0o755); err != nil {
		return fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", s.cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}

	if err := configureSQLite(db); err != nil {
		_ = db.Close()
		return err
	}

	if err := applyMigrations(db); err != nil {
		_ = db.Close()
		return err
	}

	if err := s.ensureDeviceInfo(db); err != nil {
		_ = db.Close()
		return err
	}

	s.db = db
	s.started = true
	return nil
}

func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started || s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	s.started = false
	return err
}

func (s *Service) Health() (HealthStatus, error) {
	db, err := s.getDB()
	if err != nil {
		return HealthStatus{
			Initialized: false,
			Mode:        s.cfg.Mode,
			SyncEnabled: s.cfg.Mode.SyncEnabled(),
			DBPath:      s.cfg.DBPath,
			DeviceID:    s.cfg.DeviceID,
			DeviceName:  s.cfg.DeviceName,
		}, nil
	}

	if err := db.Ping(); err != nil {
		return HealthStatus{}, fmt.Errorf("ping database: %w", err)
	}

	return HealthStatus{
		Initialized: true,
		Mode:        s.cfg.Mode,
		SyncEnabled: s.cfg.Mode.SyncEnabled(),
		DBPath:      s.cfg.DBPath,
		DeviceID:    s.cfg.DeviceID,
		DeviceName:  s.cfg.DeviceName,
	}, nil
}

func (s *Service) BootstrapBusiness(input BootstrapInput) error {
	db, err := s.getDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin bootstrap transaction: %w", err)
	}

	pairs := []Setting{
		{Key: "business_name", Value: strings.TrimSpace(input.BusinessName)},
		{Key: "location", Value: strings.TrimSpace(input.Location)},
		{Key: "currency", Value: strings.TrimSpace(input.Currency)},
		{Key: "vat_rate", Value: strings.TrimSpace(input.VATRate)},
	}

	for _, kv := range pairs {
		if kv.Key == "" {
			continue
		}
		if err := s.upsertSettingTx(tx, kv); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit bootstrap transaction: %w", err)
	}
	return nil
}

func (s *Service) UpsertSetting(setting Setting) error {
	db, err := s.getDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin setting transaction: %w", err)
	}

	if err := s.upsertSettingTx(tx, setting); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Service) ListSettings() ([]Setting, error) {
	db, err := s.getDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`SELECT key, value FROM settings WHERE deleted_at IS NULL ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("query settings: %w", err)
	}
	defer rows.Close()

	settings := make([]Setting, 0)
	for rows.Next() {
		var setting Setting
		if err := rows.Scan(&setting.Key, &setting.Value); err != nil {
			return nil, fmt.Errorf("scan setting: %w", err)
		}
		settings = append(settings, setting)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate settings: %w", err)
	}
	return settings, nil
}

func (s *Service) CreateStaff(input CreateStaffInput) (Staff, error) {
	db, err := s.getDB()
	if err != nil {
		return Staff{}, err
	}

	role := strings.ToLower(strings.TrimSpace(input.Role))
	if role != "admin" && role != "cashier" {
		return Staff{}, errors.New("role must be admin or cashier")
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Staff{}, errors.New("staff name is required")
	}
	username := strings.ToLower(strings.TrimSpace(input.Username))
	if username == "" {
		return Staff{}, errors.New("username is required")
	}
	if len(strings.TrimSpace(input.Password)) < 4 {
		return Staff{}, errors.New("password must have at least 4 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return Staff{}, fmt.Errorf("hash password: %w", err)
	}

	staff := Staff{
		ID:        uuid.NewString(),
		Name:      name,
		Username:  username,
		Role:      role,
		IsActive:  true,
		CreatedAt: nowTS(),
		UpdatedAt: nowTS(),
	}

	tx, err := db.Begin()
	if err != nil {
		return Staff{}, fmt.Errorf("begin create staff transaction: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO staff (
			id, name, username, role, pin_hash, is_active, created_at, updated_at, deleted_at, device_id, synced_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, NULL)`,
		staff.ID, staff.Name, staff.Username, staff.Role, string(hash), 1, staff.CreatedAt, staff.UpdatedAt, s.cfg.DeviceID,
	)
	if err != nil {
		_ = tx.Rollback()
		return Staff{}, fmt.Errorf("insert staff: %w", err)
	}

	if err := s.appendSyncLogTx(tx, "staff", staff.ID, "insert", staff); err != nil {
		_ = tx.Rollback()
		return Staff{}, err
	}

	if err := tx.Commit(); err != nil {
		return Staff{}, fmt.Errorf("commit create staff transaction: %w", err)
	}

	return staff, nil
}

func (s *Service) ListStaff() ([]Staff, error) {
	db, err := s.getDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT id, name, username, role, is_active, created_at, updated_at
FROM staff
WHERE deleted_at IS NULL
ORDER BY name ASC
`)
	if err != nil {
		return nil, fmt.Errorf("query staff: %w", err)
	}
	defer rows.Close()

	out := make([]Staff, 0)
	for rows.Next() {
		var sRow Staff
		var activeInt int
		if err := rows.Scan(
			&sRow.ID,
			&sRow.Name,
			&sRow.Username,
			&sRow.Role,
			&activeInt,
			&sRow.CreatedAt,
			&sRow.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan staff: %w", err)
		}
		sRow.IsActive = activeInt == 1
		out = append(out, sRow)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate staff: %w", err)
	}
	return out, nil
}

func (s *Service) CreateCategory(input CreateCategoryInput) (Category, error) {
	db, err := s.getDB()
	if err != nil {
		return Category{}, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Category{}, errors.New("category name is required")
	}

	now := nowTS()
	category := Category{
		ID:           uuid.NewString(),
		Name:         name,
		Emoji:        strings.TrimSpace(input.Emoji),
		DisplayOrder: input.DisplayOrder,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	tx, err := db.Begin()
	if err != nil {
		return Category{}, fmt.Errorf("begin create category transaction: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO categories (
			id, name, emoji, display_order, created_at, updated_at, deleted_at, device_id, synced_at
		) VALUES (?, ?, ?, ?, ?, ?, NULL, ?, NULL)`,
		category.ID,
		category.Name,
		category.Emoji,
		category.DisplayOrder,
		category.CreatedAt,
		category.UpdatedAt,
		s.cfg.DeviceID,
	)
	if err != nil {
		_ = tx.Rollback()
		return Category{}, fmt.Errorf("insert category: %w", err)
	}

	if err := s.appendSyncLogTx(tx, "categories", category.ID, "insert", category); err != nil {
		_ = tx.Rollback()
		return Category{}, err
	}

	if err := tx.Commit(); err != nil {
		return Category{}, fmt.Errorf("commit create category transaction: %w", err)
	}
	return category, nil
}

func (s *Service) ListCategories() ([]Category, error) {
	db, err := s.getDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT id, name, emoji, display_order, created_at, updated_at
FROM categories
WHERE deleted_at IS NULL
ORDER BY display_order ASC, name ASC
`)
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	defer rows.Close()

	out := make([]Category, 0)
	for rows.Next() {
		var c Category
		if err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.Emoji,
			&c.DisplayOrder,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}
	return out, nil
}

func (s *Service) UpdateCategory(input UpdateCategoryInput) (Category, error) {
	db, err := s.getDB()
	if err != nil {
		return Category{}, err
	}

	id := strings.TrimSpace(input.ID)
	name := strings.TrimSpace(input.Name)
	if id == "" {
		return Category{}, errors.New("category id is required")
	}
	if name == "" {
		return Category{}, errors.New("category name is required")
	}

	var current Category
	err = db.QueryRow(`
SELECT id, name, emoji, display_order, created_at, updated_at
FROM categories
WHERE id = ? AND deleted_at IS NULL
`, id).Scan(
		&current.ID,
		&current.Name,
		&current.Emoji,
		&current.DisplayOrder,
		&current.CreatedAt,
		&current.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Category{}, errors.New("category not found")
		}
		return Category{}, fmt.Errorf("load category: %w", err)
	}

	updated := current
	updated.Name = name
	updated.Emoji = strings.TrimSpace(input.Emoji)
	updated.DisplayOrder = input.DisplayOrder
	updated.UpdatedAt = nowTS()

	tx, err := db.Begin()
	if err != nil {
		return Category{}, fmt.Errorf("begin update category transaction: %w", err)
	}

	_, err = tx.Exec(`
UPDATE categories
SET name = ?, emoji = ?, display_order = ?, updated_at = ?, device_id = ?, synced_at = NULL
WHERE id = ? AND deleted_at IS NULL
`, updated.Name, updated.Emoji, updated.DisplayOrder, updated.UpdatedAt, s.cfg.DeviceID, updated.ID)
	if err != nil {
		_ = tx.Rollback()
		return Category{}, fmt.Errorf("update category: %w", err)
	}

	if err := s.appendSyncLogTx(tx, "categories", updated.ID, "upsert", updated); err != nil {
		_ = tx.Rollback()
		return Category{}, err
	}

	if err := tx.Commit(); err != nil {
		return Category{}, fmt.Errorf("commit update category transaction: %w", err)
	}
	return updated, nil
}

func (s *Service) DeleteCategory(categoryID string) error {
	db, err := s.getDB()
	if err != nil {
		return err
	}

	categoryID = strings.TrimSpace(categoryID)
	if categoryID == "" {
		return errors.New("category id is required")
	}

	now := nowTS()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin delete category transaction: %w", err)
	}

	res, err := tx.Exec(`
UPDATE categories
SET deleted_at = ?, updated_at = ?, device_id = ?, synced_at = NULL
WHERE id = ? AND deleted_at IS NULL
`, now, now, s.cfg.DeviceID, categoryID)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete category: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete category rows affected: %w", err)
	}
	if affected == 0 {
		_ = tx.Rollback()
		return errors.New("category not found")
	}

	if err := s.appendSyncLogTx(tx, "categories", categoryID, "delete", map[string]any{
		"id":        categoryID,
		"deletedAt": now,
	}); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete category transaction: %w", err)
	}
	return nil
}

func (s *Service) CreateProduct(input CreateProductInput) (Product, error) {
	db, err := s.getDB()
	if err != nil {
		return Product{}, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Product{}, errors.New("product name is required")
	}
	if input.PriceCents < 0 {
		return Product{}, errors.New("price cannot be negative")
	}
	if input.StartingStock < 0 {
		return Product{}, errors.New("starting stock cannot be negative")
	}

	now := nowTS()
	product := Product{
		ID:            uuid.NewString(),
		Name:          name,
		SKU:           strings.TrimSpace(input.SKU),
		Barcode:       strings.TrimSpace(input.Barcode),
		CategoryID:    strings.TrimSpace(input.CategoryID),
		PriceCents:    input.PriceCents,
		StartingStock: input.StartingStock,
		StockQty:      input.StartingStock,
		ReorderLevel:  input.ReorderLevel,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	tx, err := db.Begin()
	if err != nil {
		return Product{}, fmt.Errorf("begin create product transaction: %w", err)
	}

	var categoryID any = nil
	if product.CategoryID != "" {
		categoryID = product.CategoryID
	}

	_, err = tx.Exec(
		`INSERT INTO products (
			id, name, sku, barcode, category_id, supplier_id, price_cents, starting_stock, stock_qty, reorder_level, is_active,
			created_at, updated_at, deleted_at, device_id, synced_at
		) VALUES (?, ?, ?, ?, ?, NULL, ?, ?, ?, ?, ?, ?, ?, NULL, ?, NULL)`,
		product.ID,
		product.Name,
		nullableString(product.SKU),
		nullableString(product.Barcode),
		categoryID,
		product.PriceCents,
		product.StartingStock,
		product.StockQty,
		product.ReorderLevel,
		1,
		product.CreatedAt,
		product.UpdatedAt,
		s.cfg.DeviceID,
	)
	if err != nil {
		_ = tx.Rollback()
		return Product{}, fmt.Errorf("insert product: %w", err)
	}

	if err := s.appendSyncLogTx(tx, "products", product.ID, "insert", product); err != nil {
		_ = tx.Rollback()
		return Product{}, err
	}

	if err := tx.Commit(); err != nil {
		return Product{}, fmt.Errorf("commit create product transaction: %w", err)
	}
	return product, nil
}

func (s *Service) ListProducts() ([]Product, error) {
	db, err := s.getDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT id, name, COALESCE(sku, ''), COALESCE(barcode, ''), COALESCE(category_id, ''), price_cents, starting_stock, stock_qty, reorder_level, is_active, created_at, updated_at
FROM products
WHERE deleted_at IS NULL
ORDER BY name ASC
`)
	if err != nil {
		return nil, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	out := make([]Product, 0)
	for rows.Next() {
		var p Product
		var activeInt int
		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.SKU,
			&p.Barcode,
			&p.CategoryID,
			&p.PriceCents,
			&p.StartingStock,
			&p.StockQty,
			&p.ReorderLevel,
			&activeInt,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		p.IsActive = activeInt == 1
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}
	return out, nil
}

func (s *Service) UpdateProduct(input UpdateProductInput) (Product, error) {
	db, err := s.getDB()
	if err != nil {
		return Product{}, err
	}

	id := strings.TrimSpace(input.ID)
	if id == "" {
		return Product{}, errors.New("product id is required")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Product{}, errors.New("product name is required")
	}
	if input.PriceCents < 0 {
		return Product{}, errors.New("price cannot be negative")
	}
	if input.ReorderLevel < 0 {
		return Product{}, errors.New("reorder level cannot be negative")
	}

	tx, err := db.Begin()
	if err != nil {
		return Product{}, fmt.Errorf("begin update product transaction: %w", err)
	}

	var current Product
	var activeInt int
	err = tx.QueryRow(
		`SELECT id, name, COALESCE(sku,''), COALESCE(barcode,''), COALESCE(category_id,''), price_cents, starting_stock, stock_qty, reorder_level, is_active, created_at, updated_at
		 FROM products
		 WHERE id = ? AND deleted_at IS NULL`,
		id,
	).Scan(
		&current.ID,
		&current.Name,
		&current.SKU,
		&current.Barcode,
		&current.CategoryID,
		&current.PriceCents,
		&current.StartingStock,
		&current.StockQty,
		&current.ReorderLevel,
		&activeInt,
		&current.CreatedAt,
		&current.UpdatedAt,
	)
	if err != nil {
		_ = tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return Product{}, errors.New("product not found")
		}
		return Product{}, fmt.Errorf("query product: %w", err)
	}
	current.IsActive = activeInt == 1

	updated := current
	updated.Name = name
	updated.SKU = strings.TrimSpace(input.SKU)
	updated.Barcode = strings.TrimSpace(input.Barcode)
	updated.CategoryID = strings.TrimSpace(input.CategoryID)
	updated.PriceCents = input.PriceCents
	updated.ReorderLevel = input.ReorderLevel
	updated.IsActive = input.IsActive
	updated.UpdatedAt = nowTS()

	var categoryID any = nil
	if updated.CategoryID != "" {
		categoryID = updated.CategoryID
	}
	active := 0
	if updated.IsActive {
		active = 1
	}

	_, err = tx.Exec(
		`UPDATE products
		 SET name = ?, sku = ?, barcode = ?, category_id = ?, price_cents = ?, reorder_level = ?, is_active = ?, updated_at = ?, device_id = ?, synced_at = NULL
		 WHERE id = ? AND deleted_at IS NULL`,
		updated.Name,
		nullableString(updated.SKU),
		nullableString(updated.Barcode),
		categoryID,
		updated.PriceCents,
		updated.ReorderLevel,
		active,
		updated.UpdatedAt,
		s.cfg.DeviceID,
		updated.ID,
	)
	if err != nil {
		_ = tx.Rollback()
		return Product{}, fmt.Errorf("update product: %w", err)
	}

	if err := s.appendSyncLogTx(tx, "products", updated.ID, "upsert", updated); err != nil {
		_ = tx.Rollback()
		return Product{}, err
	}

	if err := tx.Commit(); err != nil {
		return Product{}, fmt.Errorf("commit update product transaction: %w", err)
	}
	return updated, nil
}

func (s *Service) DeleteProduct(productID string) error {
	db, err := s.getDB()
	if err != nil {
		return err
	}

	id := strings.TrimSpace(productID)
	if id == "" {
		return errors.New("product id is required")
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin delete product transaction: %w", err)
	}

	var exists int
	if err := tx.QueryRow(`SELECT COUNT(1) FROM products WHERE id = ? AND deleted_at IS NULL`, id).Scan(&exists); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("check product existence: %w", err)
	}
	if exists == 0 {
		_ = tx.Rollback()
		return errors.New("product not found")
	}

	now := nowTS()
	_, err = tx.Exec(
		`UPDATE products
		 SET deleted_at = ?, updated_at = ?, is_active = 0, device_id = ?, synced_at = NULL
		 WHERE id = ? AND deleted_at IS NULL`,
		now,
		now,
		s.cfg.DeviceID,
		id,
	)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("soft delete product: %w", err)
	}

	if err := s.appendSyncLogTx(tx, "products", id, "delete", map[string]any{
		"id":        id,
		"updatedAt": now,
		"deletedAt": now,
	}); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete product transaction: %w", err)
	}
	return nil
}

func (s *Service) upsertSettingTx(tx *sql.Tx, setting Setting) error {
	key := strings.TrimSpace(setting.Key)
	if key == "" {
		return errors.New("setting key is required")
	}
	value := strings.TrimSpace(setting.Value)
	now := nowTS()

	id := uuid.NewString()
	_, err := tx.Exec(
		`INSERT INTO settings (
			id, key, value, created_at, updated_at, deleted_at, device_id, synced_at
		) VALUES (?, ?, ?, ?, ?, NULL, ?, NULL)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = excluded.updated_at,
			device_id = excluded.device_id,
			deleted_at = NULL,
			synced_at = NULL`,
		id,
		key,
		value,
		now,
		now,
		s.cfg.DeviceID,
	)
	if err != nil {
		return fmt.Errorf("upsert setting %q: %w", key, err)
	}

	var rowID string
	if err := tx.QueryRow(`SELECT id FROM settings WHERE key = ?`, key).Scan(&rowID); err != nil {
		return fmt.Errorf("resolve setting %q id: %w", key, err)
	}

	if err := s.appendSyncLogTx(tx, "settings", rowID, "upsert", setting); err != nil {
		return err
	}
	return nil
}

func (s *Service) ensureDeviceInfo(db *sql.DB) error {
	now := nowTS()

	_, err := db.Exec(
		`INSERT INTO device_info (id, device_name, mode, server_address, last_sync_at, created_at, updated_at)
		 VALUES (?, ?, ?, NULL, NULL, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
			device_name = excluded.device_name,
			mode = excluded.mode,
			updated_at = excluded.updated_at`,
		s.cfg.DeviceID,
		s.cfg.DeviceName,
		string(s.cfg.Mode),
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("upsert device info: %w", err)
	}
	return nil
}

func (s *Service) appendSyncLogTx(tx *sql.Tx, tableName, recordID, operation string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal sync payload for %s/%s: %w", tableName, recordID, err)
	}
	_, err = tx.Exec(
		`INSERT INTO sync_log (id, table_name, record_id, operation, payload, created_at, synced_at, source_device_id)
		 VALUES (?, ?, ?, ?, ?, ?, NULL, ?)`,
		uuid.NewString(),
		tableName,
		recordID,
		operation,
		string(raw),
		nowTS(),
		s.cfg.DeviceID,
	)
	if err != nil {
		return fmt.Errorf("insert sync log for %s/%s: %w", tableName, recordID, err)
	}
	return nil
}

func (s *Service) getDB() (*sql.DB, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.started || s.db == nil {
		return nil, errNotStarted
	}
	return s.db, nil
}

func configureSQLite(db *sql.DB) error {
	queries := []string{
		`PRAGMA foreign_keys = ON`,
		`PRAGMA journal_mode = WAL`,
		`PRAGMA busy_timeout = 5000`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("exec %q: %w", q, err)
		}
	}
	return nil
}

func nullableString(s string) any {
	v := strings.TrimSpace(s)
	if v == "" {
		return nil
	}
	return v
}

func nowTS() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
