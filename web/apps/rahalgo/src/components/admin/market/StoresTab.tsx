"use client";

/**
 * **تبويبُ المتاجر — صفحةٌ موحّدة.**
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠٤: «المتاجرُ تُدار في اللوحة ولا تُعرض على الزبون —
 * واسمُها «المتاجر» لا «المورّدون»».)
 *
 * **بطاقةٌ لكلّ متجر**: أصنافُه وما يتوفّر منها وما أُضيف حديثاً وما بلا صورة،
 * **وعلامةٌ إن لم يضبط دوامَه** — وكلُّ بطاقةٍ تفتح صفحتَه.
 *
 * **وزرُّ «متجر مميّز» على البطاقة نفسِها** (طلبُ المالك ٢٠٢٦-١٠-١٨) — ضغطةٌ
 * واحدةٌ تميّزه أو تشيل تمييزَه، بلا فتح نافذة التعديل.
 */

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import Link from "next/link";
import {
  getMessages,
  defaultLocale,
  fmtNum,
  fmtStoreNo,
  errorText,
} from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Input,
  Pagination,
  EmptyState,
  LoadingState,
  useLiveData,
  IconStore,
  IconStar,
  IconBalance,
  IconSwap,
  Select,
  Modal,
  Chips,
  FormActions,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import { useImagePicker } from "@/components/admin/ImageUpload";
import { StoreActions } from "@/components/admin/StoreActions";
import { MerchantModal, type Merchant } from "@/components/admin/MerchantModal";
import StateBadge from "./StateBadge";

const m = getMessages(defaultLocale);
const T = m.admin.market.stores;

interface MarketStore {
  id: string;
  /** **رقمُه القصير** — «القصور #M-1045» و«القصور #M-1062» لا يختلطان. */
  number: number;
  name: string;
  status: string;
  emergency_closed: boolean;
  logo_url: string | null;
  items: number;
  live_items: number;
  new_items: number;
  no_image_items: number;
  has_hours: boolean;
  /** **«متجر مميّز»** وترتيبُه في شريط السوق (٢٠٢٦-١٠-١٨). */
  featured?: boolean;
  featured_order?: number;
}

export default function StoresTab() {
  const [q, setQ] = useState("");
  const [page, setPage] = useState(1);
  const query = new URLSearchParams({ q, page: String(page) }).toString();
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  // **والمتجرُ يُدار من السوق** (قرارُ المالك ٢٠٢٦-١٠-٢٥: «التحكّم بالمتاجر من السوق أسهل من الحسابات —
  // هنيك صار أصحاب المتاجر بس»): إضافةٌ من هنا، وأفعالُ كلِّ متجرٍ تحت بطاقته.
  const canCreate = useCanCall()("POST", "/merchants");
  const [creating, setCreating] = useState(false);
  // **الضغطُ على اللوغو يفتح ملفّات الجهاز فوراً** (قرارُ المالك ٢٠٢٦-١٠-٢٦) — بلا نافذةٍ وسيطة.
  const logoTarget = useRef<string | null>(null);
  const logoPicker = useImagePicker({
    kind: "merchant_logo",
    onUploaded: async (mediaID) => {
      const id = logoTarget.current;
      if (!id) return;
      await api(`/api/v1/admin/merchants/${id}`, { method: "PATCH", body: JSON.stringify({ logo_media_id: mediaID }) });
      reload();
    },
    onError: (msg) => setError(msg),
  });
  const { data, reload } = useLiveData<{
    stores: MarketStore[];
    total: number;
    per_page: number;
  }>(
    () => api(`/api/v1/admin/market/stores?${query}`),
    ["menu", "catalog"],
    [query],
  );

  async function toggleFeatured(s: MarketStore) {
    setBusy(s.id);
    setError("");
    try {
      await api(`/api/v1/admin/merchants/${s.id}`, {
        method: "PATCH",
        body: JSON.stringify({ featured: !s.featured }),
      });
      reload();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy("");
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <div className="min-w-0 flex-1">
          <Input
            placeholder={T.search}
            value={q}
            onChange={(e) => {
              setQ(e.target.value);
              setPage(1);
            }}
          />
        </div>
        {canCreate && (
          <Button onClick={() => setCreating(true)} className="flex shrink-0 items-center gap-1.5">
            <IconStore size={16} />
            {T.addStore}
          </Button>
        )}
      </div>
      {logoPicker.input}
      {creating && (
        <MerchantModal
          merchant={null}
          onClose={() => setCreating(false)}
          onSaved={() => {
            setCreating(false);
            reload();
          }}
        />
      )}
      {error && <Alert tone="error">{error}</Alert>}
      {!data ? (
        <LoadingState />
      ) : data.stores.length === 0 ? (
        <EmptyState icon={IconStore} title={T.empty} />
      ) : (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {data.stores.map((s) => (
            <div
              key={s.id}
              className="surface flex flex-col gap-3 p-3 transition-shadow hover:elev-2"
            >
              <div className="flex items-center gap-3">
              {/* **نجمةُ «متجر مميّز» في الزاوية العلويّة يميناً** (قرارُ المالك ٢٠٢٦-١٠-٢٥). */}
              <Button
                variant="ghost"
                disabled={busy === s.id}
                onClick={() => toggleFeatured(s)}
                title={s.featured ? T.unfeature : T.makeFeatured}
                aria-label={s.featured ? T.unfeature : T.makeFeatured}
                className={`!h-10 !w-10 inline-flex items-center justify-center !min-h-0 !min-w-0 !p-0 shrink-0 rounded-full ${s.featured ? "!text-warning" : "text-ink-muted hover:text-ink"}`}
              >
                <IconStar size={19} className={s.featured ? "fill-current" : ""} />
              </Button>
              {/* **اللوغو يُبدَّل بضغطةٍ عليه** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — بلا فتح نافذة التعديل. */}
              <button
                type="button"
                title={T.changeLogo}
                aria-label={T.changeLogo}
                disabled={logoPicker.busy}
                onClick={() => {
                  logoTarget.current = s.id;
                  logoPicker.open();
                }}
                className="flex h-14 w-14 shrink-0 items-center justify-center overflow-hidden rounded-full bg-field ring-1 ring-line hover:ring-primary"
              >
                  {s.logo_url ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={mediaUrl(s.logo_url) ?? ""}
                      alt={s.name}
                      className="h-full w-full object-cover"
                    />
                  ) : (
                    <IconStore size={20} />
                  )}
              </button>
              {/* **الاسمُ والكودُ أوّلاً** — والبطاقةُ تفتح ملفَّ المتجر. */}
              <Link href={`/dashboard/merchants/${s.id}`} className="min-w-0 flex-1">
                <p className="truncate font-bold">{s.name}</p>
                <p className="text-xs text-ink-muted" dir="ltr">{fmtStoreNo(s.number)}</p>
              </Link>
              <span className="flex shrink-0 flex-col items-end gap-1">
                <StateBadge
                  variant={s.status === "active" ? "success" : "danger"}
                  label={(m.admin.merchants as unknown as Record<string, string>)[s.status] ?? s.status}
                />
                {s.emergency_closed && <StateBadge variant="warning" label={T.emergency} />}
              </span>
              </div>
              {/* **أيقوناتٌ بلا كتابة** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — النجمةُ للتمييز، والباقي أفعالُ المتجر. */}
              <div className="flex items-center gap-1 border-t border-line-soft pt-3">
                <div className="min-w-0 flex-1">
                  <StoreControl id={s.id} summary={s} onChanged={reload} />
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
      {data && (
        <Pagination
          page={page}
          total={data.total}
          perPage={data.per_page}
          onChange={setPage}
        />
      )}
    </div>
  );
}

/**
 * **أفعالُ المتجر تحت بطاقته في السوق** — تُجلب بيانات المتجر الكاملة عند الطلب وحدَه (لا مع كلّ
 * بطاقة)، ثمّ تُرسم `StoreActions` نفسُها التي كانت في الحسابات: التعديل والدوام والفتحُ والإغلاق
 * والمخالفاتُ والحظر.
 */
function StoreControl({ id, summary, onChanged }: { id: string; summary: MarketStore; onChanged: () => void }) {
  const [store, setStore] = useState<Merchant | null>(null);
  const [err, setErr] = useState("");
  // **تُجلب بياناتُ المتجر مع البطاقة** — فتظهر الأزرارُ مباشرةً بلا ضغطة «تحكّم» (٢٠٢٦-١٠-٢٥).
  const load = useCallback(async () => {
    try {
      setStore(await api<Merchant>(`/api/v1/admin/merchants/${id}`));
      setErr("");
    } catch (e) {
      setErr(errorText(e));
    }
  }, [id]);
  useEffect(() => {
    void load();
  }, [load]);
  if (err) return <span className="text-xs text-danger">{err}</span>;
  if (!store) return <LoadingState variant="inline" />;
  return (
    <div className="flex flex-col gap-2">
      {/* **بطاقةٌ مرتّبة** (قرارُ المالك ٢٠٢٦-١٠-٢٥): صاحبُ المتجر ورقمُه، وأصنافُه، **وما ينقصه ليكتمل**. */}
      <dl className="space-y-1 text-sm">
        <InfoRow label={T.owner} value={<b>{store.owner_name || "—"}</b>} />
        <InfoRow label={T.phone} value={<span dir="ltr" className="font-medium">{store.owner_phone || store.phone || "—"}</span>} />
        <InfoRow
          label={T.itemsLabel}
          value={
            <span>
              <b>{fmtNum(summary.items)}</b>
              <span className="text-ink-muted">{" · "}{T.live.replace("{n}", fmtNum(summary.live_items))}</span>
              {summary.new_items > 0 && <span className="text-accent">{" · "}{T.newItems.replace("{n}", fmtNum(summary.new_items))}</span>}
            </span>
          }
        />
      </dl>
      <Missing summary={summary} store={store} />
      <RepLine store={store} onSaved={() => { onChanged(); void load(); }} />
      <CommissionLine store={store} onSaved={() => { onChanged(); void load(); }} />
      <StoreActions
        direct
        store={store}
        onChanged={() => {
          onChanged();
          void load();
        }}
      />
    </div>
  );
}

/**
 * **العمولةُ على البطاقة وتُبدَّل بأيقونة** (قرارُ المالك ٢٠٢٦-١٠-٢٥: «حطّ شقد العمولة، وإذا بدنا نغيّرها
 * بزر أيقونة بدون ما نفوت ع تعديل») — عامّةٌ أو خاصّة، بباب تعديل المتجر نفسِه (`-1` يعيدها للعامّة).
 */
function CommissionLine({ store, onSaved }: { store: Merchant; onSaved: () => void }) {
  const canEdit = useCanCall()("PATCH", "/merchants/{id}");
  const own = store.commission_percent_exact !== undefined ? store.commission_percent_exact : store.commission_percent;
  const general = (store as unknown as { general_commission_percent_exact?: number; general_commission_percent?: number });
  const gen = general.general_commission_percent_exact ?? general.general_commission_percent ?? 0;
  const [open, setOpen] = useState(false);
  const [mode, setMode] = useState<"general" | "special">(own == null ? "general" : "special");
  const [val, setVal] = useState(String(own ?? ""));
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  async function save() {
    setBusy(true);
    try {
      await api(`/api/v1/admin/merchants/${store.id}`, {
        method: "PATCH",
        body: JSON.stringify({ commission_percent: mode === "general" ? -1 : Number(val.trim()) || 0 }),
      });
      setOpen(false);
      setErr("");
      onSaved();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="flex items-center justify-between gap-2 text-sm">
      <span>
        <span className="text-ink-muted">{T.commission}: </span>
        <b>{own == null ? T.commissionGeneral.replace("{p}", fmtNum(gen)) : T.commissionOwn.replace("{p}", fmtNum(own))}</b>
      </span>
      {canEdit && (
        <Button
          variant="ghost"
          title={T.changeCommission}
          aria-label={T.changeCommission}
          onClick={() => {
            setMode(own == null ? "general" : "special");
            setVal(String(own ?? ""));
            setOpen(true);
          }}
          className="!h-9 !w-9 inline-flex items-center justify-center !min-h-0 !min-w-0 !p-0 rounded-full text-ink-muted hover:text-ink"
        >
          <IconBalance size={18} />
        </Button>
      )}
      {open && (
        <Modal open onClose={() => setOpen(false)} title={`${T.changeCommission}: ${store.name}`}>
          <div className="space-y-3">
            <Chips
              wrap
              items={[
                { id: "general", label: T.commissionGeneral.replace("{p}", fmtNum(gen)) },
                { id: "special", label: T.commissionSpecial },
              ]}
              value={mode}
              onChange={(k) => setMode(k as "general" | "special")}
            />
            {mode === "special" && (
              <Input
                id={`com-${store.id}`}
                type="number"
                inputMode="decimal"
                dir="ltr"
                label={T.commissionPct}
                value={val}
                onChange={(e) => setVal(e.target.value)}
              />
            )}
            {err && <Alert tone="error">{err}</Alert>}
            <FormActions busy={busy || (mode === "special" && val.trim() === "")} onSave={() => void save()} onCancel={() => setOpen(false)} />
          </div>
        </Modal>
      )}
    </div>
  );
}

function InfoRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex items-baseline justify-between gap-2">
      <dt className="shrink-0 text-ink-muted">{label}</dt>
      <dd className="min-w-0 truncate">{value}</dd>
    </div>
  );
}

/** **ما ينقص المتجرَ ليكتمل** — دوامٌ وموقعٌ على الخريطة ولوغو وأصنافٌ وصورُها؛ وإلّا «مكتمل». */
function Missing({ summary, store }: { summary: MarketStore; store: Merchant }) {
  const gaps: string[] = [];
  if (!summary.has_hours) gaps.push(T.noHours);
  if (store.lat == null || store.lng == null) gaps.push(T.noLocation);
  if (!summary.logo_url) gaps.push(T.noLogo);
  if (summary.items === 0) gaps.push(T.noItems);
  else if (summary.no_image_items > 0) gaps.push(T.noImageItems.replace("{n}", fmtNum(summary.no_image_items)));
  if (gaps.length === 0) {
    return <p className="text-xs font-bold text-success">{T.complete}</p>;
  }
  return (
    <div className="flex flex-wrap items-center gap-1">
      <span className="text-xs text-ink-muted">{T.missing}:</span>
      {gaps.map((g) => (
        <Badge key={g} variant="warning">{g}</Badge>
      ))}
    </div>
  );
}

/**
 * **المندوبُ على البطاقة وتبديلُه بأيقونة** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «تسهّل علينا نقل المتجر من مندوب
 * لمندوب»). البابُ نفسُه الذي في نموذج المتجر (`sales_rep_code` + `transfer_reason`): **النقلُ يحوّل
 * عمولةَ الطلبات القادمة ونسبةَ الهدف، فسببُه إلزاميٌّ ويُقيَّد في سجلّ الأحداث.**
 */
function RepLine({ store, onSaved }: { store: Merchant; onSaved: () => void }) {
  const canCall = useCanCall();
  const canEdit = canCall("PATCH", "/merchants/{id}") && canCall("GET", "/salesreps");
  const cur = store as unknown as { sales_rep_name?: string | null; sales_rep_code?: string | null };
  const [open, setOpen] = useState(false);
  const [reps, setReps] = useState<{ id: string; full_name: string; invite_code: string | null; status: string }[] | null>(null);
  const [code, setCode] = useState("");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  async function openIt() {
    setOpen(true);
    setCode("");
    setReason("");
    setErr("");
    if (reps) return;
    try {
      const list = await api<{ id: string; full_name: string; invite_code: string | null; status: string }[]>(
        "/api/v1/admin/salesreps",
      );
      setReps((list ?? []).filter((r) => r.invite_code && r.status === "active"));
    } catch (e) {
      setErr(errorText(e));
    }
  }

  async function save() {
    if (!code) return setErr(T.pickRep);
    if (!reason.trim()) return setErr(m.admin.merchants.transferReasonRequired);
    setBusy(true);
    try {
      await api(`/api/v1/admin/merchants/${store.id}`, {
        method: "PATCH",
        body: JSON.stringify({ sales_rep_code: code, transfer_reason: reason.trim() }),
      });
      setOpen(false);
      onSaved();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex items-center justify-between gap-2 text-sm">
      <span className="min-w-0 truncate">
        <span className="text-ink-muted">{T.rep}: </span>
        {cur.sales_rep_name || cur.sales_rep_code ? (
          <>
            <b>{cur.sales_rep_name || "—"}</b>
            {cur.sales_rep_code && (
              <span dir="ltr" className="ms-1.5 rounded-full bg-field px-2 py-0.5 text-2xs text-ink-muted">
                {cur.sales_rep_code}
              </span>
            )}
          </>
        ) : (
          <span className="text-ink-muted">{T.noRep}</span>
        )}
      </span>
      {canEdit && (
        <Button
          variant="ghost"
          title={T.changeRep}
          aria-label={T.changeRep}
          onClick={() => void openIt()}
          className="!h-9 !w-9 inline-flex items-center justify-center !min-h-0 !min-w-0 !p-0 rounded-full text-ink-muted hover:text-ink"
        >
          <IconSwap size={18} />
        </Button>
      )}
      {open && (
        <Modal open onClose={() => setOpen(false)} title={`${T.changeRep}: ${store.name}`}>
          <div className="space-y-3">
            <Select label={T.newRep} value={code} onChange={(e) => setCode(e.target.value)}>
              <option value="">{T.pickRep}</option>
              {(reps ?? [])
                .filter((r) => r.invite_code !== cur.sales_rep_code)
                .map((r) => (
                  <option key={r.id} value={r.invite_code ?? ""}>
                    {`${r.full_name || "—"} · ${r.invite_code}`}
                  </option>
                ))}
            </Select>
            <Input
              id={`rep-reason-${store.id}`}
              label={m.admin.merchants.transferReason}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            />
            <p className="text-xs text-ink-muted">{m.admin.merchants.transferReasonHint}</p>
            {err && <Alert tone="error">{err}</Alert>}
            <FormActions busy={busy} onSave={() => void save()} onCancel={() => setOpen(false)} />
          </div>
        </Modal>
      )}
    </div>
  );
}
