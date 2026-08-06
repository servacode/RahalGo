"use client";

/**
 * صفحة المحفظة — **نسخة واحدة مركزية** لكل من له رصيد.
 *
 * كانت مبنيّةً مرّتين: مرّة في بوابة المندوب ومرّة في موقع الزبون — 574 سطراً
 * لمنطقٍ واحد. وكل تحسين كان يُنفَّذ مرّتين أو يُنسى في إحداهما، وهذا كيف تنشأ
 * الفروق التي لا يقصدها أحد. (GROUND-RULES §1.2: يُبنى مرة هنا ويرثه الجميع.)
 *
 * ما يختلف بين الأدوار **مُعامِلات لا نسخ**: مسار النقطة، ونصّ عنوان الرصيد،
 * وهل يملك صاحبُه طلبَ سحب.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDate, fmtTime } from "@rahalgo/i18n";
import { Badge, Button, Input, Modal } from "./components";
import { Alert } from "./feedback";
import { PageContainer, PageHeader, Card, EmptyState, LoadingState, ListRow, TabCards } from "./layout";
import { DataView, ViewToggle, useViewMode, type DataColumn } from "./dataview";
import type { TabItem } from "./layout";
import { StatementSheet, currentMonthRange, type StatementData } from "./Statement";
import { useLiveData } from "./Notifications";
import {
  IconArrowIn,
  IconArrowOut, IconWallet, IconWarning, IconPrint } from "./icons";

const m = getMessages(defaultLocale);
const P = m.shared.payout;
const W = m.shared.walletTabs;
const KIND_LABELS: Record<string, string> = m.shared.txKinds;

const ALL = "all";
const REQUESTS = "requests";
const STATEMENT = "statement";

type ApiFn = <T>(path: string, init?: RequestInit) => Promise<T>;

interface Tx {
  id: string | number;
  kind: string;
  amount: number;
  note: string;
  created_at: string;
  /**
   * **رقمُ الطلب — يُرسله الخادمُ وكانت الواجهةُ ترميه.**
   *
   * فتعرض شظيّةَ المعرّف من نصّ الملاحظة: «دفع طلب ‎#1d448c90» — وهي حروفٌ
   * لا يعرفها صاحبُ المحفظة ولا يجدها في شيء. **ورقمٌ لا يُبحث به ليس
   * رقماً، هو ضجيج.**
   */
  order_number?: number | null;
  ticket_number?: number | null;
}

interface Payout {
  id: string;
  amount: number;
  status: "pending" | "paid" | "rejected";
  note: string;
  decision: string;
  created_at: string;
}

const STATUS_VARIANT: Record<Payout["status"], "warning" | "success" | "danger"> = {
  pending: "warning",
  paid: "success",
  rejected: "danger",
};

function errText(err: unknown): string {
  const key =
    typeof err === "object" && err && "body" in err
      ? ((err as { body?: { message_key?: string } }).body?.message_key ?? "").split(".").pop() ?? ""
      : "";
  return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
}

export function WalletPage({
  api,
  path,
  balanceLabel,
  payouts = false,
  holderName,
  holderPhone,
  width = "full",
}: {
  api: ApiFn;
  /** نقطة كشف المحفظة — تختلف بالدور: `/api/v1/rep/wallet` أو `/api/v1/my/wallet` */
  path: string;
  balanceLabel: string;
  /**
   * **`hint` لم تعد تُعرض** — وتبقى في النوع كي لا تنكسر خمسُ صفحاتٍ تمرّرها.
   *
   * كانت جملةً تحت الرصيد: «المحفظة اختيارية…» — **تُقرأ مرّةً ثمّ تشغل
   * بطاقةَ الرصيد كلَّ يوم.** (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «بلاها».)
   */
  hint?: string;
  /** هل يملك صاحب الحساب طلب سحب رصيده (المندوب نعم، الزبون لا) */
  payouts?: boolean;
  holderName?: string;
  holderPhone?: string;
  width?: "full" | "medium";
}) {
  const [tab, setTab] = useState(ALL);
  const [asking, setAsking] = useState(false);

  const { data: statement, loading, reload } = useLiveData<{ balance: number; transactions: Tx[] }>(
    () => api(path),
    ["wallet"],
  );
  const { data: requests, reload: reloadPayouts } = useLiveData<Payout[]>(
    () => (payouts ? api("/api/v1/me/payouts") : Promise.resolve([] as Payout[])),
    ["wallet"],
  );

  // كشف الحساب يُجلب بمداه الخاص — لا يشتقّ من لمحة الصفحة (آخر 50) وإلا كان
  // مستنداً مالياً ناقصاً بصمت.
  const [range, setRange] = useState(currentMonthRange());
  const [sheet, setSheet] = useState<StatementData | null>(null);
  const [sheetLoading, setSheetLoading] = useState(false);

  useEffect(() => {
    if (tab !== STATEMENT) return;
    setSheetLoading(true);
    api<StatementData>(`${path}?from=${range.from}&to=${range.to}`)
      .then(setSheet)
      .catch(() => setSheet(null))
      .finally(() => setSheetLoading(false));
  }, [api, path, tab, range]);

  const refreshAll = useCallback(() => {
    reload();
    reloadPayouts();
  }, [reload, reloadPayouts]);

  const txs = useMemo(() => statement?.transactions ?? [], [statement]);
  const reqs = useMemo(() => requests ?? [], [requests]);

  /** **وتفضيلُ العرضِ محفوظٌ لصاحبه** — جدولاً أو بطاقات. */
  const [view, setView] = useViewMode("wallet-tx");

  /**
   * **أعمدةُ الحركة — تعريفٌ واحدٌ يغذّي الجدولَ والبطاقات.**
   *
   * **والمبلغُ أوّلُ ما يُقرأ**: صاحبُ المحفظة ينظر كم دخل وكم خرج، **والنوعُ
   * يشرح لماذا.** فكلاهما `primary` — وهما ما تعرضه البطاقةُ في رأسها.
   */
  const txColumns = useMemo<DataColumn<Tx>[]>(
    () => [
      {
        id: "kind",
        header: m.shared.txCard.kind,
        icon: <IconWallet />,
        primary: true,
        cell: (tx) => (
          <span className="flex items-center gap-2">
            <span
              aria-hidden
              className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-control ${
                tx.amount >= 0 ? "bg-success-tint text-success" : "bg-danger-tint text-danger"
              }`}
            >
              {tx.amount >= 0 ? <IconArrowIn size={14} /> : <IconArrowOut size={14} />}
            </span>
            <span className="min-w-0 truncate">{KIND_LABELS[tx.kind] ?? tx.kind}</span>
          </span>
        ),
      },
      {
        id: "amount",
        header: m.shared.txCard.amount,
        primary: true,
        cell: (tx) => (
          /* **والإشارةُ تُكتب ولا يُترك اللونُ وحدَه** — من لا يفرّق الأخضرَ
             عن الأحمر يقرأ الرقمَ ولا يعرف أدخل أم خرج. */
          <span
            dir="ltr"
            className={`font-bold tabular-nums ${tx.amount >= 0 ? "text-success" : "text-danger"}`}
          >
            {tx.amount >= 0 ? "+" : "−"}
            {fmtNum(Math.abs(tx.amount))} <span className="text-2xs font-normal opacity-70">{m.common.currency}</span>
          </span>
        ),
      },
      {
        id: "about",
        header: m.shared.txCard.about,
        /* **ورقمُ الطلب معرّفٌ لا مبلغ — بلا فاصلةِ آلاف.** */
        cell: (tx) =>
          tx.order_number ? (
            <span dir="ltr" className="tabular-nums">#{tx.order_number}</span>
          ) : tx.ticket_number ? (
            <span dir="ltr" className="tabular-nums">#{tx.ticket_number}</span>
          ) : (
            <span className="text-ink-muted">—</span>
          ),
      },
      {
        id: "when",
        header: m.shared.txCard.when,
        cell: (tx) => (
          <span dir="ltr" className="tabular-nums text-ink-muted">
            {fmtDate(tx.created_at)} · {fmtTime(tx.created_at)}
          </span>
        ),
      },
      {
        id: "note",
        header: m.shared.txCard.note,
        block: true,
        hide: (tx) => !tx.note,
        cell: (tx) => <span className="text-ink-muted">{tx.note}</span>,
      },
    ],
    [],
  );

  /* ══════════════════════════════════════════════════════════════════
     **ولا بطاقاتِ تصنيفٍ فوق الجدول — الرصيدُ والحركاتُ وحدَهما**
     ══════════════════════════════════════════════════════════════════

     (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «الكروت الذكيّة ما لها لازمة، فقط المحفظة
      والجدول للحركات».)

     # ما كان

     بطاقةٌ لكلِّ نوعٍ من الحركات بعددِه ومجموعِه: «شحن رصيد ‎+٢٥٠٬٠٠٠» ·
     «دفع طلب ‎−١٢٤٬٠٠٠» · «استرجاع ‎+٧٧٬٠٠٠» — **وهي تقول ما يقوله عمودُ
     النوع في الجدول تحتها**، سطراً سطراً وبتاريخِه ومرجعِه.

     # ولماذا كان ضرراً لا تلخيصاً

     **الجدولُ صار يحمل التصنيفَ في عموده الأوّل** — فبطاقةٌ تصفّي على
     «استرجاع» تُخفي أربعةَ أخماسِ السجلّ لتُظهر ما كان يُقرأ بنظرةٍ إلى
     العمود. **ومرشِّحٌ يحذف ما لا يُشتكى منه ليس ربحاً.**

     **وأربعُ بطاقاتٍ تدفع الجدولَ — وهو ما جاء صاحبُ المحفظة ليقرأه —
     تحت الطيّة.**

     # وما بقي تنقّلٌ لا إحصاء

     **«طلبات السحب» قائمةٌ أخرى ليست في الجدول أصلاً** — وحذفُها يُخفيها
     عن الدرّاج والتاجر والمندوب **بلا بابٍ إليها.** فتبقى ومعها «الكلّ»
     لِيُرجَع منها، **ولا تظهر أصلاً لمن لا سحبَ له** (الزبونُ والإدارة) —
     **فيرى هؤلاء الرصيدَ والجدولَ ولا شيءَ بينهما.**
     ══════════════════════════════════════════════════════════════════ */
  const tabs = useMemo<TabItem[]>(
    () =>
      reqs.length
        ? [
            // ولا عددَ على «الكلّ» ولا مجموع: **بطاقةُ رجوعٍ لا لوحةُ إحصاء.**
            { key: ALL, label: W.all },
            // طلبات السحب بلا مجموع عمداً: خلط المعلّق بالمدفوع بالمرفوض في رقم
            // واحد يعطي مبلغاً لا يعني شيئاً — العدد وحده هو الصادق هنا.
            { key: REQUESTS, label: W.requests, count: reqs.length },
          ]
        : [],
    [reqs],
  );

  if (loading) return <LoadingState />;

  const balance = statement?.balance ?? 0;
  const hasPending = reqs.some((p) => p.status === "pending");
  // التبويب المختار قد يختفي بعد تحديث حيّ (آخر طلب سحب أُلغي) — نرتدّ للكل
  const current = tab === STATEMENT || tabs.some((t) => t.key === tab) ? tab : ALL;

  const statementBtn = (
    /* **كشفُ الحساب بتعبئةٍ خضراءَ مصمتةٍ لا بالنبرة** — وهو لونُ زرّ
       الطباعة نفسِه: **كلاهما إخراجُ ورقة**، والنبرةُ للفعل الذي يُنشئ
       شيئاً. (قرارُ المالك ٢٠٢٦-٠٨-٠٣) */
    <Button
      onClick={() => setTab(tab === STATEMENT ? ALL : STATEMENT)}
      /* **ونصُّه داكنٌ لا أبيض**: الأبيضُ على الأخضر المصمت **٣٫٥١** — دون
         حدّ النصّ العاديّ. **والداكنُ ٤٫٨٠.** (كشفه جردُ ٢٠٢٦-٠٨-٠٦.) */
      className="flex items-center gap-2 !bg-success-solid !text-on-bright hover:!opacity-90"
    >
      <IconPrint size={16} />
      {m.shared.statement.open}
    </Button>
  );

  return (
    <PageContainer width={width}>
      <PageHeader
        icon={IconWallet}
        title={m.terms.wallet}
        actions={
          <>
            {/* كشف الحساب إجراء لا تصنيف: لا رقم له فلا مكان له بين البطاقات */}
            {statementBtn}
            {payouts && (
              <Button onClick={() => setAsking(true)} disabled={balance <= 0 || hasPending}>
                {P.request}
              </Button>
            )}
          </>
        }
      />

      {/* ══════════════════════════════════════════════════════════════
          **الرصيدُ مع البطاقات الذكيّة في صفٍّ واحدٍ أعلى الصفحة**
          ══════════════════════════════════════════════════════════════

          (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «الكروت الذكيّة بالأعلى، تحطّ المحفظة».)

          **كان الرصيدُ وحدَه في الأعلى والبطاقاتُ الذكيّةُ مدفونةً داخل بطاقة
          المعاملات** — فيقرأ صاحبُ المحفظة رقماً واحداً، **ثمّ ينزل ليكتشف
          أنّ ثمّة تصنيفاً بأرقامٍ أخرى.**

          **والأرقامُ التي تُقارَن تُوضع متجاورة**: الرصيدُ وما دخل وما خرج
          وعمولاتُه — **نظرةٌ واحدةٌ تقول الحال.**
          ══════════════════════════════════════════════════════════════ */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-stretch">
        {/* الرصيد — الرقم الذي يهمّ صاحبه أولاً */}
        {/* **بطاقةُ الرصيد بالنبرة** — وهي أهمُّ رقمٍ في الصفحة.
            والأخضرُ صار خلفيةَ كلّ شيء، **فبطاقةٌ بلون البطاقة لا تُميَّز**؛
            والنبرةُ تقطعها بإشباعٍ ضِعفين ونصفٍ فيقع الرصيدُ في العين أوّلاً.
            **ونصُّها داكنٌ لا أبيض**: الأبيضُ على النبرة ١٫٣٠ يذوب، والداكنُ
            ١٣٫٩٢. (قرارُ المالك ٢٠٢٦-٠٨-٠٣، ولوناً ٢٠٢٦-٠٨-٠٦) */}
        {/* **وبقدرِ ما تقول لا بقدرِ أهميّتها.**

            كانت لوحاً يملأ عرضَ الشاشة لسطرين: **عنوانٌ ورقم**. والمساحةُ
            الفارغةُ حولهما لا تزيدهما وضوحاً، **وتدفع سجلَّ الحركات — وهو ما
            جاء الزبونُ ليقرأه — إلى ما تحت الطيّة.**

            فصارت سطراً واحداً: العنوانُ والرقمُ متجاورين، **والشرحُ تحتهما
            بخطٍّ صغير.**

            **ولا تمتدّ بعرض الصفحة**: رقمٌ من ستّة أرقامٍ في لوحٍ عرضُه شاشةٌ
            كاملة **يترك فراغاً لا يقول شيئاً**، وتُقرأ البطاقةُ بحجم ما فيها لا
            بحجم ما حولها. (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «كرت المحفظة كبير، اجعله
            صغيراً مناسباً · اجعل بادينغ للكرت، لا تجعله بامتداد الصفحة».) */}
        <div className="w-full rounded-card bg-accent px-4 py-3 text-center text-on-bright sm:mx-0 sm:w-auto sm:min-w-56 sm:text-start">
          <p className="text-sm font-medium opacity-90">{balanceLabel}</p>
          {/* **والشرحُ حُذف**: «المحفظة اختيارية…» جملةٌ تُقرأ مرّةً ثمّ تبقى
              تشغل بطاقةَ الرصيد كلَّ يوم. **وما يُقال مرّةً لا يُكتب دائماً.**
              (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «بلاها».) */}
          <p className="mt-0.5 text-2xl font-bold" dir="ltr">
            {fmtNum(balance)} <span className="text-sm font-normal">{m.common.currency}</span>
          </p>
        </div>


        {/* **وما بقي من التبويب يملأ ما بعد الرصيد** — «طلبات السحب» وحدَها
            لمن له سحب. **ولا شيءَ هنا عند الزبون والإدارة**، فيقع الجدولُ
            مباشرةً تحت الرصيد. */}
        {current !== STATEMENT && tabs.length > 0 && (
          <div className="min-w-0 flex-1">
            <TabCards items={tabs} active={current} onChange={setTab} />
          </div>
        )}
      </div>

      <Card
        title={m.terms.transactions}
        icon={IconWallet}
        /* **ومبدّلُ العرض في ترويسة البطاقة.** بلاه يبقى الجدولُ بلا طريقٍ
           إلى البطاقات — **ومن يفتحها على جوّالٍ يقرأ جدولاً بأربعة أعمدةٍ في
           ثلاثمئةٍ وستّين.** (والتفضيلُ يُحفظ فلا يُعاد اختيارُه كلَّ زيارة.) */
        actions={current !== STATEMENT ? <ViewToggle
              view={view}
              onChange={setView}
              tableLabel={m.common.viewTable}
              cardsLabel={m.common.viewCards}
            /> : undefined}
      >
        {/* **وذهب شرحُ النوع مع مرشِّحاته** — كان سطراً يشرح «استرجاع» لمن
            ضغط بطاقتَها، **ولا بطاقةَ الآن تُضغط.** (ونصوصُه حُذفت من المعجم
            كذلك: **نصٌّ يُترجَم ويُراجَع ولا يُرسَم دَينٌ صامت.**) */}

        {current === STATEMENT ? (
          <StatementSheet
            data={sheet}
            loading={sheetLoading}
            from={range.from}
            to={range.to}
            onFrom={(from) => setRange((r) => ({ ...r, from }))}
            onTo={(to) => setRange((r) => ({ ...r, to }))}
            onQuick={(from, to) => setRange({ from, to })}
            holderName={holderName}
            holderPhone={holderPhone}
          />
        ) : current === REQUESTS ? (
          <ul className="space-y-2">
            {reqs.map((p) => (
              <ListRow
                key={p.id}
                title={
                  <span dir="ltr">
                    {fmtNum(p.amount)} {m.common.currency}
                  </span>
                }
                subtitle={p.decision || p.note || undefined}
                trailing={
                  <>
                    <Badge variant={STATUS_VARIANT[p.status]}>{P.status[p.status]}</Badge>
                    <span className="text-xs text-ink-muted" dir="ltr">
                      {fmtDate(p.created_at)}
                    </span>
                  </>
                }
              />
            ))}
          </ul>
        ) : txs.length === 0 ? (
          <EmptyState icon={IconWallet} title={m.terms.noTransactions} />
        ) : (
          // لا شريط مجموع هنا: البطاقة النشطة تعرضه فوق — تكراره ضجيج
          /* ══════════════════════════════════════════════════════════
             **جدولٌ وبطاقاتٌ من المركز — لا قائمةٌ مبنيّةٌ باليد**
             ══════════════════════════════════════════════════════════

             (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «نسوّيها جدولاً احترافيّاً».)

             **والقاعدةُ الهندسيّةُ تسبق الطلب** (`GROUND-RULES` §٢): «كلُّ
             قائمةٍ تُعرض جدولاً وبطاقاتٍ عبر `DataView` المركزيّ — تعريفُ
             أعمدةٍ واحدٌ يغذّي الوضعين، **وممنوعٌ بناءُ جدولٍ يدويّ**.»

             **وكانت المحفظةُ تخالفها**: بطاقاتٌ مكتوبةٌ بيدها (`TxCard`)
             **بلا جدولٍ أصلاً** — فمن أراد أن يقارن عشرين حركةً يقرأ عشرين
             صندوقاً بدل عشرين سطر.

             **وتفضيلُ العرض يُحفظ لكلّ مستخدم** (`useViewMode`) — فمن اختار
             الجدولَ مرّةً لا يُعيد اختيارَه كلَّ زيارة. */
          <DataView
            items={txs}
            view={view}
            getKey={(tx) => String(tx.id)}
            empty={m.shared.txCard.empty}
            columns={txColumns}
          />
        )}
      </Card>

      {asking && (
        <PayoutModal
          api={api}
          balance={balance}
          onClose={() => setAsking(false)}
          onDone={() => {
            setAsking(false);
            refreshAll();
          }}
        />
      )}
    </PageContainer>
  );
}

function PayoutModal({
  api,
  balance,
  onClose,
  onDone,
}: {
  api: ApiFn;
  balance: number;
  onClose: () => void;
  onDone: () => void;
}) {
  const [amount, setAmount] = useState(String(balance));
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    const value = Number(amount);
    if (!Number.isFinite(value) || value <= 0 || value > balance) {
      return setError(m.errors.insufficient_balance);
    }
    setBusy(true);
    try {
      await api("/api/v1/me/payouts", {
        method: "POST",
        body: JSON.stringify({ amount: value, note }),
      });
      onDone();
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={P.title}>
      <form onSubmit={submit} className="space-y-4">
        <div>
          <Input
            id="payout-amount"
            label={P.amount}
            type="number"
            dir="ltr"
            min="1"
            max={balance}
            required
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            className="text-center text-lg font-bold"
          />
          <button
            type="button"
            onClick={() => setAmount(String(balance))}
            className="mt-1.5 text-sm font-medium text-primary hover:underline"
          >
            {P.all} ({fmtNum(balance)} {m.common.currency})
          </button>
        </div>

        <Input
          id="payout-note"
          label={P.note}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder={P.notePlaceholder}
        />

        {error && (
          <Alert>{error}</Alert>
        )}

        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button type="submit" disabled={busy}>
            {busy ? m.common.loading : P.submit}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
