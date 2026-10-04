"use client";

/**
 * **نافذةُ سبب الإيقاف أو الحظر.**
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠٤.) **تقول للموظّف لحظةَ الضغط الفرقَ بين الإيقاف والحظر**،
 * **وما بيد الشخص الآن** (طلبٌ مفتوحٌ ونقدٌ مع سائق) — ليعرف ما يتابعه بعدها.
 * **وخطأُ المحرّك يُعرض فيها** — كانت تبقى مفتوحةً بلا تفسيرٍ إن رُدّ الفعل
 * (كإيقاف حساب المالك المحميّ).
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtMoney, errorText } from "@rahalgo/i18n";
import { Alert, Input, Modal, FormActions } from "@rahalgo/ui";

const m = getMessages(defaultLocale);
const U = m.admin.users;
const A = m.admin.acc;

export function StatusReasonModal({
  status,
  onSubmit,
  onClose,
  holds,
}: {
  /** `suspended` إيقافٌ مؤقّت · `blocked` حظرٌ نهائيّ. */
  status: string;
  onSubmit: (reason: string) => void | Promise<void>;
  onClose: () => void;
  /** **ما بيده الآن** — طلباتٌ مفتوحةٌ ونقدٌ بذمّته (للسائق). */
  holds?: { orders: number; cash: number };
}) {
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const label = status === "suspended" ? U.suspend : U.block;
  const explain = status === "suspended" ? A.statusExplain.suspended : A.statusExplain.blocked;

  return (
    <Modal open onClose={onClose} title={U.statusReasonTitle.replace("{action}", label)}>
      <form
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          setError("");
          try {
            await onSubmit(reason.trim());
          } catch (err) {
            setError(errorText(err));
          } finally {
            setBusy(false);
          }
        }}
        className="space-y-4"
      >
        <p className="rounded-control bg-field px-3 py-2 text-xs leading-relaxed text-ink-muted">{explain}</p>
        {holds && (holds.orders > 0 || holds.cash > 0) && (
          <Alert tone="warning">
            {A.holdsNow.replace("{orders}", fmtNum(holds.orders)).replace("{cash}", fmtMoney(holds.cash))}
          </Alert>
        )}
        <Input
          id="status-reason"
          label={U.statusReasonLabel}
          required
          autoFocus
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        {error && <Alert>{error}</Alert>}
        {/* **والحظرُ أحمرُ والإيقافُ ليس كذلك** — فعلان في نافذةٍ واحدةٍ
            ومعناهما مختلف، **ولونٌ واحدٌ لهما يجعل الضغطةَ قرعة.** */}
        <FormActions
          submit
          onCancel={onClose}
          busy={busy}
          saveLabel={label}
          tone={status === "blocked" ? "danger" : undefined}
        />
      </form>
    </Modal>
  );
}

export default StatusReasonModal;
