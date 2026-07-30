"use client";

import { createContext, useCallback, useContext, useEffect, useState } from "react";
import { authApi, tokenStore, type AuthUser } from "./api";

/** الأدوار المسموح لها بدخول اللوحة */
const PANEL_ROLES = ["admin", "ops", "finance"];

interface AuthState {
  user: AuthUser | null;
  loading: boolean;
  login: (phone: string, password: string) => Promise<AuthUser>;
  logout: () => void;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!tokenStore.refresh) {
      setLoading(false);
      return;
    }
    authApi
      .me()
      .then(setUser)
      .catch(() => tokenStore.clear())
      .finally(() => setLoading(false));
  }, []);

  const login = useCallback(async (phone: string, password: string) => {
    const result = await authApi.loginPassword(phone, password);
    tokenStore.set(result.tokens);
    setUser(result.user);
    return result.user;
  }, []);

  const logout = useCallback(() => {
    authApi.logout();
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider value={{ user, loading, login, logout }}>{children}</AuthContext.Provider>
  );
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}

export function canAccessPanel(user: AuthUser | null): boolean {
  return !!user && user.roles.some((r) => PANEL_ROLES.includes(r));
}
