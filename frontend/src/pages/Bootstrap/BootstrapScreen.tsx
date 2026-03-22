import React, { useState } from "react";
import { Store, MapPin, BadgeDollarSign, Receipt } from "lucide-react";
import type { BootstrapInput } from "../../types/api";

export const BootstrapScreen = ({ onComplete }: { onComplete: (data: BootstrapInput) => void }) => {
  const [formData, setFormData] = useState<BootstrapInput>({
    businessName: "",
    location: "",
    currency: "KES",
    vatRate: "16",
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (formData.businessName && formData.location) {
      onComplete(formData);
    }
  };

  return (
    <div className="flex min-h-screen bg-slate-900 text-slate-100 font-sans selection:bg-blue-500/30 w-full">
      <div className="flex-1 flex flex-col justify-center px-8 sm:px-12 lg:px-24">
        <div className="w-full max-w-md mx-auto">
          <div className="mb-10 flex flex-col items-center text-center">
            <div className="w-16 h-16 bg-blue-600 rounded-2xl flex items-center justify-center mb-6 shadow-lg shadow-blue-600/20">
              <span className="text-3xl text-white font-black tracking-tighter">✦</span>
            </div>
            <h1 className="text-4xl font-extrabold tracking-tight text-white mb-3">Welcome to MyDuka</h1>
            <p className="text-slate-400 text-lg">Let's set up your store's database to get started.</p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="space-y-4">
              {/* Business Name */}
              <div className="relative">
                <label className="block text-sm font-semibold text-slate-300 mb-1.5 ml-1">Store Name</label>
                <div className="relative">
                  <Store className="absolute left-4 top-1/2 -translate-y-1/2 text-slate-500" size={20} />
                  <input
                    autoFocus
                    required
                    type="text"
                    placeholder="e.g. Mama Naivas Groceries"
                    className="w-full pl-12 pr-4 py-3.5 bg-slate-800 border-2 border-slate-700/50 rounded-xl focus:outline-none focus:border-blue-500 focus:bg-slate-800 text-white placeholder-slate-500 transition-all font-medium"
                    value={formData.businessName}
                    onChange={(e) => setFormData({ ...formData, businessName: e.target.value })}
                  />
                </div>
              </div>

              {/* Location */}
              <div className="relative">
                <label className="block text-sm font-semibold text-slate-300 mb-1.5 ml-1">Location</label>
                <div className="relative">
                  <MapPin className="absolute left-4 top-1/2 -translate-y-1/2 text-slate-500" size={20} />
                  <input
                    required
                    type="text"
                    placeholder="e.g. Nairobi, CBD"
                    className="w-full pl-12 pr-4 py-3.5 bg-slate-800 border-2 border-slate-700/50 rounded-xl focus:outline-none focus:border-blue-500 focus:bg-slate-800 text-white placeholder-slate-500 transition-all font-medium"
                    value={formData.location}
                    onChange={(e) => setFormData({ ...formData, location: e.target.value })}
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                {/* Currency */}
                <div className="relative">
                  <label className="block text-sm font-semibold text-slate-300 mb-1.5 ml-1">Currency</label>
                  <div className="relative">
                    <BadgeDollarSign className="absolute left-4 top-1/2 -translate-y-1/2 text-slate-500" size={20} />
                    <input
                      required
                      type="text"
                      placeholder="KES"
                      className="w-full pl-12 pr-4 py-3.5 bg-slate-800 border-2 border-slate-700/50 rounded-xl focus:outline-none focus:border-blue-500 focus:bg-slate-800 text-white placeholder-slate-500 transition-all font-medium"
                      value={formData.currency}
                      onChange={(e) => setFormData({ ...formData, currency: e.target.value })}
                    />
                  </div>
                </div>

                {/* VAT Rate */}
                <div className="relative">
                  <label className="block text-sm font-semibold text-slate-300 mb-1.5 ml-1">VAT Rate (%)</label>
                  <div className="relative">
                    <Receipt className="absolute left-4 top-1/2 -translate-y-1/2 text-slate-500" size={20} />
                    <input
                      required
                      type="number"
                      placeholder="e.g. 16"
                      className="w-full pl-12 pr-4 py-3.5 bg-slate-800 border-2 border-slate-700/50 rounded-xl focus:outline-none focus:border-blue-500 focus:bg-slate-800 text-white placeholder-slate-500 transition-all font-medium"
                      value={formData.vatRate}
                      onChange={(e) => setFormData({ ...formData, vatRate: e.target.value })}
                    />
                  </div>
                </div>
              </div>
            </div>

            <button
              type="submit"
              className="w-full bg-blue-600 hover:bg-blue-500 py-4 rounded-xl text-white font-bold text-lg shadow-lg shadow-blue-600/20 active:scale-[0.98] transition-all flex items-center justify-center gap-2 mt-8"
            >
              Initialize Store
            </button>
          </form>
        </div>
      </div>
      
      {/* Decorative Side Pane */}
      <div className="hidden lg:flex flex-1 bg-slate-800 relative overflow-hidden items-center justify-center border-l border-slate-700/50">
        <div className="absolute top-0 right-0 w-[800px] h-[800px] bg-blue-500/10 rounded-full blur-[120px] translate-x-1/3 -translate-y-1/4 mix-blend-screen pointer-events-none"></div>
        <div className="absolute bottom-0 left-0 w-[600px] h-[600px] bg-indigo-500/10 rounded-full blur-[100px] -translate-x-1/4 translate-y-1/3 mix-blend-screen pointer-events-none"></div>
        
        <div className="max-w-md z-10 text-center relative p-12 bg-slate-900/40 backdrop-blur-xl border border-slate-700/50 rounded-3xl shadow-2xl">
           <h2 className="text-2xl font-bold text-white mb-4">Local First & Offline Ready</h2>
           <p className="text-slate-400 font-medium">MyDuka is designed to run seamlessly in your shop, with or without internet. Your data stays yours, reconciled smoothly via local network sync.</p>
        </div>
      </div>
    </div>
  );
};
