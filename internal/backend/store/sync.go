package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func (s *Service) ListPendingSyncRecords(limit int64) ([]SyncRecord, error) {
	db, err := s.getDB()
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 1000 {
		limit = 200
	}

	rows, err := db.Query(`
SELECT id, table_name, record_id, operation, payload, created_at
FROM sync_log
WHERE synced_at IS NULL
ORDER BY created_at ASC
LIMIT ?
`, limit)
	if err != nil {
		return nil, fmt.Errorf("query pending sync records: %w", err)
	}
	defer rows.Close()

	out := make([]SyncRecord, 0)
	for rows.Next() {
		var r SyncRecord
		if err := rows.Scan(
			&r.ID,
			&r.TableName,
			&r.RecordID,
			&r.Operation,
			&r.Payload,
			&r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan pending sync record: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending sync records: %w", err)
	}
	return out, nil
}

func (s *Service) PendingSyncCount() (int64, error) {
	db, err := s.getDB()
	if err != nil {
		return 0, err
	}
	var count int64
	if err := db.QueryRow(`SELECT COUNT(1) FROM sync_log WHERE synced_at IS NULL`).Scan(&count); err != nil {
		return 0, fmt.Errorf("query pending sync count: %w", err)
	}
	return count, nil
}

func (s *Service) MarkSyncRecordsSynced(recordIDs []string) error {
	db, err := s.getDB()
	if err != nil {
		return err
	}
	if len(recordIDs) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin mark synced transaction: %w", err)
	}

	now := nowTS()
	for _, id := range recordIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, err := tx.Exec(
			`UPDATE sync_log SET synced_at = ? WHERE id = ?`,
			now,
			id,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("mark sync record %s synced: %w", id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit mark synced transaction: %w", err)
	}
	return nil
}

func (s *Service) ApplyIncomingMutations(sourceDeviceID string, mutations []SyncMutation) (SyncPushResponse, error) {
	db, err := s.getDB()
	if err != nil {
		return SyncPushResponse{}, err
	}

	sourceDeviceID = strings.TrimSpace(sourceDeviceID)
	if sourceDeviceID == "" {
		return SyncPushResponse{}, errors.New("source device id is required")
	}

	tx, err := db.Begin()
	if err != nil {
		return SyncPushResponse{}, fmt.Errorf("begin apply incoming mutations transaction: %w", err)
	}

	resp := SyncPushResponse{}
	affectedProducts := make(map[string]struct{})

	for _, m := range mutations {
		mutationSourceDeviceID := sourceDeviceID
		if strings.TrimSpace(m.SourceDeviceID) != "" {
			mutationSourceDeviceID = strings.TrimSpace(m.SourceDeviceID)
		}

		mutationID := strings.TrimSpace(m.MutationID)
		if mutationID == "" {
			_ = tx.Rollback()
			return SyncPushResponse{}, errors.New("mutation id is required")
		}

		insertResult, err := tx.Exec(
			`INSERT OR IGNORE INTO sync_inbox(mutation_id, source_device_id, received_at) VALUES (?, ?, ?)`,
			mutationID,
			mutationSourceDeviceID,
			nowTS(),
		)
		if err != nil {
			_ = tx.Rollback()
			return SyncPushResponse{}, fmt.Errorf("insert sync inbox mutation %s: %w", mutationID, err)
		}
		rowsAffected, _ := insertResult.RowsAffected()
		if rowsAffected == 0 {
			resp.Skipped++
			continue
		}

		if err := s.applySingleMutationTx(tx, mutationSourceDeviceID, m, affectedProducts); err != nil {
			_ = tx.Rollback()
			return SyncPushResponse{}, err
		}
		resp.Applied++
	}

	if err := s.recomputeProductStocksTx(tx, affectedProducts); err != nil {
		_ = tx.Rollback()
		return SyncPushResponse{}, err
	}

	if err := tx.Commit(); err != nil {
		return SyncPushResponse{}, fmt.Errorf("commit apply incoming mutations transaction: %w", err)
	}
	return resp, nil
}

func (s *Service) PullMutationsForDevice(deviceID, since string, limit int64) ([]SyncMutation, error) {
	db, err := s.getDB()
	if err != nil {
		return nil, err
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, errors.New("device id is required")
	}
	if limit <= 0 || limit > 1000 {
		limit = 200
	}

	rows, err := db.Query(`
SELECT id, source_device_id, table_name, record_id, operation, payload, created_at
FROM sync_log
WHERE source_device_id <> ?
  AND created_at > ?
ORDER BY created_at ASC
LIMIT ?
`, deviceID, strings.TrimSpace(since), limit)
	if err != nil {
		return nil, fmt.Errorf("query pull mutations: %w", err)
	}
	defer rows.Close()

	out := make([]SyncMutation, 0)
	for rows.Next() {
		var m SyncMutation
		if err := rows.Scan(
			&m.MutationID,
			&m.SourceDeviceID,
			&m.TableName,
			&m.RecordID,
			&m.Operation,
			&m.Payload,
			&m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan pull mutation: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pull mutations: %w", err)
	}
	return out, nil
}

func (s *Service) applySingleMutationTx(tx *sql.Tx, sourceDeviceID string, m SyncMutation, affectedProducts map[string]struct{}) error {
	tableName := strings.TrimSpace(m.TableName)
	recordID := strings.TrimSpace(m.RecordID)
	operation := strings.TrimSpace(m.Operation)
	payload := strings.TrimSpace(m.Payload)
	createdAt := strings.TrimSpace(m.CreatedAt)
	if createdAt == "" {
		createdAt = nowTS()
	}

	switch tableName {
	case "settings":
		var p struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return fmt.Errorf("unmarshal settings payload: %w", err)
		}
		now := nowTS()
		id := recordID
		if id == "" {
			id = uuidOrFallback()
		}
		if _, err := tx.Exec(
			`INSERT INTO settings(id, key, value, created_at, updated_at, deleted_at, device_id, synced_at)
			 VALUES (?, ?, ?, ?, ?, NULL, ?, ?)
			 ON CONFLICT(key) DO UPDATE SET
			   value = excluded.value,
			   updated_at = excluded.updated_at,
			   deleted_at = NULL,
			   device_id = excluded.device_id`,
			id, p.Key, p.Value, now, now, sourceDeviceID, now,
		); err != nil {
			return fmt.Errorf("apply settings mutation: %w", err)
		}
	case "categories":
		var p struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			Emoji        string `json:"emoji"`
			DisplayOrder int64  `json:"displayOrder"`
			CreatedAt    string `json:"createdAt"`
			UpdatedAt    string `json:"updatedAt"`
		}
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return fmt.Errorf("unmarshal categories payload: %w", err)
		}
		if p.ID == "" {
			p.ID = recordID
		}
		if p.ID == "" {
			return errors.New("categories mutation missing id")
		}
		if p.CreatedAt == "" {
			p.CreatedAt = createdAt
		}
		if p.UpdatedAt == "" {
			p.UpdatedAt = createdAt
		}
		if _, err := tx.Exec(
			`INSERT INTO categories(id, name, emoji, display_order, created_at, updated_at, deleted_at, device_id, synced_at)
			 VALUES (?, ?, ?, ?, ?, ?, NULL, ?, ?)
			 ON CONFLICT(id) DO UPDATE SET
			   name = excluded.name,
			   emoji = excluded.emoji,
			   display_order = excluded.display_order,
			   updated_at = excluded.updated_at,
			   deleted_at = NULL,
			   device_id = excluded.device_id`,
			p.ID, p.Name, p.Emoji, p.DisplayOrder, p.CreatedAt, p.UpdatedAt, sourceDeviceID, nowTS(),
		); err != nil {
			return fmt.Errorf("apply categories mutation: %w", err)
		}
	case "products":
		var p struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			SKU           string `json:"sku"`
			Barcode       string `json:"barcode"`
			CategoryID    string `json:"categoryId"`
			PriceCents    int64  `json:"priceCents"`
			StartingStock int64  `json:"startingStock"`
			StockQty      int64  `json:"stockQty"`
			ReorderLevel  int64  `json:"reorderLevel"`
			IsActive      bool   `json:"isActive"`
			CreatedAt     string `json:"createdAt"`
			UpdatedAt     string `json:"updatedAt"`
		}
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return fmt.Errorf("unmarshal products payload: %w", err)
		}
		if p.ID == "" {
			p.ID = recordID
		}
		if p.ID == "" {
			return errors.New("products mutation missing id")
		}
		if p.CreatedAt == "" {
			p.CreatedAt = createdAt
		}
		if p.UpdatedAt == "" {
			p.UpdatedAt = createdAt
		}
		active := 0
		if p.IsActive {
			active = 1
		}
		if _, err := tx.Exec(
			`INSERT INTO products(id, name, sku, barcode, category_id, supplier_id, price_cents, starting_stock, stock_qty, reorder_level, is_active, created_at, updated_at, deleted_at, device_id, synced_at)
			 VALUES (?, ?, ?, ?, ?, NULL, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?)
			 ON CONFLICT(id) DO UPDATE SET
			   name = excluded.name,
			   sku = excluded.sku,
			   barcode = excluded.barcode,
			   category_id = excluded.category_id,
			   price_cents = excluded.price_cents,
			   starting_stock = excluded.starting_stock,
			   reorder_level = excluded.reorder_level,
			   is_active = excluded.is_active,
			   updated_at = excluded.updated_at,
			   deleted_at = NULL,
			   device_id = excluded.device_id`,
			p.ID,
			p.Name,
			nullableString(p.SKU),
			nullableString(p.Barcode),
			nullableString(p.CategoryID),
			p.PriceCents,
			p.StartingStock,
			p.StockQty,
			p.ReorderLevel,
			active,
			p.CreatedAt,
			p.UpdatedAt,
			sourceDeviceID,
			nowTS(),
		); err != nil {
			return fmt.Errorf("apply products mutation: %w", err)
		}
	case "stock_transactions":
		var p struct {
			ID        string `json:"id"`
			ProductID string `json:"product_id"`
			QtyChange int64  `json:"qty_change"`
			Reason    string `json:"reason"`
			RefType   string `json:"ref_type"`
			RefID     string `json:"ref_id"`
		}
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return fmt.Errorf("unmarshal stock_transactions payload: %w", err)
		}
		if p.ID == "" {
			p.ID = recordID
		}
		if p.ID == "" || p.ProductID == "" {
			return errors.New("stock_transactions mutation missing id or product_id")
		}
		if _, err := tx.Exec(
			`INSERT INTO stock_transactions(id, product_id, qty_change, reason, ref_type, ref_id, created_at, updated_at, deleted_at, device_id, synced_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?)
			 ON CONFLICT(id) DO NOTHING`,
			p.ID,
			p.ProductID,
			p.QtyChange,
			p.Reason,
			p.RefType,
			p.RefID,
			createdAt,
			createdAt,
			sourceDeviceID,
			nowTS(),
		); err != nil {
			return fmt.Errorf("apply stock_transactions mutation: %w", err)
		}
		affectedProducts[p.ProductID] = struct{}{}
	case "staff":
		var p struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Username  string `json:"username"`
			Role      string `json:"role"`
			IsActive  bool   `json:"isActive"`
			CreatedAt string `json:"createdAt"`
			UpdatedAt string `json:"updatedAt"`
		}
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return fmt.Errorf("unmarshal staff payload: %w", err)
		}
		if p.ID == "" {
			p.ID = recordID
		}
		if p.ID == "" {
			return errors.New("staff mutation missing id")
		}
		if p.CreatedAt == "" {
			p.CreatedAt = createdAt
		}
		if p.UpdatedAt == "" {
			p.UpdatedAt = createdAt
		}
		active := 0
		if p.IsActive {
			active = 1
		}
		// Remote sync does not include pin hash in payload, preserve existing hash if present.
		if _, err := tx.Exec(
			`INSERT INTO staff(id, name, username, role, pin_hash, is_active, created_at, updated_at, deleted_at, device_id, synced_at)
			 VALUES (?, ?, ?, ?, '', ?, ?, ?, NULL, ?, ?)
			 ON CONFLICT(id) DO UPDATE SET
			   name = excluded.name,
			   username = excluded.username,
			   role = excluded.role,
			   is_active = excluded.is_active,
			   updated_at = excluded.updated_at,
			   deleted_at = NULL,
			   device_id = excluded.device_id`,
			p.ID, p.Name, p.Username, p.Role, active, p.CreatedAt, p.UpdatedAt, sourceDeviceID, nowTS(),
		); err != nil {
			return fmt.Errorf("apply staff mutation: %w", err)
		}
	case "sales":
		var p struct {
			ID             string `json:"id"`
			CashierStaffID string `json:"cashierStaffId"`
			PaymentMethod  string `json:"paymentMethod"`
			Status         string `json:"status"`
			SubtotalCents  int64  `json:"subtotalCents"`
			VATCents       int64  `json:"vatCents"`
			TotalCents     int64  `json:"totalCents"`
			CreatedAt      string `json:"createdAt"`
		}
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return fmt.Errorf("unmarshal sales payload: %w", err)
		}
		if p.ID == "" {
			p.ID = recordID
		}
		if p.ID == "" {
			return errors.New("sales mutation missing id")
		}
		if p.CreatedAt == "" {
			p.CreatedAt = createdAt
		}
		if p.Status == "" {
			p.Status = "completed"
		}
		if _, err := tx.Exec(
			`INSERT INTO sales(id, cashier_staff_id, payment_method, status, subtotal_cents, vat_cents, total_cents, created_at, updated_at, deleted_at, device_id, synced_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?)
			 ON CONFLICT(id) DO UPDATE SET
			   cashier_staff_id = excluded.cashier_staff_id,
			   payment_method = excluded.payment_method,
			   status = excluded.status,
			   subtotal_cents = excluded.subtotal_cents,
			   vat_cents = excluded.vat_cents,
			   total_cents = excluded.total_cents,
			   updated_at = excluded.updated_at,
			   deleted_at = NULL,
			   device_id = excluded.device_id`,
			p.ID,
			p.CashierStaffID,
			p.PaymentMethod,
			p.Status,
			p.SubtotalCents,
			p.VATCents,
			p.TotalCents,
			p.CreatedAt,
			createdAt,
			sourceDeviceID,
			nowTS(),
		); err != nil {
			return fmt.Errorf("apply sales mutation: %w", err)
		}
	case "sale_items":
		var p struct {
			ID             string `json:"id"`
			SaleID         string `json:"saleId"`
			ProductID      string `json:"productId"`
			Quantity       int64  `json:"quantity"`
			UnitPriceCents int64  `json:"unitPriceCents"`
			LineTotalCents int64  `json:"lineTotalCents"`
		}
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return fmt.Errorf("unmarshal sale_items payload: %w", err)
		}
		if p.ID == "" {
			p.ID = recordID
		}
		if p.ID == "" || p.SaleID == "" || p.ProductID == "" {
			return errors.New("sale_items mutation missing required ids")
		}
		if _, err := tx.Exec(
			`INSERT INTO sale_items(id, sale_id, product_id, quantity, unit_price_cents, discount_cents, line_total_cents, created_at, updated_at, deleted_at, device_id, synced_at)
			 VALUES (?, ?, ?, ?, ?, 0, ?, ?, ?, NULL, ?, ?)
			 ON CONFLICT(id) DO UPDATE SET
			   sale_id = excluded.sale_id,
			   product_id = excluded.product_id,
			   quantity = excluded.quantity,
			   unit_price_cents = excluded.unit_price_cents,
			   line_total_cents = excluded.line_total_cents,
			   updated_at = excluded.updated_at,
			   deleted_at = NULL,
			   device_id = excluded.device_id`,
			p.ID,
			p.SaleID,
			p.ProductID,
			p.Quantity,
			p.UnitPriceCents,
			p.LineTotalCents,
			createdAt,
			createdAt,
			sourceDeviceID,
			nowTS(),
		); err != nil {
			return fmt.Errorf("apply sale_items mutation: %w", err)
		}
	case "suppliers", "purchase_orders":
		// Accepted as pass-through for forward compatibility until CRUD is fully implemented.
	default:
		return fmt.Errorf("unsupported mutation table: %s", tableName)
	}

	if _, err := tx.Exec(
		`INSERT OR IGNORE INTO sync_log(id, table_name, record_id, operation, payload, created_at, synced_at, source_device_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		m.MutationID,
		tableName,
		recordID,
		operation,
		payload,
		createdAt,
		nowTS(),
		sourceDeviceID,
	); err != nil {
		return fmt.Errorf("append server sync log for mutation %s: %w", m.MutationID, err)
	}

	return nil
}

func uuidOrFallback() string {
	return uuid.NewString()
}
