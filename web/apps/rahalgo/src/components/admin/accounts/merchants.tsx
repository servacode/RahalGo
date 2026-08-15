"use client";

import { useCallback, useEffect, useState } from "react";
import dynamic from "next/dynamic";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";

const PickMap = dynamic(() => import("@rahalgo/ui/map").then((mod) => mod.PickMap), { ssr: false });
import {
  Pagination,
  Alert,
  CategoryIcon,
  CategoryIconPicker,
  type CategoryIconKey,
  IconPrev,
  useLiveRefresh,
  Button,
  Input,
  Select,
  Badge,
  Modal,
  FormSection,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  IconStore,
  IconPhone,
  IconUser,
  IconStatus,
  IconSearch,
  IconAdd,
  IconEdit,
  IconLocation,
  IconSettings,
  IconOrder as IconMenu,
  IconDate,
  Checkbox,
  LoadingState,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { StoreActions } from "@/components/admin/StoreActions";
import { useAuth } from "@/lib/auth";
import ImageUpload, { MediaThumb } from "@/components/admin/ImageUpload";

const m = getMessages(defaultLocale);

interface Category {
  id: string;
  name: string;
  icon: string;
  sort_order: number;
  active: boolean;
}

/** **صفُّ متجرٍ كما يرسله المحرّك** — **ويُقرأ من ملفّ صاحبه أيضاً**،
 *  فصار مُصدَّراً: **نسختان من الشكل تفترقان يوماً.** */
export interface Merchant {
  id: string;
  name: string;
  description: string;
  category_id: string;
  category_name: string;
  category_icon: string;
  phone: string;
  address_text: string;
  owner_phone: string | null;
  sales_rep_phone: string | null;
  sales_rep_code: string | null;
  lat: number | null;
  lng: number | null;
  logo_url: string | null;
  logo_thumb_url: string | null;
  status: string;
  /** إلغاءاتُ المتجر داخل نافذة الحظر وبعد آخر عفو */
  violations: number;
  commission_percent: number;
  emergency_closed: boolean;
  created_at: string;
}

interface MerchantPage {
  merchants: Merchant[];
  total: number;
  page: number;
  per_page: number;
}

function translateKey(key: string): string {
  let node: unknown = m;
  for (const part of key.split(".")) {
    if (typeof node !== "object" || node === null) return m.errors.internal;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" ? node : m.errors.internal;
}

function errText(err: unknown): string {
  return err instanceof ApiError ? translateKey(err.body.message_key) : m.errors.internal;
}

export default function MerchantsTable() {
  const { user: me } = useAuth();
  const router = useRouter();
  const isAdmin = !!me?.roles.includes("admin");

  const [data, setData] = useState<MerchantPage | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [query, setQuery] = useState("");
  const [category, setCategory] = useState("");
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState<Merchant | null | "new">(null);
  const [catsOpen, setCatsOpen] = useState(false);
  /** **سجلُّ مخالفاتِ متجرٍ بعينه** — ومنه يُصدَر الإنذار. */
  const [view, setView] = useViewMode("merchants", "cards");

  const loadCategories = useCallback(async () => {
    try {
      setCategories(await api<Category[]>("/api/v1/admin/categories"));
    } catch (err) {
      setError(errText(err));
    }
  }, []);

  const load = useCallback(async () => {
    try {
      const params = new URLSearchParams({
        query,
        category_id: category,
        status,
        page: String(page),
        per_page: "9",
      });
      setData(await api<MerchantPage>(`/api/v1/admin/merchants?${params}`));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [query, category, status, page]);

  useEffect(() => {
    void loadCategories();
  }, [loadCategories]);

  useEffect(() => {
    const t = setTimeout(load, 250);
    return () => clearTimeout(t);
  }, [load]);

  useLiveRefresh(["lead", "account"], load);

  const columns: DataColumn<Merchant>[] = [
    {
      id: "name",
      header: m.admin.merchants.name,
      icon: <IconStore />,
      primary: true,
      cell: (mr) => (
        <span className="inline-flex items-center gap-2">
          <MediaThumb url={mr.logo_thumb_url} alt={mr.name} fallback={mr.name} size={34} />
          <span className="inline-flex items-center gap-1.5">
            {/* **والأيقونةُ تُرسم لا تُطبع** — والمكوّنُ مستوردٌ في هذا
                الملفّ نفسِه ويُستعمل في نافذة التصنيفات. (٢٠٢٦-٠٨-٠٨.) */}
            <CategoryIcon name={mr.category_icon} size={15} />
            {mr.name}
          </span>
        </span>
      ),
    },
    {
      id: "category",
      header: m.admin.merchants.category,
      cell: (mr) => <Badge variant="primary">{mr.category_name}</Badge>,
    },
    {
      id: "phone",
      header: m.admin.merchants.merchantPhone,
      icon: <IconPhone />,
      cell: (mr) =>
        mr.phone ? (
          <span dir="ltr" className="font-medium">
            {mr.phone}
          </span>
        ) : (
          "—"
        ),
    },
    {
      id: "address",
      header: m.admin.merchants.address,
      icon: <IconLocation />,
      cell: (mr) => <span className="text-ink-muted">{mr.address_text || "—"}</span>,
    },
    {
      id: "owner",
      header: m.admin.merchants.owner,
      icon: <IconUser />,
      cell: (mr) =>
        mr.owner_phone ? (
          <span dir="ltr">{mr.owner_phone}</span>
        ) : (
          <span className="text-ink-muted">{m.admin.merchants.noOwner}</span>
        ),
    },
    {
      id: "status",
      header: m.admin.merchants.status,
      icon: <IconStatus />,
      cell: (mr) => (
        <div className="flex flex-wrap items-center gap-1.5">
          <Badge
            variant={
              mr.status === "active"
                ? "success"
                : mr.status === "suspended"
                  ? "danger"
                  : "neutral"
            }
          >
            {mr.status === "active"
              ? m.admin.merchants.active
              : mr.status === "suspended"
                ? m.admin.merchants.suspended
                : m.admin.merchants.inactive}
          </Badge>
          {/* **العدّادُ يُرى قبل أن يبلغ.**

              متجرٌ على ٤ من ٥ تتّصل به العملياتُ فتنقذ الطرفين، وحظرٌ يقع
              فجأةً يُفاجئ من لم يكن يعلم أن هناك عدّاداً. ولا يُعرض صفراً:
              **لا مخالفةَ خبرٌ سارّ لا تحذير.** */}
          {mr.violations > 0 && (
            <Badge variant="warning">
              {m.admin.merchants.violations.replace("{n}", String(mr.violations))}
            </Badge>
          )}
        </div>
      ),
    },
  ];

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <h1 className="heading-page">{m.admin.merchants.title}</h1>
        {isAdmin && (
          <div className="flex gap-2">
            <Button
              variant="secondary"
              onClick={() => setCatsOpen(true)}
              className="flex items-center gap-1.5"
            >
              <IconSettings size={16} />
              {m.admin.merchants.manageCategories}
            </Button>
            <Button onClick={() => setEditing("new")} className="flex items-center gap-1.5">
              <IconAdd size={16} />
              {m.admin.merchants.create}
            </Button>
          </div>
        )}
      </div>

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="w-64">
          <Input
            icon={<IconSearch />}
            placeholder={m.admin.merchants.searchPlaceholder}
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setPage(1);
            }}
          />
        </div>
        <div className="w-44">
          <Select
            value={category}
            onChange={(e) => {
              setCategory(e.target.value);
              setPage(1);
            }}
          >
            <option value="">{m.admin.merchants.allCategories}</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                <CategoryIcon name={c.icon} size={15} />
                {c.name}
              </option>
            ))}
          </Select>
        </div>
        <div className="w-36">
          <Select
            value={status}
            onChange={(e) => {
              setStatus(e.target.value);
              setPage(1);
            }}
          >
            <option value="">{m.admin.merchants.allStatuses}</option>
            <option value="active">{m.admin.merchants.active}</option>
            <option value="inactive">{m.admin.merchants.inactive}</option>
          </Select>
        </div>
        <div className="ms-auto">
          <ViewToggle
            view={view}
            onChange={setView}
            tableLabel={m.common.viewTable}
            cardsLabel={m.common.viewCards}
          />
        </div>
      </div>

      {error && (
        <Alert className="mb-4">{error}</Alert>
      )}

      <DataView
        items={data?.merchants ?? []}
        getKey={(mr) => mr.id}
        columns={columns}
        view={view}
        empty={m.admin.merchants.noResults}
        actions={
          isAdmin
            ? (mr) => (
                /* ══════════════════════════════════════════════════════
                   **والأزرارُ مكوّنٌ واحدٌ يُقرأ من موضعين**
                   ══════════════════════════════════════════════════════

                   (قرارُ المالك ٢٠٢٦-٠٨-١٦: تنتقل إلى ملفّ صاحب المتجر
                    تمهيداً لحذف هذا التبويب.)

                   **ونسختان تفترقان يوماً**: يُضاف فعلٌ هنا ويُنسى هناك،
                   **فيُحذف التبويبُ وقد ضاع الفعل.** */
                <StoreActions store={mr} onChanged={load} />
              )
            : undefined
        }
      />

      {data && (
        <div className="mt-4 flex flex-wrap items-center justify-between gap-2 text-sm text-ink-muted">
          <span>{m.admin.users.totalCount.replace("{count}", fmtNum(data.total))}</span>
          {/* **والترقيمُ من المكوّن المشترك.**

              كان مكتوباً هنا وفي ثلاثة ملفّاتٍ أخرى بالشكل نفسِه، **وأرقامُه
              لاتينيّةٌ في واجهةٍ عربية** (`{page} / {totalPages}`) لأنّها لم
              تمرّ بـ`fmtNum`. */}
          <Pagination
            page={page}
            total={data.total}
            perPage={data.per_page}
            onChange={setPage}
          />
        </div>
      )}

      {editing && (
        <MerchantModal
          merchant={editing === "new" ? null : editing}
          categories={categories}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            void load();
          }}
        />
      )}
      <CategoriesModal
        open={catsOpen}
        categories={categories}
        onClose={() => setCatsOpen(false)}
        onChanged={loadCategories}
      />
    </div>
  );
}


export function MerchantModal({
  merchant,
  categories,
  onClose,
  onSaved,
}: {
  merchant: Merchant | null;
  /** **وفارغةٌ تعني «اجلبها بنفسك»** — فتُفتح من أيّ شاشة. */
  categories?: Category[];
  onClose: () => void;
  onSaved: () => void;
}) {
  // ══════════════════════════════════════════════════════════════════
  // **والتصنيفاتُ تُجلب هنا إن لم تصل**
  // ══════════════════════════════════════════════════════════════════
  //
  // **ونافذةٌ تعتمد على من ناداها تُفتح ناقصةً من الباب الثاني** —
  // فيرى فاتحُها قائمةَ تصنيفاتٍ فارغةً ولا يعرف لماذا.
  const [cats, setCats] = useState<Category[]>(categories ?? []);
  const [catsError, setCatsError] = useState("");
  useEffect(() => {
    if (cats.length > 0) return;
    // **وهي مصفوفةٌ لا كائن** — انظر `CategoriesModal`.
    void api<Category[]>("/api/v1/admin/categories")
      .then((r) => setCats(r ?? []))
      // **وفشلُ الجلب يُقال لا يُبتلع** — **وقائمةٌ فارغةٌ تُقرأ «لا
      // تصنيفاتِ في المنصّة»** وهي في الحقيقة نداءٌ سقط.
      .catch((e) => setCatsError(errText(e)));
  }, [cats.length]);

  const [name, setName] = useState(merchant?.name ?? "");
  const [description, setDescription] = useState(merchant?.description ?? "");
  const [categoryId, setCategoryId] = useState(merchant?.category_id ?? cats[0]?.id ?? "");
  const [phone, setPhone] = useState(merchant?.phone ?? "");
  const [address, setAddress] = useState(merchant?.address_text ?? "");
  const [ownerPhone, setOwnerPhone] = useState(merchant?.owner_phone ?? "");
  // **واسمُه** — (قرارُ المالك ٢٠٢٦-٠٨-١٥): كان الحسابُ يُنشأ باسمٍ
  // فارغ، **فيصير في الحسابات صفٌّ برقمٍ بلا اسم.**
  const [ownerName, setOwnerName] = useState("");
  // **وكلمتُه المؤقّتة** — (قرارُ المالك ٢٠٢٦-٠٨-١٥): يخرج من النموذج
  // **حسابٌ جاهزٌ ومتجرٌ جاهز.**
  const [ownerPass, setOwnerPass] = useState("");
  const [ownerPass2, setOwnerPass2] = useState("");
  const [repCode, setRepCode] = useState(merchant?.sales_rep_code ?? "");
  const [lat, setLat] = useState<number | null>(merchant?.lat ?? null);
  const [lng, setLng] = useState<number | null>(merchant?.lng ?? null);
  const [commission, setCommission] = useState(String(merchant?.commission_percent ?? 10));
  // null = لم يُلمس (لا يُرسل)، "" = إزالة، معرف = شعار جديد
  const [logoID, setLogoID] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    // **والتطابقُ يُفحص قبل النداء** — **ومن أخطأ في التأكيد يعرفها
    // هنا لا بعد أن يُنشأ الحسابُ بكلمةٍ لا يعرفها.**
    if (!merchant && ownerPass !== ownerPass2) {
      setError(m.admin.merchants.passwordMismatch);
      return;
    }
    setBusy(true);
    setError("");
    const body = {
      name,
      description,
      category_id: categoryId,
      phone,
      address_text: address,
      owner_phone: ownerPhone,
      owner_name: ownerName,
      owner_password: ownerPass,
      sales_rep_code: repCode,
      lat,
      lng,
      commission_percent: Number(commission) || 0,
      ...(logoID !== null ? { logo_media_id: logoID } : {}),
    };
    try {
      if (merchant) {
        await api(`/api/v1/admin/merchants/${merchant.id}`, {
          method: "PATCH",
          body: JSON.stringify(body),
        });
      } else {
        await api("/api/v1/admin/merchants", { method: "POST", body: JSON.stringify(body) });
      }
      onSaved();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      size="2xl"
      title={merchant ? m.admin.merchants.editTitle : m.admin.merchants.createTitle}
    >
      {/* ══════════════════════════════════════════════════════════════
          **عمودان لا عمود** — والنموذجُ يُرى كلُّه بلا سكرول
          ══════════════════════════════════════════════════════════════

          (شكوى المالك ٢٠٢٦-٠٨-٠٨: «تعديل المتجر فورم طول بسكرول مزعج».)

          **وقيس فوجد أنّه يَسكرُل في كلّ مقاس** لا في نافذته وحدها: يحتاج
          ١١٦٤ بكسلاً، **والمتاح ٧٣٥ في نافذته و٩٧٠ في شاشة 1080p كاملة.**

          **والسببُ أنّ الأقسامَ الثلاثةَ تتراصّ طولاً** بينما نصفُ الشاشة
          فارغٌ عرضاً — النافذةُ كانت ٨٩٦ بكسلاً على شاشةٍ ضعفَ ذلك.

          **فالبياناتُ والحساباتُ في عمود، والموقعُ في عمود** — والخريطةُ
          هي أطولُ ما في النموذج فتقف وحدها بدل أن تُضاف إلى الطول. */}
      <form onSubmit={submit} className="space-y-6">
        <div className="grid grid-cols-1 items-start gap-5 lg:grid-cols-[3fr_2fr]">
        <div className="space-y-5">
        {/* القسم 1: بيانات المتجر */}
        <FormSection title={m.admin.merchants.sectionAccounts} icon={<IconUser />}>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              {/* **ولا متجرَ بلا صاحب** — (قرارُ المالك ٢٠٢٦-٠٨-١٥).

                  **ومتجرٌ بلا صاحبٍ لا يفتح بوّابتَه أحد**: لا يقبل
                  طلباً ولا يحضّره، **وطلبٌ يُسنَد إليه يقف** — ولا
                  يظهر في أيّ شاشةٍ أنّ السببَ حسابٌ ناقص. */}
              <Input
                id="m-owner"
                label={m.admin.merchants.ownerPhone}
                required
                dir="ltr"
                value={ownerPhone}
                onChange={(e) => setOwnerPhone(e.target.value)}
                className="text-end"
                placeholder="09xxxxxxxx"
              />
              <p className="mt-1 text-xs text-ink-muted">{m.admin.merchants.ownerHint}</p>
            </div>
            <div>
              <Input
                id="m-owner-name"
                label={m.admin.merchants.ownerName}
                value={ownerName}
                onChange={(e) => setOwnerName(e.target.value)}
              />
            </div>
            {/* **ولا كلمةَ لحسابٍ قائم** — من كان في المنصّة يدخل
                بكلمته التي يعرفها، **وتبديلُها من نافذة متجرٍ يُوقفه
                على بابه ولا يعرف لماذا.** */}
            {!merchant && (
              <>
                <div>
                  <Input
                    id="m-owner-pass"
                    label={m.admin.merchants.ownerPassword}
                    type="password"
                    required
                    value={ownerPass}
                    onChange={(e) => setOwnerPass(e.target.value)}
                  />
                  <p className="mt-1 text-xs text-ink-muted">
                    {m.admin.merchants.ownerPasswordHint}
                  </p>
                </div>
                <div>
                  <Input
                    id="m-owner-pass2"
                    label={m.admin.merchants.ownerPasswordConfirm}
                    type="password"
                    required
                    value={ownerPass2}
                    onChange={(e) => setOwnerPass2(e.target.value)}
                  />
                </div>
              </>
            )}
            <div>
              <Input
                id="m-rep"
                label={m.admin.merchants.repCode}
                dir="ltr"
                value={repCode}
                onChange={(e) => setRepCode(e.target.value.toUpperCase())}
                className="text-center font-mono uppercase tracking-widest"
                placeholder="RH-XXXXX"
              />
              <p className="mt-1 text-xs text-ink-muted">{m.admin.merchants.repCodeHint}</p>
            </div>
          </div>
        </FormSection>

        {/* القسم 2: الحسابات المرتبطة */}
        {/* القسم 1: بيانات المتجر */}
        <FormSection title={m.admin.merchants.sectionInfo} icon={<IconStore />}>
          {/* **وثلاثةُ أعمدةٍ على الشاشة الواسعة** — ستّةُ حقولٍ في صفّين لا
              ثلاثة. **وحقلُ الشعار هو أطولُ صفٍّ** فبقاؤه في صفٍّ ثالثٍ وحده
              كان يضيف ١١٠ بكسلاً إلى الطول بلا حاجة. */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <Input
              id="m-name"
              label={m.admin.merchants.name}
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
            <Select
              id="m-cat"
              label={m.admin.merchants.category}
              value={categoryId}
              onChange={(e) => setCategoryId(e.target.value)}
            >
              {cats.map((c) => (
                <option key={c.id} value={c.id}>
                  <CategoryIcon name={c.icon} size={15} />
                {c.name}
                </option>
              ))}
            </Select>
            <Input
              id="m-phone"
              label={m.admin.merchants.merchantPhone}
              icon={<IconPhone />}
              dir="ltr"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              className="text-end"
            />
            <Input
              id="m-desc"
              label={m.admin.merchants.descriptionField}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
            <Input
              id="m-commission"
              label={m.admin.merchants.commission}
              type="number"
              min="0"
              max="100"
              value={commission}
              onChange={(e) => setCommission(e.target.value)}
            />
            <ImageUpload
              kind="merchant_logo"
              label={m.admin.merchants.logo}
              initialUrl={merchant?.logo_thumb_url}
              onChange={setLogoID}
            />
          </div>
        </FormSection>
        </div>

        {/* القسم 3: الموقع — عمودٌ قائمٌ بذاته لأنّ الخريطةَ أطولُ ما فيه */}
        <FormSection title={m.admin.merchants.sectionLocation} icon={<IconLocation />}>
          <div className="space-y-3">
            <Input
              id="m-address"
              label={m.admin.merchants.address}
              value={address}
              onChange={(e) => setAddress(e.target.value)}
              placeholder={m.admin.merchants.addressPlaceholder}
            />
            {/* **وارتفاعُ الخريطة قُصَّ إلى ٢٠٨** — صارت أطولَ الأعمدة بعد
                أن قصُر عمودُ البيانات، **وهي وحدَها ما بقي يُطيل النموذج.**
                والنقرُ على مربّعٍ بهذا الحجم كافٍ لتحديد نقطةٍ في مدينة. */}
            <div className="overflow-hidden rounded-control border border-line">
              <PickMap
                lat={lat}
                lng={lng}
                height="h-52"
                onPick={(la, ln) => {
                  setLat(la);
                  setLng(ln);
                }}
              />
            </div>
            {/* **والتنبيهُ تحت الخريطة لا فوقها** — من رآها عرف أنّها تُنقر،
                ومن لم يعرف قرأ السطرَ تحتها. **وصندوقٌ ملوّنٌ فوق الخريطة
                يأخذ مكانَها ويؤخّرها.** */}
            <div className="flex flex-wrap items-center justify-between gap-2">
              <p className="text-xs leading-relaxed text-ink-muted">
                {m.admin.merchants.locationHint}
              </p>
              <Badge variant={lat != null ? "success" : "warning"}>
                {lat != null ? m.admin.merchants.locationSet : m.admin.merchants.locationUnset}
              </Badge>
            </div>
          </div>
        </FormSection>
        </div>

        {error && (
          <Alert>{error}</Alert>
        )}
        <div className="flex justify-end gap-2 border-t border-line-soft pt-4">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          {/* ══════════════════════════════════════════════════════════
              **ولا يُحفظ متجرٌ بلا دبّوس**
              ══════════════════════════════════════════════════════════

              (قرارُ المالك ٢٠٢٦-٠٨-١٢: «نسوي دبّوس المتجر إلزامي مو
               اختياري عند فتح الحساب».)

              **وكانت الخريطةُ تُعرض ولا تُلزم** — فيُحفظ المتجرُ بلا
              موضع، **ويظهر للسائق بلا مسافة** ولا نقطةٍ يمشي إليها.

              **والحارسُ في المحرّك أيضاً** (`ErrLocationRequired`) — هذا
              يمنع الضغطة، **وذاك يمنع الالتفاف من أيّ واجهةٍ أخرى.** */}
          <Button type="submit" disabled={busy || lat == null || lng == null}>
            {m.common.save}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

export function CategoriesModal({
  open,
  categories,
  onClose,
  onChanged,
}: {
  open: boolean;
  /** **وفارغةٌ تعني «اجلبها بنفسك»** — فتُفتح من أيّ شاشة. */
  categories?: Category[];
  onClose: () => void;
  onChanged?: () => Promise<void> | void;
}) {
  // ══════════════════════════════════════════════════════════════════
  // **والقائمةُ تُجلب هنا إن لم تصل**
  // ══════════════════════════════════════════════════════════════════
  //
  // (قرارُ المالك ٢٠٢٦-٠٨-١٥: زرُّ إدارة التصنيفات يصعد إلى أعلى
  //  الحسابات — **وهو من شؤون المنصّة لا من شؤون متجر.**)
  //
  // **ونافذةٌ تعتمد على من ناداها تُفتح فارغةً من الباب الثاني.**
  const [rows, setRows] = useState<Category[]>(categories ?? []);
  const [loadErr, setLoadErr] = useState("");
  const reload = useCallback(async () => {
    try {
      // **والنقطةُ تردّ مصفوفةً لا كائناً** — **وقراءتُها `r.categories`
      // تُعطي `undefined` فتُقرأ قائمةً فارغةً بلا خطأ**، فيرى فاتحُها
      // «لا تصنيفات» وهي في القاعدة. (قِيس ٢٠٢٦-٠٨-١٥.)
      setRows(await api<Category[]>("/api/v1/admin/categories"));
    } catch (e) {
      setLoadErr(errText(e));
    }
  }, []);
  useEffect(() => {
    if (!open) return;
    if ((categories?.length ?? 0) > 0) {
      setRows(categories ?? []);
      return;
    }
    void reload();
  }, [open, categories, reload]);

  const [name, setName] = useState("");
  const [icon, setIcon] = useState<CategoryIconKey>("other");
  const [error, setError] = useState("");

  async function addCategory(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await api("/api/v1/admin/categories", {
        method: "POST",
        body: JSON.stringify({ name, icon, sort_order: rows.length + 1 }),
      });
      setName("");
      setIcon("other");
      await reload();
      await onChanged?.();
    } catch (err) {
      setError(errText(err));
    }
  }

  async function toggleActive(c: Category) {
    setError("");
    try {
      await api(`/api/v1/admin/categories/${c.id}`, {
        method: "PATCH",
        body: JSON.stringify({ active: !c.active }),
      });
      await reload();
      await onChanged?.();
    } catch (err) {
      setError(errText(err));
    }
  }

  return (
    // ══════════════════════════════════════════════════════════════════
    // **وعريضةٌ لا طويلةٌ بمرّاح**
    // ══════════════════════════════════════════════════════════════════
    //
    // (شكوى المالك ٢٠٢٦-٠٨-١٥ — وهي شكواه في نموذج المتجر ٢٠٢٦-٠٨-٠٨:
    //  «فورم طولٌ بسكرول مزعج».)
    //
    // **والطولُ يُسكرَل والعرضُ لا**: عشرون تصنيفاً في عمودٍ واحدٍ
    // تعني مرّاحاً في كلّ فتحة، **وثلاثةُ أعمدةٍ تُريها كلَّها في
    // نظرة.**
    <Modal open={open} onClose={onClose} size="2xl" title={m.admin.merchants.categoriesTitle}>
      {/* **وفشلُ الجلب يُقال** — **وقائمةٌ فارغةٌ تُقرأ «لا تصنيفاتِ في
          المنصّة» وهي في الحقيقة نداءٌ سقط.** */}
      {loadErr && <Alert>{loadErr}</Alert>}
      <ul className="mb-4 grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
        {rows.map((c) => (
          <li
            key={c.id}
            className="flex items-center justify-between rounded-control border border-line px-3 py-2"
          >
            {/* **والأيقونةُ بجانب الاسم لا فوقه** — كانتا في `span`
                بلا صفّ، **فنزل الاسمُ سطراً ثانياً** فارتفع الصفُّ
                ضِعفاً بلا سبب. */}
            <span
              className={`flex min-w-0 items-center gap-2 ${
                c.active ? "" : "text-ink-muted line-through"
              }`}
            >
              <CategoryIcon name={c.icon} size={15} />
              <span className="truncate">{c.name}</span>
            </span>
            <Button variant={c.active ? "danger" : "secondary"} onClick={() => toggleActive(c)}>
              {c.active ? m.admin.merchants.deactivate : m.admin.merchants.activate}
            </Button>
          </li>
        ))}
      </ul>
      {/* **والإضافةُ صفٌّ في شاشةٍ عريضة** — الاسمُ والأيقونةُ والزرُّ
          في سطرٍ واحد، **وعمودٌ منها يُطيل النافذةَ بلا داعٍ.** */}
      <form onSubmit={addCategory} className="space-y-3 border-t border-line-soft pt-4">
        <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-[1fr_2fr_auto]">
          <Input
            id="cat-name"
            label={m.admin.merchants.categoryName}
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <CategoryIconPicker
            label={m.admin.merchants.categoryIcon}
            value={icon}
            onChange={setIcon}
          />
          <Button type="submit" className="flex items-center gap-1 lg:mt-7">
            <IconAdd size={15} />
            {m.admin.merchants.addCategory}
          </Button>
        </div>
      </form>
      {error && (
        <Alert className="mt-3">{error}</Alert>
      )}
    </Modal>
  );
}
