import { Suspense } from "react";
import CustomerAuth from "@/components/customer/CustomerAuth";

/**
 * **‏/login — بابُ الزبون** (قرارُ المالك ٢٠٢٦-١٠-١٧).
 *
 * **وبابُ الإدارة غيرُه** (`/adminrahalgo`) — ولا يتبدّل.
 */
export default function Page() {
  return (
    <Suspense>
      <CustomerAuth mode="login" />
    </Suspense>
  );
}
