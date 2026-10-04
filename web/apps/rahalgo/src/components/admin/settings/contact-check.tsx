"use client";

/**
 * **تنبيهُ الرئيسيّة: بياناتُ التواصل ما زالت تجريبيّة**
 * (قرارُ المالك ٢٠٢٦-١٠-٠٤، الإعدادات البند ١٥).
 *
 * رقمُ الدعم والعنوانُ وموقعُ المكتب يضبطها المالكُ من اللوحة قبل الإطلاق —
 * **وموقعُ المكتب وجهةُ السائق حين يُرجع البضاعة**، فقيمةٌ تجريبيّةٌ تُرسله إلى
 * مكانٍ خطأ. **فيبقى التنبيهُ ظاهراً ما دامت فارغةً أو تجريبيّة.**
 *
 * **ولمن يقرأ الإعدادات وحدَه** — ومن لا يملكها لا يُسأل عن شيءٍ لا يضبطه.
 */

import { useEffect, useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Alert } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const U = m.admin.settings.ui;

/** **رقمٌ تجريبيّ**: فارغٌ أو ذيلُه أصفارٌ متتالية (مثل ‎+963933000000‎). */
function testPhone(v: string): boolean {
  const d = v.replace(/\D/g, "");
  return d === "" || /0{6,}$/.test(d);
}

/** **عنوانٌ تجريبيّ**: فارغٌ أو يقول عن نفسه «تجريبي». */
function testAddress(v: string): boolean {
  return v.trim() === "" || v.includes(U.testMarker);
}

export function looksLikeTestContact(vals: Record<string, unknown>): boolean {
  const s = (k: string) => (typeof vals[k] === "string" ? (vals[k] as string) : "");
  return testPhone(s("platform.support_phone")) || testAddress(s("platform.address")) ||
    s("platform.location").trim() === "";
}

export default function ContactTestAlert() {
  const { can } = useAuth();
  const [show, setShow] = useState(false);
  const allowed = can("settings.read");

  useEffect(() => {
    if (!allowed) return;
    api<{ settings?: { key: string; value: unknown }[] }>("/api/v1/admin/settings")
      .then((r) => {
        const vals: Record<string, unknown> = {};
        for (const it of r.settings ?? []) vals[it.key] = it.value;
        setShow(looksLikeTestContact(vals));
      })
      .catch(() => setShow(false));
  }, [allowed]);

  if (!show) return null;
  return (
    <Alert tone="warning">
      {U.contactTestAlert}{" "}
      <Link href="/dashboard/settings?topic=site" className="underline">
        {U.contactTestLink}
      </Link>
    </Alert>
  );
}
