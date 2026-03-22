import React, { useState, useEffect, useRef } from "react";
import { Search, Plus, Minus, Trash2, ShoppingBag } from "lucide-react";
import type { Product, Category, SaleItemInput } from "../../types/api";
import { ListCategories, ListProducts, CreateSale, StartMPesaCharge, VerifyMPesaCharge } from "../../../wailsjs/go/main/App";

export const POSScreen: React.FC<{ cashierId: string }> = ({ cashierId }) => {
  const [categories, setCategories] = useState<Category[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [activeCategoryId, setActiveCategoryId] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [cart, setCart] = useState<(Product & { cartQty: number })[]>([]);
  const [isProcessing, setIsProcessing] = useState(false);
  const [checkoutMethod, setCheckoutMethod] = useState<"cash" | "mpesa" | null>(null);
  const [cashReceived, setCashReceived] = useState("");
  const [mpesaPhone, setMpesaPhone] = useState("+254");
  const [mpesaStage, setMpesaStage] = useState<"idle" | "initiated" | "confirmed">("idle");
  const [mpesaReference, setMpesaReference] = useState("");
  const [mpesaStatusText, setMpesaStatusText] = useState("");
  const [mpesaSimulated, setMpesaSimulated] = useState(false);
  const [mpesaPolling, setMpesaPolling] = useState(false);
  const [mpesaPollCount, setMpesaPollCount] = useState(0);
  const pollTimerRef = useRef<number | null>(null);

  const fetchCatalog = async () => {
    try {
      const cats: any = await ListCategories();
      const prods: any = await ListProducts();
      setCategories(cats || []);
      setProducts(prods || []);
    } catch (err) {
      console.error("Failed to load catalog", err);
    }
  };

  useEffect(() => {
    fetchCatalog();
  }, []);

  const stopMpesaPolling = () => {
    if (pollTimerRef.current !== null) {
      window.clearInterval(pollTimerRef.current);
      pollTimerRef.current = null;
    }
    setMpesaPolling(false);
    setMpesaPollCount(0);
  };

  useEffect(() => {
    return () => stopMpesaPolling();
  }, []);

  const filteredProducts = products.filter((p) => {
    if (!p.isActive) return false;
    const matchesCat = activeCategoryId ? p.categoryId === activeCategoryId : true;
    const matchesSearch = p.name.toLowerCase().includes(searchQuery.toLowerCase()) || p.sku.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesCat && matchesSearch;
  });

  const addToCart = (product: Product) => {
    setCart((prev) => {
      const existing = prev.find((item) => item.id === product.id);
      if (existing) {
         if (existing.cartQty + 1 > product.stockQty) return prev; // Avoid ordering more than stock
         return prev.map((item) => item.id === product.id ? { ...item, cartQty: item.cartQty + 1 } : item);
      }
      if (product.stockQty < 1) return prev;
      return [...prev, { ...product, cartQty: 1 }];
    });
  };

  const updateCartQty = (id: string, delta: number) => {
    setCart((prev) =>
      prev.map((item) => {
        if (item.id === id) {
          const newQty = Math.max(0, Math.min(item.cartQty + delta, item.stockQty));
          return { ...item, cartQty: newQty };
        }
        return item;
      }).filter((item) => item.cartQty > 0)
    );
  };

  const handleCheckout = async (paymentMethod: "cash" | "mpesa" | "card") => {
    if (cart.length === 0) return;
    setIsProcessing(true);
    try {
        const items: SaleItemInput[] = cart.map(c => ({
            productId: c.id,
            quantity: c.cartQty
        }));
        await CreateSale({ cashierStaffId: cashierId, paymentMethod, items } as any);
        setCart([]);
        setCheckoutMethod(null);
        setCashReceived("");
        setMpesaPhone("+254");
        setMpesaStage("idle");
        setMpesaReference("");
        setMpesaStatusText("");
        setMpesaSimulated(false);
        stopMpesaPolling();
        await fetchCatalog(); // refresh stock
    } catch(err: any) {
        alert("Failed to checkout: " + err);
    } finally {
        setIsProcessing(false);
    }
  };

  const subtotal = cart.reduce((sum, item) => sum + (item.priceCents * item.cartQty), 0);
  const totalStr = (subtotal / 100).toFixed(2);
  const cashReceivedCents = Number(cashReceived || "0") * 100;
  const changeCents = Math.max(0, cashReceivedCents - subtotal);

  const normalizePhoneInput = (value: string) => {
    let digits = value.replace(/[^\d]/g, "");
    if (digits.startsWith("254")) {
      digits = digits.slice(3);
    } else if (digits.startsWith("0")) {
      digits = digits.slice(1);
    }
    digits = digits.slice(0, 9);
    return `+254${digits}`;
  };

  const initiateMpesa = async () => {
    if (mpesaPhone.trim().length !== 13) return;
    setIsProcessing(true);
    setMpesaStatusText("");
    setMpesaSimulated(false);
    try {
      const session: any = await StartMPesaCharge({
        phone: mpesaPhone.trim(),
        amountCents: subtotal,
      } as any);
      setMpesaReference(session?.reference || "");
      setMpesaStage("initiated");
      setMpesaStatusText(session?.displayText || session?.message || "Charge initiated. Ask customer to approve on phone.");
      if (session?.reference) {
        setMpesaPolling(true);
        setMpesaPollCount(0);
        const ref = String(session.reference);
        pollTimerRef.current = window.setInterval(() => {
          verifyMpesa(true, ref);
        }, 3000);
      }
    } catch (err: any) {
      const msg = err?.toString?.() || "Failed to initiate M-Pesa payment";
      if (msg.toLowerCase().includes("paystack is not configured")) {
        setMpesaSimulated(true);
        setMpesaStage("initiated");
        setMpesaStatusText("Paystack not configured. Running in simulated confirmation mode.");
      } else {
        alert("M-Pesa initiation failed: " + msg);
      }
    } finally {
      setIsProcessing(false);
    }
  };

  const verifyMpesa = async (silent = false, referenceOverride?: string) => {
    const reference = referenceOverride || mpesaReference;
    if (!reference) return;
    if (!silent) setIsProcessing(true);
    try {
      const status: any = await VerifyMPesaCharge(reference);
      if (status?.paid) {
        stopMpesaPolling();
        setMpesaStage("confirmed");
        setMpesaStatusText("Payment confirmed by Paystack.");
        return;
      }
      if (silent) {
        setMpesaPollCount((prev) => {
          const next = prev + 1;
          if (next >= 40) {
            stopMpesaPolling();
            setMpesaStatusText("Payment still pending after 2 minutes. You can keep checking manually.");
          }
          return next;
        });
      }
      setMpesaStatusText(status?.displayText || status?.gatewayResponse || status?.message || `Current status: ${status?.status || "pending"}`);
    } catch (err: any) {
      if (!silent) {
        setMpesaStatusText("Could not verify payment yet. Retry in a few seconds.");
      }
    } finally {
      if (!silent) setIsProcessing(false);
    }
  };

  return (
    <div className="flex h-full w-full bg-slate-50">
      <div className="flex-1 flex flex-col p-6 overflow-hidden">
        <div className="flex gap-4 mb-6 shrink-0">
          <div className="relative flex-1 max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={20} />
            <input
              type="text"
              placeholder="Search products, barcode, SKU..."
              className="w-full pl-10 pr-4 py-3 bg-white border border-slate-200 rounded-xl shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all font-medium"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>
        </div>

        <div className="flex gap-3 mb-6 overflow-x-auto pb-2 shrink-0 scrollbar-hide">
          <button onClick={() => setActiveCategoryId(null)} className={`px-5 py-2.5 rounded-full whitespace-nowrap font-medium transition-all ${activeCategoryId === null ? "bg-slate-800 text-white shadow-md shadow-slate-800/20" : "bg-white text-slate-600 border border-slate-200 hover:border-slate-300"}`}>All Items</button>
          {categories.map((c) => (
            <button key={c.id} onClick={() => setActiveCategoryId(c.id)} className={`px-5 py-2.5 rounded-full whitespace-nowrap font-medium transition-all flex items-center gap-2 ${activeCategoryId === c.id ? "bg-slate-800 text-white shadow-md shadow-slate-800/20" : "bg-white text-slate-600 border border-slate-200 hover:border-slate-300"}`}>
              <span>{c.emoji}</span>{c.name}
            </button>
          ))}
        </div>

        <div className="flex-1 overflow-y-auto">
          <div className="grid grid-cols-3 xl:grid-cols-4 gap-4 pb-6">
            {filteredProducts.map((p) => (
              <button key={p.id} onClick={() => addToCart(p)} disabled={p.stockQty <= 0} className={`bg-white p-4 rounded-2xl border shadow-sm transition-all text-left flex flex-col h-40 group ${p.stockQty > 0 ? 'border-slate-100 hover:shadow-md hover:border-blue-200 active:scale-95' : 'border-slate-100 opacity-50 cursor-not-allowed'}`}>
                <div className="text-xl mb-auto font-medium text-slate-800 leading-tight line-clamp-2">{p.name}</div>
                <div className="mt-auto">
                  <div className="text-sm font-semibold text-blue-600 mb-1">KES {(p.priceCents / 100).toFixed(2)}</div>
                  <div className={`text-xs font-semibold px-2 py-0.5 rounded outline w-max ${p.stockQty <= 0 ? 'bg-red-50 text-red-600 outline-red-100' : p.stockQty <= p.reorderLevel ? 'bg-amber-50 text-amber-600 outline-amber-100' : 'bg-slate-50 text-slate-500 outline-slate-100'}`}>
                    {p.stockQty > 0 ? `${p.stockQty} in stock` : 'Out of Stock'}
                  </div>
                </div>
              </button>
            ))}
          </div>
        </div>
      </div>

      <div className="w-96 bg-white border-l border-slate-100 flex flex-col shadow-[-4px_0_24px_-12px_rgba(0,0,0,0.05)] z-10">
        <div className="p-6 border-b border-slate-100 flex items-center justify-between bg-slate-50/50 shrink-0">
          <h2 className="text-lg font-bold text-slate-800 flex items-center gap-2"><ShoppingBag size={20} className="text-blue-600"/> Current Sale</h2>
          <span className="bg-slate-800 text-white text-xs font-bold px-2 py-1 rounded-md">{cart.reduce((s, i) => s + i.cartQty, 0)} Items</span>
        </div>

        <div className="flex-1 overflow-y-auto p-4 space-y-3">
          {cart.map((item) => (
             <div key={item.id} className="flex gap-3 bg-white p-3 rounded-xl border border-slate-100 shadow-sm">
               <div className="flex-1">
                 <div className="font-semibold text-sm text-slate-800 line-clamp-1">{item.name}</div>
                 <div className="text-xs text-slate-500 mt-1">KES {(item.priceCents / 100).toFixed(2)}</div>
               </div>
               <div className="flex flex-col items-end justify-between">
                 <div className="font-bold text-slate-800 mb-2">KES {((item.priceCents * item.cartQty) / 100).toFixed(2)}</div>
                 <div className="flex items-center gap-2 bg-slate-50 rounded-lg outline outline-1 outline-slate-200 h-8">
                   <button onClick={() => updateCartQty(item.id, -1)} className="w-8 h-full flex items-center justify-center text-slate-500 hover:text-red-500"><Minus size={14} /></button>
                   <span className="w-6 text-center text-sm font-semibold text-slate-800">{item.cartQty}</span>
                   <button onClick={() => updateCartQty(item.id, 1)} className="w-8 h-full flex items-center justify-center text-slate-500 hover:text-blue-600"><Plus size={14} /></button>
                 </div>
               </div>
             </div>
          ))}
        </div>

        <div className="p-6 bg-slate-50 border-t border-slate-100 shrink-0">
          <div className="flex justify-between mb-2 text-slate-500 text-sm"><span>Subtotal</span><span>KES {totalStr}</span></div>
          <div className="flex justify-between mb-6 text-slate-500 text-sm"><span>Tax</span><span>Included</span></div>
          <div className="flex justify-between mb-6 text-xl font-black text-slate-800"><span>Total</span><span>KES {totalStr}</span></div>

          {checkoutMethod === null && (
            <div className="flex gap-2">
              <button disabled={cart.length === 0 || isProcessing} onClick={() => { setMpesaPhone("+254"); setCheckoutMethod("mpesa"); }} className="flex-1 bg-emerald-600 disabled:bg-slate-300 text-white py-3 rounded-xl font-bold text-sm hover:bg-emerald-700 active:scale-[0.98] transition-all">M-Pesa</button>
              <button disabled={cart.length === 0 || isProcessing} onClick={() => setCheckoutMethod("cash")} className="flex-1 bg-blue-600 disabled:bg-slate-300 text-white py-3 rounded-xl font-bold text-sm hover:bg-blue-700 active:scale-[0.98] transition-all">Cash</button>
            </div>
          )}

          {checkoutMethod === "cash" && (
            <div className="space-y-3">
              <label className="block">
                <span className="text-xs text-slate-500 block mb-1">Amount received (KES)</span>
                <input
                  type="number"
                  min="0"
                  step="0.01"
                  value={cashReceived}
                  onChange={(e) => setCashReceived(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg border border-slate-300 outline-none focus:border-blue-500 bg-white"
                  placeholder="0.00"
                />
              </label>
              <div className="text-sm flex justify-between">
                <span className="text-slate-500">Change</span>
                <span className="font-bold text-slate-800">KES {(changeCents / 100).toFixed(2)}</span>
              </div>
              {cashReceived !== "" && cashReceivedCents < subtotal && (
                <p className="text-xs text-red-500">Received amount is less than total.</p>
              )}
              <div className="flex gap-2">
                <button onClick={() => setCheckoutMethod(null)} className="flex-1 py-2.5 rounded-xl border border-slate-300 text-slate-600 font-semibold">Back</button>
                <button
                  disabled={isProcessing || cart.length === 0 || cashReceivedCents < subtotal}
                  onClick={() => handleCheckout("cash")}
                  className="flex-1 bg-blue-600 disabled:bg-slate-300 text-white py-2.5 rounded-xl font-bold text-sm hover:bg-blue-700"
                >
                  Confirm Cash
                </button>
              </div>
            </div>
          )}

          {checkoutMethod === "mpesa" && (
            <div className="space-y-3">
              <label className="block">
                <span className="text-xs text-slate-500 block mb-1">M-Pesa Phone Number</span>
                <input
                  type="tel"
                  value={mpesaPhone}
                  onChange={(e) => {
                    setMpesaPhone(normalizePhoneInput(e.target.value));
                    if (mpesaStage !== "idle") setMpesaStage("idle");
                    setMpesaReference("");
                    setMpesaStatusText("");
                    setMpesaSimulated(false);
                    stopMpesaPolling();
                  }}
                  className="w-full px-3 py-2 rounded-lg border border-slate-300 outline-none focus:border-emerald-500 bg-white"
                  placeholder="+2547XXXXXXXX"
                />
              </label>

              {mpesaStage === "idle" && (
                <button
                  disabled={mpesaPhone.trim().length !== 13 || isProcessing}
                  onClick={initiateMpesa}
                  className="w-full bg-emerald-600 disabled:bg-slate-300 text-white py-2.5 rounded-xl font-bold text-sm hover:bg-emerald-700"
                >
                  Initiate M-Pesa Charge
                </button>
              )}

              {mpesaStage === "initiated" && (
                <div className="space-y-2">
                  <p className="text-xs text-amber-700 bg-amber-50 rounded p-2">
                    {mpesaStatusText || `Charge created for ${mpesaPhone}.`}
                  </p>
                  {mpesaReference && (
                    <p className="text-[11px] text-slate-500 font-mono bg-slate-50 rounded p-2 break-all">Ref: {mpesaReference}</p>
                  )}
                  {!mpesaSimulated && (
                    <>
                      <p className="text-[11px] text-slate-500">
                        {mpesaPolling ? "Auto-checking payment every 3 seconds..." : "Auto-check stopped."}
                      </p>
                      <button
                        onClick={() => verifyMpesa(false)}
                        disabled={isProcessing}
                        className="w-full bg-emerald-700 text-white py-2.5 rounded-xl font-bold text-sm hover:bg-emerald-800 disabled:bg-slate-300"
                      >
                        Check Paystack Confirmation
                      </button>
                    </>
                  )}
                  {mpesaSimulated && (
                    <button
                      onClick={() => setMpesaStage("confirmed")}
                      className="w-full bg-emerald-700 text-white py-2.5 rounded-xl font-bold text-sm hover:bg-emerald-800"
                    >
                      Confirm Customer Paid (Simulated)
                    </button>
                  )}
                </div>
              )}

              {mpesaStage === "confirmed" && (
                <div className="space-y-2">
                  <p className="text-xs text-emerald-700 bg-emerald-50 rounded p-2">Payment confirmed. Complete sale now.</p>
                  <button
                    disabled={isProcessing || cart.length === 0}
                    onClick={() => handleCheckout("mpesa")}
                    className="w-full bg-emerald-600 disabled:bg-slate-300 text-white py-2.5 rounded-xl font-bold text-sm hover:bg-emerald-700"
                  >
                    Complete M-Pesa Sale
                  </button>
                </div>
              )}

              <button onClick={() => { stopMpesaPolling(); setCheckoutMethod(null); setMpesaStage("idle"); setMpesaPhone("+254"); }} className="w-full py-2.5 rounded-xl border border-slate-300 text-slate-600 font-semibold">Back</button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
