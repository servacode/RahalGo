"use client";

/**
 * **صفحةُ القسم — بضاعةُ المتاجر مجموعةً في بابٍ واحد.**
 *
 * # لماذا صفحةٌ لا نافذة
 *
 * قرارُ المالك (٢٠٢٦-٠٨-٠٤): «عرضُ القسم يجب أن يفتح صفحةً منفصلة وليس نافذةً
 * منبثقة».
 *
 * **والنافذةُ تُخفي ما خلفها**: من يقارن قسمين يفتح واحدةً ويُغلقها ليفتح
 * الأخرى. **ولا تُشارَك برابط**: من أراد أن يقول «انظر هذا القسم» لا يملك
 * عنواناً يرسله. **ولا يعود إليها زرُّ الرجوع** — يُغلق الصفحةَ كلَّها.
 *
 * **وقائمةٌ قد تبلغ خمسمئةِ صنفٍ ليست محتوى نافذة.**
 *
 * # ولماذا «كما هي»
 *
 * نقطةُ التصفّح العامّة تُرشِّح: متجرٌ فعّالٌ وقسمٌ فعّالٌ وصنفٌ مُقَرّ. **وهي
 * الصواب للزبون وخطأٌ للإدارة**: من يفتح قسماً ليقرّر إطفاءَه يريد ما فيه
 * كلَّه — **بما لا يظهر ولماذا لا يظهر.**
 *
 * # والصنفُ يأتي من متجر — من البابين
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «كلُّ شيءٍ يُضاف بالمتاجر يجب أن يأتي إلى هنا
 * تلقائياً، أو من هنا نضيف عنصراً جديداً ونحدّد هو تابعٌ لأيّ متجر».)
 *
 * **ولا صنفَ تملكه المنصة**: كلُّ صفٍّ هنا له `merchant_id`، **وإضافةٌ من هذه
 * الصفحة تسأل عن المتجر أوّلاً** — لا لتزيد خطوة، **بل لأنّ صنفاً بلا مصدرٍ
 * لا يُطبخ ولا يُحاسَب عليه أحد.**
 */

import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  Input,
  Select,
  Modal,
  PageContainer,
  Pagination,
  PageHeader,
  EmptyState,
  LoadingState,
  StatGrid,
  StatCard,
  IconStore,
  IconPrev,
  IconSearch,
  IconOrder,
  IconWarning,
  IconCamera,
} from "@rahalgo/ui";
import { api, ApiError, mediaUrl } from "@/lib/api";
import ImageUpload from "@/components/admin/ImageUpload";

const m = getMessages(defaultLocale);
const S = m.admin.sections;

interface Section {
  id: string;
  name: string;
  active: boolean;
  items: number;
  image_url: string | null;
  image_thumb_url: string | null;
}

interface SectionItem {
  id: string;
  name: string;
  /** **سعرُ الشراء** — ما وضعه المتجر، وأصلُ الحسبتين. */
  merchant_price: number;
  available: boolean;
  approved: boolean;
  merchant_name: string;
  merchant_status: string;
  thumb_url: string | null;
  image_url: string | null;
  /**
   * **الحسبتان — واحدةٌ تنزل وأخرى تصعد.**
   *
   *	عمولةُ المنصة  ←  تُقتطع من المتجر   ←  `merchant_net` ما يقبضه
   *	هامشُ المنصة   ←  يُضاف على الزبون   ←  `sale_price`  ما يدفعه
   *
   * **ولا تُجمعان في رقمٍ واحد**: من رأى «ربحُ المنصة ٣٬٦٠٠» لا يعرف أيُّهما
   * يُعدَّل حين يشتكي المتجرُ أو يشتكي الزبون. **وربحُ المنصة مجموعُهما.**
   */
  commission_percent: number;
  commission: number;
  merchant_net: number;
  /** هامشُ الصنف النافذ بالليرة — **بعد الوراثة**. */
  margin_value: number;
  margin: number;
  sale_price: number;
  margin_override: number | null;
}

/**
 * **حالُ الصنف — وسببُ ظهوره أو غيابه.**
 *
 * **والترتيبُ مقصود**: يُقرأ أوّلُ سببٍ يمنع الظهور. صنفٌ غيرُ مُقَرٍّ ومتجرُه
 * مُطفأٌ **لا يُقال عنه «متجرُه مُطفأ»** — المراجعةُ أوّلُ بابٍ يجب أن يُفتح.
 */
function itemState(it: SectionItem): { label: string; variant: "success" | "warning" | "danger" } {
  if (!it.approved) return { label: S.itemPending, variant: "warning" };
  if (it.merchant_status !== "active") return { label: S.itemStoreOff, variant: "danger" };
  if (!it.available) return { label: S.itemOut, variant: "warning" };
  return { label: S.itemLive, variant: "success" };
}

export default function SectionPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [sec, setSec] = useState<Section | null>(null);
  const [rows, setRows] = useState<SectionItem[] | null>(null);
  /** **صفحةُ أصناف القسم** — (قرارُ المالك ٢٠٢٦-٠٨-١٠).

      **والقسمُ يجمع أصنافَ كلّ المتاجر** — ينمو بعدد المتاجر لا بعدد
      الأقسام. **وخمسُمئةٍ صامتةٌ تعني أنّ صنفاً لا يُوافَق عليه لأنّ أحداً
      لم يره.** */
  const [page, setPage] = useState(1);
  const [count, setCount] = useState(0);
  const [perPage, setPerPage] = useState(50);
  const [q, setQ] = useState("");
  const [state, setState] = useState("");
  const [adding, setAdding] = useState(false);
  const [editing, setEditing] = useState<SectionItem | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      // **القسمُ من قائمته** — والقائمةُ قصيرةٌ دائماً (عشرةُ أقسامٍ أو نحوها)،
      // **ونقطةٌ ثانيةٌ لصفٍّ واحدٍ سطحٌ يُصان بلا حاجة.**
      const list = await api<{ sections: Section[] }>("/api/v1/admin/sections");
      setSec((list.sections ?? []).find((x) => x.id === id) ?? null);
      const res = await api<{ items: SectionItem[]; count: number; per_page: number }>(
        `/api/v1/admin/sections/${id}/items?page=${page}`,
      );
      setRows(res.items ?? []);
      setCount(res.count ?? 0);
      setPerPage(res.per_page || 50);
      setError("");
    } catch {
      setError(m.errors.internal);
    }
  }, [id, page]);

  /**
   * **قلبُ الإتاحة — زرٌّ واحدٌ يقول الحال.**
   *
   * **ولا يُخفى حين يكون المتجرُ مُطفأً**: «نفد» قرارُ مطبخٍ في صنفه،
   * **وإطفاءُ المتجر حالٌ أخرى** — ومن خلطهما وجد صنفاً لا يستطيع إعادتَه
   * حتى يُفتح متجرُه.
   */
  async function toggle(it: SectionItem) {
    await api(`/api/v1/admin/menu/items/${it.id}`, {
      method: "PATCH",
      body: JSON.stringify({ available: !it.available }),
    });
    await load();
  }

  useEffect(() => {
    void load();
  }, [load]);

  if (error) return <p className="py-10 text-center text-danger">{error}</p>;
  if (!rows || !sec) return <LoadingState />;

  const term = q.trim();
  const shown = rows.filter(
    (it) =>
      (term === "" || it.name.includes(term) || it.merchant_name.includes(term)) &&
      (state === "" || itemState(it).label === state),
  );
  const live = rows.filter((it) => itemState(it).variant === "success").length;

  return (
    <PageContainer width="full">
      <button
        onClick={() => router.push("/dashboard/sections")}
        className="mb-3 flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink"
      >
        <IconPrev size={15} />
        {S.backToMarket}
      </button>

      <div className="mb-4 flex flex-wrap items-start gap-3">
        {/* **صورةُ القسم في ترويسته** — هي ما يعرفه بها من يفتحها. */}
        <span className="h-20 w-28 shrink-0 overflow-hidden rounded-card bg-field">
          {sec.image_url || sec.image_thumb_url ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={mediaUrl(sec.image_url ?? sec.image_thumb_url) ?? ""}
              alt={sec.name}
              className="h-full w-full object-cover"
            />
          ) : (
            <span className="flex h-full items-center justify-center">
              <IconStore size={22} className="text-ink-muted" />
            </span>
          )}
        </span>
        <div className="min-w-0 flex-1">
          <PageHeader icon={IconStore} title={sec.name} />
        </div>
        <Badge variant={sec.active ? "success" : "neutral"}>
          {sec.active ? S.available : S.unavailable}
        </Badge>
        <Button onClick={() => setAdding(true)}>{S.addItem}</Button>
      </div>

      <StatGrid>
        <StatCard label={S.statAll} value={fmtNum(rows.length)} icon={IconOrder} />
        {/* **والمعروضُ فعلاً لا المسجَّل** — قسمٌ فيه اثنا عشر ويُعرض منه ثلاثةٌ
            حالةٌ تُعالَج، **ورقمٌ واحدٌ يخفيها.** */}
        <StatCard label={S.statLive} value={fmtNum(live)} icon={IconStore} />
        <StatCard label={S.statHidden} value={fmtNum(rows.length - live)} icon={IconWarning} />
      </StatGrid>

      <div className="mb-4 mt-4 flex flex-wrap items-end gap-3">
        <div className="w-64">
          <Input
            id="sec-q"
            icon={<IconSearch />}
            placeholder={S.searchItems}
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
        </div>
        <div className="w-48">
          <Select value={state} onChange={(e) => setState(e.target.value)}>
            <option value="">{S.allStates}</option>
            {[S.itemLive, S.itemPending, S.itemOut, S.itemStoreOff].map((x) => (
              <option key={x} value={x}>
                {x}
              </option>
            ))}
          </Select>
        </div>
      </div>

      {/* **بطاقةٌ لكلّ صنفٍ بصورته — كبطاقة القسم.**
          (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «لازم يكون كرت يحوي على صورة لكلّ طعام
          واسمِ المتجر الخاصّ بالصنف والسعر ومعروضٌ أو لا».)

          **والشكلُ واحدٌ في الشاشتين**: من يمسح السوقَ بالعين ثمّ يفتح قسماً
          **لا يُعيد تعلُّمَ أين يقع كلُّ شيء.** */}
      {shown.length === 0 ? (
        <EmptyState icon={IconStore} title={S.noItems} />
      ) : (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
          {shown.map((it) => {
            const st = itemState(it);
            return (
              <div
                key={it.id}
                className={`flex flex-col overflow-hidden surface transition-shadow hover:elev-2 ${
                  st.variant === "success" ? "" : "opacity-70"
                }`}
              >
                <div className="relative flex aspect-[4/3] items-center justify-center bg-field">
                  {it.image_url || it.thumb_url ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={mediaUrl(it.image_url ?? it.thumb_url) ?? ""}
                      alt={it.name}
                      className="h-full w-full object-cover"
                    />
                  ) : (
                    /* **ونقصُ الصورة يُقال لا يُملأ برمز** — رمزٌ رماديٌّ يجعل
                       الصنفَ يبدو تامّاً وهو ناقص، فيُنسى أنّ صورتَه لم تُرفع. */
                    <span className="flex flex-col items-center gap-1 text-xs text-ink-muted">
                      <IconCamera size={18} />
                      {S.noImage}
                    </span>
                  )}
                  {/* **الحالُ فوق الصورة** — تُقرأ قبل الاسم، وهي أوّلُ ما يُسأل عنه. */}
                  <span className="absolute end-1.5 top-1.5">
                    <Badge variant={st.variant}>{st.label}</Badge>
                  </span>
                </div>

                <div className="flex flex-1 flex-col gap-2 p-3">
                  <div>
                    <p className="truncate font-bold">{it.name}</p>
                    {/* **واسمُ المتجر يبقى هنا وحدَه** — شاشتُنا لا شاشةُ الزبون.
                        السوقُ يُخفي المصدر عن الزبون، **والإدارةُ لا تُدير ما
                        لا ترى مصدرَه.** */}
                    <p className="truncate text-xs text-ink-muted">{it.merchant_name}</p>
                  </div>

                  {/* **الحسبتان مفصولتان — واحدةٌ تنزل وأخرى تصعد.**

                      (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «عمولة المنصة والسعر بعد
                      العمولة · هامش الربح والسعر بعد الهامش».)

                      **والعمولةُ تُقتطع من المتجر ولا تُضاف على الزبون**: هي
                      حصّتُنا من سعر الشراء. **والهامشُ يُضاف فوقه** فيصير سعرَ
                      البيع. **ولا تُجمعان في رقمٍ واحد**: من رأى «ربحُ المنصة»
                      مجموعاً لا يعرف أيَّهما يُعدّل حين يشتكي أحدُ الطرفين. */}
                  <dl className="space-y-1 border-t border-line-soft pt-2 text-xs tabular-nums">
                    <Row label={S.buyPrice} value={fmtNum(it.merchant_price)} />
                    <Row
                      label={S.commission.replace("{n}", fmtNum(it.commission_percent))}
                      value={"−" + fmtNum(it.commission)}
                      tone="muted"
                    />
                    <Row label={S.afterCommission} value={fmtNum(it.merchant_net)} strong />
                    <Row
                      label={S.marginFixed}
                      value={"+" + fmtNum(it.margin)}
                      tone="muted"
                    />
                    <Row label={S.afterMargin} value={fmtNum(it.sale_price)} strong />
                  </dl>

                  <div className="mt-auto flex gap-1.5 pt-1 [&_button]:flex-1 [&_button]:!px-2 [&_button]:text-xs">
                    <Button variant="secondary" onClick={() => setEditing(it)}>
                      {m.common.edit}
                    </Button>
                    {/* **زرٌّ واحدٌ يقلب الحال** — كزرّ القسم، لا زرّان أحدُهما
                        معطّلٌ دائماً. */}
                    <Button
                      variant={it.available ? "ghost" : "primary"}
                      onClick={() => void toggle(it)}
                    >
                      {it.available ? S.makeUnavailable : S.makeAvailable}
                    </Button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {adding && (
        <AddItemModal
          sectionID={id}
          sectionName={sec.name}
          onClose={() => setAdding(false)}
          onSaved={() => {
            setAdding(false);
            void load();
          }}
        />
      )}
      {editing && (
        <EditItemModal
          item={editing}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            void load();
          }}
        />
      )}
      {/* **والترقيمُ من المكوّن المشترك** — ولا يظهر لصفحةٍ واحدة. */}
      {count > perPage && (
        <div className="mt-4 flex justify-center">
          <Pagination page={page} total={count} perPage={perPage} onChange={setPage} />
        </div>
      )}
    </PageContainer>
  );
}

/** سطرُ حسبة — **اللفظُ يميناً والرقمُ يساراً**، فالعينُ تمسح عموداً واحداً. */
function Row({
  label,
  value,
  strong,
  tone,
}: {
  label: string;
  value: string;
  strong?: boolean;
  tone?: "muted";
}) {
  return (
    <div className="flex items-baseline justify-between gap-2">
      <dt className={tone === "muted" ? "text-ink-muted" : ""}>{label}</dt>
      <dd dir="ltr" className={strong ? "font-bold" : tone === "muted" ? "text-ink-muted" : ""}>
        {value}
      </dd>
    </div>
  );
}

interface MerchantRow {
  id: string;
  name: string;
}
interface MenuSectionRow {
  id: string;
  name: string;
}

/**
 * **إضافةُ صنفٍ من بابِ السوق — والمتجرُ يُسأل عنه أوّلاً.**
 *
 * # ولماذا قسمُ المتجر أيضاً
 *
 * الصنفُ يعيش في قائمة متجره (`section_id`) **ويُعرض في قسم السوق**
 * (`platform_section_id`) — **موضعان لا واحد**: الأوّلُ ترتيبُ المطبخ يراه
 * صاحبُه، والثاني بابُ الزبون. **ولا يُنشأ صنفٌ بلا موضعٍ في قائمة صاحبه**:
 * يصير يتيماً لا يجده من يملكه.
 *
 * **وقسمُ السوق مثبَّتٌ هنا** — هو الصفحةُ التي فُتح منها، فلا يُسأل عنه.
 */
function AddItemModal({
  sectionID,
  sectionName,
  onClose,
  onSaved,
}: {
  sectionID: string;
  sectionName: string;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [merchants, setMerchants] = useState<MerchantRow[]>([]);
  const [merchantID, setMerchantID] = useState("");
  const [menuSections, setMenuSections] = useState<MenuSectionRow[] | null>(null);
  const [menuSectionID, setMenuSectionID] = useState("");
  const [name, setName] = useState("");
  const [price, setPrice] = useState("");
  const [imageID, setImageID] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    api<{ merchants: MerchantRow[] }>("/api/v1/admin/merchants")
      .then((r) => setMerchants(r.merchants ?? []))
      .catch(() => setError(m.errors.internal));
  }, []);

  // **وقائمةُ المتجر تُقرأ حين يُختار** — لا قبله: أقسامُ متجرٍ لم يُختر
  // بعدُ لا معنى لها، **وقراءةُ قوائم كلّ المتاجر سلفاً حملٌ بلا سبب.**
  useEffect(() => {
    if (!merchantID) {
      setMenuSections(null);
      setMenuSectionID("");
      return;
    }
    setMenuSections(null);
    api<MenuSectionRow[]>(`/api/v1/admin/merchants/${merchantID}/menu`)
      .then((r) => {
        const list = r ?? [];
        setMenuSections(list);
        setMenuSectionID(list[0]?.id ?? "");
      })
      .catch(() => setError(m.errors.internal));
  }, [merchantID]);

  async function submit() {
    if (!merchantID) return setError(S.pickMerchant);
    if (!menuSectionID) return setError(S.pickMenuSection);
    if (!name.trim()) return setError(S.itemNameRequired);
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/merchants/${merchantID}/menu/items`, {
        method: "POST",
        body: JSON.stringify({
          section_id: menuSectionID,
          name: name.trim(),
          price: Number(price) || 0,
          platform_section_id: sectionID,
          ...(imageID !== null ? { image_media_id: imageID } : {}),
        }),
      });
      onSaved();
    } catch (err) {
      const key = err instanceof ApiError ? (err.body.message_key.split(".").pop() ?? "") : "";
      setError((m.errors as Record<string, string>)[key] ?? m.errors.internal);
      setBusy(false);
    }
  }

  return (
    <Modal open title={S.addItemTo.replace("{s}", sectionName)} onClose={onClose}>
      <div className="space-y-3">
        <Select
          label={S.merchant}
          value={merchantID}
          onChange={(e) => setMerchantID(e.target.value)}
        >
          <option value="">{S.pickMerchant}</option>
          {merchants.map((x) => (
            <option key={x.id} value={x.id}>
              {x.name}
            </option>
          ))}
        </Select>

        {/* **وقسمُ المطبخ لا يُسأل عنه قبل المتجر** — حقلٌ فارغٌ معطَّلٌ يُقرأ
            عطباً، **وغيابُه يقول «اختر المتجر أوّلاً» بلا كلمة.** */}
        {merchantID &&
          (menuSections === null ? (
            <LoadingState variant="inline" />
          ) : menuSections.length === 0 ? (
            <p className="text-sm text-danger">{S.merchantHasNoSections}</p>
          ) : (
            <Select
              label={S.menuSection}
              value={menuSectionID}
              onChange={(e) => setMenuSectionID(e.target.value)}
            >
              {menuSections.map((x) => (
                <option key={x.id} value={x.id}>
                  {x.name}
                </option>
              ))}
            </Select>
          ))}

        <ImageUpload kind="menu_item" label={S.itemImage} onChange={setImageID} />
        <Input label={S.itemName} value={name} onChange={(e) => setName(e.target.value)} />
        {/* **سعرُ الشراء لا سعرُ البيع** — ما يقبضه المتجر، **والمنصةُ تحسب
            الهامشَ فوقه.** واللفظُ يقول أيَّهما، فـ«السعر» وحدَها تحتمل الاثنين. */}
        <Input
          label={S.itemPrice}
          type="number"
          value={price}
          onChange={(e) => setPrice(e.target.value)}
        />
        {error && <p className="text-sm text-danger">{error}</p>}
        <div className="flex gap-2">
          <Button disabled={busy} onClick={submit}>
            {m.common.save}
          </Button>
          <Button variant="ghost" onClick={onClose}>
            {m.common.cancel}
          </Button>
        </div>
      </div>
    </Modal>
  );
}

/**
 * **تعديلُ الصنف من بابِ السوق.**
 *
 * # وما يُعدَّل هنا وما لا يُعدَّل
 *
 * **سعرُ الشراء وهامشُ الصنف** — لأنّهما الرقمان اللذان تعرضهما البطاقة،
 * **وعرضُ رقمٍ لا يُمسّ من موضعه يجعل الشاشةَ تقريراً لا لوحةَ تحكّم.**
 *
 * **والعمولةُ لا تُعدَّل هنا**: هي على المتجر كلِّه لا على صنفه — **ورقمٌ
 * يُغيَّر من شاشة صنفٍ ويقع على مئة صنفٍ آخر خللٌ ينتظر.** موضعُها صفحةُ
 * المتجر.
 *
 * **والمتجرُ لا يُنقل**: صنفٌ يُنقل من متجرٍ إلى متجرٍ يترك طلباتِ الأمس تشير
 * إلى مصدرٍ لم يحضّرها. **من أراد صنفاً عند غيره أنشأه عنده.**
 */
function EditItemModal({
  item,
  onClose,
  onSaved,
}: {
  item: SectionItem;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(item.name);
  const [price, setPrice] = useState(String(item.merchant_price));
  /**
   * **تجاوزُ هامش الصنف — والفراغُ «اتبع قسمَك» لا «بلا هامش».**
   *
   * الصفرُ قرارٌ («لا هامشَ على هذا») والفراغُ غيابُ قرار. **ومن خلط بينهما
   * جعل كلَّ صنفٍ لم يُلمس بلا هامش** — فبِيع كلُّ شيءٍ بسعر شرائه.
   */
  const [margin, setMargin] = useState(
    item.margin_override === null ? "" : String(item.margin_override),
  );
  const [imageID, setImageID] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit() {
    if (!name.trim()) return setError(S.itemNameRequired);
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/menu/items/${item.id}`, {
        method: "PATCH",
        body: JSON.stringify({
          name: name.trim(),
          price: Number(price) || 0,
          // **وسالبُ الواحد يمحو التجاوز** — هو ما يقرؤه الخادمُ «اتبع قسمَك»،
          // **وفراغٌ يُقرأ «بلا تغيير»** فلا يملك أحدٌ إعادةَ صنفٍ إلى وراثته.
          margin_override: margin.trim() === "" ? -1 : Number(margin) || 0,
          ...(imageID !== null ? { image_media_id: imageID } : {}),
        }),
      });
      onSaved();
    } catch (err) {
      const key = err instanceof ApiError ? (err.body.message_key.split(".").pop() ?? "") : "";
      setError((m.errors as Record<string, string>)[key] ?? m.errors.internal);
      setBusy(false);
    }
  }

  return (
    <Modal open title={S.editItem} onClose={onClose}>
      <div className="space-y-3">
        {/* **واسمُ المتجر يُقال ولا يُغيَّر** — من فتح النافذة يعرف على مالِ من
            يعمل، **وحقلٌ معطَّلٌ يُغري بمحاولةٍ تفشل.** */}
        <p className="text-sm text-ink-muted">
          {S.merchant}: <span className="font-medium text-ink">{item.merchant_name}</span>
        </p>
        <ImageUpload
          kind="menu_item"
          label={S.itemImage}
          initialUrl={item.image_url}
          onChange={setImageID}
        />
        <Input label={S.itemName} value={name} onChange={(e) => setName(e.target.value)} />
        <Input
          label={S.itemPrice}
          type="number"
          value={price}
          onChange={(e) => setPrice(e.target.value)}
        />
        <Input
          label={S.marginFieldFixed}
          type="number"
          placeholder={S.marginInheritHint}
          value={margin}
          onChange={(e) => setMargin(e.target.value)}
        />
        <p className="text-xs text-ink-muted">{S.marginHintItem}</p>
        {error && <p className="text-sm text-danger">{error}</p>}
        <div className="flex gap-2">
          <Button disabled={busy} onClick={submit}>
            {m.common.save}
          </Button>
          <Button variant="ghost" onClick={onClose}>
            {m.common.cancel}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
