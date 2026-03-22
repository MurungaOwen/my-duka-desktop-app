package store

type DeploymentMode string

const (
	DeploymentModeStandalone DeploymentMode = "standalone"
	DeploymentModeLANSync    DeploymentMode = "lan_sync"
)

func (m DeploymentMode) Valid() bool {
	return m == DeploymentModeStandalone || m == DeploymentModeLANSync
}

func (m DeploymentMode) SyncEnabled() bool {
	return m == DeploymentModeLANSync
}

type HealthStatus struct {
	Initialized bool           `json:"initialized"`
	Mode        DeploymentMode `json:"mode"`
	SyncEnabled bool           `json:"syncEnabled"`
	DBPath      string         `json:"dbPath"`
	DeviceID    string         `json:"deviceId"`
	DeviceName  string         `json:"deviceName"`
}

type BootstrapInput struct {
	BusinessName string `json:"businessName"`
	Location     string `json:"location"`
	Currency     string `json:"currency"`
	VATRate      string `json:"vatRate"`
}

type CreateStaffInput struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

type Staff struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type CreateCategoryInput struct {
	Name         string `json:"name"`
	Emoji        string `json:"emoji"`
	DisplayOrder int64  `json:"displayOrder"`
}

type Category struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Emoji        string `json:"emoji"`
	DisplayOrder int64  `json:"displayOrder"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type CreateProductInput struct {
	Name          string `json:"name"`
	SKU           string `json:"sku"`
	Barcode       string `json:"barcode"`
	CategoryID    string `json:"categoryId"`
	PriceCents    int64  `json:"priceCents"`
	StartingStock int64  `json:"startingStock"`
	ReorderLevel  int64  `json:"reorderLevel"`
}

type Product struct {
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

type Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type SaleItemInput struct {
	ProductID string `json:"productId"`
	Quantity  int64  `json:"quantity"`
}

type CreateSaleInput struct {
	CashierStaffID string          `json:"cashierStaffId"`
	PaymentMethod  string          `json:"paymentMethod"`
	PaymentRef     string          `json:"paymentRef"`
	Items          []SaleItemInput `json:"items"`
}

type StartMPesaChargeInput struct {
	Phone       string `json:"phone"`
	AmountCents int64  `json:"amountCents"`
	Email       string `json:"email"`
	Reference   string `json:"reference"`
}

type MPesaChargeSession struct {
	Reference   string `json:"reference"`
	Status      string `json:"status"`
	DisplayText string `json:"displayText"`
	Message     string `json:"message"`
}

type MPesaChargeStatus struct {
	Reference       string `json:"reference"`
	Status          string `json:"status"`
	Paid            bool   `json:"paid"`
	AmountCents     int64  `json:"amountCents"`
	Currency        string `json:"currency"`
	Channel         string `json:"channel"`
	GatewayResponse string `json:"gatewayResponse"`
	DisplayText     string `json:"displayText"`
	Message         string `json:"message"`
}

type ListRecentMPesaPaymentsInput struct {
	WindowMinutes int64 `json:"windowMinutes"`
	AmountCents   int64 `json:"amountCents"`
	Limit         int64 `json:"limit"`
}

type RecentMPesaPayment struct {
	Reference        string `json:"reference"`
	AmountCents      int64  `json:"amountCents"`
	Currency         string `json:"currency"`
	Channel          string `json:"channel"`
	PaidAt           string `json:"paidAt"`
	GatewayResponse  string `json:"gatewayResponse"`
	CustomerEmail    string `json:"customerEmail"`
	CustomerName     string `json:"customerName"`
	AuthorizationKey string `json:"authorizationKey"`
}

type Sale struct {
	ID             string `json:"id"`
	CashierStaffID string `json:"cashierStaffId"`
	PaymentMethod  string `json:"paymentMethod"`
	Status         string `json:"status"`
	SubtotalCents  int64  `json:"subtotalCents"`
	VATCents       int64  `json:"vatCents"`
	TotalCents     int64  `json:"totalCents"`
	CreatedAt      string `json:"createdAt"`
}

type SaleItem struct {
	ID             string `json:"id"`
	SaleID         string `json:"saleId"`
	ProductID      string `json:"productId"`
	Quantity       int64  `json:"quantity"`
	UnitPriceCents int64  `json:"unitPriceCents"`
	LineTotalCents int64  `json:"lineTotalCents"`
}

type SaleDetail struct {
	Sale
	Items []SaleItem `json:"items"`
}

type StockAdjustmentInput struct {
	ProductID string `json:"productId"`
	QtyChange int64  `json:"qtyChange"`
	Reason    string `json:"reason"`
}

type ProductStockView struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	StockQty     int64  `json:"stockQty"`
	ReorderLevel int64  `json:"reorderLevel"`
}

type DashboardSummary struct {
	RevenueTodayCents      int64 `json:"revenueTodayCents"`
	TransactionsTodayCount int64 `json:"transactionsTodayCount"`
	LowStockCount          int64 `json:"lowStockCount"`
	OutOfStockCount        int64 `json:"outOfStockCount"`
}

type PINVerificationInput struct {
	StaffID string `json:"staffId"`
	PIN     string `json:"pin"`
}

type StaffLoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type SyncRecord struct {
	ID        string `json:"id"`
	TableName string `json:"tableName"`
	RecordID  string `json:"recordId"`
	Operation string `json:"operation"`
	Payload   string `json:"payload"`
	CreatedAt string `json:"createdAt"`
}

type SyncMutation struct {
	MutationID     string `json:"mutationId"`
	SourceDeviceID string `json:"sourceDeviceId,omitempty"`
	TableName      string `json:"tableName"`
	RecordID       string `json:"recordId"`
	Operation      string `json:"operation"`
	Payload        string `json:"payload"`
	CreatedAt      string `json:"createdAt"`
}

type SyncPushRequest struct {
	DeviceID  string         `json:"deviceId"`
	Mutations []SyncMutation `json:"mutations"`
}

type SyncPushResponse struct {
	Applied int `json:"applied"`
	Skipped int `json:"skipped"`
}

type SyncPullResponse struct {
	Mutations []SyncMutation `json:"mutations"`
}

type DemoCredentials struct {
	Role     string `json:"role"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
	Notes    string `json:"notes"`
}

type SeedResult struct {
	BusinessName    string            `json:"businessName"`
	StaffAdded      int               `json:"staffAdded"`
	CategoriesAdded int               `json:"categoriesAdded"`
	SuppliersAdded  int               `json:"suppliersAdded"`
	ProductsAdded   int               `json:"productsAdded"`
	OrdersAdded     int               `json:"ordersAdded"`
	SalesAdded      int               `json:"salesAdded"`
	Credentials     []DemoCredentials `json:"credentials"`
}
