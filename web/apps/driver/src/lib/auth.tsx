"use client";

/** إعادة تصدير من حزمة المصادقة المركزية + حارس هذه البوابة فقط. */

import { hasRole, type AuthUser } from "@rahalgo/auth";

export { AuthProvider, useAuth, isLoggedIn, hasRole } from "@rahalgo/auth";

export function isDriver(user: AuthUser | null): boolean {
  return hasRole(user, "driver");
}
