"use client";

/**
 * **إنذاراتُ الحساب — تُقرأ وتُوجَّه من مكانٍ واحد.**
 *
 * (شكوى المالك ٢٠٢٦-٠٨-٠٩: «وجّه إنذار للسائق… بس ما وصل الإنذار».)
 *
 * # ولماذا في ملفّ الحساب لا في شاشة الشكوى
 *
 * **الإنذارُ سجلٌّ على شخصٍ لا ردٌّ على واقعة.** ومن يقرّر أن يُنذر يسأل أوّلاً:
 * **كم مرّةً سبق؟** — والجوابُ هنا، في ملفّه، مع رصيده وطلباته وتقييماته.
 *
 * **وشاشةُ الشكوى تعالج شكوى**: تُقرأ وتُحلّ وتُغلق. **وإنذارٌ يُوجَّه من داخلها
 * يُقرأ عقوبةً على تلك الشكوى** — بينما هو حكمٌ على سجلٍّ كامل.
 *
 * # ويعمل لكلّ الأدوار
 *
 * **سائقٌ ومتجرٌ ومندوبٌ وزبون** (قرارُ المالك: «حتّى الزبون — ممكن يكون تعامل
 * بسوء مع السائق، بالنهاية كمان السائق بشر ويقدّم خدمة»).
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtDateTime, fmtNum, errorText } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  Select,
  Textarea,
  Alert,
  Modal,
  FormSection,
  IconWarning,
  FormActions,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";

const m = getMessages(defaultLocale);
const W = m.admin.warnings;
/** **وأسماءُ الأسباب معجمٌ يُفهرس بالرمز** — والرمزُ يأتي من المحرّك. */
const REASON_LABELS: Record<string, string> = W.reasons;
/** **وأسبابُ إنذار المتجر معجمٌ آخر** — **ومعجمٌ واحدٌ يترك نصفَها بلا اسم.** */
const STORE_REASONS: Record<string, string> = m.merchant.warnings.reasons;

interface WarningItem {
  id: string;
  reason: string;
  note: string;
  role_code: string;
  order_number: number | null;
  created_at: string;
}

export function WarningsSection({ userID }: { userID: string }) {
  const [rows, setRows] = useState<WarningItem[]>([]);
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState("");
  // ══════════════════════════════════════════════════════════════════
  // **وأسبابُ الإنذار من المحرّك — بدور صاحب الحساب**
  // ══════════════════════════════════════════════════════════════════
  //
  // (قرارُ المالك ٢٠٢٦-٠٨-١٥: «يجب أن يكون هناك أسبابٌ جاهزةٌ
  //  للإنذار».)
  //
  // **ونصٌّ حرٌّ يُكتب السببُ الواحدُ به بعشرة ألفاظ** — **و«أُنذر
  // ثلاثاً لنفس السبب» جملةٌ لا تُقال** إن كان كلُّ إنذارٍ بلفظ.
  const [reasons, setReasons] = useState<string[]>([]);
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [alertN, setAlertN] = useState(0);
  const [error, setError] = useState("");
  // **وتعذّرُ القراءة يُقال لا يُقرأ «لا إنذارات»** (قرارُ المالك ٢٠٢٦-١٠-٠٤): موظّفُ
  // الماليّة كان يرى «لا إنذارات» على سائقٍ عنده ثلاثة.
  const [loadError, setLoadError] = useState("");
  const canCall = useCanCall();
  const canRead = canCall("GET", "/users/{id}/warnings");
  const canIssue = canCall("POST", "/users/{id}/warnings");

  const load = useCallback(async () => {
    if (!canRead) return;
    try {
      const r = await api<{ warnings: WarningItem[] }>(
        `/api/v1/admin/users/${userID}/warnings`,
      );
      setRows(r.warnings ?? []);
      setLoadError("");
    } catch (err) {
      // **وتعذّرُ القراءة لا يُسقط الملفّ** — ويُقال سببُه.
      setRows([]);
      setLoadError(errorText(err));
    }
  }, [userID, canRead]);

  useEffect(() => {
    void load();
  }, [load]);

  // **وتُجلب عند فتح النافذة** — لا مع كلّ فتحةِ ملفّ.
  useEffect(() => {
    if (!open || reasons.length > 0) return;
    void api<{ reasons: { code: string }[] }>(
      `/api/v1/admin/users/${userID}/warn-reasons`,
    )
      .then((r) => setReasons((r.reasons ?? []).map((x) => x.code)))
      .catch((e) => setError(errorText(e)));
  }, [open, reasons.length, userID]);

  async function issue() {
    // **والسببُ إلزاميّ** — إنذارٌ بلا سببٍ لا يُصحَّح ولا يُحتجّ به.
    if (!reason.trim() || busy) return;
    setBusy(true);
    setError("");
    try {
      const res = await api<{ recent_30d?: number }>(`/api/v1/admin/users/${userID}/warnings`, {
        method: "POST",
        body: JSON.stringify({ reason: reason.trim(), note: note.trim() }),
      });
      // **ولا إيقافَ آليّاً** — تنبيهٌ يقول العدد، والقرارُ للموظّف.
      setAlertN(res?.recent_30d ?? 0);
      setReason("");
      setNote("");
      setOpen(false);
      await load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  if (!canRead) {
    return (
      <FormSection title={W.title} icon={<IconWarning />}>
        <Alert tone="warning">{m.admin.acc.notAllowed}</Alert>
      </FormSection>
    );
  }

  return (
    <FormSection title={W.title} icon={<IconWarning />}>
      {loadError && <Alert className="mb-2">{loadError}</Alert>}
      {alertN >= 3 && (
        <Alert tone="warning" className="mb-2">
          {m.admin.acc.warnAlert.replace("{n}", fmtNum(alertN))}
        </Alert>
      )}
      {loadError ? null : rows.length === 0 ? (
        <p className="py-4 text-center text-sm text-ink-muted">{W.empty}</p>
      ) : (
        <ul className="mb-3 space-y-1.5">
          {rows.map((x) => (
            <li
              key={x.id}
              className="flex items-start justify-between gap-3 rounded-control bg-field px-3 py-2"
            >
              <div className="min-w-0 flex-1">
                {/* **والرمزُ يُترجَم عند القراءة** — المحرّكُ يخزّن
                    `abuse_driver`، **ومن عرضه كما هو أرى المكتبَ
                    إنكليزيّةً في لوحةٍ عربيّة.** (شكوى المالك
                    ٢٠٢٦-٠٨-١٥.)
                    **والقديمُ يُقرأ كما كُتب**: إنذاراتٌ سُجّلت نصّاً
                    حرّاً قبل القائمة لا رمزَ لها، **فتُعرض بنصّها.** */}
                <p className="flex flex-wrap items-center gap-2 text-sm font-medium">
                  {/* **وإنذارُ المتجر يُميَّز** — (قرارُ المالك ٢٠٢٦-٠٨-١٦).

                      **جدولان لا واحد**: على الحساب وعلى المتجر —
                      **ولكلٍّ أسبابُه.** ومن خلطهما لم يعرف **أيَّ
                      شيءٍ يُحظر**: الإنسانُ أم اللافتة. */}
                  {x.role_code === "store" && <Badge variant="warning">{W.onStore}</Badge>}
                  {REASON_LABELS[x.reason] ?? STORE_REASONS[x.reason] ?? x.reason}
                </p>
                {x.note && <p className="mt-0.5 text-xs text-ink-muted">{x.note}</p>}
              </div>
              <span className="shrink-0 text-2xs text-ink-muted">
                {fmtDateTime(x.created_at)}
              </span>
            </li>
          ))}
        </ul>
      )}

      {canIssue && (
        <Button variant="secondary" onClick={() => setOpen(true)}>
          {W.issue}
        </Button>
      )}

      <Modal open={open} onClose={() => setOpen(false)} title={W.issue}>
        <div className="space-y-3">
          {/* **ويُقال ما يعنيه** — الإنذارُ يصل صاحبَه بنصّه. */}
          <p className="text-sm text-ink-muted">{W.hint}</p>
          {/* **وسببٌ من قائمةٍ لا نصٌّ حرّ** — ومن السبب يُشتقّ
              التكرارُ والحكم. **والأسبابُ تتبع دورَ صاحب الحساب**:
              «رفضُ الاستلام» لا يُنذَر به سائق. */}
          <Select
            id="warn-reason"
            label={W.reason}
            required
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          >
            <option value="">{W.pickReason}</option>
            {reasons.map((c) => (
              <option key={c} value={c}>
                {REASON_LABELS[c] ?? c}
              </option>
            ))}
          </Select>
          <Textarea
            id="warn-note"
            label={W.note}
            rows={3}
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
          {error && <Alert>{error}</Alert>}
          <FormActions onSave={() => void issue()} onCancel={() => setOpen(false)} saveLabel={busy ? m.common.loading : W.issue} />
        </div>
      </Modal>
    </FormSection>
  );
}
