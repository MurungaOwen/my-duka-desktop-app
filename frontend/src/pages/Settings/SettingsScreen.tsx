import React, { useEffect, useState } from "react";
import { CreateStaff, ListStaff } from "../../../wailsjs/go/main/App";
import type { Staff, StaffRole } from "../../types/api";

export const SettingsScreen: React.FC = () => {
  const [staff, setStaff] = useState<Staff[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [name, setName] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState<StaffRole>("cashier");

  const loadStaff = async () => {
    setLoading(true);
    try {
      const list: any = await ListStaff();
      setStaff((list || []) as Staff[]);
    } catch (err: any) {
      setError(err?.toString?.() || "Failed to load staff");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadStaff();
  }, []);

  const createUser = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    if (!name.trim() || !username.trim() || !password.trim()) {
      setError("Name, username, and password are required.");
      return;
    }

    try {
      await CreateStaff({
        name: name.trim(),
        username: username.trim().toLowerCase(),
        role,
        password,
      } as any);
      setName("");
      setUsername("");
      setPassword("");
      setRole("cashier");
      await loadStaff();
    } catch (err: any) {
      setError(err?.toString?.() || "Failed to create user");
    }
  };

  return (
    <div className="p-8 max-w-6xl mx-auto w-full h-full overflow-y-auto">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-slate-800">Settings</h1>
        <p className="text-slate-500 mt-1">Manage staff accounts and access.</p>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
        <section className="bg-white rounded-2xl border border-slate-200 shadow-sm p-6">
          <h2 className="text-xl font-bold text-slate-800 mb-4">Create User</h2>
          <form onSubmit={createUser} className="space-y-4">
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Full name"
              className="w-full px-3 py-2.5 rounded-lg border border-slate-300 outline-none focus:border-blue-500"
            />
            <input
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="Username"
              className="w-full px-3 py-2.5 rounded-lg border border-slate-300 outline-none focus:border-blue-500"
            />
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Password"
              className="w-full px-3 py-2.5 rounded-lg border border-slate-300 outline-none focus:border-blue-500"
            />
            <select
              value={role}
              onChange={(e) => setRole(e.target.value as StaffRole)}
              className="w-full px-3 py-2.5 rounded-lg border border-slate-300 outline-none focus:border-blue-500"
            >
              <option value="cashier">Cashier</option>
              <option value="admin">Admin</option>
            </select>
            <button className="w-full bg-blue-600 hover:bg-blue-700 text-white py-2.5 rounded-lg font-semibold">
              Create User
            </button>
            {error && <p className="text-sm text-red-600">{error}</p>}
          </form>
        </section>

        <section className="bg-white rounded-2xl border border-slate-200 shadow-sm p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-bold text-slate-800">Staff Users</h2>
            <button onClick={loadStaff} className="text-sm px-3 py-1.5 rounded-md border border-slate-300 hover:bg-slate-50">Refresh</button>
          </div>
          {loading ? (
            <p className="text-slate-500 text-sm">Loading users...</p>
          ) : (
            <div className="space-y-3">
              {staff.map((s) => (
                <div key={s.id} className="flex items-center justify-between border border-slate-200 rounded-lg p-3">
                  <div>
                    <p className="font-semibold text-slate-800">{s.name}</p>
                    <p className="text-xs text-slate-500">@{s.username} • {s.role}</p>
                  </div>
                  <span className={`text-xs px-2 py-1 rounded-full ${s.isActive ? "bg-emerald-100 text-emerald-700" : "bg-slate-100 text-slate-600"}`}>
                    {s.isActive ? "active" : "inactive"}
                  </span>
                </div>
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  );
};
