import { Suspense } from "react";
import CustomerAuth from "@/components/customer/CustomerAuth";

/**
 * **‏/signup — حسابُ زبونٍ جديد** (قرارُ المالك ٢٠٢٦-١٠-١٧).
 *
 * **ورمزُ الدعوة من الرابط** (`?ref=`) — لا من يده.
 */
export default function Page() {
  return (
    <Suspense>
      <CustomerAuth mode="signup" />
    </Suspense>
  );
}
