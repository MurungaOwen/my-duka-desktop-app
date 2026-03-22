package store

import (
	"database/sql"
	"fmt"
	"strings"
)

type migration struct {
	id  int
	sql string
}

var migrations = []migration{
	{
		id: 1,
		sql: `
CREATE TABLE IF NOT EXISTS schema_migrations (
  id INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS device_info (
  id TEXT PRIMARY KEY,
  device_name TEXT NOT NULL,
  mode TEXT NOT NULL,
  server_address TEXT,
  last_sync_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS settings (
  id TEXT PRIMARY KEY,
  key TEXT NOT NULL UNIQUE,
  value TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT,
  device_id TEXT NOT NULL,
  synced_at TEXT
);

CREATE TABLE IF NOT EXISTS categories (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  emoji TEXT NOT NULL DEFAULT '',
  display_order INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT,
  device_id TEXT NOT NULL,
  synced_at TEXT
);

CREATE TABLE IF NOT EXISTS suppliers (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  phone TEXT NOT NULL DEFAULT '',
  email TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT,
  device_id TEXT NOT NULL,
  synced_at TEXT
);

CREATE TABLE IF NOT EXISTS products (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  sku TEXT UNIQUE,
  barcode TEXT UNIQUE,
  category_id TEXT,
  supplier_id TEXT,
  price_cents INTEGER NOT NULL,
  starting_stock INTEGER NOT NULL DEFAULT 0,
  stock_qty INTEGER NOT NULL DEFAULT 0,
  reorder_level INTEGER NOT NULL DEFAULT 0,
  is_active INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT,
  device_id TEXT NOT NULL,
  synced_at TEXT,
  FOREIGN KEY (category_id) REFERENCES categories(id),
  FOREIGN KEY (supplier_id) REFERENCES suppliers(id)
);

CREATE INDEX IF NOT EXISTS idx_products_category_id ON products(category_id);
CREATE INDEX IF NOT EXISTS idx_products_name ON products(name);

CREATE TABLE IF NOT EXISTS stock_transactions (
  id TEXT PRIMARY KEY,
  product_id TEXT NOT NULL,
  qty_change INTEGER NOT NULL,
  reason TEXT NOT NULL,
  ref_type TEXT NOT NULL DEFAULT '',
  ref_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT,
  device_id TEXT NOT NULL,
  synced_at TEXT,
  FOREIGN KEY (product_id) REFERENCES products(id)
);

CREATE INDEX IF NOT EXISTS idx_stock_transactions_product_id ON stock_transactions(product_id);

CREATE TABLE IF NOT EXISTS staff (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  username TEXT NOT NULL UNIQUE,
  role TEXT NOT NULL CHECK(role IN ('admin','cashier')),
  pin_hash TEXT NOT NULL,
  is_active INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT,
  device_id TEXT NOT NULL,
  synced_at TEXT
);

CREATE TABLE IF NOT EXISTS sales (
  id TEXT PRIMARY KEY,
  cashier_staff_id TEXT NOT NULL,
  payment_method TEXT NOT NULL CHECK(payment_method IN ('cash','mpesa','card')),
  status TEXT NOT NULL DEFAULT 'completed',
  subtotal_cents INTEGER NOT NULL,
  vat_cents INTEGER NOT NULL,
  total_cents INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT,
  device_id TEXT NOT NULL,
  synced_at TEXT,
  FOREIGN KEY (cashier_staff_id) REFERENCES staff(id)
);

CREATE INDEX IF NOT EXISTS idx_sales_created_at ON sales(created_at);
CREATE INDEX IF NOT EXISTS idx_sales_cashier_staff_id ON sales(cashier_staff_id);

CREATE TABLE IF NOT EXISTS sale_items (
  id TEXT PRIMARY KEY,
  sale_id TEXT NOT NULL,
  product_id TEXT NOT NULL,
  quantity INTEGER NOT NULL,
  unit_price_cents INTEGER NOT NULL,
  discount_cents INTEGER NOT NULL DEFAULT 0,
  line_total_cents INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT,
  device_id TEXT NOT NULL,
  synced_at TEXT,
  FOREIGN KEY (sale_id) REFERENCES sales(id),
  FOREIGN KEY (product_id) REFERENCES products(id)
);

CREATE INDEX IF NOT EXISTS idx_sale_items_sale_id ON sale_items(sale_id);

CREATE TABLE IF NOT EXISTS purchase_orders (
  id TEXT PRIMARY KEY,
  supplier_id TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'raised',
  notes TEXT NOT NULL DEFAULT '',
  expected_date TEXT,
  received_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT,
  device_id TEXT NOT NULL,
  synced_at TEXT,
  FOREIGN KEY (supplier_id) REFERENCES suppliers(id)
);

CREATE TABLE IF NOT EXISTS sync_log (
  id TEXT PRIMARY KEY,
  table_name TEXT NOT NULL,
  record_id TEXT NOT NULL,
  operation TEXT NOT NULL,
  payload TEXT NOT NULL,
  created_at TEXT NOT NULL,
  synced_at TEXT,
  source_device_id TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_sync_log_synced_at ON sync_log(synced_at, created_at);
`,
	},
	{
		id: 2,
		sql: `
CREATE TABLE IF NOT EXISTS sync_inbox (
  mutation_id TEXT PRIMARY KEY,
  source_device_id TEXT NOT NULL,
  received_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sync_log_created_at ON sync_log(created_at);
CREATE INDEX IF NOT EXISTS idx_sync_log_source_device_id ON sync_log(source_device_id, created_at);
`,
	},
}

func applyMigrations(db *sql.DB) error {
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
  id INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL
)`); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	for _, m := range migrations {
		var exists int
		if err := db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE id = ?`, m.id).Scan(&exists); err != nil {
			return fmt.Errorf("check migration %d: %w", m.id, err)
		}
		if exists > 0 {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", m.id, err)
		}

		if _, err := tx.Exec(m.sql); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", m.id, err)
		}

		if _, err := tx.Exec(
			`INSERT INTO schema_migrations(id, applied_at) VALUES (?, ?)`,
			m.id,
			nowTS(),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", m.id, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", m.id, err)
		}
	}

	if err := applyUnifiedMigration3(db); err != nil {
		return err
	}
	if err := applyUnifiedMigration4(db); err != nil {
		return err
	}

	return nil
}

func applyUnifiedMigration3(db *sql.DB) error {
	var exists int
	if err := db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE id = 3`).Scan(&exists); err != nil {
		return fmt.Errorf("check migration 3: %w", err)
	}
	if exists > 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration 3: %w", err)
	}

	hasUsername, err := hasColumnTx(tx, "staff", "username")
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if !hasUsername {
		if _, err := tx.Exec(`ALTER TABLE staff ADD COLUMN username TEXT NOT NULL DEFAULT ''`); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration 3 add staff.username: %w", err)
		}
		if _, err := tx.Exec(`UPDATE staff SET username = LOWER(REPLACE(name, ' ', '_')) WHERE username = ''`); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration 3 backfill staff.username: %w", err)
		}
	}

	if _, err := tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_staff_username ON staff(username)`); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("migration 3 ensure idx_staff_username: %w", err)
	}
	if _, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS sync_inbox (
  mutation_id TEXT PRIMARY KEY,
  source_device_id TEXT NOT NULL,
  received_at TEXT NOT NULL
)`); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("migration 3 ensure sync_inbox table: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_sync_log_created_at ON sync_log(created_at)`); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("migration 3 ensure idx_sync_log_created_at: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_sync_log_source_device_id ON sync_log(source_device_id, created_at)`); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("migration 3 ensure idx_sync_log_source_device_id: %w", err)
	}

	if _, err := tx.Exec(
		`INSERT INTO schema_migrations(id, applied_at) VALUES (?, ?)`,
		3,
		nowTS(),
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("record migration 3: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration 3: %w", err)
	}

	return nil
}

func hasColumnTx(tx *sql.Tx, tableName, columnName string) (bool, error) {
	rows, err := tx.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, tableName))
	if err != nil {
		return false, fmt.Errorf("query table info for %s: %w", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var colType string
		var notNull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			return false, fmt.Errorf("scan pragma table_info(%s): %w", tableName, err)
		}
		if strings.EqualFold(name, columnName) {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate pragma table_info(%s): %w", tableName, err)
	}
	return false, nil
}

func applyUnifiedMigration4(db *sql.DB) error {
	var exists int
	if err := db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE id = 4`).Scan(&exists); err != nil {
		return fmt.Errorf("check migration 4: %w", err)
	}
	if exists > 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration 4: %w", err)
	}

	if _, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS sale_payments (
  id TEXT PRIMARY KEY,
  sale_id TEXT NOT NULL,
  provider TEXT NOT NULL,
  reference TEXT NOT NULL UNIQUE,
  amount_cents INTEGER NOT NULL,
  currency TEXT NOT NULL DEFAULT 'KES',
  created_at TEXT NOT NULL,
  verified_by_staff_id TEXT NOT NULL DEFAULT '',
  device_id TEXT NOT NULL,
  synced_at TEXT,
  FOREIGN KEY (sale_id) REFERENCES sales(id)
)`); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("migration 4 create sale_payments: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_sale_payments_sale_id ON sale_payments(sale_id)`); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("migration 4 create idx_sale_payments_sale_id: %w", err)
	}

	if _, err := tx.Exec(
		`INSERT INTO schema_migrations(id, applied_at) VALUES (?, ?)`,
		4,
		nowTS(),
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("record migration 4: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration 4: %w", err)
	}
	return nil
}
