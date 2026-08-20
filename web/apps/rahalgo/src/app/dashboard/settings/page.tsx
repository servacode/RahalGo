"use client";

/**
 * الإعدادات — أخطر شاشة في المنصة.
 *
 * كانت محرّر JSON خاماً: اسمٌ لاتينيّ (`drivers.share_value`) وقيمةٌ في مربّع
 * نصٍّ حرّ. ويُطلب من صاحب المنصة — رجلٍ في الرقة لا مبرمج — أن يكتب JSON
 * صحيحاً في حقلٍ يحكم رواتب سائقيه. وحرفٌ زائد يكسر خطّ التوصيل كلَّه.
 *
 * والتعريف كلُّه من الخادم: نوع الحقل ومداه وخياراته. **المدى الذي يحرسه
 * الخادم هو المدى الذي يعرضه الحقل** — ولو كُتب هنا لانحرف عنه يوماً، فيرى
 * المالك حقلاً يقبل ما يرفضه الحفظ.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { getMessages, defaultLocale, fmtNum, errorText } from "@rahalgo/i18n";
import {
  Tabs,
  Alert,
  PageHeader, Button, Input, Textarea, Select, Switch, Badge, Card, EmptyState, FormSection,
  IconSettings, IconWarning, IconCheck,
  LoadingState,
} from "@rahalgo/ui";
import ImageUpload from "@/components/admin/ImageUpload";
import { FileUpload } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import dynamic from "next/dynamic";
import ZonesPanel from "@/components/admin/settings/zones";
import CitiesPanel from "@/components/admin/settings/cities";
import SitePagesPanel from "@/components/admin/settings/site-pages";
import BannersPanel from "@/components/admin/settings/banners";
import WhatsAppPanel from "@/components/admin/settings/whatsapp";
import BroadcastPanel from "@/components/admin/BroadcastPanel";

/** **«عرض,طول» ← رقمان** — وفارغٌ أو مشوَّهٌ يعني «لا موقع». */
function geoOf(v: string): [number, number] | null {
  const p = v.split(",");
  if (p.length !== 2) return null;
  const la = Number(p[0]);
  const ln = Number(p[1]);
  return Number.isFinite(la) && Number.isFinite(ln) ? [la, ln] : null;
}

const m = getMessages(defaultLocale);


/* **والخريطةُ لا تُصيَّر في الخادم** — `leaflet` يقرأ `window` عند التحميل. */
const PickMap = dynamic(() => import("@rahalgo/ui/map").then((mod) => mod.PickMap), {
  ssr: false,
});
const S = m.admin.settings;

/**
 * **يقسّم مفاتيحَ المجموعة إلى صناديق.**
 *
 * **وموضعُ الصندوق أوّلُ ظهورٍ لاسمه** — فترتيبُ الفهرس هو ما يُرى، **ولا
 * فرزَ بالاسم** يقلب ما قصده من كتبه.
 *
 * **ويُجمع المتفرّقُ تحت اسمه.**
 *
 * (عطبٌ ظهر ٢٠٢٦-٠٨-٠٩ بعد نقل مفاتيحَ إلى «إعدادات الموقع»: كان الجمعُ
 *  بالتجاور — **فمفاتيحُ الهويّة جاءت في دفعتين بينهما مفاتيحُ التواصل،
 *  فظهر صندوقُ «الهويّة البصريّة» مرّتين.**)
 *
 * **وترتيبُ الفهرس لا يُملي أن يكون كلُّ قسمٍ متلاصقاً** — ومن أضاف مفتاحاً
 * في موضعه المنطقيّ لا يجب أن يشقّ صندوقاً بلا أن يدري.
 */
function sectionsOf(items: Setting[]): { name: string; items: Setting[] }[] {
  const out: { name: string; items: Setting[] }[] = [];
  const at = new Map<string, number>();
  for (const it of items) {
    const name = it.section ?? "";
    const i = at.get(name);
    if (i === undefined) {
      at.set(name, out.length);
      out.push({ name, items: [it] });
    } else {
      out[i]!.items.push(it);
    }
  }
  return out;
}


/** **واسمُ الصندوق من المعجم** — وغيابُه يُظهر مفتاحَه لا فراغاً. */
function sectionLabel(name: string): string {
  return (S.sections as Record<string, string>)?.[name] ?? name;
}


/** **يطابق أنواعَ الكتالوج في المحرّك** — ونوعٌ يُضاف هناك ولا يُضاف هنا يسقط إلى الحقل النصّيّ. */
type Kind =
  | "int"
  | "money"
  | "bool"
  | "choice"
  | "text"
  | "media"
  | "percent"
  | "file"
  | "geo"
  | "longtext";

interface Setting {
  key: string;
  /** **مسارُ الصورة المشتقُّ من المعرّف** — يُرسله الخادمُ مع إعدادات الصور. */
  media_url?: string | null;
  group: string;
  kind: Kind;
  /** **صندوقٌ داخل المجموعة** — وفارغٌ يعني «في المجرى». */
  section?: string;
  min?: number;
  max?: number;
  options?: string[];
  unit?: string;
  default: unknown;
  sensitive?: boolean;
  value: unknown;
  updated_at: string | null;
  updated_by: string | null;
  /** **شرطُ الظهور** — مفتاحٌ آخرُ بإحدى قيمٍ بعينها. */
  /**
   * شرطُ الظهور من الفهرس.
   *
   * **و`equals` قد تصل `null`**: Go تُسلسل `[]string` الفارغةَ `null` لا `[]`
   * — **فشرطٌ بلا قيمٍ يُسقط الصفحةَ كلَّها** بـ«Cannot read properties of
   * null». (وقع فعلاً ٢٠٢٦-٠٨-٠٦ عند إضافة `not_empty`.)
   */
  show_when?: { key: string; equals: string[] | null; not_empty?: boolean };
}

/* **والتلميحُ اختياريّ**: مفتاحٌ يُفهم من عنوانه لا يُشرح. */
const label = (k: string) =>
  (S.keys as Record<string, { label?: string; hint?: string }>)[k]?.label ?? k;
const hint = (k: string) =>
  (S.keys as Record<string, { label?: string; hint?: string }>)[k]?.hint ?? "";
const unitText = (u?: string) => (u ? (S.units as Record<string, string>)[u] ?? "" : "");
const choiceText = (c: string) => (S.choices as Record<string, string>)[c] ?? c;
/**
 * **ما يعنيه الصفرُ في هذا المفتاح** — إن كان له معنًى خاصّ.
 *
 * «أجرة التوصيل = ٠» رقمٌ صحيحٌ لا خطأ، **ومعناه «التوصيل مجّانيّ»** — وهو
 * قرارٌ كبيرٌ يُتّخذ بحرفٍ واحد. **ورقمٌ لا يقول ما يفعله يُترك على صفره سهواً
 * ويُكتشف في آخر الشهر.**
 *
 * **ولا يُقال إلّا حين يقع**: شرحٌ دائمٌ تحت الحقل زحامٌ، **وشارةٌ تظهر عند
 * الصفر وحدَه تُقرأ.** (قرارُ المالك ٢٠٢٦-٠٨-٠٤.)
 */
const zeroNote = (k: string) =>
  (S.keys as Record<string, { zero?: string }>)[k]?.zero ?? "";
/**
 * حالُ مفتاحٍ منطقيّ بالكلمات — **«المنصة تدير الطلبات» لا «مُطفأ».**
 *
 * **و«نعم/لا» لا تقول شيئاً**: من قرأ «لا» تحت «المتجر يدير طلباته» عرف أنّه
 * لا يديرها **ولم يعرف من يديرها.** والبديلُ العامّ يبقى لمفاتيحَ لم تُوصَف
 * بعد — فلا تُفرَض كتابةُ وصفين لكلّ مفتاح.
 */
const boolText = (k: string, side: "on" | "off") =>
  (S.boolStates as Record<string, { on: string; off: string }>)[k]?.[side] ??
  (side === "on" ? S.boolOn : S.boolOff);

/**
 * **نوعُ الوسيط لكلّ مفتاحِ صورة** — مصدرٌ واحدٌ يطابق `validKinds` في المحرّك.
 *
 * **ومفتاحٌ جديدٌ بلا سطرٍ هنا يرفع بنوعٍ خطأ** — فالافتراضُ مكتوبٌ عند
 * الاستعمال ليُرى.
 */
const MEDIA_KIND: Record<string, "platform_logo" | "auth_background" | "site_background"> = {
  "platform.logo": "platform_logo",
  "auth.background": "auth_background",
  "auth.background_mobile": "auth_background",
  "platform.background": "site_background",
  /* **والنسخُ الجوّالةُ من نوع أخواتها.**

     (كُشف ٢٠٢٦-٠٨-٠٩ عند مراجعة الشاشة: أربعةُ مفاتيحَ جديدةٍ تسقط إلى
      `platform_logo` — **فتُرفع خلفيّةُ صفحةٍ بحدودِ شعارٍ ومصغَّرتِه.**)

     **والنوعُ يحكم الحدَّ والمصغَّرة**: شعارٌ يُصغَّر إلى مئتين، وخلفيّةٌ
     تحتاج ألفين. **ولا يظهر الخطأُ إلّا صورةً باهتةً ممطوطة.** */
  "platform.background_mobile": "site_background",
};

export default function SettingsPage() {
  const { user: me } = useAuth();
  const isAdmin = !!me?.roles.includes("admin");
  const [list, setList] = useState<Setting[] | null>(null);
  const [error, setError] = useState("");
  /** التبويبُ المفتوح — وفراغُه يعني «أوّلَ مجموعةٍ يرسلها الخادم». */
  const [tab, setTab] = useState("");
  /** **القسمُ المفتوح داخلَ المجموعة** — وفارغٌ يعني «أوّلُه». */
  const [sec, setSec] = useState("");
  /** **ترتيبُ الأقسام من الخادم** — فيه القسمُ الفارغُ الذي لا مفتاحَ فيه بعد. */
  const [order, setOrder] = useState<string[]>([]);

  const load = useCallback(async () => {
    try {
      const res = await api<{ settings: Setting[]; groups: string[] }>("/api/v1/admin/settings");
      setList(res.settings ?? []);
      setOrder(res.groups ?? []);
      setError("");
    } catch (err) {
      setError(errorText(err));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  // **نمطُ الهامش يُقرأ مرّةً للصفحة** — تتبعه لافتةُ حقل القيمة.
  const marginMode = String(
    list?.find((x) => x.key === "pricing.margin_mode")?.value ?? "percent",
  );

  /**
   * المجموعات **بترتيب الخادم** لا بترتيب أبجديّ: المفاتيح مجموعةٌ بالموضوع،
   * وبعثرتُها تفصل «مهلة القبول» عن «مهلة التوصيل».
   *
   * **والترتيبُ يأتي قائمةً صريحةً لا مشتقّاً من المفاتيح.** كان يُشتقّ —
   * أوّلُ ظهورٍ للمجموعة هو موضعُها — **فقسمٌ بلا مفاتيحَ لا يظهر أصلاً**،
   * ولا يُبنى قسمٌ يُملأ على مراحل.
   */
  /**
   * **ولا يُعرض مفتاحٌ لا أثرَ له في الوضع الحاليّ.**
   *
   * مهلةُ العرض لا تُستعمل في «الأسرع» — **وحقلٌ لا أثرَ له يُضبط ثمّ
   * يُنتظر أثرُه فلا يقع**، فيُشكّ في الشاشة كلِّها.
   *
   * **والشرطُ من الخادم لا من هنا**: الفهرسُ يقوله (`ShowWhen`)، **ولو كُتب
   * في الواجهة لَافترق عمّا يعمل به المحرّك.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «مهلةُ العرض يجب أن تظهر فقط بوضع التساوي».)
   */
  const visible = useCallback(
    (s: Setting, all: Setting[]) => {
      if (!s.show_when) return true;
      const on = all.find((x) => x.key === s.show_when!.key);
      if (!on) return false;
      // **وشرطُ «غيرِ الفارغ» للوسائط**: معرّفُ الصورة نصٌّ عشوائيّ لا يُقارن
      // بقائمة، **والسؤالُ الوحيدُ المفيدُ عنه أرُفعت أم لا.**
      if (s.show_when!.not_empty) return on.value !== "" && on.value != null;
      return (s.show_when!.equals ?? []).includes(String(on.value));
    },
    [],
  );

  const groups = useMemo(() => {
    if (!list) return [];
    return order.map((g) => ({
      g,
      items: list.filter((s) => s.group === g && visible(s, list)),
    }));
  }, [list, order, visible]);

  if (!list) return <LoadingState variant="text" />;

  /**
   * **التبويباتُ نوعان في شريطٍ واحد.**
   *
   * مجموعاتُ المفاتيح تأتي من الخادم — **ولا مجموعةَ اليومَ**، فالفهرسُ فارغٌ
   * يُبنى بقرار. **وشاشتان ليستا مفاتيح**: المناطقُ وبوتُ واتساب. وكانتا
   * قسمين مستقلّين في القائمة الجانبية **وهما ضبطٌ لا تشغيل** — ومن يفتح
   * «الإعدادات» يبحث فيهما عمّا لا يجده.
   *
   * قرارُ المالك (٢٠٢٦-٠٨-٠٣): «مناطقُ التغطية تكون بالإعدادات · بوتُ واتساب
   * أيضاً بالإعدادات · الإعداداتُ تكون تبويباتٍ لكلّ قسمٍ زرُّ تبويب».
   *
   * # ولماذا ذهب تبويبُ العمولات
   *
   * كان صفحةً تجمع نسبَ المال كلَّها في نظرةٍ واحدة **لأنّ مفاتيحَها كانت
   * متفرّقةً بين أربع مجموعات** — فلولاه لَضاعت النظرةُ الجامعة.
   *
   * **وقد ذهب سببُه**: لا مفتاحَ في المنصة يجمعه، **وشاشةٌ تجمع لا شيءَ
   * تُفتح فتُقرأ عطباً.** (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «قسم العمولات احذفه،
   * وقسم جديد أيضاً — فقط البوت والمناطق اتركها».)
   *
   * **وحين تعود مفاتيحُ المال يُقرَّر عندها**: أتُجمع في شاشةٍ أم تكفيها
   * مجموعتُها؟ — **والجوابُ يتبع أين وقعت، لا ما كان.**
   */
  /* **وأقسامٌ ليست مفاتيحَ** — محرّرٌ للمناطق، وضبطٌ لبوت واتساب، **وإعلانُ
     المنصة.** (قرارُ المالك ٢٠٢٦-٠٨-٠٨: «إعلان المنصة انقله على الإعدادات
     قسمٌ لوحده».)

     **وموضعُه كان مبدئيّاً بنصّه**: نُقل إلى بابٍ مستقلٍّ ٢٠٢٦-٠٨-٠٧
     «مبدئيّاً لبين ما ننتقل إلى لوحة الأدمن ونرتّبها» — وهذا أوانُه.

     **والإرسالُ للمالك وحدَه**: كان البابُ محجوزاً بـ`roles: ["admin"]`،
     **فيبقى محجوزاً تبويباً** — لا يُرسَم لغيره أصلاً. */
  const extra = [
    // **والمدنُ قبل المناطق** — **المنطقةُ بنتُ المدينة**، ومن قرأ
    // «مناطق» قبل أن يعرف أنّ للمنصّة مدناً ظنّ التغطيةَ طبقةً واحدة.
    { key: "cities", label: m.admin.cities.title },
    { key: "zones", label: m.terms.zones },
    { key: "whatsapp", label: m.admin.nav.whatsapp },
    ...(isAdmin ? [{ key: "broadcast", label: m.admin.broadcast.title }] : []),
  ];
  const extraKeys = extra.map((x) => x.key);
  // **وأوّلُ تبويبٍ مفتوحٍ أوّلُ ما يُعرض فعلاً.**
  //
  // كان `groups[0]` — **والمجموعاتُ فارغةٌ اليوم**، فيبقى `active` فراغاً
  // ولا يُفتح شيء: **شريطُ تبويباتٍ وتحته بياض.**
  const active = tab || groups[0]?.g || extraKeys[0] || "";
  const activeGroup = groups.find((x) => x.g === active);

  return (
    <div>
      <PageHeader icon={IconSettings} title={m.admin.settingsPage.title} />
      <p className="mb-4 text-sm text-ink-muted">{m.admin.settingsPage.hint}</p>

      {error && (
        <Alert className="mb-4">{error}</Alert>
      )}

      <Tabs
        className="mb-4"
        items={[
          ...groups.map((x) => ({ key: x.g, label: (S.groups as Record<string, string>)[x.g] ?? x.g })),
          ...extra,
        ]}
        value={active}
        /* **وتبديلُ المجموعة يُصفّر القسمَ المفتوح** — وإلّا حُمل اسمُ
           قسمٍ من مجموعةٍ إلى أخرى لا وجودَ له فيها، **فيسقط الاختيارُ إلى
           أوّلها بلا سبب يُرى.** */
        onChange={(k) => {
          setTab(k);
          setSec("");
        }}
      />

      {activeGroup &&
        (activeGroup.items.length === 0 ? (
          /* **قسمٌ فارغٌ يقول إنّه فارغٌ عمداً.**

             شاشةٌ بيضاءُ تُقرأ عطباً: يظنّ من فتحها أنّ التحميل تعثّر فيُعيد،
             **أو أنّ إعداداتِه اختفت.** والفراغُ هنا مرحلةٌ لا خلل — يُملأ
             مفتاحاً مفتاحاً. */
          <EmptyState
            icon={IconSettings}
            title={S.emptyGroup}
            action={<p className="text-xs text-ink-muted">{S.emptyGroupHint}</p>}
          />
        ) : (
          /* ══════════════════════════════════════════════════════════
             **وأقسامُ المجموعة تبويباتٌ متجاورةٌ لا صناديقُ متراكمة**
             ══════════════════════════════════════════════════════════

             (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «صفحات الموقع يجب أن تكون جانب الهويّة
              البصريّة لا تحتها — لأنّ هيك بدنا نظلّ ننزل لتحت كلّ ما بدنا
              تعديل، مو معقولة. نخلّي التبويبات الرئيسيّة بجانب بعض، وإذا في
              داخل التبويب تبويبات فرعيّة».)

             **وصندوقان متراكمان يعني تمريراً في كلّ تعديل** — ومن أراد
             الثاني مرّ بالأوّل كلِّه. **والتبويبُ يُظهر واحداً ويُخفي ما
             سواه**، فيبقى ما تضبطه في أعلى الشاشة أبداً.

             **ولا تظهر التبويباتُ إلّا حين تُغني**: قسمٌ واحدٌ لا يُبوَّب —
             **شريطُ تبويبٍ فيه واحدٌ زينةٌ تأخذ سطراً.** والمجموعاتُ القديمةُ
             بلا أقسامٍ تبقى كما كانت: مفاتيحُ في مجرًى واحد.

             **وما لا قسمَ له يعلو التبويبات** — لا يُدفن في أحدها ولا
             يُخترع له بيت. */
          <div className="space-y-3">
            {(() => {
              const secs = sectionsOf(activeGroup.items);
              const loose = secs.find((x) => !x.name);
              /* ══════════════════════════════════════════════════════
                 **وأقسامُ الصفحات لا تصعد إلى الشريط الأعلى**
                 ══════════════════════════════════════════════════════

                 (شكوى المالك ٢٠٢٦-٠٨-٠٩: «خلفيّة شاشة الدخول وخلفيّة الموقع
                  ضفتهنّ بمكانين — بتبويبٍ لحالهنّ وبنفس الوقت بإعدادات
                  الموقع، وهذا غلط».)

                 **وسببُه أنّي جمعتُ كلَّ اسمِ قسمٍ في الشريط** — وأقسامُ
                 الصفحات أسماءٌ كغيرها (`page.auth`)، **فصعدت مرّةً بنفسها
                 ومرّةً داخلَ «صفحات الموقع».** وظهرت بمفاتيحها خامّةً لأنّه
                 لا اسمَ لها في المعجم: **اسمُ الصفحة هناك لا اسمُ القسم.**

                 **فما بدأ بـ`page.` بيتُه واحد**: لوحُ الصفحات. */
              const names = [
                ...secs.filter((x) => x.name && !x.name.startsWith("page.")).map((x) => x.name),
                ...(active === "site" ? ["pages"] : []),
              ];
              const at = names.includes(sec) ? sec : (names[0] ?? "");

              return (
                <>
                  {loose?.items.map((s) => (
                    <SettingRow
                      key={s.key}
                      s={s}
                      editable={isAdmin}
                      onSaved={load}
                      marginMode={marginMode}
                    />
                  ))}

                  {names.length > 1 && (
                    <Tabs
                      items={names.map((n) => ({ key: n, label: sectionLabel(n) }))}
                      value={at}
                      onChange={setSec}
                    />
                  )}

                  {names.length === 1 && (
                    <h2 className="heading-card">{sectionLabel(at)}</h2>
                  )}

                  {at === "pages" ? (
                    <SitePagesPanel
                      render={(section) => {
                        const rows = activeGroup.items.filter((s) => s.section === section);
                        /* **ولافتاتُ التسوّق فوق مفاتيحها.**

                           (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «نضيف السلايدر بالإعدادات
                            لصفحة التسوّق مع العمل التلقائيّ حسب الثواني».)

                           **واللافتاتُ ليست مفاتيحَ إعدادات** — صفوفٌ تُضاف
                           وتُحذف. **والمهلةُ تحتها**: من بدّل لافتةً يبدّل
                           مهلتَها في الشاشة نفسِها. */
                        const shop = section === "page.shop";
                        /* **ولافتاتُ الرئيسيّة في قسمها** — (تصحيحُ المالك
                           ٢٠٢٦-٠٨-١٧: «بانرات صفحة التسوّق مختلفة برأيي عن
                           الرئيسيّة»). **والضبطُ يُطلب حيث يُرى أثرُه.** */
                        const home = section === "page.home";
                        if (!shop && !home && rows.length === 0) return null;
                        return (
                          <div className="space-y-4">
                            {/* **وبطاقةُ لافتات التسوّق حُذفت** — (قرارُ المالك
                                ٢٠٢٦-٠٨-١٨: «أريد حذفَ سلايدر التسوّق وربطَ
                                التطبيق بسلايدر الرئيسيّة»).

                                **وبطاقةٌ تبقى لموضعٍ لا يقرؤه أحدٌ بابٌ يُفتح
                                فتُرفع فيه صورةٌ لا تُرى** — وهي عائلةُ الخلل
                                نفسُها التي نُظّفت في «أبوابٌ لا ينادِيها أحد». */}
                            {(shop || home) && <BannersPanel isAdmin={isAdmin} placement="home" />}
                            <div className="space-y-3">
                            {rows.map((s) => (
                              <SettingRow
                                key={s.key}
                                s={s}
                                editable={isAdmin}
                                onSaved={load}
                                marginMode={marginMode}
                              />
                            ))}
                            </div>
                          </div>
                        );
                      }}
                    />
                  ) : (
                    <div className="space-y-3">
                      {/* **وما لا قسمَ له لا يُرسم مرّتين.**

                          (شكوى المالك ٢٠٢٦-٠٨-٠٩ على قسم «التطبيق»: «ليش
                           مكرّرٌ مرّتين».)

                          **مجموعةٌ بلا أقسامٍ اسمُ قسمها فارغ** — فتُرسم
                          مرّةً بوصفها «سائبة» فوق التبويبات، **ومرّةً لأنّ
                          القسمَ المفتوح فارغٌ هو أيضاً** فيطابقها.

                          **فيُشترط اسمٌ غيرُ فارغ** — والسائبةُ لها موضعُها
                          أعلاه. */}
                      {(at ? (secs.find((x) => x.name === at)?.items ?? []) : []).map((s) => (
                        <SettingRow
                          key={s.key}
                          s={s}
                          editable={isAdmin}
                          onSaved={load}
                          marginMode={marginMode}
                        />
                      ))}
                    </div>
                  )}
                </>
              );
            })()}
          </div>
        ))}
      {active === "cities" && <CitiesPanel />}
      {active === "zones" && <ZonesPanel />}
      {active === "whatsapp" && <WhatsAppPanel />}
      {/* **والإعلانُ فعلٌ لا إعداد** — فيُقال ما هو قبل نموذجه: رسالةٌ تُرسل
          ولا تُسحب. واللوحُ نفسُه يسأل قبل الإرسال ويقول كم حساباً ستصل. */}
      {active === "broadcast" && isAdmin && (
        <div className="space-y-3">
          <p className="text-sm text-ink-muted">{m.admin.broadcast.hint}</p>
          <BroadcastPanel />
        </div>
      )}
    </div>
  );
}

/**
 * صفٌّ واحد — يحرّر نفسه في مكانه.
 *
 * ولا نافذة منبثقة: النافذة تُخفي بقية الإعدادات، ومن يضبط «مهلة القبول» يريد
 * أن يرى «مهلة السائق» وهو يضبطها. والحفظ لكل صفٍّ على حدة لا زرّ واحد في
 * الأسفل — كي لا يحفظ من غيّر رقماً واحداً عشرين رقماً بلا قصد.
 */
function SettingRow({
  s,
  editable,
  onSaved,
  marginMode,
}: {
  s: Setting;
  editable: boolean;
  onSaved: () => void;
  /** نمطُ الهامش — **تتبعه لافتةُ حقل القيمة**: «٪» أو «ل.س». */
  marginMode: string;
}) {
  const [draft, setDraft] = useState<string>(() => String(s.value ?? ""));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    setDraft(String(s.value ?? ""));
  }, [s.value]);

  const numeric = s.kind === "int" || s.kind === "money";
  /**
   * **هل يُنتظر حفظ؟**
   *
   * **والمفاتيحُ التي تُحفظ بالضغطة لا تنتظر شيئاً**: المنطقيُّ والخيارُ
   * يُرسلان لحظةَ اللمس، **وزرُّ «حفظ» يظهر بجانبهما يقول إنّ شيئاً لم
   * يُحفظ بعد** — فيُضغط مرّةً ثانية على ما حُفظ.
   */
  const dirty =
    s.kind === "bool" || s.kind === "choice"
      ? false
      : draft !== String(s.value ?? "");

  async function save(raw: unknown) {
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/settings/${s.key}`, {
        method: "PUT",
        body: JSON.stringify({ value: raw }),
      });
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
      onSaved();
    } catch (err) {
      // الخادم يردّ `validation` لكل رفض — والمدى معروفٌ هنا، فنقول السبب
      setError(
        err instanceof ApiError && numeric
          ? S.range.replace("{min}", fmtNum(s.min ?? 0)).replace("{max}", fmtNum(s.max ?? 0))
          : m.errors.validation,
      );
    } finally {
      setBusy(false);
    }
  }

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (numeric) {
      const n = Number(draft);
      if (!Number.isInteger(n)) return setError(m.errors.validation);
      // الفحص قبل الشبكة: رحلةٌ إلى الخادم لتُردّ برسالة نعرفها سلفاً بطءٌ بلا سبب
      if ((s.min !== undefined && n < s.min) || (s.max !== undefined && n > s.max)) {
        return setError(
          S.range.replace("{min}", fmtNum(s.min ?? 0)).replace("{max}", fmtNum(s.max ?? 0)),
        );
      }
      return void save(n);
    }
    void save(draft);
  }

  return (
    <Card>
      <form onSubmit={submit}>
        <div className="mb-1 flex flex-wrap items-center gap-2">
          <span className="font-medium">{label(s.key)}</span>
          {s.sensitive && (
            <Badge variant="warning">
              <span className="flex items-center gap-1">
                <IconWarning size={12} />
                {S.sensitive}
              </span>
            </Badge>
          )}
          {zeroNote(s.key) && Number(s.value) === 0 && (
            <Badge variant="warning">{zeroNote(s.key)}</Badge>
          )}
          {saved && (
            <Badge variant="success">
              <span className="flex items-center gap-1">
                <IconCheck size={12} strokeWidth={3} />
                {S.saved}
              </span>
            </Badge>
          )}
        </div>
        {/* **وشرحٌ فارغٌ لا يترك مكانَه.**

            كان السطرُ يُرسَم دائماً — **فمفتاحٌ بلا شرحٍ يخلّف فراغاً بين
            اسمه وزرّه** يُقرأ نقصاً: أين الجملةُ التي كانت هنا؟

            **ولا كلَّ مفتاحٍ يحتاج شرحاً**: «وضع المنصة / وضع المتاجر» يقول
            نفسَه، **وجملةٌ تشرح ما لا يحتاج شرحاً تُعلّم العينَ أن تتخطّى
            الشروحَ كلَّها.** (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «بدون أيّ شرحٍ
            وأشياءَ مزعجة، فقط زرٌّ ذكيّ».) */}
        {hint(s.key) && (
          <p className="mb-3 text-xs leading-relaxed text-ink-muted">{hint(s.key)}</p>
        )}

        <div className="flex flex-wrap items-end gap-2">
          {s.kind === "bool" ? (
            /* ══════════════════════════════════════════════════════════
               **ومفتاحٌ واحدٌ يقول الحالَ بشكله**
               ══════════════════════════════════════════════════════════

               (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «سوِّهنّ أزراراً ذكيّة — زرٌّ ذكيّ:
                مفعَّل، معطَّل، وخلص. ما بدّها كلّ هالشي».)

               **كان ثلاثةَ أسطرٍ لمعنًى واحد**: اسمُ الإعداد في رأس الصفّ،
               ثمّ سطرُ «الآن: مُفعَّل»، ثمّ مربّعُ اختيارٍ يحمل الاسمَ ثانية.

               **والسطرُ الأوسطُ وُضع بحقٍّ يومَها** (قرارُ المالك ٢٠٢٦-٠٨-٠٤):
               **مربّعُ اختيارٍ لا يقول أهو وصفُ الحال أم وصفُ ما سيصير.**
               **والمفتاحُ يقوله بموضعه ولونه** — فلا يحتاج سطراً يترجمه.

               **والاسمُ في رأس الصفّ وحدَه** — فيُمرَّر فارغاً هنا. */
            <Switch
              id={s.key}
              label=""
              checked={s.value === true}
              disabled={!editable || busy}
              onChange={(next) => void save(next)}
            />
          ) : s.kind === "choice" ? (
            /* **أزرارٌ متجاورةٌ لا قائمةٌ منسدلة — والبدائلُ تُرى كلُّها.**

               المنسدلةُ تعرض المختارَ وتُخفي ما سواه: **من فتح الصفحة لا
               يعرف أنّ للإعداد بديلاً أصلاً** حتى يضغط. و«الأسرع التقاطاً»
               وحدَها في مربّعٍ تُقرأ عنواناً لا خياراً.

               **والمتجاورةُ تقول الحالَ والبديلَ معاً**: هذا ما يعمل الآن،
               وهذا ما يصير إن ضغطت — وهو معنى الزرّ الذكيّ.

               (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «خيارُ الطلبات تلقائي أو الأسرع
               أيضاً لازم يكون زرّاً ذكيّاً».) */
            <div className="flex flex-wrap gap-1 rounded-control border border-line bg-field p-1">
              {(s.options ?? []).map((o) => {
                const on = draft === o;
                return (
                  <button
                    key={o}
                    type="button"
                    disabled={!editable || busy}
                    aria-pressed={on}
                    onClick={() => {
                      if (on) return;
                      setDraft(o);
                      void save(o);
                    }}
                    className={`rounded-control px-3 py-1.5 text-sm transition-colors disabled:opacity-50 ${
                      on
                        ? "bg-primary font-bold text-on-bright elev-1"
                        : "text-ink-muted hover:bg-surface hover:text-ink"
                    }`}
                  >
                    {choiceText(o)}
                  </button>
                );
              })}
            </div>
          ) : s.kind === "percent" ? (
            /* **شريطٌ يُسحب فيُرى الأثر.** (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «شريط
               من ٠ إلى ١٠٠».)

               **وحقلُ الرقم يُكتب ثمّ يُحفظ ثمّ يُنظر** — ثلاثُ خطواتٍ لضبط
               شيءٍ يُحكَم عليه بالعين. **والرقمُ بجانبه يبقى** لمن أراد قيمةً
               بعينها أو أراد أن ينقلها إلى منصّةٍ أخرى.

               **ويُحفظ عند الإفلات لا مع كلّ بكسل** (`onMouseUp`/`onTouchEnd`
               عبر `change`): **وإلّا صار سحبُ الشريط مئةَ نداءٍ للخادم.** */
            <div className="flex flex-1 items-center gap-3">
              <input
                id={s.key}
                type="range"
                min={0}
                max={100}
                value={draft === "" ? 0 : Number(draft)}
                disabled={!editable || busy}
                onChange={(e) => setDraft(e.target.value)}
                onMouseUp={() => void save(Number(draft))}
                onTouchEnd={() => void save(Number(draft))}
                onKeyUp={() => void save(Number(draft))}
                className="h-1.5 flex-1 cursor-pointer appearance-none rounded-badge bg-field accent-accent"
              />
              <span className="w-12 shrink-0 text-end text-sm font-medium tabular-nums text-primary-dark">
                {draft === "" ? 0 : Number(draft)}%
              </span>
            </div>
          ) : s.kind === "file" ? (
            /* **وملفٌّ يُرفع لا اسمٌ يُكتب.**

               (طلبُ المالك ٢٠٢٦-٠٨-٠٨: «رفع التطبيق بشكلٍ مباشر من خيار
                رفع أيضاً».)

               **والقيمةُ اسمُ الملفّ المخزَّن** يكتبه الخادمُ عند الرفع —
               **واسمٌ يُكتب خطأً يعني زرَّ تنزيلٍ يقود إلى لا شيء.** */
            <FileUpload
              /* **ولا اسمَ ولا تلميحَ هنا** — رأسُ الصفّ كتبهما.
                 (شكوى المالك ٢٠٢٦-٠٨-٠٩.) */
              label=""
              hint=""
              accept=".apk"
              present={typeof s.value === "string" && s.value !== ""}
              path="/api/v1/admin/app-file"
              api={api}
              errorText={(e) => (e instanceof ApiError ? e.message : m.errors.internal)}
              onChange={onSaved}
            />
          ) : s.kind === "media" ? (
            /* **وصورةٌ تُرفع لا معرّفٌ يُكتب.**

               (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «لا تنسَ إضافة هوية المنصة أيضاً —
               الاسم واللوغو».)

               **وقيمةُ الإعداد معرّفُ الوسيط**: المسارُ يتغيّر إن نُقل
               التخزينُ، **والمعرّفُ يبقى** — والمسارُ يُشتقّ منه عند العرض.

               **والرافعُ هو رافعُ اللوحة نفسُه** (`ImageUpload`) — بحدوده
               وفحصه ومصغَّرته. **ورافعٌ ثانٍ يعني حدَّ حجمٍ ثانياً يفترق.** */
            <ImageUpload
              /* **ونوعُ الوسيط من المفتاح لا مثبَّتاً.**

                 كان `"platform_logo"` لكلّ مفتاحٍ من نوع `media` — **وكان
                 صحيحاً يومَ كان المفتاحُ واحداً.** ولمّا جاءت خلفيّةُ الدخول
                 **كانت سترفع صورتَها باسم «شعار المنصة»** — فتُخزَّن بنوعٍ
                 ليس نوعَها، **ولا يظهر الخطأُ إلّا لمن يقرأ القاعدة.** */
              kind={MEDIA_KIND[s.key] ?? "platform_logo"}
              /* **ولا اسمَ هنا** — رأسُ الصفّ كتبه.
                 (شكوى المالك ٢٠٢٦-٠٨-٠٩: «هون مكرّرٌ الاسمُ مرّتين بدون
                  سبب».) **والرافعُ مكوّنٌ عامٌّ يُستعمل في نماذجَ لا رأسَ
                 لها**، فيحمل اسمَه — **وفي صفِّ إعدادٍ يصير الثاني.** */
              label=""
              initialUrl={typeof s.value === "string" && s.value ? s.media_url : null}
              onChange={(id) => void save(id)}
            />
          ) : s.kind === "longtext" ? (
            /* ══════════════════════════════════════════════════════════
                **ونصُّ صفحةٍ يُكتب في صندوقٍ لا في سطر**
                ══════════════════════════════════════════════════════════

               (طلبُ المالك ٢٠٢٦-٠٨-٠٩: «جهّز الصفحات لتكون ديناميكيّةً
                أتحكّم بها من لوحة التحكّم، أعدّل النصوص الموجودة».)

               **والاتّفاقُ بسيط**: سطرٌ فارغٌ يفصل كرتاً عن كرت، **وأوّلُ
               سطرٍ في الكرت عنوانُه** وما بعده فقراتُه.

               **وفارغُه يعرض النصَّ الأصليّ** — فمن لم يمسّه لا ينكسر عنده
               شيء، **ومن أفرغه بعد أن كتب يعود إلى الأصل** لا إلى صفحةٍ
               بيضاء. */
            <div className="space-y-2">
              <Textarea
                id={s.key}
                rows={12}
                value={draft}
                disabled={!editable || busy}
                onChange={(e) => {
                  setDraft(e.target.value);
                  setError("");
                }}
                placeholder={S.longTextHint}
              />
              <p className="text-xs leading-relaxed text-ink-muted">{S.longTextHint}</p>
            </div>
          ) : s.kind === "geo" ? (
            /* ══════════════════════════════════════════════════════════
                **وموقعٌ يُنقر لا رقمان يُكتبان**
                ══════════════════════════════════════════════════════════

               (طلبُ المالك ٢٠٢٦-٠٨-٠٩: «صفحة تواصل معنا لازم صفحة خاصّة
                فيها خريطة المكتب».)

               **من يضبط موقعَ مكتبٍ لا يحفظ إحداثيّاته** — ومن كتبها بيده
               وضع فاصلةً في غير موضعها **فوقع المكتبُ في بحر**، ولا يُقال
               له لماذا لا تظهر الخريطة.

               **والقيمةُ تُبنى من النقرة** بستّ منازلَ عشريّة — نحو عشرة
               سنتيمترات، **وهو أدقُّ ممّا يحتاجه بابُ مكتب.** */
            <div className="space-y-2">
              <div className="overflow-hidden rounded-control border border-line">
                <PickMap
                  lat={geoOf(draft)?.[0] ?? null}
                  lng={geoOf(draft)?.[1] ?? null}
                  height="h-56"
                  onPick={(la, ln) => {
                    if (!editable || busy) return;
                    void save(`${la.toFixed(6)},${ln.toFixed(6)}`);
                  }}
                />
              </div>
              <p className="text-xs leading-relaxed text-ink-muted">{hint(s.key)}</p>
              {draft ? (
                <p className="text-xs text-ink-muted" dir="ltr">
                  {draft}
                </p>
              ) : null}
            </div>
          ) : (
            <>
              {/* **وحقلُ النصّ يملأ السطر.**

                  (شكوى المالك ٢٠٢٦-٠٨-٠٩: «حقلُ عنوان المكتب صغير».)

                  **الصفُّ `flex`** — وحقلُ النصّ فيه لا ينمو إلّا إن قيل له،
                  **فيبقى بعرضه الطبيعيّ** ويترك نصفَ الصفّ فارغاً.
                  **والرقمُ يبقى ضيّقاً**: حقلٌ لثلاثةِ أرقامٍ بعرض الشاشة
                  يُقرأ حقلَ نصّ. */}
              <Input
                id={s.key}
                type={numeric ? "number" : "text"}
                inputMode={numeric ? "numeric" : undefined}
                min={s.min}
                max={s.max}
                value={draft}
                disabled={!editable || busy}
                onChange={(e) => {
                  setDraft(e.target.value);
                  setError("");
                }}
                className={numeric ? "w-40" : "w-full"}
                wrapperClassName={numeric ? "" : "min-w-0 flex-1"}
              />
              {/* **ولافتةُ الهامش تتبع نمطَه.**

                  مفتاحٌ واحدٌ بمعنيين: «٣٠٠٠» ثلاثةُ آلاف ليرةٍ في الثابت،
                  **وواحدٌ وثلاثون ضعفاً في النسبة.** وقد وقع فعلاً في تجربةٍ
                  حيّة: كُتب ٣٠٠٠ قصداً للّيرة **فبِيع طلبٌ تكلفتُه ٦٥ ألفاً
                  بمليونين** — والحقلُ لا يقول أيَّهما يُكتب. */}
              {s.key === "pricing.margin_value" ? (
                <span className="pb-2 text-sm font-medium text-primary-dark">
                  {marginMode === "fixed" ? m.common.currency : "%"}
                </span>
              ) : (
                s.unit && (
                  <span className="pb-2 text-sm text-ink-muted">{unitText(s.unit)}</span>
                )
              )}
              {editable && dirty && (
                <Button type="submit" disabled={busy}>
                  {m.common.save}
                </Button>
              )}
            </>
          )}
        </div>

        {numeric && s.min !== undefined && s.max !== undefined && (
          <p className="mt-1.5 text-xs text-ink-muted">
            {S.range.replace("{min}", fmtNum(s.min)).replace("{max}", fmtNum(s.max))}
            {" · "}
            {S.defaultIs.replace("{v}", fmtNum(Number(s.default)))}
          </p>
        )}

        {error && <p className="mt-1.5 text-xs text-danger">{error}</p>}

        {/* **ولا سطرَ «آخر تغيير» تحت أيّ مفتاح.**

            كان يقول «مدير المنصة — ٤/٨/٢٠٢٦، ٣:٥٦ م». **وهو خبرٌ يُقرأ مرّةً
            ثمّ يشغل سطراً تحت كلّ بطاقةٍ إلى الأبد** — ومن يفتح الإعدادات
            يريد أن يضبط لا أن يقرأ تاريخاً.

            **ولا يُفقَد شيء**: `admin.setting_update` يُقيَّد في سجلّ الأحداث
            بـ«كان» و«صار» ومن فعل — **وهو موضعُ المراجعة، لا بطاقةُ الضبط.**

            (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «لا يوجد داعٍ لهذا بأيّ إعداد».) */}

        {/* **وما يُحفظ باللمس يقول ذلك قبل أن يُلمس.**

            التنبيهُ كان يظهر حين يُنتظر حفظ. **والزرُّ الذكيُّ لا ينتظر** —
            يُضغط فيسري، **فلا فرصةَ لتنبيهٍ يظهر بعده.** فيُقال دائماً في
            المفاتيح التي تمسّ المال وتُحفظ باللمس. */}
        {s.sensitive && (dirty || s.kind === "bool" || s.kind === "choice") && (
          <Alert tone="warning" className="mt-2">
            {S.sensitiveHint}
          </Alert>
        )}
      </form>
    </Card>
  );
}
