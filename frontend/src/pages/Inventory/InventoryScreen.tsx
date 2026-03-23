import React, { useEffect, useMemo, useState } from "react";
import { Plus, Search, Filter, Edit, Package, Trash2, X } from "lucide-react";
import type { Category, Product } from "../../types/api";
import { CreateCategory, CreateProduct, DeleteCategory, DeleteProduct, ListCategories, ListProducts, UpdateCategory, UpdateProduct } from "../../../wailsjs/go/main/App";

type ProductFormState = {
  id?: string;
  name: string;
  sku: string;
  barcode: string;
  categoryId: string;
  priceKes: string;
  startingStock: string;
  reorderLevel: string;
  isActive: boolean;
};

type CategoryFormState = {
  id?: string;
  name: string;
  emoji: string;
  displayOrder: string;
};

const emptyForm: ProductFormState = {
  name: "",
  sku: "",
  barcode: "",
  categoryId: "",
  priceKes: "",
  startingStock: "0",
  reorderLevel: "0",
  isActive: true,
};

const emptyCategoryForm: CategoryFormState = {
  name: "",
  emoji: "",
  displayOrder: "0",
};

export const InventoryScreen: React.FC = () => {
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedCategoryId, setSelectedCategoryId] = useState<string>("all");
  const [statusFilter, setStatusFilter] = useState<"all" | "active" | "inactive">("all");
  const [stockFilter, setStockFilter] = useState<"all" | "low" | "out">("all");
  const [products, setProducts] = useState<Product[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const [showModal, setShowModal] = useState(false);
  const [formMode, setFormMode] = useState<"create" | "edit">("create");
  const [form, setForm] = useState<ProductFormState>(emptyForm);
  const [showCategoryModal, setShowCategoryModal] = useState(false);
  const [categoryMode, setCategoryMode] = useState<"create" | "edit">("create");
  const [categoryForm, setCategoryForm] = useState<CategoryFormState>(emptyCategoryForm);
  const [categoryError, setCategoryError] = useState<string | null>(null);
  const [categorySaving, setCategorySaving] = useState(false);

  const fetchCatalog = async () => {
    try {
      setLoading(true);
      setError(null);
      const [prods, cats] = await Promise.all([ListProducts(), ListCategories()]);
      setProducts((prods || []) as Product[]);
      setCategories((cats || []) as Category[]);
    } catch (err: any) {
      setError(err?.toString?.() || "Failed to load inventory.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCatalog();
  }, []);

  const filtered = useMemo(
    () =>
      products.filter((p) => {
        const q = searchQuery.toLowerCase();
        const matchesSearch =
          p.name.toLowerCase().includes(q) ||
          p.sku.toLowerCase().includes(q) ||
          p.barcode.toLowerCase().includes(q);
        const matchesCategory = selectedCategoryId === "all" ? true : p.categoryId === selectedCategoryId;
        const matchesStatus = statusFilter === "all" ? true : statusFilter === "active" ? p.isActive : !p.isActive;
        const isLowStock = p.stockQty <= p.reorderLevel;
        const matchesStock = stockFilter === "all" ? true : stockFilter === "low" ? isLowStock : p.stockQty <= 0;
        return matchesSearch && matchesCategory && matchesStatus && matchesStock;
      }),
    [products, searchQuery, selectedCategoryId, statusFilter, stockFilter]
  );

  const openCreateModal = () => {
    setFormMode("create");
    setForm(emptyForm);
    setError(null);
    setShowModal(true);
  };

  const openEditModal = (product: Product) => {
    setFormMode("edit");
    setForm({
      id: product.id,
      name: product.name,
      sku: product.sku || "",
      barcode: product.barcode || "",
      categoryId: product.categoryId || "",
      priceKes: (product.priceCents / 100).toFixed(2),
      startingStock: String(product.startingStock),
      reorderLevel: String(product.reorderLevel),
      isActive: product.isActive,
    });
    setError(null);
    setShowModal(true);
  };

  const closeModal = () => {
    if (saving) return;
    setShowModal(false);
    setForm(emptyForm);
    setError(null);
  };

  const submitForm = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    const name = form.name.trim();
    if (!name) {
      setError("Product name is required.");
      return;
    }
    const priceKesNum = Number(form.priceKes);
    const priceCents = Math.round(priceKesNum * 100);
    if (!Number.isFinite(priceKesNum) || priceKesNum < 0) {
      setError("Price must be a valid positive number.");
      return;
    }
    const reorderLevel = Number(form.reorderLevel);
    if (!Number.isFinite(reorderLevel) || reorderLevel < 0) {
      setError("Reorder level must be zero or more.");
      return;
    }

    setSaving(true);
    try {
      if (formMode === "create") {
        const startingStock = Number(form.startingStock);
        if (!Number.isFinite(startingStock) || startingStock < 0) {
          throw new Error("Starting stock must be zero or more.");
        }
        await CreateProduct({
          name,
          sku: form.sku.trim(),
          barcode: form.barcode.trim(),
          categoryId: form.categoryId,
          priceCents,
          startingStock,
          reorderLevel,
        } as any);
      } else {
        if (!form.id) throw new Error("Product id missing for edit.");
        await UpdateProduct({
          id: form.id,
          name,
          sku: form.sku.trim(),
          barcode: form.barcode.trim(),
          categoryId: form.categoryId,
          priceCents,
          reorderLevel,
          isActive: form.isActive,
        } as any);
      }
      closeModal();
      await fetchCatalog();
    } catch (err: any) {
      setError(err?.toString?.() || "Failed to save product.");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (product: Product) => {
    const ok = window.confirm(`Delete product "${product.name}"?`);
    if (!ok) return;
    try {
      await DeleteProduct(product.id);
      await fetchCatalog();
    } catch (err: any) {
      setError(err?.toString?.() || "Failed to delete product.");
    }
  };

  const openCreateCategoryModal = () => {
    setCategoryMode("create");
    setCategoryForm(emptyCategoryForm);
    setCategoryError(null);
    setShowCategoryModal(true);
  };

  const openEditCategoryModal = (category: Category) => {
    setCategoryMode("edit");
    setCategoryForm({
      id: category.id,
      name: category.name,
      emoji: category.emoji || "",
      displayOrder: String(category.displayOrder),
    });
    setCategoryError(null);
    setShowCategoryModal(true);
  };

  const closeCategoryModal = () => {
    if (categorySaving) return;
    setShowCategoryModal(false);
    setCategoryForm(emptyCategoryForm);
    setCategoryError(null);
  };

  const submitCategoryForm = async (e: React.FormEvent) => {
    e.preventDefault();
    setCategoryError(null);

    const name = categoryForm.name.trim();
    if (!name) {
      setCategoryError("Category name is required.");
      return;
    }
    const displayOrder = Number(categoryForm.displayOrder);
    if (!Number.isFinite(displayOrder) || displayOrder < 0) {
      setCategoryError("Display order must be zero or more.");
      return;
    }

    setCategorySaving(true);
    try {
      if (categoryMode === "create") {
        await CreateCategory({
          name,
          emoji: categoryForm.emoji.trim(),
          displayOrder,
        } as any);
      } else {
        if (!categoryForm.id) throw new Error("Category id missing for edit.");
        await UpdateCategory({
          id: categoryForm.id,
          name,
          emoji: categoryForm.emoji.trim(),
          displayOrder,
        } as any);
      }
      closeCategoryModal();
      await fetchCatalog();
    } catch (err: any) {
      setCategoryError(err?.toString?.() || "Failed to save category.");
    } finally {
      setCategorySaving(false);
    }
  };

  const handleDeleteCategory = async (categoryID: string, categoryName: string) => {
    const ok = window.confirm(`Delete category "${categoryName}"?`);
    if (!ok) return;
    try {
      await DeleteCategory(categoryID);
      await fetchCatalog();
    } catch (err: any) {
      setCategoryError(err?.toString?.() || "Failed to delete category.");
    }
  };

  return (
    <div className="p-8 max-w-7xl mx-auto w-full h-full flex flex-col overflow-hidden">
      <div className="flex items-center justify-between mb-8 shrink-0">
        <div>
          <h1 className="text-3xl font-bold text-slate-800">Inventory</h1>
          <p className="text-slate-500 mt-1">Manage your catalog, prices, and stock levels.</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={openCreateCategoryModal}
            className="px-4 py-2.5 border border-slate-200 bg-white rounded-xl font-semibold text-slate-700 hover:bg-slate-50"
          >
            Manage Categories
          </button>
          <button
            onClick={openCreateModal}
            className="flex items-center gap-2 px-5 py-2.5 bg-blue-600 text-white rounded-xl font-semibold hover:bg-blue-700 shadow-sm transition-all shadow-blue-600/20 active:scale-[0.98]"
          >
            <Plus size={18} /> Add Product
          </button>
        </div>
      </div>

      <div className="flex items-center justify-between gap-4 mb-6 shrink-0">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={18} />
          <input
            type="text"
            placeholder="Search products, SKUs, barcode..."
            className="w-full pl-10 pr-4 py-2 bg-white border border-slate-200 rounded-lg shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-colors"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>
        <button
          onClick={fetchCatalog}
          className="flex items-center gap-2 px-4 py-2 bg-white border border-slate-200 text-slate-600 rounded-lg hover:bg-slate-50 font-medium active:scale-95 transition-transform"
        >
          <Filter size={16} /> Refresh
        </button>
      </div>

      <div className="mb-4 grid grid-cols-1 md:grid-cols-4 gap-3 shrink-0">
        <select
          value={selectedCategoryId}
          onChange={(e) => setSelectedCategoryId(e.target.value)}
          className="px-3 py-2.5 rounded-lg border border-slate-300 bg-white outline-none focus:border-blue-500"
        >
          <option value="all">All Categories</option>
          {categories.map((c) => (
            <option key={c.id} value={c.id}>
              {c.emoji} {c.name}
            </option>
          ))}
        </select>

        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value as "all" | "active" | "inactive")}
          className="px-3 py-2.5 rounded-lg border border-slate-300 bg-white outline-none focus:border-blue-500"
        >
          <option value="all">All Status</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
        </select>

        <select
          value={stockFilter}
          onChange={(e) => setStockFilter(e.target.value as "all" | "low" | "out")}
          className="px-3 py-2.5 rounded-lg border border-slate-300 bg-white outline-none focus:border-blue-500"
        >
          <option value="all">All Stock</option>
          <option value="low">Low Stock</option>
          <option value="out">Out of Stock</option>
        </select>

        <button
          onClick={() => {
            setSearchQuery("");
            setSelectedCategoryId("all");
            setStatusFilter("all");
            setStockFilter("all");
          }}
          className="px-4 py-2.5 rounded-lg border border-slate-300 text-slate-600 font-medium hover:bg-slate-50"
        >
          Reset Filters
        </button>
      </div>

      {error && <p className="mb-3 text-sm text-red-600">{error}</p>}
      {categories.length > 0 && (
        <div className="mb-4 flex flex-wrap gap-2">
          {categories.map((c) => (
            <button
              key={c.id}
              onClick={() => openEditCategoryModal(c)}
              className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-slate-100 text-slate-700 text-sm hover:bg-slate-200"
              title="Edit category"
            >
              <span>{c.emoji || "•"}</span>
              <span>{c.name}</span>
            </button>
          ))}
        </div>
      )}

      <div className="bg-white rounded-xl border border-slate-200 shadow-sm flex-1 overflow-hidden flex flex-col relative">
        {loading && (
          <div className="absolute inset-0 bg-white/50 backdrop-blur-sm z-10 flex items-center justify-center">
            <div className="animate-pulse flex items-center gap-2 font-semibold text-blue-600">Loading Inventory...</div>
          </div>
        )}
        <div className="overflow-x-auto flex-1">
          <table className="w-full text-left border-collapse">
            <thead className="sticky top-0 bg-slate-50 z-0">
              <tr className="border-b border-slate-200 text-slate-500 text-sm font-semibold uppercase tracking-wider">
                <th className="px-6 py-4">Product Name</th>
                <th className="px-6 py-4">SKU / Barcode</th>
                <th className="px-6 py-4 text-right">Price</th>
                <th className="px-6 py-4 text-right">Stock</th>
                <th className="px-6 py-4 text-center">Status</th>
                <th className="px-6 py-4 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {filtered.map((product) => (
                <tr key={product.id} className="hover:bg-slate-50/50 transition-colors">
                  <td className="px-6 py-4">
                    <div className="font-medium text-slate-800">{product.name}</div>
                  </td>
                  <td className="px-6 py-4">
                    <div className="text-sm font-mono text-slate-500">{product.sku || "-"}</div>
                    <div className="text-xs text-slate-400">{product.barcode || "-"}</div>
                  </td>
                  <td className="px-6 py-4 text-right font-medium text-slate-800">KES {(product.priceCents / 100).toFixed(2)}</td>
                  <td className="px-6 py-4 text-right">
                    <div
                      className={`inline-flex items-center px-2.5 py-1 rounded-md text-sm font-semibold ${
                        product.stockQty <= product.reorderLevel
                          ? "bg-red-50 text-red-700 outline outline-1 outline-red-200"
                          : "bg-slate-100 text-slate-700"
                      }`}
                    >
                      {product.stockQty}
                    </div>
                  </td>
                  <td className="px-6 py-4 text-center">
                    {product.isActive ? (
                      <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-green-50 text-green-700">
                        <span className="w-1.5 h-1.5 rounded-full bg-green-500"></span>
                        Active
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-600">
                        <span className="w-1.5 h-1.5 rounded-full bg-slate-400"></span>
                        Inactive
                      </span>
                    )}
                  </td>
                  <td className="px-6 py-4 text-right">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => openEditModal(product)}
                        className="p-2 text-slate-400 hover:text-blue-600 hover:bg-blue-50 rounded-lg transition-colors"
                        title="Edit Product"
                      >
                        <Edit size={16} />
                      </button>
                      <button
                        onClick={() => handleDelete(product)}
                        className="p-2 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                        title="Delete Product"
                      >
                        <Trash2 size={16} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
              {!loading && filtered.length === 0 && (
                <tr>
                  <td colSpan={6} className="px-6 py-16 text-center text-slate-500">
                    <Package size={48} className="mx-auto mb-4 text-slate-300" />
                    {products.length === 0 ? "Your catalog is empty. Add your first product." : "No products found matching your search."}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {showModal && (
        <div className="absolute inset-0 z-50 flex items-center justify-center bg-slate-900/45 backdrop-blur-sm p-4">
          <div className="w-full max-w-lg rounded-2xl bg-white border border-slate-200 shadow-2xl overflow-hidden">
            <div className="px-5 py-4 border-b border-slate-200 flex items-center justify-between">
              <h3 className="font-bold text-slate-800">{formMode === "create" ? "Add Product" : "Edit Product"}</h3>
              <button onClick={closeModal} className="p-1 rounded hover:bg-slate-100 text-slate-500">
                <X size={18} />
              </button>
            </div>
            <form onSubmit={submitForm} className="p-5 space-y-3">
              <input
                value={form.name}
                onChange={(e) => setForm((prev) => ({ ...prev, name: e.target.value }))}
                placeholder="Product name"
                className="w-full px-3 py-2.5 rounded-lg border border-slate-300 outline-none focus:border-blue-500"
              />
              <div className="grid grid-cols-2 gap-3">
                <input
                  value={form.sku}
                  onChange={(e) => setForm((prev) => ({ ...prev, sku: e.target.value }))}
                  placeholder="SKU"
                  className="w-full px-3 py-2.5 rounded-lg border border-slate-300 outline-none focus:border-blue-500"
                />
                <input
                  value={form.barcode}
                  onChange={(e) => setForm((prev) => ({ ...prev, barcode: e.target.value }))}
                  placeholder="Barcode"
                  className="w-full px-3 py-2.5 rounded-lg border border-slate-300 outline-none focus:border-blue-500"
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <input
                  type="number"
                  step="0.01"
                  min="0"
                  value={form.priceKes}
                  onChange={(e) => setForm((prev) => ({ ...prev, priceKes: e.target.value }))}
                  placeholder="Price (KES)"
                  className="w-full px-3 py-2.5 rounded-lg border border-slate-300 outline-none focus:border-blue-500"
                />
                <input
                  type="number"
                  min="0"
                  value={form.reorderLevel}
                  onChange={(e) => setForm((prev) => ({ ...prev, reorderLevel: e.target.value }))}
                  placeholder="Reorder level"
                  className="w-full px-3 py-2.5 rounded-lg border border-slate-300 outline-none focus:border-blue-500"
                />
              </div>
              <select
                value={form.categoryId}
                onChange={(e) => setForm((prev) => ({ ...prev, categoryId: e.target.value }))}
                className="w-full px-3 py-2.5 rounded-lg border border-slate-300 outline-none focus:border-blue-500"
              >
                <option value="">No category</option>
                {categories.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.emoji} {c.name}
                  </option>
                ))}
              </select>

              {formMode === "create" && (
                <input
                  type="number"
                  min="0"
                  value={form.startingStock}
                  onChange={(e) => setForm((prev) => ({ ...prev, startingStock: e.target.value }))}
                  placeholder="Starting stock"
                  className="w-full px-3 py-2.5 rounded-lg border border-slate-300 outline-none focus:border-blue-500"
                />
              )}

              {formMode === "edit" && (
                <label className="flex items-center gap-2 text-sm text-slate-700">
                  <input
                    type="checkbox"
                    checked={form.isActive}
                    onChange={(e) => setForm((prev) => ({ ...prev, isActive: e.target.checked }))}
                  />
                  Product is active
                </label>
              )}

              {error && <p className="text-sm text-red-600">{error}</p>}

              <div className="flex gap-2 pt-2">
                <button
                  type="button"
                  onClick={closeModal}
                  className="flex-1 py-2.5 rounded-lg border border-slate-300 text-slate-600 font-semibold"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={saving}
                  className="flex-1 py-2.5 rounded-lg bg-blue-600 text-white font-semibold hover:bg-blue-700 disabled:bg-slate-300"
                >
                  {saving ? "Saving..." : formMode === "create" ? "Add Product" : "Save Changes"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showCategoryModal && (
        <div className="absolute inset-0 z-50 flex items-center justify-center bg-slate-900/45 backdrop-blur-sm p-4">
          <div className="w-full max-w-xl rounded-2xl bg-white border border-slate-200 shadow-2xl overflow-hidden">
            <div className="px-5 py-4 border-b border-slate-200 flex items-center justify-between">
              <h3 className="font-bold text-slate-800">Manage Categories</h3>
              <button onClick={closeCategoryModal} className="p-1 rounded hover:bg-slate-100 text-slate-500">
                <X size={18} />
              </button>
            </div>

            <form onSubmit={submitCategoryForm} className="p-5 space-y-3 border-b border-slate-200">
              <div className="grid grid-cols-3 gap-3">
                <input
                  value={categoryForm.name}
                  onChange={(e) => setCategoryForm((prev) => ({ ...prev, name: e.target.value }))}
                  placeholder="Category name"
                  className="col-span-2 w-full px-3 py-2.5 rounded-lg border border-slate-300 outline-none focus:border-blue-500"
                />
                <input
                  value={categoryForm.emoji}
                  onChange={(e) => setCategoryForm((prev) => ({ ...prev, emoji: e.target.value }))}
                  placeholder="Emoji"
                  className="w-full px-3 py-2.5 rounded-lg border border-slate-300 outline-none focus:border-blue-500"
                />
              </div>
              <input
                type="number"
                min="0"
                value={categoryForm.displayOrder}
                onChange={(e) => setCategoryForm((prev) => ({ ...prev, displayOrder: e.target.value }))}
                placeholder="Display order"
                className="w-full px-3 py-2.5 rounded-lg border border-slate-300 outline-none focus:border-blue-500"
              />

              {categoryError && <p className="text-sm text-red-600">{categoryError}</p>}

              <div className="flex gap-2">
                {categoryMode === "edit" && categoryForm.id && (
                  <button
                    type="button"
                    onClick={() => handleDeleteCategory(categoryForm.id!, categoryForm.name)}
                    className="px-4 py-2.5 rounded-lg border border-red-200 text-red-600 font-semibold hover:bg-red-50"
                  >
                    Delete
                  </button>
                )}
                <button
                  type="button"
                  onClick={closeCategoryModal}
                  className="flex-1 py-2.5 rounded-lg border border-slate-300 text-slate-600 font-semibold"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={categorySaving}
                  className="flex-1 py-2.5 rounded-lg bg-blue-600 text-white font-semibold hover:bg-blue-700 disabled:bg-slate-300"
                >
                  {categorySaving ? "Saving..." : categoryMode === "create" ? "Add Category" : "Save Category"}
                </button>
              </div>
            </form>

            <div className="max-h-60 overflow-auto p-4">
              <p className="text-xs font-semibold uppercase text-slate-500 mb-2">Existing Categories</p>
              <div className="space-y-2">
                {categories.map((c) => (
                  <div key={c.id} className="flex items-center justify-between px-3 py-2 rounded-lg bg-slate-50 border border-slate-200">
                    <div className="text-sm text-slate-700">
                      <span className="mr-2">{c.emoji || "•"}</span>
                      {c.name}
                    </div>
                    <button
                      onClick={() => openEditCategoryModal(c)}
                      className="text-xs px-2 py-1 rounded border border-slate-300 text-slate-600 hover:bg-white"
                    >
                      Edit
                    </button>
                  </div>
                ))}
                {categories.length === 0 && <p className="text-sm text-slate-500">No categories yet.</p>}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
