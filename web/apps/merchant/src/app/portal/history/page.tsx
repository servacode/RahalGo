"use client";

/**
 * **سجلُّ طلبات المتجر — ما انتهى منها.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٠: «أضِف سجلَّ الطلبات بالحالتين — قسمٌ خاصٌّ بكلّ
 *  الطلبات من هذا المتجر… ليعرف المتجرُ ماذا سلّم وماذا أُلغي منه».)
 *
 * # ولماذا في الوضعين
 *
 * **«ماذا بعتُ وماذا ضاع منّي؟» سؤالٌ لا علاقةَ له بمن يضغط الأزرار.**
 * والجاريةُ وحدَها هي المحجوبةُ في وضع «المنصّة تدير» — لأنّه لا يملك فيها
 * زرّاً، **وشاشةٌ تُشاهَد ولا تُلمَس تُربك أكثرَ ممّا تفيد.**
 *
 * # وما انتهى لا ما يجري
 *
 * **السجلُّ تاريخٌ**: مُسلَّمٌ ومُلغًى ومرفوضٌ ومتعذّرٌ ومسترجَع. **وطلبٌ في
 * الطريق ليس تاريخاً بعد** — موضعُه «الطلبات» حيث يُعمل عليه.
 *
 * # والترشيحُ بالحال
 *
 * **من يبحث عن سببِ إلغاءٍ لا يقلّب عشرين مُسلَّماً ليجد واحداً مُلغًى.**
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDateTime } from "@rahalgo/i18n";
import {
  Badge,
  Select,
  Pagination,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  useLiveRefresh,
  IconCheck,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store";

const m = getMessages(defaultLocale);
const STATUS_LABELS: Record<string, string> = m.orders.status;

/** **نبرةُ النهاية** — وصلت أم لم تقع. */
const TONE: Record<string, "success" | "danger" | "neutral"> = {
  delivered: "success",
  cancelled: "danger",
  rejected: "danger",
  failed: "danger",
  refunded: "neutral",
};

interface Row {
  id: string;
  number: number;
  status: string;
  items_count: number;
  items_preview: string;
  subtotal: number;
  total: number;
  cancel_reason?: string;
  created_at: string;
  closed_at?: string | null;
}

/** **الحالاتُ المنتهيةُ وحدَها** — والجاريةُ ليست تاريخاً. */
const CLOSED = ["delivered", "cancelled", "rejected", "failed", "refunded"] as const;

export default function MerchantHistoryPage() {
  const { store } = useStore();
  const [rows, setRows] = useState<Row[] | null>(null);
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(1);
  const [count, setCount] = useState(0);
  const [perPage, setPerPage] = useState(20);

  const load = useCallback(() => {
    if (!store) return;
    // **و`closed_only` صريحةٌ من هنا** — فيُجاب في الوضعين، **ولا يتوقّف
    // السجلُّ على من يدير الطلبات.**
    const q = new URLSearchParams({ closed_only: "true", page: String(page) });
    if (status) q.set("status", status);
    api<{ orders: Row[]; total: number; per_page: number }>(
      `/api/v1/merchant/stores/${store.id}/orders?${q}`,
    )
      .then((r) => {
        setRows(r?.orders ?? []);
        setCount(r?.total ?? 0);
        setPerPage(r?.per_page || 20);
      })
      // @empty-ok **وسجلٌّ لا يُجلب يُقرأ فارغاً** — لا إنذارَ فوق شاشةِ تاريخ.
      .catch(() => setRows([]));
  }, [store, status, page]);

  useEffect(load, [load]);
  useLiveRefresh(["order"], load);

  if (!store || rows === null) return <LoadingState />;

  return (
    <PageContainer>
      <PageHeader
        icon={IconCheck}
        title={m.merchant.historyTitle}
        subtitle={m.merchant.historySubtitle}
        actions={
          <Select
            id="h-status"
            value={status}
            onChange={(e) => {
              /* **وتبديلُ الترشيح يعود إلى الأولى** — **ومن كان في الرابعة
                 يقع على رابعةٍ قد لا توجد**، فيرى فراغاً ويظنّ القسمَ خالياً. */
              setPage(1);
              setStatus(e.target.value);
            }}
          >
            <option value="">{m.merchant.historyAll}</option>
            {CLOSED.map((s) => (
              <option key={s} value={s}>
                {STATUS_LABELS[s] ?? s}
              </option>
            ))}
          </Select>
        }
      />

      {rows.length === 0 ? (
        <EmptyState icon={IconCheck} title={m.merchant.historyEmpty} />
      ) : (
        <ul className="space-y-2">
          {rows.map((o) => (
            <li key={o.id} className="surface p-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="flex items-center gap-2">
                  <span className="font-bold">#{fmtRef(o.number)}</span>
                  <Badge variant={TONE[o.status] ?? "neutral"}>
                    {STATUS_LABELS[o.status] ?? o.status}
                  </Badge>
                </span>
                <span className="flex items-center gap-3">
                  {/* **وما قبضه هو `subtotal` لا `total`** — الإجماليُّ فيه
                      أجرةُ التوصيل وهي للسائق. **ورقمٌ يقرؤه صاحبُ المتجر
                      دخلاً وهو ليس دخلَه يُبنى عليه حسابٌ خاطئ.** */}
                  <span className="figure text-sm" dir="ltr">
                    {fmtNum(o.subtotal)}{" "}
                    <span className="text-2xs font-normal text-ink-muted">
                      {m.common.currency}
                    </span>
                  </span>
                  <span className="text-2xs text-ink-muted" dir="ltr">
                    {fmtDateTime(o.closed_at || o.created_at)}
                  </span>
                </span>
              </div>
              {o.items_preview && (
                <p className="mt-1 truncate text-xs text-ink-muted">
                  {o.items_preview}
                  {o.items_count > 0 && ` · ${fmtNum(o.items_count)}`}
                </p>
              )}
              {/* **وسببُ ما لم يصل يُقرأ هنا** — **ومن رأى «مُلغًى» بلا كلمةٍ
                  لا يعرف أخطأ هو أم الزبون**، فلا يُصلح شيئاً. */}
              {o.cancel_reason && (
                <p className="mt-1 text-xs text-danger">{o.cancel_reason}</p>
              )}
            </li>
          ))}
        </ul>
      )}

      {count > perPage && (
        <div className="mt-4 flex justify-center">
          <Pagination page={page} total={count} perPage={perPage} onChange={setPage} />
        </div>
      )}
    </PageContainer>
  );
}
