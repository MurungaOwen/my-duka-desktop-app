package backend

import (
	"net/http"

	"inventory-desktop/internal/backend/store"
	"inventory-desktop/internal/backend/syncapi"
	"inventory-desktop/internal/backend/syncengine"
)

type DeploymentMode = store.DeploymentMode

const (
	DeploymentModeStandalone = store.DeploymentModeStandalone
	DeploymentModeLANSync    = store.DeploymentModeLANSync
)

type Config = store.Config
type HealthStatus = store.HealthStatus
type BootstrapInput = store.BootstrapInput
type CreateStaffInput = store.CreateStaffInput
type Staff = store.Staff
type CreateCategoryInput = store.CreateCategoryInput
type Category = store.Category
type CreateProductInput = store.CreateProductInput
type Product = store.Product
type Setting = store.Setting
type SaleItemInput = store.SaleItemInput
type CreateSaleInput = store.CreateSaleInput
type StartMPesaChargeInput = store.StartMPesaChargeInput
type MPesaChargeSession = store.MPesaChargeSession
type MPesaChargeStatus = store.MPesaChargeStatus
type Sale = store.Sale
type SaleItem = store.SaleItem
type SaleDetail = store.SaleDetail
type StockAdjustmentInput = store.StockAdjustmentInput
type ProductStockView = store.ProductStockView
type DashboardSummary = store.DashboardSummary
type PINVerificationInput = store.PINVerificationInput
type StaffLoginInput = store.StaffLoginInput
type SyncRecord = store.SyncRecord
type SyncMutation = store.SyncMutation
type SyncPushRequest = store.SyncPushRequest
type SyncPushResponse = store.SyncPushResponse
type SyncPullResponse = store.SyncPullResponse
type SeedResult = store.SeedResult
type DemoCredentials = store.DemoCredentials
type Service = store.Service
type SyncEngine = syncengine.Engine
type SyncEngineConfig = syncengine.Config
type StepResult = syncengine.StepResult

func DefaultConfig() (Config, error) {
	return store.DefaultConfig()
}

func NewService(cfg Config) (*Service, error) {
	return store.NewService(cfg)
}

func NewSyncHTTPHandler(s *Service) http.Handler {
	return syncapi.NewSyncHTTPHandler(s)
}

func NewSyncEngine(s *Service, cfg SyncEngineConfig) (*SyncEngine, error) {
	return syncengine.New(s, cfg)
}
