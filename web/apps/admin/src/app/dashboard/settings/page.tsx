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
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  Tabs,
  Alert,
  PageHeader, Button, Input, Select, Checkbox, Badge, Card, EmptyState,
  IconSettings, IconWarning, IconCheck,
} from "@rahalgo/ui";
import ImageUpload from "@/components/ImageUpload";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import ZonesPanel from "@/components/settings/zones";
import WhatsAppPanel from "@/components/settings/whatsapp";

const m = getMessages(defaultLocale);
const S = m.admin.settings;

type Kind = "int" | "money" | "bool" | "choice" | "text" | "media";

interface Setting {
  key: string;
  /** **مسارُ الصورة المشتقُّ من المعرّف** — يُرسله الخادمُ مع إعدادات الصور. */
  media_url?: string | null;
  group: string;
  kind: Kind;
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
  show_when?: { key: string; equals: string[] };
}

const label = (k: string) =>
  (S.keys as Record<string, { label: string; hint: string }>)[k]?.label ?? k;
const hint = (k: string) =>
  (S.keys as Record<string, { label: string; hint: string }>)[k]?.hint ?? "";
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

export default function SettingsPage() {
  const { user: me } = useAuth();
  const isAdmin = !!me?.roles.includes("admin");
  const [list, setList] = useState<Setting[] | null>(null);
  const [error, setError] = useState("");
  /** التبويبُ المفتوح — وفراغُه يعني «أوّلَ مجموعةٍ يرسلها الخادم». */
  const [tab, setTab] = useState("");
  /** **ترتيبُ الأقسام من الخادم** — فيه القسمُ الفارغُ الذي لا مفتاحَ فيه بعد. */
  const [order, setOrder] = useState<string[]>([]);

  const load = useCallback(async () => {
    try {
      const res = await api<{ settings: Setting[]; groups: string[] }>("/api/v1/admin/settings");
      setList(res.settings ?? []);
      setOrder(res.groups ?? []);
      setError("");
    } catch {
      setError(m.errors.internal);
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
      return !!on && s.show_when.equals.includes(String(on.value));
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

  if (!list) return <p className="p-6 text-center text-ink-muted">{m.common.loading}</p>;

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
  const extra = [
    { key: "zones", label: m.terms.zones },
    { key: "whatsapp", label: m.admin.nav.whatsapp },
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
        onChange={setTab}
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
          <div className="space-y-3">
            {activeGroup.items.map((s) => (
              <SettingRow
                key={s.key}
                s={s}
                editable={isAdmin}
                onSaved={load}
                marginMode={marginMode}
              />
            ))}
          </div>
        ))}
      {active === "zones" && <ZonesPanel />}
      {active === "whatsapp" && <WhatsAppPanel />}
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
            /* **زرٌّ ذكيٌّ يقول الحال لا الفعل.**

               مربّعُ اختيارٍ باسم المفتاح يسأل من ينظر: **أهذا وصفُ ما هو
               قائمٌ الآن أم وصفُ ما سيصير إن ضغطتُ؟** — والفرقُ في إعدادٍ
               يحكم من يدير الطلبات فرقُ يومٍ كامل.

               **فيُقال الحالُ صراحةً** فوق المربّع: «الآن: …».

               (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «زرٌّ ذكيٌّ للتبديل بين المنصة تدير
               المتاجر أو المتاجر تدير نفسها».) */
            <div className="flex flex-col gap-1.5">
              <span className="text-xs font-medium text-ink-muted">
                {S.nowIs.replace(
                  "{v}",
                  s.value === true ? boolText(s.key, "on") : boolText(s.key, "off"),
                )}
              </span>
              <Checkbox
                id={s.key}
                label={label(s.key)}
                checked={s.value === true}
                disabled={!editable || busy}
                onChange={(e) => void save(e.target.checked)}
              />
            </div>
          ) : s.kind === "choice" ? (
            /* **أزرارٌ متجاورةٌ لا قائمةٌ منسدلة — والبدائلُ تُرى كلُّها.**

               المنسدلةُ تعرض المختارَ وتُخفي ما سواه: **من فتح الصفحة لا
               يعرف أنّ للإعداد بديلاً أصلاً** حتى يضغط. و«الأسرع التقاطاً»
               وحدَها في مربّعٍ تُقرأ عنواناً لا خياراً.

               **والمتجاورةُ تقول الحالَ والبديلَ معاً**: هذا ما يعمل الآن،
               وهذا ما يصير إن ضغطت — وهو معنى الزرّ الذكيّ.

               (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «خيارُ الطلبات تلقائي أو الأسرع
               أيضاً لازم يكون زرّاً ذكيّاً».) */
            <div className="flex flex-wrap gap-1 rounded-control border border-line bg-page p-1">
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
                        ? "bg-primary font-bold text-on-solid elev-1"
                        : "text-ink-muted hover:bg-surface hover:text-ink"
                    }`}
                  >
                    {choiceText(o)}
                  </button>
                );
              })}
            </div>
          ) : s.kind === "media" ? (
            /* **وصورةٌ تُرفع لا معرّفٌ يُكتب.**

               (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «لا تنسَ إضافة هوية المنصة أيضاً —
               الاسم واللوغو».)

               **وقيمةُ الإعداد معرّفُ الوسيط**: المسارُ يتغيّر إن نُقل
               التخزينُ، **والمعرّفُ يبقى** — والمسارُ يُشتقّ منه عند العرض.

               **والرافعُ هو رافعُ اللوحة نفسُه** (`ImageUpload`) — بحدوده
               وفحصه ومصغَّرته. **ورافعٌ ثانٍ يعني حدَّ حجمٍ ثانياً يفترق.** */
            <ImageUpload
              kind="platform_logo"
              label={label(s.key)}
              initialUrl={typeof s.value === "string" && s.value ? s.media_url : null}
              onChange={(id) => void save(id)}
            />
          ) : (
            <>
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
