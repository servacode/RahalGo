"use client";

/**
 * **«هذا القسم يحتاج حساباً»** — للزائر في السلّة والطلبات والحساب.
 *
 * **والتصفّحُ مفتوحٌ له** (مرآةُ وضع الزائر في أندرويد) — **ويُعاد بعد
 * الدخول إلى حيث كان** (`next`).
 */

import { Suspense } from "react";
import { usePathname } from "next/navigation";
import CustomerAuth from "./CustomerAuth";

/**
 * **فورمُ الدخول نفسُه لا زرٌّ إليه** (قرارُ المالك ٢٠٢٦-١٠-١٩: «لازم فورا يفتح فورم تسجيل
 * الدخول وتحت انشاء حساب جديد») — خطوةٌ أقلّ، **ويعود بعد الدخول إلى حيث كان.**
 */
export function NeedLogin() {
  const path = usePathname() ?? "/shop";
  return (
    <Suspense>
      <CustomerAuth mode="login" next={path} />
    </Suspense>
  );
}
