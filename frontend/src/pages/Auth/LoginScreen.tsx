import React, { useState } from "react";
import { Lock, UserCircle, User, KeyRound } from "lucide-react";
import type { Staff } from "../../types/api";
import { AuthenticateStaff } from "../../../wailsjs/go/main/App";

export const LoginScreen = ({ onLogin }: { onLogin: (staff: Staff) => Promise<void> | void }) => {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    if (!username.trim() || !password) {
      setError("Enter username and password");
      return;
    }
    setLoading(true);
    try {
      const staff: any = await AuthenticateStaff({
        username: username.trim().toLowerCase(),
        password,
      } as any);
      await onLogin(staff);
    } catch (err: any) {
      setError(err?.toString?.() || "Login failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex h-screen bg-slate-900 justify-center items-center text-slate-100 font-sans w-full relative overflow-hidden">
      <div className="absolute inset-0 z-0">
        <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-blue-600/5 rounded-full blur-3xl pointer-events-none"></div>
        <div className="absolute bottom-1/4 right-1/4 w-[500px] h-[500px] bg-purple-600/5 rounded-full blur-3xl pointer-events-none"></div>
      </div>

      <div className="z-10 w-full max-w-sm">
        <div className="text-center mb-6">
          <div className="w-20 h-20 bg-slate-800 rounded-full flex flex-col items-center justify-center mx-auto mb-4 shadow-xl border border-slate-700/50 text-blue-500">
            <UserCircle size={40} strokeWidth={1.5} />
          </div>
          <h1 className="text-3xl font-bold text-white tracking-tight mb-2">Staff Login</h1>
          <p className="text-slate-400 text-sm">Use your username and password</p>
        </div>

        <form onSubmit={handleSubmit} className="bg-slate-800/80 backdrop-blur-xl p-8 rounded-3xl border border-slate-700/50 shadow-2xl space-y-4">
          <label className="block">
            <span className="text-xs text-slate-300 mb-1 block">Username</span>
            <div className="relative">
              <User size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                className="w-full pl-9 pr-3 py-2.5 rounded-xl bg-slate-900/80 border border-slate-700 text-white outline-none focus:border-blue-500"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete="username"
                placeholder="e.g. owner"
              />
            </div>
          </label>

          <label className="block">
            <span className="text-xs text-slate-300 mb-1 block">Password</span>
            <div className="relative">
              <KeyRound size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                type="password"
                className="w-full pl-9 pr-3 py-2.5 rounded-xl bg-slate-900/80 border border-slate-700 text-white outline-none focus:border-blue-500"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="current-password"
                placeholder="Enter password"
              />
            </div>
          </label>

          <button
            type="submit"
            disabled={loading}
            className="w-full h-11 rounded-xl bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 text-white font-bold transition-colors flex items-center justify-center gap-2"
          >
            <Lock size={16} />
            {loading ? "Signing in..." : "Sign In"}
          </button>

          {error && <p className="text-red-400 text-center text-sm font-medium">{error}</p>}
        </form>
      </div>
    </div>
  );
};
