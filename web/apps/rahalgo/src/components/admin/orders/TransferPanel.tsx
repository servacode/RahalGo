"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تبديلُ متجر الطلب — مرشّحون مرتّبون ثمّ مقابلُ كلِّ صنف**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠٣: «المفروض ما يكون نفس الاسم بالضبط، لأنّه ممكن
 *  يكون نفسه بتسميةٍ مختلفة… لازم نلاقي حلّ يكون نفس المتجر يقدّم نفس الأصناف».)
 *
 * **كانت الشاشةُ قائمةَ أسماءِ متاجر** — يختار الموظّفُ واحداً ويضغط،
 * **ثمّ يُقال له «لا يطابق»** لأنّ «رز مصري» كُتب هناك «أرز مصري». فيجرّب غيرَه أعمى.
 *
 * **والآن خطوتان:**
 *
 *	١ · المتاجرُ **مرتّبةً بما تقدّمه من أصناف الطلب** («٣ من ٣») ثمّ بقربها —
 *	    من المحرّك (`GET /admin/orders/{id}/transfer-candidates`).
 *	٢ · **لكلّ صنفٍ مقابلُه المقترح محدَّداً سلفاً**، وقائمةُ المتجر كاملةً
 *	    ليبدّله الموظّفُ بيده — **فالاقتراحُ لا يقرّر.**
 *
 * **ولا تحسب الشاشةُ شيئاً**: الترتيبُ والمطابقةُ والتحقّقُ في المحرّك،
 * **وسعرُ الزبون لا يُمسّ** — يُرسَل المقابلُ ويُردّ ما قاله الخادم.
 */

import { useEffect, useState } from "react";
import {
  getMessages,
  defaultLocale,
  fmtNum,
  fmtMoney,
  errorText,
} from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Input,
  Select,
  Badge,
  FormActions,
  LoadingState,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const t = m.admin.ordersPage;

type MenuItem = { menu_item_id: string; name: string; merchant_price: number };

type Candidate = {
  merchant_id: string;
  name: string;
  matched: number;
  total: number;
  distance_m: number | null;
  items: { order_item_id: string; match: MenuItem | null; score: number }[];
  menu?: MenuItem[];
};

type Candidates = {
  distance_from: "driver" | "store" | "";
  items: { order_item_id: string; name: string; qty: number; merchant_price: number }[];
  stores: Candidate[];
};

/** **المسافةُ بالمتر تحت الكيلو، وبالكيلو بعده** — ومجهولُها لا يُكتب. */
function distanceText(d: number | null): string {
  if (d == null) return "";
  if (d < 1000) return t.transferMeters.replace("{n}", fmtNum(Math.round(d)));
  return t.transferKm.replace("{n}", fmtNum(Math.round(d / 100) / 10));
}

export function TransferPanel({
  orderId,
  onDone,
  onClose,
}: {
  orderId: string;
  /** **نجح التحويل** — تُحدَّث القائمة. */
  onDone: () => void;
  onClose: () => void;
}) {
  const [data, setData] = useState<Candidates | null>(null);
  const [store, setStore] = useState<Candidate | null>(null);
  /** `order_item_id → menu_item_id` — **و`""` بلا مقابل.** */
  const [picks, setPicks] = useState<Record<string, string>>({});
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [unmatched, setUnmatched] = useState<string[]>([]);

  useEffect(() => {
    let live = true;
    api<Candidates>(`/api/v1/admin/orders/${orderId}/transfer-candidates`)
      .then((r) => {
        if (live) setData(r);
      })
      .catch((e) => {
        if (live) setErr(errorText(e));
      });
    return () => {
      live = false;
    };
  }, [orderId]);

  /** **متجرٌ بعينه يأتي بقائمته كاملةً** — للاختيار باليد. */
  async function choose(id: string) {
    setBusy(true);
    setErr("");
    setUnmatched([]);
    try {
      const r = await api<Candidates>(
        `/api/v1/admin/orders/${orderId}/transfer-candidates?merchant_id=${id}`,
      );
      const s = r.stores[0];
      // **ومتجرٌ لم يعد في القائمة يُقال** (المشكلة ٣٤) — كانت الضغطةُ لا تفعل شيئاً.
      if (!s) {
        setErr(t.board.transferStoreGone);
        return;
      }
      const p: Record<string, string> = {};
      for (const it of s.items) p[it.order_item_id] = it.match?.menu_item_id ?? "";
      setPicks(p);
      setStore(s);
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  async function submit() {
    if (!store || !data) return;
    // **والسببُ الناقصُ يُقال لا يُعطَّل الزرُّ صامتاً** (المشكلة ٣٤).
    if (!reason.trim()) {
      setErr(t.board.transferReasonNeeded);
      return;
    }
    const missing = data.items.filter((it) => !picks[it.order_item_id]);
    if (missing.length > 0) {
      setErr(t.transferMapMissing);
      return;
    }
    setBusy(true);
    setErr("");
    setUnmatched([]);
    try {
      await api(`/api/v1/admin/orders/${orderId}/transfer`, {
        method: "POST",
        body: JSON.stringify({
          merchant_id: store.merchant_id,
          note: reason.trim(),
          items: data.items.map((it) => ({
            order_item_id: it.order_item_id,
            menu_item_id: picks[it.order_item_id],
          })),
        }),
      });
      onDone();
    } catch (e) {
      const items = (e instanceof ApiError ? e.body.details?.items : undefined) as
        | string[]
        | undefined;
      if (items?.length) setUnmatched(items);
      else setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  // ── الخطوةُ الثانية: مقابلُ كلِّ صنف ──────────────────────────────
  if (store && data) {
    const menu = store.menu ?? [];
    return (
      <div className="w-full space-y-2" onClick={(e) => e.stopPropagation()}>
        <p className="text-xs font-medium text-primary-dark">
          {t.transferMapTitle.replace("{store}", store.name)}
        </p>
        <p className="text-xs text-ink-muted">{t.transferMapHint}</p>
        {data.items.map((it) => (
          <div key={it.order_item_id} className="space-y-1">
            <p className="text-xs">
              {it.name}
              {it.qty > 1 && ` × ${fmtNum(it.qty)}`}
            </p>
            <Select
              id={`tm-${it.order_item_id}`}
              value={picks[it.order_item_id] ?? ""}
              onChange={(e) =>
                setPicks((p) => ({ ...p, [it.order_item_id]: e.target.value }))
              }
            >
              <option value="">{t.transferNoMatch}</option>
              {menu.map((x) => (
                <option key={x.menu_item_id} value={x.menu_item_id}>
                  {x.name} — {t.transferBuyPrice.replace("{price}", fmtMoney(x.merchant_price))}
                </option>
              ))}
            </Select>
          </div>
        ))}
        <Input
          id={`tr-${orderId}`}
          value={reason}
          onChange={(e) => {
            setReason(e.target.value);
            setErr("");
          }}
          placeholder={t.transferReason}
        />
        {!reason.trim() && (
          <p className="text-xs text-ink-muted">{t.board.transferReasonNeeded}</p>
        )}
        {/* **وصنفٌ بلا مقابلٍ يُسمّى** — كما ردّه الخادم. */}
        {unmatched.length > 0 && (
          <Alert>
            {t.transferUnmatched} {unmatched.join(m.common.listSep)}
          </Alert>
        )}
        {err && <p className="text-xs text-danger">{err}</p>}
        <FormActions
          busy={busy}
          onSave={() => void submit()}
          onCancel={() => {
            setStore(null);
            setErr("");
            setUnmatched([]);
          }}
          saveLabel={t.transferConfirm}
          cancelLabel={m.common.back}
        />
      </div>
    );
  }

  // ── الخطوةُ الأولى: المتاجرُ المرشّحة ─────────────────────────────
  return (
    <div className="w-full space-y-2" onClick={(e) => e.stopPropagation()}>
      <p className="text-xs font-medium text-primary-dark">{t.transferTitle}</p>
      <p className="text-xs text-ink-muted">{t.transferHint}</p>
      {data?.distance_from === "driver" && (
        <p className="text-xs text-ink-muted">{t.transferFromDriver}</p>
      )}
      {data?.distance_from === "store" && (
        <p className="text-xs text-ink-muted">{t.transferFromStore}</p>
      )}
      {!data && !err && <LoadingState variant="inline" />}
      {data && data.stores.length === 0 && (
        <p className="text-xs text-ink-muted">{t.transferNoStores}</p>
      )}
      {data && data.stores.length > 0 && (
        <div className="flex max-h-72 flex-col gap-1.5 overflow-y-auto">
          {data.stores.map((s) => (
            <Button
              key={s.merchant_id}
              variant="secondary"
              disabled={busy}
              onClick={() => void choose(s.merchant_id)}
            >
              <span className="flex w-full items-center justify-between gap-2">
                <span className="truncate">{s.name}</span>
                <span className="flex shrink-0 items-center gap-1.5">
                  {s.distance_m != null && (
                    <span className="text-xs text-ink-muted">
                      {distanceText(s.distance_m)}
                    </span>
                  )}
                  <Badge
                    variant={
                      s.matched === s.total
                        ? "success"
                        : s.matched > 0
                          ? "warning"
                          : "neutral"
                    }
                  >
                    {t.transferCoverage
                      .replace("{n}", fmtNum(s.matched))
                      .replace("{total}", fmtNum(s.total))}
                  </Badge>
                </span>
              </span>
            </Button>
          ))}
        </div>
      )}
      {err && <p className="text-xs text-danger">{err}</p>}
      <div className="flex justify-end">
        <Button variant="secondary" onClick={onClose}>
          {m.common.cancel}
        </Button>
      </div>
    </div>
  );
}
