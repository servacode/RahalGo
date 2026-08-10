"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حرّاسُ الأدوار — في ملفٍّ واحدٍ لأنّ اللوحاتِ صارت بيتاً واحداً**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٠: «بابٌ واحدٌ للجميع».)
 *
 * **كانت خمسةَ ملفّاتٍ في خمسة تطبيقات**، كلٌّ يعرّف حارسَ لوحته وحدَها
 * (`canAccessPanel` · `canAccessPortal` · `isRep` · `isDriver`).
 * **وخمسةُ ملفّاتٍ تُقرأ خمسَ قراءاتٍ ولا تُقارَن** — فمن بدّل قاعدةً في
 * واحدٍ لم يعلم أنّ لها أخواتٍ أربعاً.
 *
 * **والمنطقُ كلُّه في `hasRole`** — وهذه أسماءٌ تقول من يدخل أين.
 */

import { hasRole, PANEL_ROLES, type AuthUser } from "@rahalgo/auth";

export { AuthProvider, useAuth, isLoggedIn, hasRole } from "@rahalgo/auth";

/** لوحةُ الإدارة — الأدمنُ والعملياتُ والمالية. */
export function canAccessPanel(user: AuthUser | null): boolean {
  return hasRole(user, ...PANEL_ROLES);
}

/** لوحةُ المتجر — لصاحبه وحدَه. */
export function canAccessPortal(user: AuthUser | null): boolean {
  return hasRole(user, "merchant");
}

/** لوحةُ المندوب. */
export function isRep(user: AuthUser | null): boolean {
  return hasRole(user, "sales");
}

/** لوحةُ السائق. */
export function isDriver(user: AuthUser | null): boolean {
  return hasRole(user, "driver");
}
