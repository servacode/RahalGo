"use client";

/**
 * عملائي — الصفحة الواحدة لعلاقة المندوب بمتاجره.
 *
 * دُمج فيها ما كان قسماً مستقلاً ("طلبات الانضمام"): العميل الذي سجّله المندوب
 * يظهر هنا **بحالته** (بانتظار الموافقة / مرفوض) ثم يصبح عميلاً فعّالاً عند
 * موافقة الإدارة — فلا يتنقّل المندوب بين قسمين لمتابعة العميل نفسه.
 *
 * ومنها يسجّل عميلاً جديداً وهو واقف في المحل: أنجع من إرسال رابط يُنسى.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import dynamic from "next/dynamic";
import { getMessages, defaultLocale, fmtNum, fmtDate } from "@rahalgo/i18n";
import {
  Alert,
  CategoryIcon,
  Badge,
  Button,
  Input,
  Select,
  Modal,
  PasswordMeter,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  EntityCard,
  useLiveData,
  useLiveRefresh,
  IconStore,
  IconAdd,
  IconUser,
  IconPhone,
  IconLocation,
  IconLock,
  IconSuccess,
  IconError,
  IconWallet,
  IconWarning,
  IconWhatsApp,
  IconView,
} from "@rahalgo/ui";
import { api, mediaUrl, ApiError } from "@/lib/api";

// الخريطة من الحزمة المشتركة — مدخل فرعي كي لا تُجرّ مكتبتها لكل صفحة
const PickMap = dynamic(() => import("@rahalgo/ui/map").then((mod) => mod.PickMap), {
  ssr: false,
  loading: () => <div className="h-64 w-full animate-pulse rounded-card bg-page" />,
});

interface Place {
  label: string;
  lat: number;
  lng: number;
}

const m = getMessages(defaultLocale);
const C = m.rep.clients;

interface RepMerchant {
  id: string;
  name: string;
  category_icon: string;
  category_name: string;
  logo_thumb_url: string | null;
  status: string;
  joined_at: string;
  owner_phone: string | null;
  delivered_orders: number;
  cancelled_orders: number;
  my_commission: number;
  /** كم طلباً احتُسب نحو تفعيل العمولة — بشرط القاعدة نفسه */
  activation_done: number;
  /** كم يلزم لتبدأ العمولة — 0 يعني لا عتبة */
  activation_needed: number;
  last_order_at: string | null;
}

interface Lead {
  id: string;
  store_name: string;
  owner_name: string;
  phone: string;
  area: string;
  category_icon: string | null;
  status: "new" | "converted" | "rejected";
  created_at: string;
  /**
   * **سببُ ردّ الإدارة** — يُقرأ في البطاقة لا في إشعارٍ يمرّ.
   *
   * من رُدّت فرصتُه بلا سببٍ **يلاحق عميلاً ميتاً أو يعيد إرسالها** فتُردّ
   * ثانية. **والإشعارُ يُقرأ مرّةً ويُنسى، والبطاقةُ تبقى.**
   */
  decision_note?: string;
}

interface Category {
  id: string;
  name: string;
  icon: string;
}

function errText(err: unknown): string {
  if (!(err instanceof ApiError)) return m.errors.internal;
  const key = err.body.message_key.split(".").pop() ?? "";
  return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
}

/** منذ كم يوم آخر نشاط — «منذ 12 يوماً» يقول ما لا يقوله تاريخٌ مجرّد. */
function sinceLabel(iso: string): string {
  const days = Math.floor((Date.now() - new Date(iso).getTime()) / 86_400_000);
  return days < 1 ? C.activeToday : C.activeDays.replace("{n}", fmtNum(days));
}

/** منذ كم يوم أُرسل الطلب — كي يرى المندوب ما طال انتظاره. */
function waitedLabel(iso: string): string {
  const days = Math.floor((Date.now() - new Date(iso).getTime()) / 86_400_000);
  return days < 1 ? C.waitingToday : C.waitingSince.replace("{n}", fmtNum(days));
}

export default function ClientsPage() {
  const [adding, setAdding] = useState(false);

  const {
    data: merchants,
    loading,
    reload,
  } = useLiveData<RepMerchant[]>(() => api("/api/v1/rep/merchants"), ["lead", "order", "account"]);
  const { data: leads, reload: reloadLeads } = useLiveData<Lead[]>(
    () => api("/api/v1/rep/leads"),
    ["lead"],
  );

  const refreshAll = useCallback(() => {
    reload();
    reloadLeads();
  }, [reload, reloadLeads]);
  useLiveRefresh(["lead"], refreshAll);

  if (loading) return <LoadingState />;

  const active = merchants ?? [];
  // المحوَّل صار متجراً فعلياً في القائمة أعلاه — نعرض المعلّق والمرفوض فقط
  const open = (leads ?? []).filter((l) => l.status !== "converted");
  const empty = active.length === 0 && open.length === 0;

  const addButton = (
    <Button onClick={() => setAdding(true)} className="flex items-center gap-1.5">
      <IconAdd size={16} />
      {C.add}
    </Button>
  );

  return (
    <PageContainer>
      <PageHeader icon={IconStore} title={m.terms.clients} actions={addButton} />

      {empty ? (
        <EmptyState icon={IconStore} title={m.rep.merchantsEmpty} action={addButton} />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {/* المعلّق أولاً — هو ما يحتاج متابعة، ويبقى مُتقطّع الحدّ ليُميَّز */}
          {open.map((l) => (
            <EntityCard
              key={l.id}
              muted
              media={
                <span className="flex h-12 w-12 items-center justify-center rounded-control bg-page">
                  <CategoryIcon name={l.category_icon ?? "other"} size={20} />
                </span>
              }
              title={l.store_name}
              subtitle={
                <>
                  {l.owner_name && <span>{l.owner_name} — </span>}
                  <span dir="ltr">{l.phone}</span>
                </>
              }
              badge={
                <Badge variant={l.status === "new" ? "warning" : "danger"}>
                  {l.status === "new" ? C.pending : C.rejected}
                </Badge>
              }
              footer={
                l.status === "new" ? (
                  waitedLabel(l.created_at)
                ) : l.decision_note ? (
                  <span className="text-danger">
                    {C.reason}: {l.decision_note}
                  </span>
                ) : undefined
              }
            />
          ))}

          {active.map((mr) => {
            const logo = mediaUrl(mr.logo_thumb_url);
            const live = mr.status === "active";
            return (
              <EntityCard
                key={mr.id}
                muted={!live}
                media={
                  logo ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={logo} alt="" loading="lazy" className="h-12 w-12 rounded-control object-cover" />
                  ) : (
                    <span className="flex h-12 w-12 items-center justify-center rounded-control bg-primary-light">
                      <CategoryIcon name={mr.category_icon} size={20} />
                    </span>
                  )
                }
                title={mr.name}
                subtitle={mr.category_name}
                badge={
                  <Badge variant={live ? "success" : "danger"}>
                    {live ? m.terms.active : m.terms.suspended}
                  </Badge>
                }
                stats={[
                  {
                    label: C.statDelivered,
                    value: fmtNum(mr.delivered_orders),
                    icon: <IconSuccess />,
                    tone: mr.delivered_orders > 0 ? "success" : "muted",
                  },
                  {
                    label: C.statCancelled,
                    value: fmtNum(mr.cancelled_orders),
                    icon: <IconError />,
                    // الأحمر للملغي **فقط إن وُجد**: صفر ملغي خبر سارّ لا تحذير
                    tone: mr.cancelled_orders > 0 ? "danger" : "muted",
                  },
                  {
                    label: C.statCommission,
                    value: fmtNum(mr.my_commission),
                    icon: <IconWallet />,
                    // صفرٌ ليس ربحاً ولا خسارة: تلوينه أخضر يَعِد بما ليس
                    tone: mr.my_commission > 0 ? "success" : "muted",
                  },
                ]}
                footer={
                  <>
                    {/* **لماذا لا عمولة بعد.**

                        العمولة محجوزة حتى يُثبت المتجرُ أنه يعمل. وكانت القاعدة
                        تُطبَّق في الخادم **ولا تُقال في شاشة**: يرى المندوبُ
                        طلباتٍ مُسلَّمة وعمولةً صفراً، ولا شيء يربط بينهما —
                        **فيظنّ المنصةَ أكلت حقَّه.**

                        وتختفي حين يُفعَّل: خبرٌ انتهى مفعولُه يبقى ضجيجاً. */}
                    {mr.activation_needed > 1 &&
                      mr.activation_done < mr.activation_needed && (
                        <span className="text-warning">
                          {C.activationHint
                            .replace("{done}", fmtNum(mr.activation_done))
                            .replace("{needed}", fmtNum(mr.activation_needed))}
                          {" · "}
                        </span>
                      )}
                    {m.rep.joinedAt} <span dir="ltr">{fmtDate(mr.joined_at)}</span>
                    {" · "}
                    {mr.last_order_at
                      ? `${C.lastOrder} ${sinceLabel(mr.last_order_at)}`
                      : C.noOrders}
                  </>
                }
                actions={
                  <>
                    <Link
                      href={`/portal/merchants/${mr.id}`}
                      className="flex flex-1 items-center justify-center gap-1.5 rounded-control bg-primary px-3 py-1.5 text-sm font-medium text-on-bright hover:bg-primary-dark"
                    >
                      <IconView size={15} />
                      {m.rep.merchantDetail.open}
                    </Link>
                    {mr.owner_phone && (
                      <a
                        href={`https://wa.me/${mr.owner_phone.replace(/[^0-9]/g, "")}`}
                        target="_blank"
                        rel="noreferrer"
                        className="flex flex-1 items-center justify-center gap-1.5 rounded-control border border-line px-3 py-1.5 text-sm font-medium text-ink hover:bg-row-hover"
                      >
                        <IconWhatsApp size={15} />
                        {C.contact}
                      </a>
                    )}
                  </>
                }
              />
            );
          })}
        </div>
      )}

      {adding && (
        <AddClientModal
          onClose={() => setAdding(false)}
          onDone={() => {
            setAdding(false);
            refreshAll();
          }}
        />
      )}
    </PageContainer>
  );
}

/**
 * نموذج تسجيل عميل — بلا كلمة مرور عمداً: لا يجوز أن يعرف المندوب كلمة مرور
 * صاحب المتجر (يدخل برمز واتساب ثم يضبطها بنفسه). وبدل خريطة ثقيلة، زرّ يلتقط
 * موقع المتجر من الجهاز — والمندوب واقف داخله فيكون أدقّ من أي دبوس يدوي.
 */
function AddClientModal({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const { data: categories } = useLiveData<Category[]>(() => api("/api/v1/rep/categories"));

  const [storeName, setStoreName] = useState("");
  const [ownerName, setOwnerName] = useState("");
  const [phone, setPhone] = useState("");
  const [area, setArea] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [coords, setCoords] = useState<{ lat: number; lng: number } | null>(null);
  const [suggestions, setSuggestions] = useState<Place[]>([]);
  const [areaTouched, setAreaTouched] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [done, setDone] = useState(false);

  /**
   * الموقع ← عنوان: يقترح وصفاً من الخريطة ما لم يكن المندوب قد كتب عنواناً
   * بنفسه (لا نطمس ما كتبه — تغطية العنونة في الرقة ضعيفة ووصفه أدقّ غالباً).
   */
  const fillAddress = useCallback(
    async (la: number, ln: number) => {
      if (areaTouched) return;
      try {
        const p = await api<Place>(`/api/v1/geo/reverse?lat=${la}&lng=${ln}`);
        if (p.label) setArea(p.label);
      } catch {
        /* العنونة مساعدة لا شرط — الصمت مقصود */
      }
    },
    [areaTouched],
  );

  // عنوان مكتوب ← مواقع مرشّحة (بتهدئة). قائمة فارغة نتيجة مشروعة: يبقى الدبوس يدوياً.
  const areaRef = useRef(area);
  areaRef.current = area;
  useEffect(() => {
    if (!areaTouched || area.trim().length < 3) {
      setSuggestions([]);
      return;
    }
    const t = setTimeout(async () => {
      try {
        const found = await api<Place[]>(`/api/v1/geo/search?q=${encodeURIComponent(area)}`);
        if (areaRef.current === area) setSuggestions(found.slice(0, 4));
      } catch {
        setSuggestions([]);
      }
    }, 600);
    return () => clearTimeout(t);
  }, [area, areaTouched]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (!categoryId) return setError(m.site.join.pickCategory);
    if (password.length < 8) return setError(m.errors.weak_password);
    if (password !== confirm) return setError(m.errors.password_mismatch);
    setBusy(true);
    try {
      await api("/api/v1/rep/leads", {
        method: "POST",
        body: JSON.stringify({
          store_name: storeName,
          owner_name: ownerName,
          phone,
          area,
          category_id: categoryId,
          password,
          lat: coords?.lat ?? null,
          lng: coords?.lng ?? null,
        }),
      });
      setDone(true);
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  if (done) {
    return (
      <Modal open onClose={onDone} title={C.addTitle}>
        <div className="py-4 text-center">
          <IconSuccess size={40} className="mx-auto mb-3 text-success" />
          <p className="text-sm text-ink">{C.done}</p>
          <Button onClick={onDone} className="mt-5 w-full py-2.5">
            {m.common.confirm}
          </Button>
        </div>
      </Modal>
    );
  }

  return (
    // نافذة عريضة بعمودين: البيانات يميناً والموقع يساراً — النموذج الطولي
    // كان يفرض تمريراً مزعجاً بينما نصف الشاشة فارغ.
    <Modal open onClose={onClose} title={C.addTitle} size="xl">
      <form onSubmit={submit} className="grid gap-6 lg:grid-cols-2">
        {/* ---------- العمود الأول: بيانات المتجر وصاحبه ---------- */}
        <div className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <Input
              id="store-name"
              label={m.site.join.storeName}
              icon={<IconStore />}
              required
              value={storeName}
              onChange={(e) => setStoreName(e.target.value)}
            />
            <Select
              id="category"
              label={m.site.join.category}
              required
              value={categoryId}
              onChange={(e) => setCategoryId(e.target.value)}
            >
              <option value="">{m.site.join.pickCategory}</option>
              {(categories ?? []).map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </Select>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <Input
              id="owner-name"
              label={m.site.join.ownerName}
              icon={<IconUser />}
              value={ownerName}
              onChange={(e) => setOwnerName(e.target.value)}
            />
            <Input
              id="owner-phone"
              label={m.site.join.phone}
              icon={<IconPhone />}
              dir="ltr"
              inputMode="tel"
              required
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              className="text-end"
              placeholder="09xxxxxxxx"
            />
          </div>

          {/* كلمة مرور مؤقتة يسلّمها المندوب — يُجبَر المالك على تبديلها أول دخول */}
          <div className="rounded-card border border-line bg-page p-3">
            <div className="grid gap-3 sm:grid-cols-2">
              <Input
                id="owner-pw"
                label={m.site.join.password}
                icon={<IconLock />}
                type="password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
              <Input
                id="owner-pw2"
                label={m.site.join.confirmPassword}
                icon={<IconLock />}
                type="password"
                required
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
              />
            </div>
            <div className="mt-2">
              <PasswordMeter
                value={password}
                labels={[m.auth.strength.weak, m.auth.strength.fair, m.auth.strength.strong]}
              />
            </div>
            <p className="mt-2 text-xs text-ink-muted">{C.passwordHint}</p>
          </div>
        </div>

        {/* ---------- العمود الثاني: العنوان والموقع ---------- */}
        <div className="flex flex-col gap-3">
          <div className="relative">
            <Input
              id="area"
              label={m.site.join.area}
              icon={<IconLocation />}
              value={area}
              onChange={(e) => {
                setArea(e.target.value);
                setAreaTouched(true);
              }}
              placeholder={m.site.join.areaPlaceholder}
            />
            {/* المرشّحات تطفو فوق الخريطة بدل أن تدفعها للأسفل */}
            {suggestions.length > 0 && (
              <ul className="absolute inset-x-0 top-full z-20 mt-1 overflow-hidden rounded-control border border-line bg-surface elev-3">
                {suggestions.map((p, i) => (
                  <li key={i} className="border-b border-line-soft last:border-0">
                    <button
                      type="button"
                      onClick={() => {
                        setCoords({ lat: p.lat, lng: p.lng });
                        setArea(p.label);
                        setSuggestions([]);
                      }}
                      className="flex w-full items-start gap-2 px-3 py-2 text-start text-sm transition-colors hover:bg-row-hover"
                    >
                      <IconLocation size={15} className="mt-0.5 shrink-0 text-primary" />
                      {p.label}
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>

          <PickMap
            lat={coords?.lat ?? null}
            lng={coords?.lng ?? null}
            height="h-56 lg:h-full lg:min-h-[16rem]"
            onPick={(la, ln) => {
              setCoords({ lat: la, lng: ln });
              void fillAddress(la, ln);
            }}
            onLocated={(la, ln) => void fillAddress(la, ln)}
          />

          <p className="flex items-center gap-1.5 text-xs text-ink-muted">
            {coords ? (
              <>
                <IconSuccess size={14} className="text-success" />
                {C.locationCaptured}
              </>
            ) : (
              m.map.pickHint
            )}
          </p>
        </div>

        {/* ---------- التذييل عبر العمودين ---------- */}
        <div className="lg:col-span-2">
          {error && (
            <Alert className="mb-3">{error}</Alert>
          )}
          <div className="flex justify-end gap-2 border-t border-line-soft pt-4">
            <Button type="button" variant="secondary" onClick={onClose}>
              {m.common.cancel}
            </Button>
            <Button type="submit" disabled={busy}>
              {busy ? C.sending : C.submit}
            </Button>
          </div>
        </div>
      </form>
    </Modal>
  );
}
