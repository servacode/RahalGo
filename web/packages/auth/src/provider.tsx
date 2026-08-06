"use client";

/**
 * موفّر الهوية المركزي — نسخة واحدة لكل التطبيقات (كان مكرّراً 4 مرات).
 * يوفّر معاً `login` و`setUser` كي تستعمله كل اللوحات بلا اختلاف عقد.
 */

import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from "react";
import { AuthTransition, type AuthTransitionKind } from "@rahalgo/ui";
import { authApi, tokenStore, type AuthUser } from "./client";

interface AuthState {
  user: AuthUser | null;
  loading: boolean;
  setUser: (u: AuthUser | null) => void;
  login: (phone: string, password: string) => Promise<AuthUser>;
  logout: () => void;
  /**
   * **يُعلن دخولاً ناجحاً فتُغطّى الشاشةُ حتّى تحلّ الوجهة.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «عند تسجيل الدخول لا يجوز أن يكون انتقالٌ
   *  بدون تأثير».)
   *
   * **والخروجُ لا يحتاج إعلاناً** — `logout` تُعلنه بنفسها، **فخمسةَ عشرَ
   * نداءً في خمسة تطبيقاتٍ ترثه بلا أن يُلمس واحدٌ منها.** والدخولُ يُعلَن
   * لأنّ نجاحَه يقع في `onSuccess` خارجَ الموفّر.
   */
  enter: (u: AuthUser) => void;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [loading, setLoading] = useState(true);
  /**
   * **الطبقةُ الانتقاليّة — وتنطفئ وحدَها.**
   *
   * **الانتقالُ بين بوّابتين تحميلُ صفحةٍ كاملة**، فتموت الطبقةُ مع الصفحة
   * القديمة ولا تحتاج إطفاءً. **وداخلَ التطبيق الواحد يبقى الموفّرُ حيّاً**
   * — ولو تُركت لَغطّت الوجهةَ إلى الأبد.
   *
   * **ومهلةٌ تكفي لتُقرأ ولا تُملّ**: تسعمئةُ مِلّي ثانية.
   */
  const [transit, setTransit] = useState<{ kind: AuthTransitionKind; name: string } | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const flash = useCallback((kind: AuthTransitionKind, name = "") => {
    if (timer.current) clearTimeout(timer.current);
    setTransit({ kind, name });
    timer.current = setTimeout(() => setTransit(null), 900);
  }, []);

  // **ومؤقّتٌ يبقى بعد رحيل الشاشة يُحدّث ما لا وجودَ له.**
  useEffect(() => () => {
    if (timer.current) clearTimeout(timer.current);
  }, []);

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
    // **الطبقةُ تُرفع قبل المسح** — لو رُفعت بعده لَأُعيد رسمُ الصفحة زائراً
    // **فتُرى شاشةُ الضيف لحظةً قبل الغطاء**، وهي الوميضةُ التي تُقرأ عطباً.
    flash("out");
    authApi.logout();
    setUser(null);
  }, [flash]);

  const enter = useCallback(
    (u: AuthUser) => {
      setUser(u);
      flash("in", u.full_name || "");
    },
    [flash],
  );

  return (
    <AuthContext.Provider value={{ user, loading, setUser, login, logout, enter }}>
      {children}
      {/* **والعلامةُ تُقرأ من `PlatformProvider` داخلَ الطبقة** — فلا تُمرَّر
          عبر أربعِ طبقاتٍ من الوسائط. */}
      {transit && <AuthTransition kind={transit.kind} name={transit.name} />}
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
