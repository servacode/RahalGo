"use client";

/**
 * **الأهداف والمكافآت** — من بلغ، وبكم تكافئه.
 *
 * # لماذا الهدفُ يُقرأ ولا يدفع
 *
 * **رقمٌ يدفع بلا يدٍ لا يُراجَع.** ومن بلغ الهدفَ بثلاثين طلباً صغيراً ليس
 * كمن بلغه في ليالي المطر — **والفرقُ يراه إنسانٌ ولا تراه معادلة.** وخطأٌ
 * في رقمٍ تلقائيٍّ يُصرف على الجميع قبل أن يُلاحظ.
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٥: «بيدك، والشاشةُ تقول من بلغ».)
 *
 * # والعقوبةُ في الشاشة نفسِها
 *
 * **لا شاشةٌ للثواب وأخرى للعقاب**: من يفتح ملفَّ سائقٍ يقرّر في الاثنين
 * بنظرةٍ واحدة، **وقرارٌ يحتاج شاشتين يُؤجَّل.**
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, errorText } from "@rahalgo/i18n";
import {
  Tabs,
  type TabDef,
  Alert,
  PageContainer,
  PageHeader,
  Button,
  Badge,
  Modal,
  Input,
  EmptyState,
  LoadingState,
  ReloadState,
  IconStar,
  IconDriver,
  IconUser,
  FormActions,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const P = m.admin.incentives;

interface Standing {
  user_id: string;
  name: string;
  phone: string;
  done: number;
  target: number;
  reached: boolean;
  rewarded: number;
  penalized: number;
}

type Role = "driver" | "sales";

export default function IncentivesPage() {
  const [role, setRole] = useState<Role>("driver");
  /**
   * ══════════════════════════════════════════════════════════════════
   * **والفشلُ ليس فراغاً — والفراغُ ليس فشلاً**
   * ══════════════════════════════════════════════════════════════════
   *
   * (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.)
   *
   * **كان `.catch(() => setRows([]))`** — **فيُقرأ «لا أحدَ بلغ الهدفَ
   * هذا الشهر» حين تفشل القراءة.** وهو استنتاجٌ **يُبنى عليه قرارُ صرفِ
   * مال.**
   *
   * **وشاشةُ الأحداث تحمل هذا الدرسَ مكتوباً**: «وسجلٌّ فارغٌ على خطأٍ
   * شهادةُ زور».
   */
  const [rows, setRows] = useState<Standing[] | null | "failed">(null);
  const [error, setError] = useState("");
  const [granting, setGranting] = useState<Standing | null>(null);
  const [kind, setKind] = useState<"reward" | "penalty">("reward");
  const [amount, setAmount] = useState("");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");

  const load = useCallback(() => {
    setRows(null);
    setError("");
    api<{ standings: Standing[] }>(`/api/v1/admin/incentives/${role}`)
      .then((r) => setRows(r.standings ?? []))
      .catch((err) => {
        setRows("failed");
        // **وسببُ الخادم بنصّه** — **ورسالةٌ واحدةٌ لكلّ العلل تُسكت ما
        // يُفيد.**
        setError(errorText(err));
      });
  }, [role]);

  useEffect(load, [load]);

  async function submit() {
    const who = granting;
    if (!who) return;
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/users/${who.user_id}/incentive`, {
        method: "POST",
        body: JSON.stringify({
          kind,
          amount: Number(amount),
          reason: reason.trim(),
          // **وعن الهدف حين يكون قد بلغه** — يُقرأ لاحقاً في تقاريره،
          // فيُعرف كم صُرف على الأهداف وكم على وقائعَ بعينها.
          for_target: kind === "reward" && who.reached,
        }),
      });
      setGranting(null);
      setNotice(P.sent);
      load();
    } catch (e) {
      // **ونسخةٌ ثانيةٌ من الترجمة تفترق عن المركزيّة يوماً** — يُضاف
      // مفتاحٌ في إحداهما ويُنسى في الأخرى.
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  const TABS: TabDef<Role>[] = [
    { key: "driver", label: P.tabDrivers, icon: IconDriver },
    { key: "sales", label: P.tabSales, icon: IconUser },
  ];

  return (
    <PageContainer>
      <PageHeader icon={IconStar} title={P.title} subtitle={P.subtitle} />

      <Tabs items={TABS} value={role} onChange={setRole} />

      {error && <p className="text-sm text-danger">{error}</p>}
      {notice && (
        <Alert tone="success">{notice}</Alert>
      )}

      {rows === null ? (
        <LoadingState />
      ) : rows === "failed" ? (
        /* **وتعذّرُ القراءة يُقال ويُعاد** — **ولا يُقرأ «لا أحدَ بلغ»**،
           وهو استنتاجٌ يُبنى عليه قرارُ صرفِ مال. */
        <ReloadState onRetry={load} />
      ) : rows.length === 0 ? (
        <EmptyState icon={IconUser} title={P.empty} />
      ) : (
        <div className="surface sm:overflow-x-auto">
          {/* **وعلى الجوّال بطاقاتٌ لا أعمدة** — انظر `.table-stack` في
              `theme.css`. (شهده المالك ٢٠٢٦-٠٨-١١: الزرُّ مقصوصٌ خارجَ
              البطاقة، وخمسةُ أعمدةٍ لا تتّسع في ٣٦٠ بكسلاً.) */}
          <table className="table-stack w-full text-sm">
            <thead className="border-b border-line-soft text-ink-muted">
              <tr>
                <th className="p-3 text-start font-medium">{P.person}</th>
                <th className="p-3 text-start font-medium">{P.progress}</th>
                <th className="p-3 text-start font-medium">{P.rewarded}</th>
                <th className="p-3 text-start font-medium">{P.penalized}</th>
                <th className="p-3" />
              </tr>
            </thead>
            <tbody>
              {rows.map((x) => (
                <tr key={x.user_id} className="border-b border-line-soft last:border-0">
                  <td className="p-3" data-label={P.person}>
                    <span className="font-medium">{x.name || x.phone}</span>
                    {x.reached && (
                      <Badge variant="success" className="ms-2">
                        {m.shared.incentives.reached}
                      </Badge>
                    )}
                  </td>
                  {/* **وبلا هدفٍ يُقال ذلك** — لا يُعرض «٣ / ٠» فيُقرأ لغزاً. */}
                  <td className="p-3 tabular-nums" dir="ltr" data-label={P.progress}>
                    {x.target > 0 ? (
                      `${fmtNum(x.done)} / ${fmtNum(x.target)}`
                    ) : (
                      <span className="text-xs text-ink-muted">{P.noTarget}</span>
                    )}
                  </td>
                  <td className="p-3 tabular-nums text-success" dir="ltr" data-label={P.rewarded}>
                    {fmtNum(x.rewarded)}
                  </td>
                  <td className="p-3 tabular-nums text-danger" dir="ltr" data-label={P.penalized}>
                    {fmtNum(x.penalized)}
                  </td>
                  <td className="p-3 text-end">
                    <Button
                      variant="secondary"
                      onClick={() => {
                        setKind("reward");
                        setAmount("");
                        setReason("");
                        setNotice("");
                        setGranting(x);
                      }}
                    >
                      {P.grant}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Modal
        open={granting !== null}
        onClose={() => setGranting(null)}
        title={`${P.grantTitle} — ${granting?.name || granting?.phone || ""}`}
      >
        <div className="space-y-3">
          <p className="text-sm font-medium">{P.kind}</p>
          <div className="flex gap-1">
            {(["reward", "penalty"] as const).map((k) => (
              <button
                key={k}
                type="button"
                onClick={() => setKind(k)}
                className={`rounded-control border px-3 py-1.5 text-sm transition-colors ${
                  kind === k
                    ? "border-accent bg-accent-tint font-medium"
                    : "border-line text-ink-muted hover:border-accent-edge"
                }`}
              >
                {k === "reward" ? m.shared.incentives.reward : m.shared.incentives.penalty}
              </button>
            ))}
          </div>

          <Input
            id="incentive-amount"
            label={P.amount}
            type="number"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
          {/* **والسببُ إلزاميٌّ في الخادم** — مالٌ يخرج بلا كلمةٍ لا يُراجَع
              ولا يُقاس من يُكثره. **ويراه صاحبُه**، فمن عوقب يعرف بماذا. */}
          <Input
            id="incentive-reason"
            label={P.reason}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />

          <FormActions onSave={() => void submit()} onCancel={() => setGranting(null)} saveLabel={P.send} />
        </div>
      </Modal>
    </PageContainer>
  );
}
