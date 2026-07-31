"use client";

/** إعادة تصدير من حزمة المصادقة المركزية + حارس هذه اللوحة فقط. */

import { hasRole, PANEL_ROLES, type AuthUser } from "@rahalgo/auth";

export { AuthProvider, useAuth, isLoggedIn, hasRole } from "@rahalgo/auth";

export function canAccessPortal(user: AuthUser | null): boolean {
  return hasRole(user, "merchant");
}
