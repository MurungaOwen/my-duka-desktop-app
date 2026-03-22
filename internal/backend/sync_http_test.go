package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func setupNamedService(t *testing.T, deviceID string) *Service {
	t.Helper()

	cfg := Config{
		DBPath:     filepath.Join(t.TempDir(), "myduka-"+deviceID+".db"),
		Mode:       DeploymentModeLANSync,
		DeviceID:   deviceID,
		DeviceName: deviceID,
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

func toMutations(records []SyncRecord) []SyncMutation {
	out := make([]SyncMutation, 0, len(records))
	for _, r := range records {
		out = append(out, SyncMutation{
			MutationID: r.ID,
			TableName:  r.TableName,
			RecordID:   r.RecordID,
			Operation:  r.Operation,
			Payload:    r.Payload,
			CreatedAt:  r.CreatedAt,
		})
	}
	return out
}

func TestSyncPushPullAndIdempotency(t *testing.T) {
	serverSvc := setupNamedService(t, "server-device")
	clientSvc := setupNamedService(t, "client-device-1")

	_, err := clientSvc.CreateCategory(CreateCategoryInput{Name: "Sync Cat", Emoji: "C"})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	_, err = clientSvc.CreateProduct(CreateProductInput{
		Name:          "Sync Product",
		PriceCents:    550,
		StartingStock: 7,
		ReorderLevel:  2,
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	pending, err := clientSvc.ListPendingSyncRecords(100)
	if err != nil {
		t.Fatalf("list pending sync records: %v", err)
	}
	if len(pending) < 2 {
		t.Fatalf("expected at least 2 pending records, got %d", len(pending))
	}

	handler := NewSyncHTTPHandler(serverSvc)

	pushReq := SyncPushRequest{
		DeviceID:  "client-device-1",
		Mutations: toMutations(pending),
	}
	body, err := json.Marshal(pushReq)
	if err != nil {
		t.Fatalf("marshal push request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/sync/push", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("push status: got %d body=%s", rec.Code, rec.Body.String())
	}

	var pushResp SyncPushResponse
	if err := json.NewDecoder(rec.Body).Decode(&pushResp); err != nil {
		t.Fatalf("decode push response: %v", err)
	}
	if pushResp.Applied != len(pending) {
		t.Fatalf("expected applied=%d got %d", len(pending), pushResp.Applied)
	}
	if pushResp.Skipped != 0 {
		t.Fatalf("expected skipped=0 got %d", pushResp.Skipped)
	}

	serverProducts, err := serverSvc.ListProducts()
	if err != nil {
		t.Fatalf("list server products: %v", err)
	}
	if len(serverProducts) != 1 {
		t.Fatalf("expected 1 server product, got %d", len(serverProducts))
	}
	if serverProducts[0].Name != "Sync Product" {
		t.Fatalf("unexpected server product name: %s", serverProducts[0].Name)
	}

	// Push same mutations again: should be idempotent and skipped.
	req2 := httptest.NewRequest(http.MethodPost, "/sync/push", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("second push status: got %d body=%s", rec2.Code, rec2.Body.String())
	}
	var pushResp2 SyncPushResponse
	if err := json.NewDecoder(rec2.Body).Decode(&pushResp2); err != nil {
		t.Fatalf("decode second push response: %v", err)
	}
	if pushResp2.Applied != 0 {
		t.Fatalf("expected second push applied=0 got %d", pushResp2.Applied)
	}
	if pushResp2.Skipped != len(pending) {
		t.Fatalf("expected second push skipped=%d got %d", len(pending), pushResp2.Skipped)
	}

	// Pull from a different device should receive pushed records.
	pullReq := httptest.NewRequest(http.MethodGet, "/sync/pull?device_id=client-device-2&since=", nil)
	pullRec := httptest.NewRecorder()
	handler.ServeHTTP(pullRec, pullReq)
	if pullRec.Code != http.StatusOK {
		t.Fatalf("pull status: got %d body=%s", pullRec.Code, pullRec.Body.String())
	}

	var pullResp SyncPullResponse
	if err := json.NewDecoder(pullRec.Body).Decode(&pullResp); err != nil {
		t.Fatalf("decode pull response: %v", err)
	}
	if len(pullResp.Mutations) < len(pending) {
		t.Fatalf("expected at least %d pulled mutations, got %d", len(pending), len(pullResp.Mutations))
	}

	// Pull from same source device should not receive its own records.
	pullSameReq := httptest.NewRequest(http.MethodGet, "/sync/pull?device_id=client-device-1&since=", nil)
	pullSameRec := httptest.NewRecorder()
	handler.ServeHTTP(pullSameRec, pullSameReq)
	if pullSameRec.Code != http.StatusOK {
		t.Fatalf("pull same-device status: got %d body=%s", pullSameRec.Code, pullSameRec.Body.String())
	}
	var pullSameResp SyncPullResponse
	if err := json.NewDecoder(pullSameRec.Body).Decode(&pullSameResp); err != nil {
		t.Fatalf("decode same-device pull response: %v", err)
	}
	if len(pullSameResp.Mutations) != 0 {
		t.Fatalf("expected same-device pull to return 0 mutations, got %d", len(pullSameResp.Mutations))
	}
}

func TestSyncApplySalesAndSaleItemsMutation(t *testing.T) {
	serverSvc := setupNamedService(t, "server-device-sales")

	staff, err := serverSvc.CreateStaff(CreateStaffInput{
		Name:     "Sync Cashier",
		Username: "sync_cashier",
		Role:     "cashier",
		Password: "1234",
	})
	if err != nil {
		t.Fatalf("create staff: %v", err)
	}
	product, err := serverSvc.CreateProduct(CreateProductInput{
		Name:          "Sales Sync Product",
		PriceCents:    300,
		StartingStock: 10,
		ReorderLevel:  2,
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	// Flush local pending setup records to keep assertions predictable.
	pending, err := serverSvc.ListPendingSyncRecords(200)
	if err != nil {
		t.Fatalf("list pending sync records: %v", err)
	}
	flushIDs := make([]string, 0, len(pending))
	for _, p := range pending {
		flushIDs = append(flushIDs, p.ID)
	}
	if err := serverSvc.MarkSyncRecordsSynced(flushIDs); err != nil {
		t.Fatalf("mark pending synced: %v", err)
	}

	mutations := []SyncMutation{
		{
			MutationID:     "sale-mut-1",
			SourceDeviceID: "remote-device-1",
			TableName:      "sales",
			RecordID:       "sale-1",
			Operation:      "insert",
			Payload:        `{"id":"sale-1","cashierStaffId":"` + staff.ID + `","paymentMethod":"cash","status":"completed","subtotalCents":600,"vatCents":96,"totalCents":696,"createdAt":"2026-01-01T10:00:00Z"}`,
			CreatedAt:      "2026-01-01T10:00:00Z",
		},
		{
			MutationID:     "sale-item-mut-1",
			SourceDeviceID: "remote-device-1",
			TableName:      "sale_items",
			RecordID:       "sale-item-1",
			Operation:      "insert",
			Payload:        `{"id":"sale-item-1","saleId":"sale-1","productId":"` + product.ID + `","quantity":2,"unitPriceCents":300,"lineTotalCents":600}`,
			CreatedAt:      "2026-01-01T10:00:01Z",
		},
		{
			MutationID:     "stock-mut-1",
			SourceDeviceID: "remote-device-1",
			TableName:      "stock_transactions",
			RecordID:       "stock-1",
			Operation:      "insert",
			Payload:        `{"id":"stock-1","product_id":"` + product.ID + `","qty_change":-2,"reason":"sale","ref_type":"sale","ref_id":"sale-1"}`,
			CreatedAt:      "2026-01-01T10:00:01Z",
		},
	}

	resp, err := serverSvc.ApplyIncomingMutations("remote-device-1", mutations)
	if err != nil {
		t.Fatalf("apply incoming mutations: %v", err)
	}
	if resp.Applied != 3 {
		t.Fatalf("expected applied=3 got %d", resp.Applied)
	}

	sales, err := serverSvc.ListSales(10)
	if err != nil {
		t.Fatalf("list sales: %v", err)
	}
	if len(sales) != 1 {
		t.Fatalf("expected 1 sale after sync apply, got %d", len(sales))
	}
	if sales[0].TotalCents != 696 {
		t.Fatalf("expected total 696, got %d", sales[0].TotalCents)
	}

	detail, err := serverSvc.GetSaleDetail("sale-1")
	if err != nil {
		t.Fatalf("get sale detail: %v", err)
	}
	if len(detail.Items) != 1 {
		t.Fatalf("expected 1 sale item, got %d", len(detail.Items))
	}

	products, err := serverSvc.ListProducts()
	if err != nil {
		t.Fatalf("list products: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}
	if products[0].StockQty != 8 {
		t.Fatalf("expected stock 8 after remote sale sync, got %d", products[0].StockQty)
	}
}
