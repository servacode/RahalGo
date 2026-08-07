"use client";

/**
 * سجلّ الأحداث.
 *
 * كان `audit_log` يُكتب ولا يُقرأ إلا في نشاط مستخدمٍ بعينه. **وسجلٌّ لا يُقرأ
 * ليس سجلاً** — هو تكلفةُ كتابةٍ بلا فائدةِ قراءة.
 *
 * والصفحة محجوبة عن العمليات: تحوي مبالغ التعويضات والسحوبات وأرصدة المحافظ،
 * وموظّف العمليات ليس طرفاً في المال.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtDateTime, fmtNum } from "@rahalgo/i18n";
import {
  Alert,
  PageHeader, TabCards, EmptyState, Badge,
  IconStatus, IconUser,
  LoadingState,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const A = m.admin.audit;

interface Entry {
  id: number;
  actor_id: string | null;
  actor_name: string | null;
  action: string;
  entity: string;
  entity_id: string | null;
  details: Record<string, unknown> | null;
  ip: string | null;
  created_at: string;
}

const FILTERS = ["", "finance", "ops", "admin", "auth", "menu"] as const;

const actionText = (a: string) =>
  (A.actions as Record<string, string>)[a] ?? a;

/** الأفعال المالية تُبرَز: هي ما يُبحث عنه حين يُبحث في هذا السجلّ. */
const toneOf = (action: string): "danger" | "warning" | "neutral" =>
  action.startsWith("finance.") ? "danger"
    : action.startsWith("ops.") || action.startsWith("admin.") ? "warning"
      : "neutral";

/**
 * تفاصيل الحدث بلغةٍ لا بـJSON.
 *
 * وعرضُ الكائن خاماً كان سيُعيد الخطأ الذي أصلحناه في صفحة الإعدادات: من يقرأ
 * سجلاً يريد أن يعرف «كم» و«لمن»، لا أن يفكّ ترميزاً.
 */
function DetailLine({ e }: { e: Entry }) {
  const d = e.details;
  if (!d) return null;
  const bits: string[] = [];
  if (typeof d.amount === "number") bits.push(`${fmtNum(d.amount)} ${m.common.currency}`);
  if (typeof d.compensation === "number" && d.compensation > 0)
    bits.push(`${fmtNum(d.compensation)} ${m.common.currency}`);
  if (typeof d.status === "string") bits.push(d.status);
  if (typeof d.to === "string")
    bits.push(m.orders.status[d.to as keyof typeof m.orders.status] ?? d.to);
  if (typeof d.kind === "string") bits.push(d.kind);
  if (typeof d.note === "string" && d.note) bits.push(d.note);
  if (typeof d.resolution === "string" && d.resolution) bits.push(d.resolution);
  // تغيير الإعداد: الفرق لا النتيجة — «صار ٧٠» بلا «كان ٥٠» يُثبت الفعل ولا يُظهر أثره
  if (d.before !== undefined || d.after !== undefined)
    bits.push(`${JSON.stringify(d.before ?? null)} ← ${JSON.stringify(d.after ?? null)}`);

  if (bits.length === 0) return null;
  return (
    <p className="mt-0.5 truncate text-xs text-ink-muted">{bits.join(m.common.listSeparator)}</p>
  );
}

export default function AuditPage() {
  const [prefix, setPrefix] = useState<string>("");
  const [list, setList] = useState<Entry[] | null>(null);

  const [failed, setFailed] = useState(false);
  /**
   * **وسجلٌّ فارغٌ على خطأٍ شهادةُ زور.**
   *
   * كان `.catch(() => setList([]))` — **فيقرأ المدقّقُ «لا أحداث»** حين تفشل
   * القراءة، **ويستنتج أنّ شيئاً لم يقع.** وهذه شاشةٌ تُفتح للتحقّق من
   * واقعةٍ بعينها: **من بحث عن تعويضٍ صُرف ولم يجده يظنّ أنّه لم يُصرف.**
   *
   * **والسجلُّ الذي لا يُقرأ خيرٌ من سجلٍّ يكذب.**
   */
  const load = useCallback(() => {
    setFailed(false);
    api<Entry[]>(`/api/v1/admin/audit?limit=200${prefix ? `&prefix=${prefix}` : ""}`)
      .then(setList)
      .catch(() => setFailed(true));
  }, [prefix]);

  useEffect(load, [load]);

  return (
    <div>
      <PageHeader icon={IconStatus} title={A.title} />
      {failed && (
        <Alert tone="warning" title={m.errors.offline} className="mb-3">
          {m.errors.offlineHint}
        </Alert>
      )}
      <p className="mb-4 text-sm text-ink-muted">{A.hint}</p>

      <TabCards
        items={FILTERS.map((f) => ({
          key: f,
          label: f ? ((A.filters as Record<string, string>)[f] ?? f) : A.all,
        }))}
        active={prefix}
        onChange={setPrefix}
      />

      {list === null ? (
        <LoadingState variant="text" />
      ) : list.length === 0 ? (
        <EmptyState icon={IconStatus} title={A.empty} />
      ) : (
        <ul className="mt-4 space-y-2">
          {list.map((e) => (
            <li
              key={e.id}
              className="flex items-start gap-3 surface p-3"
            >
              <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-control bg-field">
                <IconUser size={15} className="text-ink-muted" />
              </span>
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant={toneOf(e.action)}>{actionText(e.action)}</Badge>
                  <span className="truncate text-sm font-medium">
                    {e.actor_name ?? A.system}
                  </span>
                </div>
                <DetailLine e={e} />
              </div>
              <time className="shrink-0 text-xs text-ink-muted">
                {fmtDateTime(e.created_at)}
              </time>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
