"use client";

/**
 * **«الموقع المباشر اليوم: ٥ من ١٥»** (قرارُ المالك ٢٠٢٦-١١-١٨: «لازم يوصلنا موقع مباشر»)
 *
 * سطرٌ تحت عدّادات لوحة الطلبات: كم طلب صار بالطريق اليوم، وبكم وصل موقع مباشر (من التطبيق أو
 * من الواتساب)، وكم انبعت تذكير. **يقيس إن كانت التصليحات ترفع الرقم** — والأرقام من المحرّك
 * (`GET /admin/orders/live-stats`)، وتتحدّث مع كلّ تحديثٍ للّوحة.
 */
import { useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { IconLocation } from "@rahalgo/ui";
import { api } from "@/lib/api";

const L = getMessages(defaultLocale).admin.ordersPage.liveStats;

interface Stats {
  on_the_way: number;
  with_live: number;
  from_app: number;
  from_whatsapp: number;
  nudged_push: number;
  nudged_wa: number;
}

const fill = (s: string, v: Record<string, string>) => s.replace(/\{(\w+)\}/g, (_, k: string) => v[k] ?? "");

export function LiveShareStats({ tick }: { tick: number }) {
  const [st, setSt] = useState<Stats | null>(null);
  useEffect(() => {
    let alive = true;
    api<Stats>("/api/v1/admin/orders/live-stats")
      .then((r) => alive && setSt(r))
      .catch(() => {});
    return () => {
      alive = false;
    };
  }, [tick]);
  if (!st || st.on_the_way === 0) return null;
  const pct = Math.round((st.with_live / st.on_the_way) * 100);
  const tone = pct >= 80 ? "text-success" : pct >= 50 ? "text-warning" : "text-danger";
  return (
    <p className="mb-3 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-ink-muted">
      <span className={`inline-flex items-center gap-1.5 font-bold ${tone}`}>
        <IconLocation size={16} />
        {fill(L.line, { n: fmtNum(st.with_live), total: fmtNum(st.on_the_way), pct: fmtNum(pct) })}
      </span>
      <span>{fill(L.sources, { app: fmtNum(st.from_app), wa: fmtNum(st.from_whatsapp) })}</span>
      <span>{fill(L.nudges, { push: fmtNum(st.nudged_push), wa: fmtNum(st.nudged_wa) })}</span>
    </p>
  );
}
