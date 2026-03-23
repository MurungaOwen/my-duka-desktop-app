package main

import (
	"bufio"
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"inventory-desktop/internal/backend"
)

// App struct
type App struct {
	ctx        context.Context
	cfg        backend.Config
	svc        *backend.Service
	syncEngine *backend.SyncEngine
	syncCancel context.CancelFunc
	startupErr string

	syncMu     sync.RWMutex
	syncStatus AppSyncStatus
}

type AppSyncStatus struct {
	Mode                string `json:"mode"`
	Enabled             bool   `json:"enabled"`
	Running             bool   `json:"running"`
	PendingCount        int64  `json:"pendingCount"`
	LastPushed          int    `json:"lastPushed"`
	LastPulled          int    `json:"lastPulled"`
	ConsecutiveFailures int    `json:"consecutiveFailures"`
	LastSuccessAt       string `json:"lastSuccessAt"`
	LastError           string `json:"lastError"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	loadDotEnvIfPresent(".env")

	cfg, err := backend.DefaultConfig()
	if err != nil {
		return &App{startupErr: err.Error()}
	}

	if mode := strings.TrimSpace(strings.ToLower(os.Getenv("MYDUKA_MODE"))); mode != "" {
		cfg.Mode = backend.DeploymentMode(mode)
	}
	if dbPath := strings.TrimSpace(os.Getenv("MYDUKA_DB_PATH")); dbPath != "" {
		cfg.DBPath = dbPath
	}

	svc, err := backend.NewService(cfg)
	if err != nil {
		return &App{startupErr: err.Error()}
	}

	app := &App{
		cfg: cfg,
		svc: svc,
		syncStatus: AppSyncStatus{
			Mode:    string(cfg.Mode),
			Enabled: cfg.Mode == backend.DeploymentModeLANSync,
		},
	}
	return app
}

func loadDotEnvIfPresent(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key := strings.TrimSpace(k)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		value := strings.TrimSpace(v)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		_ = os.Setenv(key, value)
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.startupErr != "" || a.svc == nil {
		return
	}
	if err := a.svc.Start(ctx); err != nil {
		a.startupErr = err.Error()
		return
	}

	if a.cfg.Mode == backend.DeploymentModeLANSync {
		a.startSyncEngine(ctx)
	}
}

func (a *App) shutdown(_ context.Context) {
	if a.syncCancel != nil {
		a.syncCancel()
	}
	if a.svc == nil {
		return
	}
	_ = a.svc.Close()
}

func (a *App) startSyncEngine(parent context.Context) {
	baseURL := strings.TrimSpace(os.Getenv("MYDUKA_SYNC_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://myduka.local:8080"
	}
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	interval := 5 * time.Second
	if raw := strings.TrimSpace(os.Getenv("MYDUKA_SYNC_INTERVAL_SECONDS")); raw != "" {
		if secs, err := strconv.Atoi(raw); err == nil && secs > 0 {
			interval = time.Duration(secs) * time.Second
		}
	}

	batchLimit := int64(200)
	if raw := strings.TrimSpace(os.Getenv("MYDUKA_SYNC_BATCH_LIMIT")); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil && v > 0 {
			batchLimit = v
		}
	}

	syncCtx, cancel := context.WithCancel(parent)
	a.syncCancel = cancel

	engine, err := backend.NewSyncEngine(a.svc, backend.SyncEngineConfig{
		BaseURL:    baseURL,
		DeviceID:   a.cfg.DeviceID,
		Interval:   interval,
		BatchLimit: batchLimit,
		OnCycle:    a.onSyncCycle,
	})
	if err != nil {
		a.setSyncStatus(func(s *AppSyncStatus) {
			s.Running = false
			s.LastError = err.Error()
		})
		return
	}
	a.syncEngine = engine
	a.setSyncStatus(func(s *AppSyncStatus) {
		s.Running = true
		s.LastError = ""
	})

	go func() {
		err := engine.Start(syncCtx)
		if err != nil && !errors.Is(err, context.Canceled) {
			a.setSyncStatus(func(s *AppSyncStatus) {
				s.Running = false
				s.LastError = err.Error()
			})
			return
		}
		a.setSyncStatus(func(s *AppSyncStatus) {
			s.Running = false
		})
	}()
}

func (a *App) onSyncCycle(result backend.StepResult, err error, failures int) {
	var pending int64
	if a.svc != nil {
		if count, countErr := a.svc.PendingSyncCount(); countErr == nil {
			pending = count
		}
	}

	a.setSyncStatus(func(s *AppSyncStatus) {
		s.PendingCount = pending
		s.ConsecutiveFailures = failures
		if err != nil {
			s.LastError = err.Error()
			return
		}
		s.LastPushed = result.Pushed
		s.LastPulled = result.Pulled
		s.LastSuccessAt = time.Now().UTC().Format(time.RFC3339)
		s.LastError = ""
	})
}

func (a *App) setSyncStatus(update func(*AppSyncStatus)) {
	a.syncMu.Lock()
	defer a.syncMu.Unlock()
	update(&a.syncStatus)
}

func (a *App) GetSyncStatus() AppSyncStatus {
	a.syncMu.RLock()
	defer a.syncMu.RUnlock()
	return a.syncStatus
}

func (a *App) StartupStatus() string {
	if a.startupErr == "" {
		return "ok"
	}
	return a.startupErr
}

func (a *App) BackendHealth() (backend.HealthStatus, error) {
	if a.svc == nil {
		return backend.HealthStatus{}, errors.New("backend service unavailable")
	}
	return a.svc.Health()
}

func (a *App) BootstrapBusiness(input backend.BootstrapInput) error {
	if a.svc == nil {
		return errors.New("backend service unavailable")
	}
	return a.svc.BootstrapBusiness(input)
}

func (a *App) UpsertSetting(setting backend.Setting) error {
	if a.svc == nil {
		return errors.New("backend service unavailable")
	}
	return a.svc.UpsertSetting(setting)
}

func (a *App) ListSettings() ([]backend.Setting, error) {
	if a.svc == nil {
		return nil, errors.New("backend service unavailable")
	}
	return a.svc.ListSettings()
}

func (a *App) CreateStaff(input backend.CreateStaffInput) (backend.Staff, error) {
	if a.svc == nil {
		return backend.Staff{}, errors.New("backend service unavailable")
	}
	return a.svc.CreateStaff(input)
}

func (a *App) ListStaff() ([]backend.Staff, error) {
	if a.svc == nil {
		return nil, errors.New("backend service unavailable")
	}
	return a.svc.ListStaff()
}

func (a *App) AuthenticateStaff(input backend.StaffLoginInput) (backend.Staff, error) {
	if a.svc == nil {
		return backend.Staff{}, errors.New("backend service unavailable")
	}
	return a.svc.AuthenticateStaff(input)
}

func (a *App) CreateCategory(input backend.CreateCategoryInput) (backend.Category, error) {
	if a.svc == nil {
		return backend.Category{}, errors.New("backend service unavailable")
	}
	return a.svc.CreateCategory(input)
}

func (a *App) UpdateCategory(input backend.UpdateCategoryInput) (backend.Category, error) {
	if a.svc == nil {
		return backend.Category{}, errors.New("backend service unavailable")
	}
	return a.svc.UpdateCategory(input)
}

func (a *App) DeleteCategory(categoryID string) error {
	if a.svc == nil {
		return errors.New("backend service unavailable")
	}
	return a.svc.DeleteCategory(categoryID)
}

func (a *App) ListCategories() ([]backend.Category, error) {
	if a.svc == nil {
		return nil, errors.New("backend service unavailable")
	}
	return a.svc.ListCategories()
}

func (a *App) CreateProduct(input backend.CreateProductInput) (backend.Product, error) {
	if a.svc == nil {
		return backend.Product{}, errors.New("backend service unavailable")
	}
	return a.svc.CreateProduct(input)
}

func (a *App) UpdateProduct(input backend.UpdateProductInput) (backend.Product, error) {
	if a.svc == nil {
		return backend.Product{}, errors.New("backend service unavailable")
	}
	return a.svc.UpdateProduct(input)
}

func (a *App) DeleteProduct(productID string) error {
	if a.svc == nil {
		return errors.New("backend service unavailable")
	}
	return a.svc.DeleteProduct(productID)
}

func (a *App) ListProducts() ([]backend.Product, error) {
	if a.svc == nil {
		return nil, errors.New("backend service unavailable")
	}
	return a.svc.ListProducts()
}

func (a *App) VerifyStaffPIN(input backend.PINVerificationInput) (bool, error) {
	if a.svc == nil {
		return false, errors.New("backend service unavailable")
	}
	return a.svc.VerifyStaffPIN(input)
}

func (a *App) CreateSale(input backend.CreateSaleInput) (backend.SaleDetail, error) {
	if a.svc == nil {
		return backend.SaleDetail{}, errors.New("backend service unavailable")
	}
	return a.svc.CreateSale(input)
}

func (a *App) StartMPesaCharge(input backend.StartMPesaChargeInput) (backend.MPesaChargeSession, error) {
	if a.svc == nil {
		return backend.MPesaChargeSession{}, errors.New("backend service unavailable")
	}
	return a.svc.StartMPesaCharge(input)
}

func (a *App) VerifyMPesaCharge(reference string) (backend.MPesaChargeStatus, error) {
	if a.svc == nil {
		return backend.MPesaChargeStatus{}, errors.New("backend service unavailable")
	}
	return a.svc.VerifyMPesaCharge(reference)
}

func (a *App) ListRecentMPesaPayments(input backend.ListRecentMPesaPaymentsInput) ([]backend.RecentMPesaPayment, error) {
	if a.svc == nil {
		return nil, errors.New("backend service unavailable")
	}
	return a.svc.ListRecentMPesaPayments(input)
}

func (a *App) ListSales(limit int64) ([]backend.Sale, error) {
	if a.svc == nil {
		return nil, errors.New("backend service unavailable")
	}
	return a.svc.ListSales(limit)
}

func (a *App) GetSaleDetail(saleID string) (backend.SaleDetail, error) {
	if a.svc == nil {
		return backend.SaleDetail{}, errors.New("backend service unavailable")
	}
	return a.svc.GetSaleDetail(saleID)
}

func (a *App) AdjustStock(input backend.StockAdjustmentInput) error {
	if a.svc == nil {
		return errors.New("backend service unavailable")
	}
	return a.svc.AdjustStock(input)
}

func (a *App) ListLowStockProducts() ([]backend.ProductStockView, error) {
	if a.svc == nil {
		return nil, errors.New("backend service unavailable")
	}
	return a.svc.ListLowStockProducts()
}

func (a *App) DashboardSummary() (backend.DashboardSummary, error) {
	if a.svc == nil {
		return backend.DashboardSummary{}, errors.New("backend service unavailable")
	}
	return a.svc.DashboardSummary()
}

func (a *App) SeedDemoData() (backend.SeedResult, error) {
	if a.svc == nil {
		return backend.SeedResult{}, errors.New("backend service unavailable")
	}
	return a.svc.SeedDemoData()
}

func (a *App) ListPendingSyncRecords(limit int64) ([]backend.SyncRecord, error) {
	if a.svc == nil {
		return nil, errors.New("backend service unavailable")
	}
	return a.svc.ListPendingSyncRecords(limit)
}

func (a *App) MarkSyncRecordsSynced(recordIDs []string) error {
	if a.svc == nil {
		return errors.New("backend service unavailable")
	}
	return a.svc.MarkSyncRecordsSynced(recordIDs)
}

func (a *App) ApplyIncomingMutations(sourceDeviceID string, mutations []backend.SyncMutation) (backend.SyncPushResponse, error) {
	if a.svc == nil {
		return backend.SyncPushResponse{}, errors.New("backend service unavailable")
	}
	return a.svc.ApplyIncomingMutations(sourceDeviceID, mutations)
}

func (a *App) PullMutationsForDevice(deviceID, since string, limit int64) ([]backend.SyncMutation, error) {
	if a.svc == nil {
		return nil, errors.New("backend service unavailable")
	}
	return a.svc.PullMutationsForDevice(deviceID, since, limit)
}
