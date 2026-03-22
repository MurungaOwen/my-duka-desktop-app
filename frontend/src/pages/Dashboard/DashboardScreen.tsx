import React, { useEffect, useState } from "react";
import { TrendingUp, Package, AlertCircle, RefreshCw } from "lucide-react";
import { DashboardSummary, ListSales } from "../../../wailsjs/go/main/App";
import type { Sale } from "../../types/api";

type DailyPoint = { label: string; totalCents: number };

export const DashboardScreen: React.FC = () => {
  const [summary, setSummary] = useState({
    revenueTodayCents: 0,
    transactionsTodayCount: 0,
    lowStockCount: 0,
    outOfStockCount: 0
  });
  const [dailyRevenue, setDailyRevenue] = useState<DailyPoint[]>([]);
  const [paymentSplit, setPaymentSplit] = useState<Record<string, number>>({
    cash: 0,
    mpesa: 0,
    card: 0,
  });

  const fetchSummary = async () => {
     try {
       const [res, salesRaw] = await Promise.all([DashboardSummary(), ListSales(500)]);
       if (res) setSummary(res);

       const sales = ((salesRaw || []) as Sale[]).filter((s) => s.status === "completed");
       const byPayment = { cash: 0, mpesa: 0, card: 0 };
       for (const sale of sales) {
        if (sale.paymentMethod === "cash") byPayment.cash += sale.totalCents;
        if (sale.paymentMethod === "mpesa") byPayment.mpesa += sale.totalCents;
        if (sale.paymentMethod === "card") byPayment.card += sale.totalCents;
       }
       setPaymentSplit(byPayment);

       const map = new Map<string, number>();
       for (let i = 6; i >= 0; i--) {
        const d = new Date();
        d.setHours(0, 0, 0, 0);
        d.setDate(d.getDate() - i);
        map.set(d.toISOString().slice(0, 10), 0);
       }

       for (const sale of sales) {
        const key = new Date(sale.createdAt).toISOString().slice(0, 10);
        if (!map.has(key)) continue;
        map.set(key, (map.get(key) || 0) + sale.totalCents);
       }

       const points: DailyPoint[] = Array.from(map.entries()).map(([dateKey, totalCents]) => {
        const d = new Date(dateKey);
        return { label: d.toLocaleDateString(undefined, { weekday: "short" }), totalCents };
       });
       setDailyRevenue(points);
     } catch(err) {
       console.error("Failed to load dashboard summary", err);
     }
  };

  useEffect(() => {
     fetchSummary();
  }, []);

  const maxDaily = dailyRevenue.reduce((m, d) => Math.max(m, d.totalCents), 1);
  const splitTotal = paymentSplit.cash + paymentSplit.mpesa + paymentSplit.card;

  return (
    <div className="p-8 pb-12 max-w-6xl mx-auto w-full h-full overflow-y-auto">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-slate-800">Overview</h1>
          <p className="text-slate-500 mt-1">Your business at a glance today.</p>
        </div>
        <button onClick={fetchSummary} className="flex items-center gap-2 px-4 py-2 bg-white border border-slate-200 text-slate-600 rounded-lg hover:bg-slate-50 font-medium text-sm transition-colors shadow-sm active:scale-95">
          <RefreshCw size={16} /> Refresh
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <div className="bg-white p-6 rounded-2xl border border-slate-100 shadow-sm flex flex-col">
          <div className="flex items-center gap-3 text-slate-500 font-medium mb-4">
            <div className="p-2 bg-blue-50 text-blue-600 rounded-lg"><TrendingUp size={20} /></div>
            Today's Revenue
          </div>
          <div className="text-3xl font-black text-slate-800 mt-auto">
            <span className="text-lg text-slate-400 font-bold mr-1">KES</span>
            {(summary.revenueTodayCents / 100).toFixed(2)}
          </div>
        </div>

        <div className="bg-white p-6 rounded-2xl border border-slate-100 shadow-sm flex flex-col">
          <div className="flex items-center gap-3 text-slate-500 font-medium mb-4">
            <div className="p-2 bg-indigo-50 text-indigo-600 rounded-lg"><RefreshCw size={20} /></div>
            Transactions
          </div>
          <div className="text-3xl font-black text-slate-800 mt-auto">{summary.transactionsTodayCount}</div>
        </div>

        <div className="bg-white p-6 rounded-2xl border border-slate-100 shadow-sm flex flex-col">
          <div className="flex items-center gap-3 text-slate-500 font-medium mb-4">
            <div className="p-2 bg-amber-50 text-amber-600 rounded-lg"><AlertCircle size={20} /></div>
            Low Stock Alerts
          </div>
          <div className="text-3xl font-black text-amber-600 mt-auto">{summary.lowStockCount}</div>
        </div>

        <div className="bg-white p-6 rounded-2xl border border-slate-100 shadow-sm flex flex-col">
          <div className="flex items-center gap-3 text-slate-500 font-medium mb-4">
            <div className="p-2 bg-red-50 text-red-600 rounded-lg"><Package size={20} /></div>
            Out of Stock
          </div>
          <div className="text-3xl font-black text-red-600 mt-auto">{summary.outOfStockCount}</div>
        </div>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
        <section className="bg-white p-6 rounded-2xl border border-slate-100 shadow-sm">
          <div className="mb-4">
            <h2 className="text-lg font-bold text-slate-800">Reports: Revenue (Last 7 Days)</h2>
            <p className="text-sm text-slate-500">Daily completed sales totals in KES.</p>
          </div>
          <div className="h-56 flex items-end gap-3">
            {dailyRevenue.map((point) => {
              const pct = Math.max(4, Math.round((point.totalCents / maxDaily) * 100));
              return (
                <div key={point.label} className="flex-1 min-w-0">
                  <div className="h-44 flex items-end">
                    <div
                      className="w-full rounded-t-md bg-blue-500/90 hover:bg-blue-500 transition-colors"
                      style={{ height: `${pct}%` }}
                      title={`KES ${(point.totalCents / 100).toFixed(2)}`}
                    />
                  </div>
                  <p className="mt-2 text-xs text-slate-500 text-center">{point.label}</p>
                </div>
              );
            })}
          </div>
        </section>

        <section className="bg-white p-6 rounded-2xl border border-slate-100 shadow-sm">
          <div className="mb-4">
            <h2 className="text-lg font-bold text-slate-800">Reports: Payment Mix</h2>
            <p className="text-sm text-slate-500">Distribution across cash, M-Pesa, and card.</p>
          </div>
          <div className="space-y-4">
            {[
              { key: "cash", label: "Cash", color: "bg-emerald-500", value: paymentSplit.cash },
              { key: "mpesa", label: "M-Pesa", color: "bg-green-600", value: paymentSplit.mpesa },
              { key: "card", label: "Card", color: "bg-indigo-500", value: paymentSplit.card },
            ].map((row) => {
              const pct = splitTotal > 0 ? Math.round((row.value / splitTotal) * 100) : 0;
              return (
                <div key={row.key}>
                  <div className="flex justify-between text-sm mb-1">
                    <span className="font-medium text-slate-700">{row.label}</span>
                    <span className="text-slate-500">
                      KES {(row.value / 100).toFixed(2)} ({pct}%)
                    </span>
                  </div>
                  <div className="h-2 rounded-full bg-slate-100 overflow-hidden">
                    <div className={`h-full ${row.color}`} style={{ width: `${pct}%` }} />
                  </div>
                </div>
              );
            })}
          </div>
        </section>
      </div>
    </div>
  );
};
