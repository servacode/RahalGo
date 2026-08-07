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
import { useAuth } from "@/lib/auth";
import ImageUpload, { MediaThumb } from "@/components/ImageUpload";
import ViolationsModal from "@/components/ViolationsModal";

const m = getMessages(defaultLocale);

interface Category {
  id: string;
  name: string;
  icon: string;
  sort_order: number;
  active: boolean;
}

interface Merchant {
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

interface DayHours {
  day_of_week: number;
  closed: boolean;
  open_time: string;
  close_time: string;
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
  const [hoursFor, setHoursFor] = useState<Merchant | null>(null);
  /** **سجلُّ مخالفاتِ متجرٍ بعينه** — ومنه يُصدَر الإنذار. */
  const [violationsFor, setViolationsFor] = useState<Merchant | null>(null);
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

  async function toggleStatus(mr: Merchant) {
    try {
      await api(`/api/v1/admin/merchants/${mr.id}`, {
        method: "PATCH",
        body: JSON.stringify({ status: mr.status === "active" ? "inactive" : "active" }),
      });
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  /**
   * الحظرُ ورفعُه.
   *
   * **`suspended` لا `inactive`**: الثانيةُ يملكها المتجر — إجازةٌ أو ترميم —
   * ولو حُظر بها لرفع الحظرَ عن نفسه من بوابته.
   */
  async function suspend(mr: Merchant, on: boolean) {
    try {
      await api(`/api/v1/admin/merchants/${mr.id}/suspend`, {
        method: "POST",
        body: JSON.stringify({ suspended: on, note: "" }),
      });
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  /**
   * العفو — يُصفَّر العدّاد ولا يُمحى الماضي.
   *
   * **وهو منفصلٌ عن رفع الحظر عمداً**: رفعُ الحظر وحده يُعيده يعمل وعدّادُه
   * كما هو — «أعدناك على وعد». ودمجُهما يجعل كلَّ رفعِ حظرٍ عفواً، **فيتعلّم
   * المتجرُ أن الإلغاء بلا ثمن.**
   */
  async function forgive(mr: Merchant) {
    try {
      await api(`/api/v1/admin/merchants/${mr.id}/clear-violations`, { method: "POST" });
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  const totalPages = data ? Math.max(1, Math.ceil(data.total / data.per_page)) : 1;

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
            <span>{mr.category_icon}</span>
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
    {
      id: "ban",
      header: m.admin.merchants.banActions,
      cell: (mr) =>
        isAdmin ? (
          <div className="flex flex-wrap gap-1.5">
            <Button
              variant={mr.status === "suspended" ? "primary" : "secondary"}
              onClick={() => void suspend(mr, mr.status !== "suspended")}
            >
              {mr.status === "suspended"
                ? m.admin.merchants.unban
                : m.admin.merchants.ban}
            </Button>
            {/* **السجلُّ قبل الحكم.**

                كان زرُّ العفو وحدَه بجانب عدّادٍ مجرّد — **فيُعفى أو يُحظر بلا
                أن يُرى ما وقع.** (الثغرة `G-01`.) */}
            <Button variant="ghost" onClick={() => setViolationsFor(mr)}>
              {m.admin.merchants.violationsLog.viewLog}
            </Button>
            {mr.violations > 0 && (
              <Button variant="ghost" onClick={() => void forgive(mr)}>
                {m.admin.merchants.forgive}
              </Button>
            )}
          </div>
        ) : (
          <span className="text-ink-muted">—</span>
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
                <>
                  {/* **الملفُّ قبل القائمة.**

                      كان الزرُّ يفتح القائمةَ مباشرةً، **وكلُّ ما سواها في
                      نوافذَ منبثقة**: تُفتح واحدةً وتُغلق لتُفتح أخرى، ولا
                      تُرى صورةُ المتجر مجتمعة. **والقائمةُ صارت تبويباً فيه.** */}
                  <Button
                    variant="secondary"
                    onClick={() => router.push(`/dashboard/merchants/${mr.id}`)}
                    className="flex items-center gap-1.5"
                  >
                    <IconMenu size={15} />
                    {m.admin.merchants.openProfile}
                  </Button>
                  <Button
                    variant="ghost"
                    onClick={() => setHoursFor(mr)}
                    className="flex items-center gap-1.5"
                  >
                    <IconDate size={15} />
                    {m.admin.hours.manageHours}
                  </Button>
                  <Button
                    variant="ghost"
                    onClick={() => setEditing(mr)}
                    className="flex items-center gap-1.5"
                  >
                    <IconEdit size={15} />
                    {m.admin.merchants.edit}
                  </Button>
                  <Button
                    variant={mr.status === "active" ? "danger" : "secondary"}
                    onClick={() => toggleStatus(mr)}
                  >
                    {mr.status === "active"
                      ? m.admin.merchants.deactivate
                      : m.admin.merchants.activate}
                  </Button>
                </>
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
      {hoursFor && (
        <HoursModal
          merchant={hoursFor}
          onClose={() => setHoursFor(null)}
          onChanged={load}
        />
      )}
      {violationsFor && (
        <ViolationsModal
          merchant={violationsFor}
          onClose={() => setViolationsFor(null)}
          onChanged={load}
        />
      )}
    </div>
  );
}

function HoursModal({
  merchant,
  onClose,
  onChanged,
}: {
  merchant: Merchant;
  onClose: () => void;
  onChanged: () => Promise<void> | void;
}) {
  const [days, setDays] = useState<DayHours[] | null>(null);
  const [emergency, setEmergency] = useState(merchant.emergency_closed);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api<DayHours[]>(`/api/v1/admin/merchants/${merchant.id}/hours`)
      .then(setDays)
      .catch((err) => setError(errText(err)));
  }, [merchant.id]);

  function updateDay(i: number, patch: Partial<DayHours>) {
    setDays((ds) => ds && ds.map((d, di) => (di === i ? { ...d, ...patch } : d)));
  }

  async function save() {
    if (!days) return;
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/merchants/${merchant.id}/hours`, {
        method: "PUT",
        body: JSON.stringify({ days }),
      });
      if (emergency !== merchant.emergency_closed) {
        await api(`/api/v1/admin/merchants/${merchant.id}`, {
          method: "PATCH",
          body: JSON.stringify({ emergency_closed: emergency }),
        });
      }
      await onChanged();
      onClose();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={`${m.admin.hours.title}: ${merchant.name}`}>
      <Checkbox
        id="mr-hours-emergency"
        checked={emergency}
        onChange={(e) => setEmergency(e.target.checked)}
        label={
          <span className="block">
            <span className="block text-sm font-medium text-danger">
              {m.admin.hours.emergencyClose}
            </span>
            <span className="text-xs text-ink-muted">{m.admin.hours.emergencyHint}</span>
          </span>
        }
        className="mb-4 rounded-control border border-danger-edge bg-danger-tint px-3 py-2.5"
      />

      {!days ? (
        <LoadingState variant="inline" />
      ) : (
        <div className="space-y-2">
          {days.map((d, i) => (
            <div key={d.day_of_week} className="flex items-center gap-3 text-sm">
              <span className="w-16 shrink-0 font-medium">{m.admin.hours.days[i]}</span>
              <Checkbox
                id={`mr-hours-closed-${d.day_of_week}`}
                checked={d.closed}
                onChange={(e) => updateDay(i, { closed: e.target.checked })}
                label={m.admin.hours.closedDay}
                className="gap-1.5 text-ink-muted"
              />
              <input
                type="time"
                disabled={d.closed}
                value={d.open_time}
                onChange={(e) => updateDay(i, { open_time: e.target.value })}
                className="rounded-control border border-line px-2 py-1 disabled:opacity-40"
              />
              <IconPrev size={14} className="text-ink-muted" />
              <input
                type="time"
                disabled={d.closed}
                value={d.close_time}
                onChange={(e) => updateDay(i, { close_time: e.target.value })}
                className="rounded-control border border-line px-2 py-1 disabled:opacity-40"
              />
              {/* **دوامٌ يعبر منتصفَ الليل مقبولٌ ومُعلَن.**

                  ساعةُ إغلاقٍ أصغرُ من ساعة الفتح تعني «إلى ما بعد منتصف
                  الليل» — **ومطاعمُ الشاورما في الرقّة تعمل هكذا.** وبلا
                  هذه الكلمة يظنّها من يضبطها خطأً فيتراجع، **أو يضبطها
                  ولا يثق أنّها فُهمت.** */}
              {!d.closed && d.close_time <= d.open_time && (
                <span className="text-2xs text-ink-muted">{m.admin.hours.overnight}</span>
              )}
            </div>
          ))}
        </div>
      )}

      {error && (
        <Alert className="mt-3">{error}</Alert>
      )}
      <div className="mt-5 flex justify-end gap-2">
        <Button variant="secondary" onClick={onClose}>
          {m.common.cancel}
        </Button>
        <Button onClick={save} disabled={busy || !days}>
          {m.common.save}
        </Button>
      </div>
    </Modal>
  );
}

function MerchantModal({
  merchant,
  categories,
  onClose,
  onSaved,
}: {
  merchant: Merchant | null;
  categories: Category[];
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(merchant?.name ?? "");
  const [description, setDescription] = useState(merchant?.description ?? "");
  const [categoryId, setCategoryId] = useState(merchant?.category_id ?? categories[0]?.id ?? "");
  const [phone, setPhone] = useState(merchant?.phone ?? "");
  const [address, setAddress] = useState(merchant?.address_text ?? "");
  const [ownerPhone, setOwnerPhone] = useState(merchant?.owner_phone ?? "");
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
    setBusy(true);
    setError("");
    const body = {
      name,
      description,
      category_id: categoryId,
      phone,
      address_text: address,
      owner_phone: ownerPhone,
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
      size="xl"
      title={merchant ? m.admin.merchants.editTitle : m.admin.merchants.createTitle}
    >
      <form onSubmit={submit} className="space-y-6">
        {/* القسم 1: بيانات المتجر */}
        <FormSection title={m.admin.merchants.sectionInfo} icon={<IconStore />}>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
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
              {categories.map((c) => (
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

        {/* القسم 2: الموقع — العنوان والخريطة جنباً إلى جنب */}
        <FormSection title={m.admin.merchants.sectionLocation} icon={<IconLocation />}>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="space-y-3">
              <Input
                id="m-address"
                label={m.admin.merchants.address}
                value={address}
                onChange={(e) => setAddress(e.target.value)}
                placeholder={m.admin.merchants.addressPlaceholder}
              />
              <div className="rounded-control bg-field p-3 text-xs leading-relaxed text-ink-muted">
                {m.admin.merchants.locationHint}
              </div>
              <Badge variant={lat != null ? "success" : "warning"}>
                {lat != null ? m.admin.merchants.locationSet : m.admin.merchants.locationUnset}
              </Badge>
            </div>
            <div className="overflow-hidden rounded-control border border-line">
              <PickMap
                lat={lat}
                lng={lng}
                onPick={(la, ln) => {
                  setLat(la);
                  setLng(ln);
                }}
              />
            </div>
          </div>
        </FormSection>

        {/* القسم 3: الحسابات المرتبطة */}
        <FormSection title={m.admin.merchants.sectionAccounts} icon={<IconUser />}>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <Input
                id="m-owner"
                label={m.admin.merchants.ownerPhone}
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

        {error && (
          <Alert>{error}</Alert>
        )}
        <div className="flex justify-end gap-2 border-t border-line-soft pt-4">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button type="submit" disabled={busy}>
            {m.common.save}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

function CategoriesModal({
  open,
  categories,
  onClose,
  onChanged,
}: {
  open: boolean;
  categories: Category[];
  onClose: () => void;
  onChanged: () => Promise<void> | void;
}) {
  const [name, setName] = useState("");
  const [icon, setIcon] = useState<CategoryIconKey>("other");
  const [error, setError] = useState("");

  async function addCategory(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await api("/api/v1/admin/categories", {
        method: "POST",
        body: JSON.stringify({ name, icon, sort_order: categories.length + 1 }),
      });
      setName("");
      setIcon("other");
      await onChanged();
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
      await onChanged();
    } catch (err) {
      setError(errText(err));
    }
  }

  return (
    <Modal open={open} onClose={onClose} title={m.admin.merchants.categoriesTitle}>
      <ul className="mb-4 space-y-2">
        {categories.map((c) => (
          <li
            key={c.id}
            className="flex items-center justify-between rounded-control border border-line px-3 py-2"
          >
            <span className={c.active ? "" : "text-ink-muted line-through"}>
              <CategoryIcon name={c.icon} size={15} />
                {c.name}
            </span>
            <Button variant={c.active ? "danger" : "secondary"} onClick={() => toggleActive(c)}>
              {c.active ? m.admin.merchants.deactivate : m.admin.merchants.activate}
            </Button>
          </li>
        ))}
      </ul>
      <form onSubmit={addCategory} className="space-y-3">
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
        <Button type="submit" className="flex items-center gap-1">
          <IconAdd size={15} />
          {m.admin.merchants.addCategory}
        </Button>
      </form>
      {error && (
        <Alert className="mt-3">{error}</Alert>
      )}
    </Modal>
  );
}
