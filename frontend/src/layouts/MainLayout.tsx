import React, { useState } from "react";
import { Staff, HealthStatus } from "../types/api";
import { Sidebar } from "../components/Sidebar";

import { POSScreen } from "../pages/POS/POSScreen";
import { DashboardScreen } from "../pages/Dashboard/DashboardScreen";
import { InventoryScreen } from "../pages/Inventory/InventoryScreen";
import { SettingsScreen } from "../pages/Settings/SettingsScreen";

export type ViewState = "pos" | "dashboard" | "inventory" | "settings";

interface MainLayoutProps {
  currentUser: Staff;
  health: HealthStatus | null;
  onLogout: () => void;
}

export const MainLayout: React.FC<MainLayoutProps> = ({ currentUser, health, onLogout }) => {
  const [activeView, setActiveView] = useState<ViewState>("pos");

  const renderView = () => {
    switch (activeView) {
      case "pos":
        return <POSScreen cashierId={currentUser.id} />;
      case "dashboard":
        return <DashboardScreen />;
      case "inventory":
        return <InventoryScreen />;
      case "settings":
        return <SettingsScreen />;
      default:
        return <POSScreen cashierId={currentUser.id} />;
    }
  };

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-white text-slate-900 selection:bg-blue-200">
      <Sidebar 
        activeView={activeView} 
        setActiveView={setActiveView} 
        onLogout={onLogout} 
        role={currentUser.role} 
      />
      <main className="flex-1 flex flex-col h-full relative">
        {/* Topbar */}
        <header className="h-16 border-b border-slate-100 flex items-center justify-end px-8 bg-white shrink-0 z-20 shadow-sm shadow-slate-100/50">
          <div className="flex items-center gap-6 text-sm">
            <span className="text-slate-500 font-medium tracking-wide">
              Logged in as <span className="font-bold text-slate-800">{currentUser.name}</span>
            </span>
            <div className={`px-3 py-1.5 rounded-full text-xs font-bold tracking-wide flex items-center gap-2 shadow-sm ${
              health?.mode === 'lan_sync' 
                ? 'bg-emerald-50 text-emerald-700 border border-emerald-100' 
                : 'bg-slate-50 text-slate-600 border border-slate-200'
            }`}>
              {health?.mode === 'lan_sync' && <span className="w-1.5 h-1.5 bg-emerald-500 rounded-full animate-pulse"></span>}
              {health?.mode === 'lan_sync' ? 'SYNC ONLINE' : 'STANDALONE MODE'}
            </div>
          </div>
        </header>

        {/* Content Area */}
        <div className="flex-1 overflow-hidden relative bg-slate-50/30">
          {renderView()}
        </div>
      </main>
    </div>
  );
};
