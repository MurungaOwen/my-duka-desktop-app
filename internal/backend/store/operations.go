package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func (s *Service) VerifyStaffPIN(input PINVerificationInput) (bool, error) {
	db, err := s.getDB()
	if err != nil {
		return false, err
	}

	staffID := strings.TrimSpace(input.StaffID)
	if staffID == "" {
		return false, errors.New("staff id is required")
	}

	var pinHash string
	var active int
	err = db.QueryRow(
		`SELECT pin_hash, is_active FROM staff WHERE id = ? AND deleted_at IS NULL`,
		staffID,
	).Scan(&pinHash, &active)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, errors.New("staff not found")
		}
		return false, fmt.Errorf("query staff pin: %w", err)
	}
	if active != 1 {
		return false, errors.New("staff is inactive")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(pinHash), []byte(input.PIN)); err != nil {
		return false, nil
	}
	return true, nil
}

func (s *Service) AuthenticateStaff(input StaffLoginInput) (Staff, error) {
	db, err := s.getDB()
	if err != nil {
		return Staff{}, err
	}

	username := strings.ToLower(strings.TrimSpace(input.Username))
	if username == "" {
		return Staff{}, errors.New("username is required")
	}
	if strings.TrimSpace(input.Password) == "" {
		return Staff{}, errors.New("password is required")
	}

	var staff Staff
	var hash string
	var active int
	err = db.QueryRow(
		`SELECT id, name, username, role, pin_hash, is_active, created_at, updated_at
		 FROM staff
		 WHERE username = ? AND deleted_at IS NULL`,
		username,
	).Scan(
		&staff.ID,
		&staff.Name,
		&staff.Username,
		&staff.Role,
		&hash,
		&active,
		&staff.CreatedAt,
		&staff.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Staff{}, errors.New("invalid username or password")
		}
		return Staff{}, fmt.Errorf("query staff auth: %w", err)
	}
	if active != 1 {
		return Staff{}, errors.New("staff is inactive")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(input.Password)); err != nil {
		return Staff{}, errors.New("invalid username or password")
	}
	staff.IsActive = true
	return staff, nil
}

func (s *Service) CreateSale(input CreateSaleInput) (SaleDetail, error) {
	db, err := s.getDB()
	if err != nil {
		return SaleDetail{}, err
	}

	paymentMethod := strings.ToLower(strings.TrimSpace(input.PaymentMethod))
	if paymentMethod != "cash" && paymentMethod != "mpesa" && paymentMethod != "card" {
		return SaleDetail{}, errors.New("payment method must be cash, mpesa, or card")
	}

	cashierID := strings.TrimSpace(input.CashierStaffID)
	paymentRef := strings.TrimSpace(input.PaymentRef)
	if cashierID == "" {
		return SaleDetail{}, errors.New("cashier staff id is required")
	}

	if len(input.Items) == 0 {
		return SaleDetail{}, errors.New("sale must include at least one item")
	}

	tx, err := db.Begin()
	if err != nil {
		return SaleDetail{}, fmt.Errorf("begin create sale transaction: %w", err)
	}

	if err := s.ensureStaffActiveTx(tx, cashierID); err != nil {
		_ = tx.Rollback()
		return SaleDetail{}, err
	}

	now := nowTS()
	sale := Sale{
		ID:             uuid.NewString(),
		CashierStaffID: cashierID,
		PaymentMethod:  paymentMethod,
		Status:         "completed",
		CreatedAt:      now,
	}

	saleItems := make([]SaleItem, 0, len(input.Items))
	var subtotal int64
	affectedProducts := make(map[string]struct{})

	for _, item := range input.Items {
		productID := strings.TrimSpace(item.ProductID)
		if productID == "" || item.Quantity <= 0 {
			_ = tx.Rollback()
			return SaleDetail{}, errors.New("each item must include a valid product id and quantity")
		}

		var productName string
		var unitPriceCents int64
		var stockQty int64
		var activeInt int
		err := tx.QueryRow(
			`SELECT name, price_cents, stock_qty, is_active FROM products WHERE id = ? AND deleted_at IS NULL`,
			productID,
		).Scan(&productName, &unitPriceCents, &stockQty, &activeInt)
		if err != nil {
			_ = tx.Rollback()
			if errors.Is(err, sql.ErrNoRows) {
				return SaleDetail{}, fmt.Errorf("product not found: %s", productID)
			}
			return SaleDetail{}, fmt.Errorf("query product %s: %w", productID, err)
		}
		if activeInt != 1 {
			_ = tx.Rollback()
			return SaleDetail{}, fmt.Errorf("product is inactive: %s", productName)
		}
		if stockQty < item.Quantity {
			_ = tx.Rollback()
			return SaleDetail{}, fmt.Errorf("insufficient stock for %s", productName)
		}

		lineTotal := unitPriceCents * item.Quantity
		subtotal += lineTotal

		saleItems = append(saleItems, SaleItem{
			ID:             uuid.NewString(),
			SaleID:         sale.ID,
			ProductID:      productID,
			Quantity:       item.Quantity,
			UnitPriceCents: unitPriceCents,
			LineTotalCents: lineTotal,
		})
		affectedProducts[productID] = struct{}{}
	}

	vatRate, err := s.getVATRateTx(tx)
	if err != nil {
		_ = tx.Rollback()
		return SaleDetail{}, err
	}

	vatCents := (subtotal * vatRate) / 10000
	totalCents := subtotal + vatCents
	sale.SubtotalCents = subtotal
	sale.VATCents = vatCents
	sale.TotalCents = totalCents

	_, err = tx.Exec(
		`INSERT INTO sales (
			id, cashier_staff_id, payment_method, status, subtotal_cents, vat_cents, total_cents,
			created_at, updated_at, deleted_at, device_id, synced_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, NULL)`,
		sale.ID,
		sale.CashierStaffID,
		sale.PaymentMethod,
		sale.Status,
		sale.SubtotalCents,
		sale.VATCents,
		sale.TotalCents,
		now,
		now,
		s.cfg.DeviceID,
	)
	if err != nil {
		_ = tx.Rollback()
		return SaleDetail{}, fmt.Errorf("insert sale: %w", err)
	}
	if err := s.appendSyncLogTx(tx, "sales", sale.ID, "insert", sale); err != nil {
		_ = tx.Rollback()
		return SaleDetail{}, err
	}

	for _, item := range saleItems {
		_, err = tx.Exec(
			`INSERT INTO sale_items (
				id, sale_id, product_id, quantity, unit_price_cents, discount_cents, line_total_cents,
				created_at, updated_at, deleted_at, device_id, synced_at
			) VALUES (?, ?, ?, ?, ?, 0, ?, ?, ?, NULL, ?, NULL)`,
			item.ID,
			item.SaleID,
			item.ProductID,
			item.Quantity,
			item.UnitPriceCents,
			item.LineTotalCents,
			now,
			now,
			s.cfg.DeviceID,
		)
		if err != nil {
			_ = tx.Rollback()
			return SaleDetail{}, fmt.Errorf("insert sale item: %w", err)
		}
		if err := s.appendSyncLogTx(tx, "sale_items", item.ID, "insert", item); err != nil {
			_ = tx.Rollback()
			return SaleDetail{}, err
		}

		stockTxnID := uuid.NewString()
		_, err = tx.Exec(
			`INSERT INTO stock_transactions (
				id, product_id, qty_change, reason, ref_type, ref_id, created_at, updated_at, deleted_at, device_id, synced_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, NULL)`,
			stockTxnID,
			item.ProductID,
			-item.Quantity,
			"sale",
			"sale",
			sale.ID,
			now,
			now,
			s.cfg.DeviceID,
		)
		if err != nil {
			_ = tx.Rollback()
			return SaleDetail{}, fmt.Errorf("insert stock transaction: %w", err)
		}
		if err := s.appendSyncLogTx(tx, "stock_transactions", stockTxnID, "insert", map[string]any{
			"id":         stockTxnID,
			"product_id": item.ProductID,
			"qty_change": -item.Quantity,
			"reason":     "sale",
			"ref_type":   "sale",
			"ref_id":     sale.ID,
		}); err != nil {
			_ = tx.Rollback()
			return SaleDetail{}, err
		}
	}

	if err := s.recomputeProductStocksTx(tx, affectedProducts); err != nil {
		_ = tx.Rollback()
		return SaleDetail{}, err
	}

	if paymentMethod == "mpesa" && paymentRef != "" {
		paymentID := uuid.NewString()
		_, err = tx.Exec(
			`INSERT INTO sale_payments(id, sale_id, provider, reference, amount_cents, currency, created_at, verified_by_staff_id, device_id, synced_at)
			 VALUES (?, ?, 'mpesa', ?, ?, 'KES', ?, ?, ?, NULL)`,
			paymentID,
			sale.ID,
			paymentRef,
			sale.TotalCents,
			now,
			cashierID,
			s.cfg.DeviceID,
		)
		if err != nil {
			_ = tx.Rollback()
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				return SaleDetail{}, errors.New("payment reference already used by another sale")
			}
			return SaleDetail{}, fmt.Errorf("insert sale payment binding: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return SaleDetail{}, fmt.Errorf("commit create sale transaction: %w", err)
	}

	return SaleDetail{Sale: sale, Items: saleItems}, nil
}

func (s *Service) ListSales(limit int64) ([]Sale, error) {
	db, err := s.getDB()
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	rows, err := db.Query(`
SELECT id, cashier_staff_id, payment_method, status, subtotal_cents, vat_cents, total_cents, created_at
FROM sales
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT ?
`, limit)
	if err != nil {
		return nil, fmt.Errorf("query sales: %w", err)
	}
	defer rows.Close()

	out := make([]Sale, 0)
	for rows.Next() {
		var sale Sale
		if err := rows.Scan(
			&sale.ID,
			&sale.CashierStaffID,
			&sale.PaymentMethod,
			&sale.Status,
			&sale.SubtotalCents,
			&sale.VATCents,
			&sale.TotalCents,
			&sale.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan sale: %w", err)
		}
		out = append(out, sale)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sales: %w", err)
	}
	return out, nil
}

func (s *Service) GetSaleDetail(saleID string) (SaleDetail, error) {
	db, err := s.getDB()
	if err != nil {
		return SaleDetail{}, err
	}

	id := strings.TrimSpace(saleID)
	if id == "" {
		return SaleDetail{}, errors.New("sale id is required")
	}

	var out SaleDetail
	err = db.QueryRow(
		`SELECT id, cashier_staff_id, payment_method, status, subtotal_cents, vat_cents, total_cents, created_at
		 FROM sales WHERE id = ? AND deleted_at IS NULL`,
		id,
	).Scan(
		&out.ID,
		&out.CashierStaffID,
		&out.PaymentMethod,
		&out.Status,
		&out.SubtotalCents,
		&out.VATCents,
		&out.TotalCents,
		&out.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SaleDetail{}, errors.New("sale not found")
		}
		return SaleDetail{}, fmt.Errorf("query sale: %w", err)
	}

	rows, err := db.Query(
		`SELECT id, sale_id, product_id, quantity, unit_price_cents, line_total_cents
		 FROM sale_items
		 WHERE sale_id = ? AND deleted_at IS NULL
		 ORDER BY created_at ASC`,
		id,
	)
	if err != nil {
		return SaleDetail{}, fmt.Errorf("query sale items: %w", err)
	}
	defer rows.Close()

	out.Items = make([]SaleItem, 0)
	for rows.Next() {
		var item SaleItem
		if err := rows.Scan(
			&item.ID,
			&item.SaleID,
			&item.ProductID,
			&item.Quantity,
			&item.UnitPriceCents,
			&item.LineTotalCents,
		); err != nil {
			return SaleDetail{}, fmt.Errorf("scan sale item: %w", err)
		}
		out.Items = append(out.Items, item)
	}
	if err := rows.Err(); err != nil {
		return SaleDetail{}, fmt.Errorf("iterate sale items: %w", err)
	}
	return out, nil
}

func (s *Service) AdjustStock(input StockAdjustmentInput) error {
	db, err := s.getDB()
	if err != nil {
		return err
	}

	productID := strings.TrimSpace(input.ProductID)
	if productID == "" {
		return errors.New("product id is required")
	}
	if input.QtyChange == 0 {
		return errors.New("qty change cannot be zero")
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = "adjustment"
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin stock adjustment transaction: %w", err)
	}

	var existing int
	if err := tx.QueryRow(
		`SELECT COUNT(1) FROM products WHERE id = ? AND deleted_at IS NULL`,
		productID,
	).Scan(&existing); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("query product for adjustment: %w", err)
	}
	if existing == 0 {
		_ = tx.Rollback()
		return errors.New("product not found")
	}

	now := nowTS()
	stockTxnID := uuid.NewString()
	_, err = tx.Exec(
		`INSERT INTO stock_transactions (
			id, product_id, qty_change, reason, ref_type, ref_id, created_at, updated_at, deleted_at, device_id, synced_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, NULL)`,
		stockTxnID,
		productID,
		input.QtyChange,
		reason,
		"adjustment",
		productID,
		now,
		now,
		s.cfg.DeviceID,
	)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("insert stock adjustment: %w", err)
	}

	if err := s.appendSyncLogTx(tx, "stock_transactions", stockTxnID, "insert", map[string]any{
		"id":         stockTxnID,
		"product_id": productID,
		"qty_change": input.QtyChange,
		"reason":     reason,
		"ref_type":   "adjustment",
		"ref_id":     productID,
	}); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := s.recomputeProductStocksTx(tx, map[string]struct{}{productID: {}}); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit stock adjustment transaction: %w", err)
	}
	return nil
}

func (s *Service) ListLowStockProducts() ([]ProductStockView, error) {
	db, err := s.getDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT id, name, stock_qty, reorder_level
FROM products
WHERE deleted_at IS NULL
  AND is_active = 1
  AND stock_qty <= reorder_level
ORDER BY stock_qty ASC, name ASC
`)
	if err != nil {
		return nil, fmt.Errorf("query low stock products: %w", err)
	}
	defer rows.Close()

	out := make([]ProductStockView, 0)
	for rows.Next() {
		var p ProductStockView
		if err := rows.Scan(&p.ID, &p.Name, &p.StockQty, &p.ReorderLevel); err != nil {
			return nil, fmt.Errorf("scan low stock product: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate low stock products: %w", err)
	}
	return out, nil
}

func (s *Service) DashboardSummary() (DashboardSummary, error) {
	db, err := s.getDB()
	if err != nil {
		return DashboardSummary{}, err
	}

	start, end := dayBoundsUTC(time.Now().UTC())
	var summary DashboardSummary

	if err := db.QueryRow(
		`SELECT COALESCE(SUM(total_cents), 0), COUNT(1)
		 FROM sales
		 WHERE deleted_at IS NULL
		   AND status = 'completed'
		   AND created_at >= ?
		   AND created_at < ?`,
		start,
		end,
	).Scan(&summary.RevenueTodayCents, &summary.TransactionsTodayCount); err != nil {
		return DashboardSummary{}, fmt.Errorf("query revenue summary: %w", err)
	}

	if err := db.QueryRow(
		`SELECT COUNT(1) FROM products WHERE deleted_at IS NULL AND is_active = 1 AND stock_qty <= reorder_level`,
	).Scan(&summary.LowStockCount); err != nil {
		return DashboardSummary{}, fmt.Errorf("query low stock count: %w", err)
	}

	if err := db.QueryRow(
		`SELECT COUNT(1) FROM products WHERE deleted_at IS NULL AND is_active = 1 AND stock_qty <= 0`,
	).Scan(&summary.OutOfStockCount); err != nil {
		return DashboardSummary{}, fmt.Errorf("query out of stock count: %w", err)
	}

	return summary, nil
}

func (s *Service) ensureStaffActiveTx(tx *sql.Tx, staffID string) error {
	var role string
	var active int
	err := tx.QueryRow(
		`SELECT role, is_active FROM staff WHERE id = ? AND deleted_at IS NULL`,
		staffID,
	).Scan(&role, &active)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("cashier staff not found")
		}
		return fmt.Errorf("query cashier staff: %w", err)
	}
	if active != 1 {
		return errors.New("cashier staff is inactive")
	}
	if role != "cashier" && role != "admin" {
		return errors.New("staff role not allowed for sales")
	}
	return nil
}

func (s *Service) getVATRateTx(tx *sql.Tx) (int64, error) {
	var vatRaw string
	err := tx.QueryRow(
		`SELECT value FROM settings WHERE key = 'vat_rate' AND deleted_at IS NULL`,
	).Scan(&vatRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("query vat rate: %w", err)
	}

	vatRaw = strings.TrimSpace(vatRaw)
	if vatRaw == "" {
		return 0, nil
	}

	var vatPercent int64
	if _, err := fmt.Sscan(vatRaw, &vatPercent); err != nil {
		return 0, fmt.Errorf("invalid vat rate value %q", vatRaw)
	}
	if vatPercent < 0 {
		vatPercent = 0
	}
	return vatPercent * 100, nil
}

func (s *Service) recomputeProductStocksTx(tx *sql.Tx, affected map[string]struct{}) error {
	for productID := range affected {
		_, err := tx.Exec(
			`UPDATE products
			 SET stock_qty = (
			   COALESCE(starting_stock, 0) + COALESCE((
			     SELECT SUM(qty_change)
			     FROM stock_transactions
			     WHERE product_id = products.id
			       AND deleted_at IS NULL
			   ), 0)
			 )
			 WHERE id = ?`,
			productID,
		)
		if err != nil {
			return fmt.Errorf("recompute stock for product %s: %w", productID, err)
		}
	}
	return nil
}

func dayBoundsUTC(t time.Time) (string, string) {
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	return start.Format(time.RFC3339Nano), end.Format(time.RFC3339Nano)
}
