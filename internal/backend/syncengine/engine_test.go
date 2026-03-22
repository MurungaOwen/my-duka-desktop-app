package syncengine

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"inventory-desktop/internal/backend/store"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonHTTPResponse(statusCode int, body any) *http.Response {
	raw, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(raw))),
	}
}

func setupStore(t *testing.T, deviceID string) *store.Service {
	t.Helper()
	cfg := store.Config{
		DBPath:     filepath.Join(t.TempDir(), "sync-engine-"+deviceID+".db"),
		Mode:       store.DeploymentModeLANSync,
		DeviceID:   deviceID,
		DeviceName: deviceID,
	}
	svc, err := store.NewService(cfg)
	if err != nil {
		t.Fatalf("new store service: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("start store service: %v", err)
	}
	t.Cleanup(func() {
		_ = svc.Close()
	})
	return svc
}

func productStock(t *testing.T, svc *store.Service, productID string) int64 {
	t.Helper()
	products, err := svc.ListProducts()
	if err != nil {
		t.Fatalf("list products: %v", err)
	}
	for _, p := range products {
		if p.ID == productID {
			return p.StockQty
		}
	}
	t.Fatalf("product not found: %s", productID)
	return 0
}

func TestRunOncePushMarksPendingAsSynced(t *testing.T) {
	local := setupStore(t, "local-1")

	_, err := local.CreateCategory(store.CreateCategoryInput{Name: "Beverages", Emoji: "B"})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	_, err = local.CreateProduct(store.CreateProductInput{
		Name:          "Soda",
		PriceCents:    120,
		StartingStock: 5,
		ReorderLevel:  1,
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	pendingBefore, err := local.ListPendingSyncRecords(100)
	if err != nil {
		t.Fatalf("pending before: %v", err)
	}
	if len(pendingBefore) == 0 {
		t.Fatalf("expected pending records before sync")
	}

	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/sync/push":
				return jsonHTTPResponse(http.StatusOK, store.SyncPushResponse{Applied: len(pendingBefore), Skipped: 0}), nil
			case "/sync/pull":
				return jsonHTTPResponse(http.StatusOK, store.SyncPullResponse{Mutations: []store.SyncMutation{}}), nil
			default:
				return jsonHTTPResponse(http.StatusNotFound, map[string]string{"error": "not found"}), nil
			}
		}),
	}

	engine, err := New(local, Config{
		BaseURL:    "http://sync.test",
		DeviceID:   "local-1",
		HTTPClient: client,
	})
	if err != nil {
		t.Fatalf("new sync engine: %v", err)
	}

	res, err := engine.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if res.Pushed == 0 {
		t.Fatalf("expected pushed records > 0")
	}

	pendingAfter, err := local.ListPendingSyncRecords(100)
	if err != nil {
		t.Fatalf("pending after: %v", err)
	}
	if len(pendingAfter) != 0 {
		t.Fatalf("expected no pending records after sync, got %d", len(pendingAfter))
	}
}

func TestBackoffForFailures(t *testing.T) {
	tests := []struct {
		failures int
		want     time.Duration
	}{
		{0, 5 * time.Second},
		{1, 5 * time.Second},
		{2, 15 * time.Second},
		{3, 30 * time.Second},
		{4, 60 * time.Second},
		{10, 60 * time.Second},
	}
	for _, tc := range tests {
		got := BackoffForFailures(tc.failures)
		if got != tc.want {
			t.Fatalf("failures=%d got=%s want=%s", tc.failures, got, tc.want)
		}
	}
}

func TestRunOncePullAppliesStockMutation(t *testing.T) {
	local := setupStore(t, "local-2")

	product, err := local.CreateProduct(store.CreateProductInput{
		Name:          "Sugar",
		PriceCents:    200,
		StartingStock: 10,
		ReorderLevel:  1,
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	pending, err := local.ListPendingSyncRecords(100)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	ids := make([]string, 0, len(pending))
	for _, p := range pending {
		ids = append(ids, p.ID)
	}
	if err := local.MarkSyncRecordsSynced(ids); err != nil {
		t.Fatalf("mark initial pending synced: %v", err)
	}

	mutation := store.SyncMutation{
		MutationID:     "mut-stock-1",
		SourceDeviceID: "remote-1",
		TableName:      "stock_transactions",
		RecordID:       "stx-1",
		Operation:      "insert",
		Payload:        `{"id":"stx-1","product_id":"` + product.ID + `","qty_change":-2,"reason":"sale","ref_type":"sale","ref_id":"sale-1"}`,
		CreatedAt:      "2026-01-01T10:00:00Z",
	}

	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/sync/push":
				return jsonHTTPResponse(http.StatusOK, store.SyncPushResponse{Applied: 0, Skipped: 0}), nil
			case "/sync/pull":
				return jsonHTTPResponse(http.StatusOK, store.SyncPullResponse{Mutations: []store.SyncMutation{mutation}}), nil
			default:
				return jsonHTTPResponse(http.StatusNotFound, map[string]string{"error": "not found"}), nil
			}
		}),
	}

	engine, err := New(local, Config{
		BaseURL:    "http://sync.test",
		DeviceID:   "local-2",
		HTTPClient: client,
	})
	if err != nil {
		t.Fatalf("new sync engine: %v", err)
	}

	res, err := engine.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if res.Pulled != 1 {
		t.Fatalf("expected pulled=1, got %d", res.Pulled)
	}

	if got := productStock(t, local, product.ID); got != 8 {
		t.Fatalf("expected stock=8 after pull, got %d", got)
	}
}

func TestRunOnceReplayIsIdempotent(t *testing.T) {
	local := setupStore(t, "local-3")

	product, err := local.CreateProduct(store.CreateProductInput{
		Name:          "Flour",
		PriceCents:    250,
		StartingStock: 10,
		ReorderLevel:  1,
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	pending, err := local.ListPendingSyncRecords(100)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	ids := make([]string, 0, len(pending))
	for _, p := range pending {
		ids = append(ids, p.ID)
	}
	if err := local.MarkSyncRecordsSynced(ids); err != nil {
		t.Fatalf("mark initial pending synced: %v", err)
	}

	mutation := store.SyncMutation{
		MutationID:     "mut-stock-replay-1",
		SourceDeviceID: "remote-2",
		TableName:      "stock_transactions",
		RecordID:       "stx-replay-1",
		Operation:      "insert",
		Payload:        `{"id":"stx-replay-1","product_id":"` + product.ID + `","qty_change":-2,"reason":"sale","ref_type":"sale","ref_id":"sale-replay-1"}`,
		CreatedAt:      "2026-01-02T10:00:00Z",
	}

	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/sync/push":
				return jsonHTTPResponse(http.StatusOK, store.SyncPushResponse{Applied: 0, Skipped: 0}), nil
			case "/sync/pull":
				// Return the exact same mutation every call to simulate replay.
				return jsonHTTPResponse(http.StatusOK, store.SyncPullResponse{Mutations: []store.SyncMutation{mutation}}), nil
			default:
				return jsonHTTPResponse(http.StatusNotFound, map[string]string{"error": "not found"}), nil
			}
		}),
	}

	engine, err := New(local, Config{
		BaseURL:    "http://sync.test",
		DeviceID:   "local-3",
		HTTPClient: client,
	})
	if err != nil {
		t.Fatalf("new sync engine: %v", err)
	}

	if _, err := engine.RunOnce(context.Background()); err != nil {
		t.Fatalf("first run once: %v", err)
	}
	firstStock := productStock(t, local, product.ID)
	if firstStock != 8 {
		t.Fatalf("expected stock=8 after first pull, got %d", firstStock)
	}

	if _, err := engine.RunOnce(context.Background()); err != nil {
		t.Fatalf("second run once: %v", err)
	}
	secondStock := productStock(t, local, product.ID)
	if secondStock != 8 {
		t.Fatalf("expected idempotent replay stock=8, got %d", secondStock)
	}
}

func TestStartInvokesOnCycleCallback(t *testing.T) {
	local := setupStore(t, "local-4")

	_, err := local.CreateCategory(store.CreateCategoryInput{Name: "CycleCat", Emoji: "C"})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	cycles := 0
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/sync/push":
				return jsonHTTPResponse(http.StatusOK, store.SyncPushResponse{Applied: 1, Skipped: 0}), nil
			case "/sync/pull":
				return jsonHTTPResponse(http.StatusOK, store.SyncPullResponse{Mutations: []store.SyncMutation{}}), nil
			default:
				return jsonHTTPResponse(http.StatusNotFound, map[string]string{"error": "not found"}), nil
			}
		}),
	}

	engine, err := New(local, Config{
		BaseURL:    "http://sync.test",
		DeviceID:   "local-4",
		Interval:   10 * time.Millisecond,
		HTTPClient: client,
		OnCycle: func(_ StepResult, err error, _ int) {
			if err == nil {
				cycles++
			}
		},
	})
	if err != nil {
		t.Fatalf("new sync engine: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	_ = engine.Start(ctx)

	if cycles == 0 {
		t.Fatalf("expected onCycle to be invoked at least once")
	}
}
