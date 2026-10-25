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
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
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
              <Link
                href={`/dashboard/merchants/${s.id}`}
                className="flex min-w-0 flex-1 items-center gap-3"
              >
                <div className="flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-control bg-field">
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
                </div>
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
              <div className="flex flex-wrap items-center gap-2 border-t border-line-soft pt-3">
                <Button
                  variant={s.featured ? "secondary" : "primary"}
                  disabled={busy === s.id}
                  onClick={() => toggleFeatured(s)}
                  className="shrink-0"
                >
                  {s.featured ? T.unfeature : T.makeFeatured}
                </Button>
                <StoreControl id={s.id} onChanged={reload} />
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
    <StoreActions
      direct
      store={store}
      onChanged={() => {
        onChanged();
        void load();
      }}
    />
  );
}
