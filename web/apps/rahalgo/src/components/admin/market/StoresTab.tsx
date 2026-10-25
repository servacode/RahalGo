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

import { useCallback, useEffect, useState } from "react";
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
  Modal,
  Chips,
  FormActions,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import ImageUpload from "@/components/admin/ImageUpload";
import { StoreActions } from "@/components/admin/StoreActions";
import { MerchantModal, type Merchant } from "@/components/admin/MerchantModal";

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
  const [logoFor, setLogoFor] = useState<MarketStore | null>(null);
  async function saveLogo(id: string, mediaID: string) {
    try {
      await api(`/api/v1/admin/merchants/${id}`, { method: "PATCH", body: JSON.stringify({ logo_media_id: mediaID }) });
      setLogoFor(null);
      reload();
    } catch (err) {
      setError(errorText(err));
    }
  }
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
      {logoFor && (
        <Modal open onClose={() => setLogoFor(null)} title={`${T.changeLogo}: ${logoFor.name}`}>
          <ImageUpload
            kind="merchant_logo"
            label={T.changeLogo}
            initialUrl={logoFor.logo_url}
            onChange={(id) => void saveLogo(logoFor.id, id)}
          />
        </Modal>
      )}
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
                onClick={() => setLogoFor(s)}
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
              <Link
                href={`/dashboard/merchants/${s.id}`}
                className="flex min-w-0 flex-1 items-center gap-3"
              >
                <div className="min-w-0 flex-1 space-y-1">
                  <p className="truncate font-bold">
                    {s.name}{" "}
                    <span className="text-xs font-normal text-ink-muted">
                      {fmtStoreNo(s.number)}
                    </span>
                  </p>
                  <p className="text-xs text-ink-muted">
                    {T.items.replace("{n}", fmtNum(s.items))}
                    {" · "}
                    {T.live.replace("{n}", fmtNum(s.live_items))}
                  </p>
                  <div className="flex flex-wrap gap-1">
                    <Badge
                      variant={s.status === "active" ? "success" : "neutral"}
                    >
                      {(m.admin.merchants as unknown as Record<string, string>)[
                        s.status
                      ] ?? s.status}
                    </Badge>
                    {s.emergency_closed && (
                      <Badge variant="danger">{T.emergency}</Badge>
                    )}
                    {s.featured && (
                      <Badge variant="accent">{`${m.admin.merchants.featuredBadge} · ${fmtNum(s.featured_order ?? 0)}`}</Badge>
                    )}
                    {s.new_items > 0 && (
                      <Badge variant="accent">
                        {T.newItems.replace("{n}", fmtNum(s.new_items))}
                      </Badge>
                    )}
                    {s.no_image_items > 0 && (
                      <Badge variant="warning">
                        {T.noImageItems.replace(
                          "{n}",
                          fmtNum(s.no_image_items),
                        )}
                      </Badge>
                    )}
                    {!s.has_hours && (
                      <Badge variant="warning">{T.noHours}</Badge>
                    )}
                  </div>
                </div>
              </Link>
              </div>
              {/* **أيقوناتٌ بلا كتابة** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — النجمةُ للتمييز، والباقي أفعالُ المتجر. */}
              <div className="flex items-center gap-1 border-t border-line-soft pt-3">
                <div className="min-w-0 flex-1">
                  <StoreControl id={s.id} onChanged={reload} />
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
function StoreControl({ id, onChanged }: { id: string; onChanged: () => void }) {
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
      {/* **صاحبُ المتجر ورقمُه على البطاقة** (قرارُ المالك ٢٠٢٦-١٠-٢٥). */}
      <div className="flex items-center justify-between gap-2 text-sm">
        <span className="min-w-0 truncate">
          <span className="text-ink-muted">{T.owner}: </span>
          <b>{store.owner_name || "—"}</b>
        </span>
        <span dir="ltr" className="shrink-0 font-medium">
          {store.owner_phone || store.phone || "—"}
        </span>
      </div>
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
