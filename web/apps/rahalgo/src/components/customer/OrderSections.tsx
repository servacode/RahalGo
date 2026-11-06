"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«طلبك من {n} محلات» — أقسامُ الطلب المشترك** (قرارُ المالك ٢٠٢٦-١١-٠٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * طلبٌ واحدٌ بحالةٍ واحدة وكودٍ واحد ومجموعٍ واحد، **وداخلَه قسمٌ لكلّ محلّ: «القسم ١ · القسم ٢».**
 * **بلا اسمِ محلٍّ ولا شعاره** (قاعدة الخصوصيّة P-1: الزبونُ لا يرى من أين يُجلب طلبُه).
 * وقسمٌ «ما توفّر» تظهر أصنافُه مشطوبةً رماديّة — والمجموعُ نزل من الخادم.
 *
 * **الحقلُ غائبٌ في طلب المحلّ الواحد ⇒ لا يُرسم شيء.**
 */

import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { Badge } from "@rahalgo/ui";
import { fill } from "@/lib/customer/texts";
import type { OrderSection } from "@/lib/customer/types";

const S = getMessages(defaultLocale).shop.orders.sections;

const TONE = {
  pending: "warning",
  accepted: "accent",
  picked_up: "success",
  unavailable: "neutral",
} as const;

export function OrderSections({ sections }: { sections?: OrderSection[] | null }) {
  if (!sections || sections.length === 0) return null;
  const rows = [...sections].sort((a, b) => a.seq - b.seq);
  return (
    <section className="surface flex flex-col gap-3 p-4" data-testid="order-sections">
      <h2 className="heading-card">{fill(S.title, { n: fmtNum(rows.length) })}</h2>
      <ul className="flex flex-col gap-3">
        {rows.map((s) => {
          const off = s.status === "unavailable";
          return (
            <li key={s.seq} className="flex flex-col gap-1.5 rounded-control border border-line-soft bg-ink-faint p-3">
              <div className="flex items-center justify-between gap-2">
                <span className="text-sm font-bold">{fill(S.section, { seq: fmtNum(s.seq) })}</span>
                <Badge variant={TONE[s.status] ?? "neutral"}>{S.status[s.status] ?? s.status}</Badge>
              </div>
              <ul className={`flex flex-col gap-0.5 text-sm ${off ? "text-ink-muted line-through" : ""}`}>
                {(s.items ?? []).map((it, i) => (
                  <li key={`${it.name}-${i}`}>
                    {it.name}
                    {it.qty > 1 && ` × ${fmtNum(it.qty)}`}
                  </li>
                ))}
              </ul>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
