export namespace main {
	
	export class AppSyncStatus {
	    mode: string;
	    enabled: boolean;
	    running: boolean;
	    pendingCount: number;
	    lastPushed: number;
	    lastPulled: number;
	    consecutiveFailures: number;
	    lastSuccessAt: string;
	    lastError: string;
	
	    static createFrom(source: any = {}) {
	        return new AppSyncStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.enabled = source["enabled"];
	        this.running = source["running"];
	        this.pendingCount = source["pendingCount"];
	        this.lastPushed = source["lastPushed"];
	        this.lastPulled = source["lastPulled"];
	        this.consecutiveFailures = source["consecutiveFailures"];
	        this.lastSuccessAt = source["lastSuccessAt"];
	        this.lastError = source["lastError"];
	    }
	}

}

export namespace store {
	
	export class BootstrapInput {
	    businessName: string;
	    location: string;
	    currency: string;
	    vatRate: string;
	
	    static createFrom(source: any = {}) {
	        return new BootstrapInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.businessName = source["businessName"];
	        this.location = source["location"];
	        this.currency = source["currency"];
	        this.vatRate = source["vatRate"];
	    }
	}
	export class Category {
	    id: string;
	    name: string;
	    emoji: string;
	    displayOrder: number;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.emoji = source["emoji"];
	        this.displayOrder = source["displayOrder"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class CreateCategoryInput {
	    name: string;
	    emoji: string;
	    displayOrder: number;
	
	    static createFrom(source: any = {}) {
	        return new CreateCategoryInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.emoji = source["emoji"];
	        this.displayOrder = source["displayOrder"];
	    }
	}
	export class CreateProductInput {
	    name: string;
	    sku: string;
	    barcode: string;
	    categoryId: string;
	    priceCents: number;
	    startingStock: number;
	    reorderLevel: number;
	
	    static createFrom(source: any = {}) {
	        return new CreateProductInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.sku = source["sku"];
	        this.barcode = source["barcode"];
	        this.categoryId = source["categoryId"];
	        this.priceCents = source["priceCents"];
	        this.startingStock = source["startingStock"];
	        this.reorderLevel = source["reorderLevel"];
	    }
	}
	export class SaleItemInput {
	    productId: string;
	    quantity: number;
	
	    static createFrom(source: any = {}) {
	        return new SaleItemInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.productId = source["productId"];
	        this.quantity = source["quantity"];
	    }
	}
	export class CreateSaleInput {
	    cashierStaffId: string;
	    paymentMethod: string;
	    paymentRef: string;
	    items: SaleItemInput[];
	
	    static createFrom(source: any = {}) {
	        return new CreateSaleInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cashierStaffId = source["cashierStaffId"];
	        this.paymentMethod = source["paymentMethod"];
	        this.paymentRef = source["paymentRef"];
	        this.items = this.convertValues(source["items"], SaleItemInput);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CreateStaffInput {
	    name: string;
	    username: string;
	    role: string;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateStaffInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.username = source["username"];
	        this.role = source["role"];
	        this.password = source["password"];
	    }
	}
	export class DashboardSummary {
	    revenueTodayCents: number;
	    transactionsTodayCount: number;
	    lowStockCount: number;
	    outOfStockCount: number;
	
	    static createFrom(source: any = {}) {
	        return new DashboardSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.revenueTodayCents = source["revenueTodayCents"];
	        this.transactionsTodayCount = source["transactionsTodayCount"];
	        this.lowStockCount = source["lowStockCount"];
	        this.outOfStockCount = source["outOfStockCount"];
	    }
	}
	export class DemoCredentials {
	    role: string;
	    name: string;
	    username: string;
	    password: string;
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new DemoCredentials(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.name = source["name"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.notes = source["notes"];
	    }
	}
	export class HealthStatus {
	    initialized: boolean;
	    mode: string;
	    syncEnabled: boolean;
	    dbPath: string;
	    deviceId: string;
	    deviceName: string;
	
	    static createFrom(source: any = {}) {
	        return new HealthStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.initialized = source["initialized"];
	        this.mode = source["mode"];
	        this.syncEnabled = source["syncEnabled"];
	        this.dbPath = source["dbPath"];
	        this.deviceId = source["deviceId"];
	        this.deviceName = source["deviceName"];
	    }
	}
	export class ListRecentMPesaPaymentsInput {
	    windowMinutes: number;
	    amountCents: number;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new ListRecentMPesaPaymentsInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.windowMinutes = source["windowMinutes"];
	        this.amountCents = source["amountCents"];
	        this.limit = source["limit"];
	    }
	}
	export class MPesaChargeSession {
	    reference: string;
	    status: string;
	    displayText: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new MPesaChargeSession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.reference = source["reference"];
	        this.status = source["status"];
	        this.displayText = source["displayText"];
	        this.message = source["message"];
	    }
	}
	export class MPesaChargeStatus {
	    reference: string;
	    status: string;
	    paid: boolean;
	    amountCents: number;
	    currency: string;
	    channel: string;
	    gatewayResponse: string;
	    displayText: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new MPesaChargeStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.reference = source["reference"];
	        this.status = source["status"];
	        this.paid = source["paid"];
	        this.amountCents = source["amountCents"];
	        this.currency = source["currency"];
	        this.channel = source["channel"];
	        this.gatewayResponse = source["gatewayResponse"];
	        this.displayText = source["displayText"];
	        this.message = source["message"];
	    }
	}
	export class PINVerificationInput {
	    staffId: string;
	    pin: string;
	
	    static createFrom(source: any = {}) {
	        return new PINVerificationInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.staffId = source["staffId"];
	        this.pin = source["pin"];
	    }
	}
	export class Product {
	    id: string;
	    name: string;
	    sku: string;
	    barcode: string;
	    categoryId: string;
	    priceCents: number;
	    startingStock: number;
	    stockQty: number;
	    reorderLevel: number;
	    isActive: boolean;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Product(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.sku = source["sku"];
	        this.barcode = source["barcode"];
	        this.categoryId = source["categoryId"];
	        this.priceCents = source["priceCents"];
	        this.startingStock = source["startingStock"];
	        this.stockQty = source["stockQty"];
	        this.reorderLevel = source["reorderLevel"];
	        this.isActive = source["isActive"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ProductStockView {
	    id: string;
	    name: string;
	    stockQty: number;
	    reorderLevel: number;
	
	    static createFrom(source: any = {}) {
	        return new ProductStockView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.stockQty = source["stockQty"];
	        this.reorderLevel = source["reorderLevel"];
	    }
	}
	export class RecentMPesaPayment {
	    reference: string;
	    amountCents: number;
	    currency: string;
	    channel: string;
	    paidAt: string;
	    gatewayResponse: string;
	    customerEmail: string;
	    customerName: string;
	    authorizationKey: string;
	
	    static createFrom(source: any = {}) {
	        return new RecentMPesaPayment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.reference = source["reference"];
	        this.amountCents = source["amountCents"];
	        this.currency = source["currency"];
	        this.channel = source["channel"];
	        this.paidAt = source["paidAt"];
	        this.gatewayResponse = source["gatewayResponse"];
	        this.customerEmail = source["customerEmail"];
	        this.customerName = source["customerName"];
	        this.authorizationKey = source["authorizationKey"];
	    }
	}
	export class Sale {
	    id: string;
	    cashierStaffId: string;
	    paymentMethod: string;
	    status: string;
	    subtotalCents: number;
	    vatCents: number;
	    totalCents: number;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Sale(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.cashierStaffId = source["cashierStaffId"];
	        this.paymentMethod = source["paymentMethod"];
	        this.status = source["status"];
	        this.subtotalCents = source["subtotalCents"];
	        this.vatCents = source["vatCents"];
	        this.totalCents = source["totalCents"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class SaleItem {
	    id: string;
	    saleId: string;
	    productId: string;
	    quantity: number;
	    unitPriceCents: number;
	    lineTotalCents: number;
	
	    static createFrom(source: any = {}) {
	        return new SaleItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.saleId = source["saleId"];
	        this.productId = source["productId"];
	        this.quantity = source["quantity"];
	        this.unitPriceCents = source["unitPriceCents"];
	        this.lineTotalCents = source["lineTotalCents"];
	    }
	}
	export class SaleDetail {
	    id: string;
	    cashierStaffId: string;
	    paymentMethod: string;
	    status: string;
	    subtotalCents: number;
	    vatCents: number;
	    totalCents: number;
	    createdAt: string;
	    items: SaleItem[];
	
	    static createFrom(source: any = {}) {
	        return new SaleDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.cashierStaffId = source["cashierStaffId"];
	        this.paymentMethod = source["paymentMethod"];
	        this.status = source["status"];
	        this.subtotalCents = source["subtotalCents"];
	        this.vatCents = source["vatCents"];
	        this.totalCents = source["totalCents"];
	        this.createdAt = source["createdAt"];
	        this.items = this.convertValues(source["items"], SaleItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class SeedResult {
	    businessName: string;
	    staffAdded: number;
	    categoriesAdded: number;
	    suppliersAdded: number;
	    productsAdded: number;
	    ordersAdded: number;
	    salesAdded: number;
	    credentials: DemoCredentials[];
	
	    static createFrom(source: any = {}) {
	        return new SeedResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.businessName = source["businessName"];
	        this.staffAdded = source["staffAdded"];
	        this.categoriesAdded = source["categoriesAdded"];
	        this.suppliersAdded = source["suppliersAdded"];
	        this.productsAdded = source["productsAdded"];
	        this.ordersAdded = source["ordersAdded"];
	        this.salesAdded = source["salesAdded"];
	        this.credentials = this.convertValues(source["credentials"], DemoCredentials);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Setting {
	    key: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new Setting(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	    }
	}
	export class Staff {
	    id: string;
	    name: string;
	    username: string;
	    role: string;
	    isActive: boolean;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Staff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.username = source["username"];
	        this.role = source["role"];
	        this.isActive = source["isActive"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class StaffLoginInput {
	    username: string;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new StaffLoginInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.password = source["password"];
	    }
	}
	export class StartMPesaChargeInput {
	    phone: string;
	    amountCents: number;
	    email: string;
	    reference: string;
	
	    static createFrom(source: any = {}) {
	        return new StartMPesaChargeInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.phone = source["phone"];
	        this.amountCents = source["amountCents"];
	        this.email = source["email"];
	        this.reference = source["reference"];
	    }
	}
	export class StockAdjustmentInput {
	    productId: string;
	    qtyChange: number;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new StockAdjustmentInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.productId = source["productId"];
	        this.qtyChange = source["qtyChange"];
	        this.reason = source["reason"];
	    }
	}
	export class SyncMutation {
	    mutationId: string;
	    sourceDeviceId?: string;
	    tableName: string;
	    recordId: string;
	    operation: string;
	    payload: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new SyncMutation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mutationId = source["mutationId"];
	        this.sourceDeviceId = source["sourceDeviceId"];
	        this.tableName = source["tableName"];
	        this.recordId = source["recordId"];
	        this.operation = source["operation"];
	        this.payload = source["payload"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class SyncPushResponse {
	    applied: number;
	    skipped: number;
	
	    static createFrom(source: any = {}) {
	        return new SyncPushResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.applied = source["applied"];
	        this.skipped = source["skipped"];
	    }
	}
	export class SyncRecord {
	    id: string;
	    tableName: string;
	    recordId: string;
	    operation: string;
	    payload: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new SyncRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.tableName = source["tableName"];
	        this.recordId = source["recordId"];
	        this.operation = source["operation"];
	        this.payload = source["payload"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class UpdateCategoryInput {
	    id: string;
	    name: string;
	    emoji: string;
	    displayOrder: number;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCategoryInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.emoji = source["emoji"];
	        this.displayOrder = source["displayOrder"];
	    }
	}
	export class UpdateProductInput {
	    id: string;
	    name: string;
	    sku: string;
	    barcode: string;
	    categoryId: string;
	    priceCents: number;
	    reorderLevel: number;
	    isActive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateProductInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.sku = source["sku"];
	        this.barcode = source["barcode"];
	        this.categoryId = source["categoryId"];
	        this.priceCents = source["priceCents"];
	        this.reorderLevel = source["reorderLevel"];
	        this.isActive = source["isActive"];
	    }
	}

}

