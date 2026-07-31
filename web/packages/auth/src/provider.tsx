"use client";

/**
 * موفّر الهوية المركزي — نسخة واحدة لكل التطبيقات (كان مكرّراً 4 مرات).
 * يوفّر معاً `login` و`setUser` كي تستعمله كل اللوحات بلا اختلاف عقد.
 */

import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { authApi, tokenStore, type AuthUser } from "./client";

interface AuthState {
  user: AuthUser | null;
  loading: boolean;
  setUser: (u: AuthUser | null) => void;
  login: (phone: string, password: string) => Promise<AuthUser>;
  logout: () => void;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
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
    <AuthContext.Provider value={{ user, loading, setUser, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}

// ---------- الأدوار: مصدر واحد ----------

/** أدوار المنصة السبعة — مطابقة لـ identity.AllRoles في الخادم. */
export const ROLES = ["customer", "driver", "merchant", "sales", "ops", "finance", "admin"] as const;
export type Role = (typeof ROLES)[number];

/** أدوار لوحة الإدارة (موظفو المنصة الداخليون). */
export const PANEL_ROLES: Role[] = ["admin", "ops", "finance"];

/** hasRole المُتحقِّق الوحيد من الأدوار — بدل canAccessPanel/canAccessPortal/isRep. */
export function hasRole(user: AuthUser | null, ...roles: Role[]): boolean {
  return !!user && roles.some((r) => user.roles.includes(r));
}

export function isLoggedIn(user: AuthUser | null): boolean {
  return !!user;
}
