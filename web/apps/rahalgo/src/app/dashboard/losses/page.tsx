"use client";

/**
 * **الخسائرُ والنزاعاتُ وتعويضُ البضاعة صارت في «المشاكل والتعويضات»** (قرارُ المالك
 * ٢٠٢٦-١٠-٢٥). **والرابطُ القديمُ يفتح تبويبَ الخسائر هناك** — روابطُ الموافقات والأحداث
 * والإشعارات تبقى تعمل.
 */

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { LoadingState } from "@rahalgo/ui";

export default function LossesRedirect() {
  const router = useRouter();
  useEffect(() => {
    const t = new URLSearchParams(window.location.search).get("tab");
    router.replace(`/dashboard/tickets?tab=${t === "disputes" || t === "goods" ? t : "losses"}`);
  }, [router]);
  return <LoadingState variant="text" />;
}
