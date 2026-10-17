import { Suspense } from "react";
import CustomerAuth from "@/components/customer/CustomerAuth";

/**
 * **‏/forgot — استعادةُ كلمة المرور عبر واتساب** (قرارُ المالك ٢٠٢٦-١٠-١٧).
 */
export default function Page() {
  return (
    <Suspense>
      <CustomerAuth mode="forgot" />
    </Suspense>
  );
}
