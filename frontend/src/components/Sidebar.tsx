import React from "react";
import { LayoutDashboard, ShoppingCart, Package, Settings, LogOut } from "lucide-react";
import type { ViewState } from "../layouts/MainLayout";

interface SidebarProps {
  activeView: ViewState;
  setActiveView: (view: ViewState) => void;
  onLogout: () => void;
  role: "admin" | "cashier";
}

export const Sidebar: React.FC<SidebarProps> = ({ activeView, setActiveView, onLogout, role }) => {
  const navItems = [
    { id: "pos" as ViewState, label: "POS", icon: ShoppingCart },
    { id: "dashboard" as ViewState, label: "Dashboard", icon: LayoutDashboard },
    { id: "inventory" as ViewState, label: "Inventory", icon: Package },
    { id: "settings" as ViewState, label: "Settings", icon: Settings },
  ];

  return (
    <aside className="w-64 bg-slate-900 border-r border-slate-800 text-slate-300 flex flex-col h-full shrink-0">
      <div className="h-16 flex items-center px-6 border-b border-slate-800 text-white font-bold text-xl tracking-tight">
        <span className="text-blue-500 mr-2">✦</span>
        MyDuka
      </div>
      
      <nav className="flex-1 overflow-y-auto py-6 flex flex-col gap-1 px-3">
        {navItems.map((item) => {
          // Cashiers only see POS. Hide other tabs if not admin.
          if (role !== "admin" && item.id !== "pos") return null;
          
          const isActive = activeView === item.id;
          const Icon = item.icon;
          
          return (
            <button
              key={item.id}
              onClick={() => setActiveView(item.id)}
              className={`flex items-center gap-3 w-full px-3 py-3 rounded-lg text-sm font-medium transition-all duration-200 ${
                isActive 
                  ? "bg-blue-600 shadow-sm shadow-blue-900/50 text-white" 
                  : "hover:bg-slate-800 hover:text-white"
              }`}
            >
              <Icon size={18} className={isActive ? "text-white" : "text-slate-400"} />
              {item.label}
            </button>
          );
        })}
      </nav>

      <div className="p-4 border-t border-slate-800">
        <button 
          onClick={onLogout}
          className="flex flex-row items-center justify-center gap-2 w-full py-2.5 px-4 rounded-lg bg-slate-800 hover:bg-slate-700 text-sm font-medium transition-colors text-slate-300 hover:text-white"
        >
          <LogOut size={16} />
          Lock UI
        </button>
      </div>
    </aside>
  );
};
