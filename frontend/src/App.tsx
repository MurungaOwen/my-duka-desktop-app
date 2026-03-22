import { useEffect, useState } from "react";
import { HealthStatus, Staff, BootstrapInput } from "./types/api";
import { MainLayout } from "./layouts/MainLayout";
import { BootstrapScreen } from "./pages/Bootstrap/BootstrapScreen";
import { LoginScreen } from "./pages/Auth/LoginScreen";
import * as API from "../wailsjs/go/main/App";

function App() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [currentUser, setCurrentUser] = useState<Staff | null>(null);

  const fetchHealth = async () => {
    try {
      const h: any = await API.BackendHealth();
      setHealth(h as HealthStatus);
    } catch (err: any) {
      if (err.includes("backend service unavailable")) {
        // Assume not initialized
        setHealth({
           initialized: false,
           mode: "standalone",
           syncEnabled: false,
           dbPath: "",
           deviceId: "",
           deviceName: ""
        });
      } else {
         setError(err.toString());
      }
    }
  };

  useEffect(() => {
    const initialize = async () => {
      setLoading(true);
      await fetchHealth();
      setLoading(false);
    };
    initialize();
  }, []);

  const handleBootstrap = async (data: BootstrapInput) => {
    try {
      setLoading(true);
      await API.BootstrapBusiness(data as any);
      await fetchHealth();
    } catch (err: any) {
      setError(err.toString() || "Setup failed");
    } finally {
      setLoading(false);
    }
  };

  const handleLogin = async (staff: Staff) => {
    setCurrentUser(staff);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-slate-900 text-white w-full">
        <div className="animate-pulse text-xl font-bold tracking-tight">Starting MyDuka Services...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center min-h-screen bg-slate-900 text-red-400 w-full p-4">
        <div className="bg-red-900/20 p-6 rounded-2xl border border-red-800/50 shadow-2xl max-w-md w-full text-center">
          <h2 className="text-xl font-bold mb-3">Startup Error</h2>
          <p className="text-slate-300 mb-4">{error}</p>
          <button onClick={() => window.location.reload()} className="px-4 py-2 bg-red-600/30 hover:bg-red-600/50 text-red-100 rounded-lg">Retry</button>
        </div>
      </div>
    );
  }

  if (health && !health.initialized) {
    return <BootstrapScreen onComplete={handleBootstrap} />;
  }

  if (!currentUser) {
    return <LoginScreen onLogin={handleLogin} />;
  }

  return <MainLayout currentUser={currentUser} health={health} onLogout={() => setCurrentUser(null)} />;
}

export default App;
