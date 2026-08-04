"use client";

/**
 * قائمة مهامّ السائق — لا لوحة تحكّم.
 *
 * اللوحات لمن يجلس إلى مكتب فيمسح بعينه شبكة أرقام. والسائق واقفٌ عند درّاجته
 * يمسك هاتفه بيدٍ واحدة، وربّما تحت الشمس: فما يحتاجه سطرٌ واحد يقول **ماذا
 * الآن**، وزرٌّ واحد كبير يُنهيه. ولذلك:
 *
 * - لا صفحة تفاصيل منفصلة: البطاقة **هي** التفصيل، وكلُّ ضغطةٍ زائدة على درّاجة
 *   ضغطةٌ لن تُضغط.
 * - الزرّ الكبير واحدٌ في كل بطاقة، ونصُّه **الفعل التالي** لا اسم الحالة:
 *   «استلمتُ الطلب» لا «picked_up».
 * - «تعذّر التسليم» زرٌّ ثانويّ صغير تحته: هو مخرجٌ حقيقي لكنه ليس الطريق.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtTime } from "@rahalgo/i18n";
import {
  Button,
  Badge,
  Modal,
  Input,
  Card,
  EmptyState,
  StatGrid,
  StatCard,
  useLiveRefresh,
  useRepeatingChime,
  useLocationBeacon,
  fmtDistance,
  IconOrder,
  IconStore,
  IconLocation,
  IconPhone,
  IconCheck,
  IconBalance,
  IconWallet,
  IconDriver,
  IconWarning,
  IconCamera,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const D = m.driver;
/** وحداتُ المسافة — من القاموس لا من نصٍّ مكتوبٍ في كلّ شاشة. */
const UNITS = m.admin.settings.units;

/** سببٌ مصنَّفٌ وذنبُه — **والذنبُ يقرّر التعويض** (`failreasons.go`). */
type FailReason = { code: string; fault: string };

/**
 * المرحلتان اللتان يملك السائقُ فيهما إعلانَ التعذّر — **وهو من هناك فهو من
 * يقول**: عند باب المتجر «المطعم لم يسلّمني»، وعند باب الزبون «لم يستلم».
 *
 * **ومصدرٌ واحدٌ للاثنتين**: الزرُّ يظهر بها والأسبابُ تُجلَب بها. وكانتا
 * مكتوبتين في موضعٍ واحدٍ فقط، **فلو أُضيفت ثالثةٌ في الزرّ وحدَه لظهر زرٌّ
 * بلا أسباب.**
 */
const FAIL_AT = ["at_pickup", "at_dropoff"] as const;

/** الفعل التالي لكل حالة — مصدرٌ واحد يقابل خارطة الحالات في الخادم. */
const NEXT: Record<string, string> = {
  assigned: "at_pickup",
  at_pickup: "picked_up",
  picked_up: "on_the_way",
  on_the_way: "at_dropoff",
  at_dropoff: "delivered",
};

interface DriverOrder {
  id: string;
  number: number;
  status: string;
  merchant_name: string;
  merchant_phone: string | null;
  address_text: string;
  lat: number;
  lng: number;
  customer_name: string;
  customer_phone: string;
  total: number;
  cash_due: number;
  /** نقطةُ استلامٍ بديلة — البضاعةُ مع سائقٍ سابقٍ وقع له طارئ، لا في المتجر. */
  pickup_lat?: number | null;
  pickup_lng?: number | null;
  pickup_note?: string;
  items_count: number;
  ready_at: string | null;
  prep_minutes: number | null;
  accepted_at: string | null;
  created_at: string;
  /** **سياسةُ هذا المتجر في الاسترجاع** — وهي ما يقرّر وجهةَ البضاعة بعد التعذّر. */
  merchant_accepts_returns: boolean;
  /** رمزُ التعذّر — يُعرَض مترجَماً على البطاقة التي بقيت بيده. */
  fail_reason: string;
  /** كم بينه وبين نقطة الاستلام — **بالمتر، وسالبٌ يعني «لا يُعرف»**. */
  to_pickup_m: number;
  /** طولُ المشوار: من الاستلام إلى باب الزبون. */
  leg_m: number;
}

interface Me {
  full_name: string;
  on_shift: boolean;
  shift_started_at: string | null;
  cash_held: number;
  cash_limit: number;
  balance: number;
  today_delivered: number;
  /** ما لم يُسلَّم اليوم — **ولا يُخفى**: يومٌ فيه تعذّرٌ ليس يومَ تسليماتٍ فقط. */
  today_failed: number;
  today_earned: number;
  /** جزءٌ من `today_earned` لا زيادةٌ عليه — يُعرض ليُعرف مصدرُه. */
  today_compensated: number;
  active_orders: number;
}

function errText(e: unknown): string {
  const key =
    e instanceof ApiError ? (e.body.message_key ?? "").split(".").pop() ?? "" : "";
  return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
}

/** رابط الملاحة — يفتحه تطبيق الخرائط في الهاتف مباشرةً. */
function mapsHref(lat: number, lng: number): string {
  return `https://www.google.com/maps/dir/?api=1&destination=${lat},${lng}`;
}

export default function TasksPage() {
  const [me, setMe] = useState<Me | null>(null);
  const [mine, setMine] = useState<DriverOrder[]>([]);
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [failing, setFailing] = useState<DriverOrder | null>(null);
  const [emergency, setEmergency] = useState<DriverOrder | null>(null);
  /** الطلبُ الذي يُطلب إثباتُ تسليمه — **قبل «سُلّم» لا بعده**. */
  const [proving, setProving] = useState<DriverOrder | null>(null);
  /** **رمزُ** السبب المختار — لا نصُّه. */
  const [reason, setReason] = useState("");
  /** التفصيلُ الحرّ بجانبه — اختياريّ. */
  const [detail, setDetail] = useState("");
  /**
   * أسبابُ التعذّر لكلّ مرحلة — **مفتاحُها الحالة**.
   *
   * `FailReasonsAt` تُرجع ما يخصّ المرحلة وحدَها: **أسبابُ باب المتجر ليست
   * أسبابَ باب الزبون**، ومن رأى «المتجر مغلق» وهو واقفٌ أمام بيت الزبون
   * يختار أقربَ لفظٍ إليه فيكذب السجلّ.
   *
   * **ومرحلةٌ غائبةٌ من الخريطة ليست مرحلةً بلا أسباب** — هي مرحلةٌ لم
   * تصل بعد. والفرقُ بينهما هو كلُّ ما في الأمر (انظر النافذة).
   */
  const [reasonsBy, setReasonsBy] = useState<Record<string, FailReason[]>>({});
  /** **وتعذّرُ الجلب يُقال** — لا يُعرض سكوتاً يشبه «لا أسباب». */
  const [reasonsErr, setReasonsErr] = useState(false);

  const load = useCallback(() => {
    api<Me>("/api/v1/driver/me").then(setMe).catch(() => undefined);
    api<DriverOrder[]>("/api/v1/driver/orders").then(setMine).catch(() => undefined);
  }, []);

  useEffect(load, [load]);

  /**
   * **تُجلَب مع الشاشة لا مع الضغطة.**
   *
   * كانت تُطلب في `onFail` ثمّ تُفتح النافذةُ في السطر التالي بلا انتظار،
   * **فتُعرض «لا أسبابَ معرَّفة» بينما الطلبُ في الطريق** — جملةٌ تحكم على
   * الإعدادات، والحقيقةُ أنّها لم تُسأل بعد. وأيُّ إخفاقٍ — شبكةٌ منقطعةٌ
   * على درّاجة، خادمٌ يُعاد تشغيله، جلسةٌ انتهت — **كان يسقط في `catch`
   * فيصير القائمةَ الفارغةَ نفسَها.**
   *
   * **وثلاثُ حالاتٍ مختلفةٍ تُعرض بجملةٍ واحدة لا يُشخَّص منها شيء**: يقف
   * السائقُ أمام زرٍّ مُطفأ ولا يعرف أيرجع أم يعيد المحاولة أم يتّصل.
   *
   * والمرحلتان معروفتان سلفاً، فتُجلبان مرّةً واحدة: **حين يضغط تكون
   * القائمةُ عنده.**
   */
  const loadReasons = useCallback(async () => {
    setReasonsErr(false);
    try {
      const pairs = await Promise.all(
        FAIL_AT.map(
          async (at) =>
            [
              at,
              (await api<{ reasons: FailReason[] }>(`/api/v1/driver/fail-reasons?at=${at}`))
                .reasons ?? [],
            ] as const,
        ),
      );
      setReasonsBy(Object.fromEntries(pairs));
    } catch {
      setReasonsErr(true);
    }
  }, []);

  useEffect(() => {
    void loadReasons();
  }, [loadReasons]);

  /**
   * **مهمّةٌ وقعت في يده بلا أن يطلبها — فتُنبّه حتى يتحرّك.**
   *
   * في العرض يضغط «خذ الطلب» **فهو ينظر أصلاً**. وفي الإسناد المباشر يقع
   * الطلبُ في مهامّه وهو على درّاجته لا ينظر، **ورنّةٌ واحدةٌ تضيع تحت
   * الخوذة.** (قرارُ المالك: تنبيهٌ متكرّرٌ لمدّة ٢٠ ثانية.)
   *
   * **ويصمت بمجرّد أن يتحرّك**: `assigned` تعني «لم يمضِ بعد»، وأوّلُ خطوةٍ
   * نحو المتجر تُخرجه منها. **والصوتُ الذي لا يتوقّف عند الفعل يُكتم الجهازُ
   * من أجله** — فلا يُسمع التنبيهُ التالي.
   */
  const waiting = mine.some((o) => o.status === "assigned");
  useRepeatingChime(waiting);

  // **ونبضةُ موضعه مع دوامه** — منها تُقاس المسافةُ إلى المتجر. **ومن أُغلق
  // دوامُه أُغلقت نبضتُه**: تعقّبٌ بلا سبب واستنزافُ بطّارية.
  useLocationBeacon(api, me?.on_shift === true);

  // **حيٌّ**: حالةُ المهمّة تتغيّر بفعل العمليات أيضاً — إلغاءٌ أو إسنادٌ يدويّ
  // — **فشاشةٌ لا تتحدّث تجعل السائقَ يضغط على ما لم يعد قائماً.**
  useLiveRefresh(["order", "wallet"], load);

  async function act(o: DriverOrder, to: string, note = "", failReason = "") {
    setBusy(o.id);
    setError("");
    try {
      await api(`/api/v1/driver/orders/${o.id}/transition`, {
        method: "POST",
        // **والرمزُ يُرسل** — كان يُغفَل، فيردّ الخادمُ `bad_fail_reason`
        // على كلّ محاولةٍ ويبقى الطلبُ قائماً بيد سائقٍ لا يفهم لماذا.
        body: JSON.stringify({ to, note, reason: failReason }),
      });
      load();
    } catch (e) {
      setError(errText(e));
      load();
    } finally {
      setBusy("");
    }
  }

  async function toggleShift() {
    if (!me) return;
    setBusy("shift");
    setError("");
    try {
      await api("/api/v1/driver/shift", {
        method: "POST",
        body: JSON.stringify({ on: !me.on_shift }),
      });
      load();
    } catch (e) {
      setError(errText(e));
    } finally {
      setBusy("");
    }
  }

  if (!me) return <p className="p-6 text-center text-ink-muted">{m.common.loading}</p>;

  const cashRatio = me.cash_limit > 0 ? me.cash_held / me.cash_limit : 0;

  /**
   * أسبابُ مرحلةِ الطلبِ المفتوحةِ نافذتُه — **و`undefined` ليست فارغة**:
   * الأولى «لم تصل»، والثانية «وصلت ولا شيء فيها».
   */
  const stageReasons = failing ? reasonsBy[failing.status] : undefined;

  return (
    /* **تتمدّد بعرض اللوحة** — كان سقفُها `max-w-2xl`: عمودٌ ضيّقٌ في وسط
       شاشةٍ واسعة **ونصفُ العرض فارغ**، والسائقُ على حاسوبٍ يرى بطاقاتٍ
       مقصوصةً بلا سبب. (قرارُ المالك ٢٠٢٦-٠٨-٠٣) */
    <div className="space-y-4">
      {error && (
        <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      {/* الدوام: العلَم الذي يرفعه هو — لا يُستنتج عنه من آخر ظهور */}
      <Card>
        <div className="flex items-center gap-3">
          <span
            className={`flex h-11 w-11 shrink-0 items-center justify-center rounded-badge ${
              me.on_shift ? "bg-success/10 text-success" : "bg-page text-ink-muted"
            }`}
          >
            <IconDriver size={20} />
          </span>
          <div className="min-w-0 flex-1">
            <p className="font-bold">{me.on_shift ? D.shift.on : D.shift.off}</p>
            <p className="text-xs text-ink-muted">
              {me.on_shift && me.shift_started_at
                ? D.shift.since.replace("{t}", fmtTime(me.shift_started_at))
                : D.shift.offHint}
            </p>
          </div>
          <Button
            variant={me.on_shift ? "secondary" : "primary"}
            disabled={busy === "shift"}
            onClick={toggleShift}
          >
            {me.on_shift ? D.shift.end : D.shift.start}
          </Button>
        </div>
      </Card>

      {/* النقد بذمّته — يُعرض دائماً لا عند الامتلاء: الرقم الذي يُرى يُسلَّم */}
      <Card>
        <div className="mb-2 flex items-center justify-between text-sm">
          <span className="flex items-center gap-2 font-medium">
            <IconBalance size={17} className="text-ink-muted" />
            {D.cash.title}
          </span>
          {/* **ما بذمّته يُكتب سالباً — لا رصيداً يملكه.**

              كان يُعرض رقماً موجباً بجانب سقفه، **فيُقرأ كأنّه له**: سائقٌ
              يرى «٥٠٠٠٠» في شاشته لا يقرؤها ديناً عليه إلّا إن قيل له.
              **والمالُ الذي في جيبه ليس ماله** — هو مال المنصة يحمله حتى
              يورّده، **وإشارةُ السالب هي كلُّ الفرق بين الأمرين.** */}
          <span dir="ltr" className={`font-bold ${me.cash_held > 0 ? "text-warning" : ""}`}>
            {me.cash_held > 0 ? "−" : ""}
            {fmtNum(me.cash_held)} / {fmtNum(me.cash_limit)}
          </span>
        </div>
        <div className="h-2 overflow-hidden rounded-badge bg-page">
          <div
            className={`h-full rounded-badge ${
              cashRatio >= 1 ? "bg-danger" : cashRatio >= 0.8 ? "bg-warning" : "bg-success"
            }`}
            style={{ width: `${Math.min(100, cashRatio * 100)}%` }}
          />
        </div>
        <p
          className={`mt-1.5 text-xs ${
            cashRatio >= 1 ? "text-danger" : cashRatio >= 0.8 ? "text-warning" : "text-ink-muted"
          }`}
        >
          {cashRatio >= 1 ? D.cash.full : cashRatio >= 0.8 ? D.cash.near : D.cash.hint}
        </p>
      </Card>

      {/* **ويومُه ما وقع فيه لا ما نجح منه.**

          كانت البطاقتان «مُسلَّمة» و«أجرُك» وحدَهما، **فيومٌ فيه ثلاثُ
          تسليماتٍ وتعذّرٌ واحدٌ يُقرأ ثلاثَ تسليمات** — والسائقُ يعرف أنّه
          وقف عند بابٍ ولم يُسلّم، **فشاشةٌ لا تذكره تُقرأ إخفاءً.**

          **والتعويضُ كان في رصيده ولا في أجر يومه** — رقمان يختلفان عن اليوم
          نفسِه ولا سطرَ يفسّر الفرق. */}
      <StatGrid>
        <StatCard
          icon={IconCheck}
          label={D.today.delivered}
          value={fmtNum(me.today_delivered)}
        />
        <StatCard
          icon={IconWarning}
          label={D.today.failed}
          value={fmtNum(me.today_failed)}
        />
        <StatCard
          icon={IconWallet}
          label={D.today.earned}
          value={`${fmtNum(me.today_earned)} ${m.common.currency}`}
          sub={
            me.today_compensated > 0
              ? `${D.today.ofWhichCompensation}: ${fmtNum(me.today_compensated)}`
              : undefined
          }
        />
      </StatGrid>

      {/* مهامّي */}
      <section>
        <h2 className="mb-2 flex items-center gap-2 font-bold">
          <IconOrder size={18} className="text-ink-muted" />
          {D.tasks.title}
        </h2>
        {mine.length === 0 ? (
          <EmptyState icon={IconOrder} title={D.tasks.empty} action={<span className="text-sm text-ink-muted">{D.tasks.emptyHint}</span>} />
        ) : (
          /* **شبكةٌ لا عمود** — بطاقةُ مهمّةٍ بعرض شاشةٍ كاملةٍ تُبعثر العينَ
             بين طرفيها، **واثنتان في السطر تُقرآن معاً.** */
          <div className="grid gap-3 xl:grid-cols-2">
            {mine.map((o) => (
              <TaskCard
                key={o.id}
                o={o}
                busy={busy === o.id}
                onAct={(to) => {
                  // **الإثباتُ قبل الإغلاق لا بعده.**
                  //
                  // بعد الإغلاق يصير الطلبُ تاريخاً، **وصورةٌ تُضاف إلى تاريخٍ
                  // مغلقٍ تُقرأ إضافةً متأخّرة** — وهي أضعفُ ما يُحتجّ به.
                  if (to === "delivered") setProving(o);
                  else act(o, to);
                }}
                onFail={() => {
                  setReason("");
                  setDetail("");
                  // **والقائمةُ حاضرةٌ سلفاً** — جُلبت مع الشاشة (`loadReasons`).
                  // وإن كان جلبُها قد أخفق تُعاد المحاولةُ من داخل النافذة.
                  if (reasonsErr) void loadReasons();
                  setFailing(o);
                }}
                onRelease={() => act(o, "dispatching")}
                onEmergency={() => setEmergency(o)}
              />
            ))}
          </div>
        )}
      </section>

      {/* **و«طلبات قادمة» صارت قسماً قائماً بذاته** — `/portal/incoming`.
          كانت هنا ذيلاً لمهامّي، **فيقرؤها السائقُ بعد أن يمرّ على مهامّه**،
          وهي أوّلُ ما يحتاجه لا آخرُه. (قرارُ المالك ٢٠٢٦-٠٨-٠٣) */}

      {proving && (
        <ProofModal
          order={proving}
          onClose={() => setProving(null)}
          onDone={() => {
            const o = proving;
            setProving(null);
            if (o) void act(o, "delivered");
          }}
        />
      )}

      {emergency && (
        <EmergencyModal
          order={emergency}
          onClose={() => setEmergency(null)}
          onDone={() => {
            setEmergency(null);
            load();
          }}
        />
      )}

      {/* تعذّر التسليم — **سببٌ مصنَّفٌ يُختار، لا نصٌّ حرٌّ يُكتب.**

          كان الحقلُ نصّاً حرّاً **والخادمُ يرفض كلَّ ما ليس رمزاً من قائمته**
          (`FaultOf(reason) == ""`) — **فكلُّ ضغطةٍ تعود بخطأ والطلبُ لا يتحرّك.**
          وشهده المالكُ في شاشته (٢٠٢٦-٠٨-٠٣): «رغم أنني قلت تعذّر التسليم ما
          زال زرّ سلّمت الطلب يظهر والطلب قائم».

          **والرموزُ تُجلَب من الخادم لا تُكتب هنا**: قائمةٌ في مكانين تفترق
          حين يُضاف سببٌ في أحدهما — **وهي عائلةُ الخلل نفسُها التي تكرّرت في
          أنواع الوسائط وسقف النقد وشرط الساعات.**

          **والذنبُ يُشتقّ من الرمز لا من تقدير أحد** — وهو ما يقرّر التعويض. */}
      <Modal
        open={failing !== null}
        onClose={() => setFailing(null)}
        title={D.act.failedTitle}
      >
        <div className="space-y-3">
          <p className="text-sm font-medium">{D.act.failedReason}</p>
          {/* **أربعُ حالاتٍ لا واحدة.**

              «لم تصل بعد» و«تعذّر الوصولُ إليها» و«وصلت فارغة» ثلاثةُ أشياء،
              **وكانت تُعرض جميعاً بجملةٍ واحدةٍ تقول إنّ الإعدادات خاليةٌ من
              أسبابٍ لهذه المرحلة** — وهي أشدُّها بُعداً عن الحقيقة.

              **والزرُّ مُطفأٌ في الثلاث**، فمن لم يُخبَر أيُّها وقع لم يعرف
              أيرجع أم يعيد المحاولة. */}
          {reasonsErr && !stageReasons ? (
            <div className="space-y-2 rounded-control border border-danger/40 bg-danger/5 px-3 py-2">
              <p className="text-sm text-danger">{D.act.failedReasonsError}</p>
              <Button variant="secondary" onClick={() => void loadReasons()}>
                {m.common.retry}
              </Button>
            </div>
          ) : !stageReasons ? (
            <p className="text-sm text-ink-muted">{m.common.loading}</p>
          ) : stageReasons.length === 0 ? (
            <p className="text-sm text-ink-muted">{D.act.failedNoReasons}</p>
          ) : (
            <div className="space-y-1.5">
              {stageReasons.map((x) => (
                <label
                  key={x.code}
                  className={`flex cursor-pointer items-center gap-2.5 rounded-control border px-3 py-2 text-sm transition-colors ${
                    reason === x.code
                      ? "border-accent bg-accent/10 font-medium"
                      : "border-line hover:border-accent/60"
                  }`}
                >
                  <input
                    type="radio"
                    name="fail-reason"
                    className="accent-accent"
                    checked={reason === x.code}
                    onChange={() => setReason(x.code)}
                  />
                  {/* **ورمزٌ بلا ترجمةٍ يُعرض كما هو** — لا فارغاً. فمن أضاف
                      سبباً في الخادم ونسي القاموسَ يرى نقصَه في الشاشة. */}
                  {m.common.failReasons[x.code as keyof typeof m.common.failReasons] ?? x.code}
                </label>
              ))}
            </div>
          )}
          {/* **والتفصيلُ اختياريّ**: «القائمةُ تُصنّف والنصُّ يشرح»
              (`failreasons.go`). ومن أُلزم بالكتابة على درّاجةٍ تحت الشمس
              كتب حرفاً ليمرّ. */}
          <Input
            id="fail-detail"
            label={D.act.failedDetail}
            value={detail}
            onChange={(e) => setDetail(e.target.value)}
          />
          <p className="text-xs text-ink-muted">{D.act.failedHint}</p>
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setFailing(null)}>
              {m.common.cancel}
            </Button>
            <Button
              variant="danger"
              disabled={!reason || busy !== ""}
              onClick={() => {
                const o = failing;
                setFailing(null);
                if (o) act(o, "failed", detail.trim(), reason);
              }}
            >
              {D.act.failed}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}

/** بطاقة المهمّة — كلُّ ما يحتاجه في مكانٍ واحد، والزرّ الكبير آخرها. */
function TaskCard({
  o,
  busy,
  onAct,
  onFail,
  onRelease,
  onEmergency,
}: {
  o: DriverOrder;
  busy: boolean;
  onAct: (to: string) => void;
  onFail: () => void;
  onRelease: () => void;
  onEmergency: () => void;
}) {
  const next = NEXT[o.status];
  // قبل الاستلام وجهتُه المتجر، وبعده وجهتُه الزبون — الملاحة تتبع الرحلة
  const heading = o.status === "assigned" || o.status === "at_pickup";
  const cash = o.cash_due > 0;

  return (
    <Card>
      <div className="mb-3 flex items-center gap-2">
        <Badge variant="primary">#{fmtRef(o.number)}</Badge>
        <Badge variant="neutral">{m.orders.status[o.status as keyof typeof m.orders.status]}</Badge>
        <span className="ms-auto text-sm font-bold" dir="ltr">
          {fmtNum(o.total)} {m.common.currency}
        </span>
      </div>

      {/* **المسافتان — ورقمٌ يُقرأ قراراً.**

          بلاهما يفتح الخريطةَ لكلّ بطاقةٍ ليعرف أيُّها أقرب. **و«إليك ٤٠٠ م»
          يقول له بأيّها يبدأ**، و«المشوار ٣٫٢ كم» يقول كم سيأخذ.

          **وسالبٌ يعني «لا تُعرف» لا «صفر»**: الجهلُ ليس قرباً — ومن أطفأ
          الموقعَ يُقال له ذلك بدل أن يُعرض عليه رقمٌ كاذب. */}
      <div className="mb-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-ink-muted">
        {o.to_pickup_m >= 0 ? (
          <span className="flex items-center gap-1">
            <IconDriver size={13} />
            {D.distance.toPickup}: {fmtDistance(o.to_pickup_m, UNITS.meter, UNITS.km)}
          </span>
        ) : (
          <span className="flex items-center gap-1">
            <IconDriver size={13} />
            {D.distance.unknown}
          </span>
        )}
        {o.leg_m >= 0 && (
          <span className="flex items-center gap-1">
            <IconLocation size={13} />
            {D.distance.leg}: {fmtDistance(o.leg_m, UNITS.meter, UNITS.km)}
          </span>
        )}
      </div>

      <Leg
        icon={IconStore}
        label={o.pickup_lat != null ? D.order.pickupOverride : D.order.pickup}
        name={o.pickup_lat != null ? D.order.pickupFromDriver : o.merchant_name}
        detail={
          o.pickup_lat != null
            ? o.pickup_note || D.order.pickupOverrideHint
            : D.order.items.replace("{n}", fmtNum(o.items_count))
        }
        href={o.pickup_lat != null ? mapsHref(o.pickup_lat, o.pickup_lng ?? 0) : undefined}
        hrefLabel={o.pickup_lat != null ? D.order.navigate : undefined}
        phone={o.merchant_phone}
        callLabel={D.order.callMerchant}
        dim={!heading}
      />
      <Leg
        icon={IconLocation}
        label={D.order.dropoff}
        name={o.customer_name}
        detail={o.address_text}
        phone={o.customer_phone}
        callLabel={D.order.callCustomer}
        href={mapsHref(o.lat, o.lng)}
        hrefLabel={D.order.navigate}
        dim={heading}
      />

      {/* ما يقبضه: يُقال قبل التسليم لا بعده */}
      <p
        className={`mt-3 flex items-center gap-2 rounded-control px-3 py-2 text-sm ${
          cash ? "bg-warning/10 font-bold text-warning" : "bg-page text-ink-muted"
        }`}
      >
        <IconBalance size={16} />
        {cash ? (
          <>
            {D.order.cashCollect}:{" "}
            <span dir="ltr">
              {fmtNum(o.cash_due)} {m.common.currency}
            </span>
          </>
        ) : (
          D.order.walletPaid
        )}
      </p>

      {/* **ولا بطاقةَ لطلبٍ مُغلَق.**

          كانت بطاقةُ `failed` تبقى حتى تُحسم بضاعتُها، **فتزاحم البطاقاتِ
          التي فيها فعلٌ ولا فعلَ فيها هي**: متجرٌ لا يستردّ يترك بطاقةً لا
          تُغلق أبداً — أوّلُ ما يراه كلَّ صباح.

          (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «مهمّاتٌ تفشل التسليم لا يجب أن تبقى
          بالمهام لدى السائق — خلص، تُعتبر مغلقةً منتهية».)

          **والقائمةُ لم تعد تحمله أصلاً** (`handleDriverOrders`). */}
      {next && (
        <Button
          size="lg"
          className="mt-3 w-full"
          disabled={busy}
          onClick={() => onAct(next)}
        >
          {D.act[next as keyof typeof D.act]}
        </Button>
      )}

      <div className="mt-2 flex items-center justify-between">
        {/* **الإعادةُ للطابور متاحةٌ حتى الاستلام — لا حتى الوصول.**

            كانت تختفي بمجرّد أن يقول «وصلتُ المتجر»، **فمن عرض له عارضٌ وهو
            عند الباب لم يبقَ له إلّا زرُّ «تعذّر التسليم»**: يُقفل طلباً
            بضاعتُه لم تخرج بعد، ويُحسب ذنبٌ لم يقع، **ويُحرم زبونٌ من طلبٍ
            كان سائقٌ آخر يوصله في دقائق.**

            **وحدُّها الاستلام**: بعد أن تصير البضاعةُ في يده لا يُترك الطلبُ
            لغيره. */}
        {o.status === "assigned" || o.status === "at_pickup" ? (
          <Button variant="ghost" disabled={busy} onClick={onRelease} title={D.order.releaseHint}>
            {D.order.release}
          </Button>
        ) : (
          <span />
        )}
        {/* **المخرجُ عند الطرفين — لا عند الزبون وحده.**

            كان الإفشالُ متاحاً عند باب الزبون فقط. فلو وصل السائقُ إلى المطعم
            ووجده **مغلقاً**، أو لم تصله الرسالةُ أصلاً، أو رفض التحضير —
            **بقي الطلبُ معلّقاً بلا نهاية ممكنة**: مالُ الزبون محجوز، والسائقُ
            مربوطٌ بطلبٍ لا يُقفل وسقفُه النقديّ مشغولٌ به.

            **وهو من هناك، فهو من يقول.** واللفظُ يختلف بالموضع: عند المطعم
            «المطعم لم يسلّمني»، وعند الزبون «الزبون لم يستلم». و«فشل» وحدها
            تُخفي ثلاثة أخطاءٍ في ثلاث جهات. */}
        <span className="flex items-center gap-1">
          {(FAIL_AT as readonly string[]).includes(o.status) && (
            <Button variant="ghost" disabled={busy} onClick={onFail} className="text-danger">
              <span className="flex items-center gap-1.5">
                <IconWarning size={15} />
                {o.status === "at_pickup" ? D.act.failedAtPickup : D.act.failed}
              </span>
            </Button>
          )}
          {/* **الطارئ — لما لا يحتمل شاشة.**

              من وقع له حادثٌ وهو حاملٌ الطعام لم يكن يملك إلّا «تعذّر
              التسليم»: **يُقفل طلبٌ كان يمكن أن يصل**، ويُحسب ذنبٌ لم يقع،
              **ويبقى هو مشغولاً بشاشةٍ وهو في حالٍ لا تحتمل الشاشات.**

              **وضغطةٌ واحدة لا ثلاث**: موقعُه يُلتقط، والعملياتُ تُنبَّه،
              والطلبُ يعود إلى الطابور، ودوامُه يُغلق. **وكلُّ حقلٍ نطلبه منه
              في تلك اللحظة حقلٌ لن يُملأ** — فيُترك الزرُّ ولا يُضغط. */}
          <Button variant="ghost" disabled={busy} onClick={onEmergency} className="text-danger">
            <span className="flex items-center gap-1.5">
              <IconWarning size={15} />
              {D.act.emergency}
            </span>
          </Button>
        </span>
      </div>
    </Card>
  );
}

/** طرفُ الرحلة — المتجر أو الزبون. الطرف الذي ليس وجهتَه الآن يبهت. */
function Leg({
  icon: Icon,
  label,
  name,
  detail,
  phone,
  callLabel,
  href,
  hrefLabel,
  dim,
}: {
  icon: React.ComponentType<{ size?: number; className?: string }>;
  label: string;
  name: string;
  detail: string;
  phone: string | null;
  callLabel: string;
  href?: string;
  hrefLabel?: string;
  dim: boolean;
}) {
  return (
    <div className={`flex items-start gap-3 py-1.5 ${dim ? "opacity-50" : ""}`}>
      <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-control bg-page">
        <Icon size={16} className="text-ink-muted" />
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-xs text-ink-muted">{label}</p>
        <p className="truncate font-medium">{name}</p>
        <p className="truncate text-xs text-ink-muted">{detail}</p>
      </div>
      <div className="flex shrink-0 gap-1">
        {href && (
          <a
            href={href}
            target="_blank"
            rel="noreferrer"
            title={hrefLabel}
            aria-label={hrefLabel}
            className="flex h-9 w-9 items-center justify-center rounded-control bg-primary-light text-primary-dark"
          >
            <IconLocation size={17} />
          </a>
        )}
        {phone && (
          <a
            href={`tel:${phone}`}
            title={callLabel}
            aria-label={callLabel}
            className="flex h-9 w-9 items-center justify-center rounded-control bg-page text-ink-muted"
          >
            <IconPhone size={17} />
          </a>
        )}
      </div>
    </div>
  );
}

/**
 * **الطارئ** — ضغطةٌ واحدة لا ثلاث.
 *
 * من كُسرت يدُه لا يملأ نموذجاً. **وكلُّ حقلٍ نطلبه منه في تلك اللحظة حقلٌ لن
 * يُملأ** — فيُترك الزرُّ ولا يُضغط، **ويبقى الطلبُ معه ولا نعلم.**
 *
 * فالملاحظةُ اختيارية، **والموقعُ يُلتقط بلا أن يُطلب**، وتعذّرُه لا يوقف
 * شيئاً: جهازٌ مرفوضُ الإذن أو داخلَ بناءٍ لا إشارةَ فيه — **وطارئٌ يُردّ لأن
 * الموقعَ لم يُقرأ طارئٌ ضاع.** والعملياتُ تتّصل به فتعرف أين هو.
 */
function EmergencyModal({
  order,
  onClose,
  onDone,
}: {
  order: DriverOrder;
  onClose: () => void;
  onDone: () => void;
}) {
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit() {
    setBusy(true);
    setError("");
    // **الموقعُ بمهلة**: انتظارٌ بلا حدٍّ يجعل الزرَّ يبدو معطّلاً في اللحظة
    // التي يجب أن يعمل فيها أسرعَ ما يكون.
    const point = await new Promise<{ lat: number; lng: number } | null>((resolve) => {
      if (!navigator.geolocation) return resolve(null);
      navigator.geolocation.getCurrentPosition(
        (p) => resolve({ lat: p.coords.latitude, lng: p.coords.longitude }),
        () => resolve(null),
        { timeout: 6000, enableHighAccuracy: true },
      );
    });
    try {
      await api(`/api/v1/driver/orders/${order.id}/emergency`, {
        method: "POST",
        body: JSON.stringify({ lat: point?.lat ?? null, lng: point?.lng ?? null, note: note.trim() }),
      });
      onDone();
    } catch (e) {
      setError(errText(e));
      setBusy(false);
    }
  }

  return (
    <Modal open title={D.emergency.title} onClose={onClose}>
      <div className="space-y-3">
        <p className="text-sm text-ink-muted">{D.emergency.hint}</p>
        <Input
          label={D.emergency.note}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder={D.emergency.notePlaceholder}
        />
        {error && <p className="text-sm text-danger">{error}</p>}
        <div className="flex gap-2">
          <Button variant="danger" disabled={busy} onClick={submit}>
            {busy ? D.emergency.sending : D.emergency.confirm}
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
 * **إثباتُ التسليم** — صورةٌ وإحداثياتٌ ووقت.
 *
 * # لماذا
 *
 * قاعدتُنا أنّ **الزبونَ يُصدَّق أوّلَ مرّة** — والمنصةُ تتحمّل. **وقاعدةٌ بلا
 * دليلٍ تكلفةٌ بلا سقف**: من عرف أنّ كلمتَه تكفي قالها مرّةً بعد مرّة، **ولا
 * يبقى للسائق الصادق ما يدفع به عن نفسه.**
 *
 * # والمعيارُ العالميّ ثلاثةٌ لا واحد
 *
 * الصورةُ تُظهر **موضعَ التسليم** لا الطردَ وحدَه · **والإحداثياتُ تُلتقط
 * آلياً** فتثبت أنّه كان هناك · والوقتُ آليٌّ كذلك.
 *
 * **وأهمُّها الإحداثيات**: صورةُ بابٍ قد تكون لأيّ باب.
 *
 * # ولا يقف التسليمُ على كاميرا
 *
 * هاتفٌ لا يعمل، أو إذنٌ مرفوض، أو ليلٌ لا يُرى فيه شيء — **وسائقٌ لا يستطيع
 * إنهاء طلبٍ سلّمه فعلاً يقف في الشارع.** فالتخطّي متاحٌ **بكلمةٍ تُقرأ يومَ
 * النزاع.**
 */
function ProofModal({
  order,
  onClose,
  onDone,
}: {
  order: DriverOrder;
  onClose: () => void;
  onDone: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [skipping, setSkipping] = useState(false);
  const [reason, setReason] = useState("");

  /** الموقعُ بمهلة — انتظارٌ بلا حدٍّ يجعل الزرَّ يبدو معطّلاً. */
  async function where(): Promise<{ lat: number; lng: number } | null> {
    return new Promise((resolve) => {
      if (!navigator.geolocation) return resolve(null);
      navigator.geolocation.getCurrentPosition(
        (p) => resolve({ lat: p.coords.latitude, lng: p.coords.longitude }),
        () => resolve(null),
        { timeout: 6000, enableHighAccuracy: true },
      );
    });
  }

  async function upload(file: File) {
    setBusy(true);
    setError("");
    const point = await where();
    const fd = new FormData();
    fd.append("file", file);
    if (point) {
      fd.append("lat", String(point.lat));
      fd.append("lng", String(point.lng));
    }
    try {
      const base = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${base}/api/v1/driver/orders/${order.id}/proof`, {
        method: "POST",
        credentials: "include",
        body: fd,
      });
      if (!res.ok) throw new Error(String(res.status));
      onDone();
    } catch {
      setError(m.errors.internal);
      setBusy(false);
    }
  }

  async function skip() {
    if (!reason.trim()) return setError(D.proof.reasonRequired);
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/driver/orders/${order.id}/proof/skip`, {
        method: "POST",
        body: JSON.stringify({ reason: reason.trim() }),
      });
      onDone();
    } catch (e) {
      setError(errText(e));
      setBusy(false);
    }
  }

  return (
    <Modal open title={D.proof.title} onClose={onClose}>
      <div className="space-y-4">
        <p className="text-sm text-ink-muted">{D.proof.hint}</p>

        {!skipping ? (
          <>
            {/* **الكاميرا مباشرةً لا معرضُ الصور.**

                `capture` تفتح آلةَ التصوير — **وصورةٌ تُختار من المعرض قد تكون
                لأيّ يومٍ ولأيّ باب**، فتسقط حجّتُها في أوّل نزاع. */}
            <label className="flex w-full cursor-pointer items-center justify-center gap-2 rounded-control border-2 border-dashed border-primary/50 py-6 text-primary-dark">
              <IconCamera size={20} />
              <span className="font-medium">{busy ? D.proof.sending : D.proof.take}</span>
              <input
                type="file"
                accept="image/*"
                capture="environment"
                disabled={busy}
                className="hidden"
                onChange={(e) => {
                  const f = e.target.files?.[0];
                  if (f) void upload(f);
                }}
              />
            </label>
            {error && <p className="text-sm text-danger">{error}</p>}
            <button
              type="button"
              onClick={() => setSkipping(true)}
              className="w-full text-xs text-ink-muted underline"
            >
              {D.proof.cannot}
            </button>
          </>
        ) : (
          <>
            <Input
              id="pod-reason"
              label={D.proof.reason}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder={D.proof.reasonHint}
            />
            {error && <p className="text-sm text-danger">{error}</p>}
            <div className="flex gap-2">
              <Button variant="secondary" disabled={busy} onClick={skip}>
                {D.proof.skipConfirm}
              </Button>
              <Button variant="ghost" onClick={() => setSkipping(false)}>
                {m.common.cancel}
              </Button>
            </div>
          </>
        )}
      </div>
    </Modal>
  );
}
