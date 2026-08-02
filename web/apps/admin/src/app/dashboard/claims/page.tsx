"use client";

/**
 * **نزاعاتُ المتاجر** — ما دفعته المنصةُ بسببهم وتُطالِبهم به.
 *
 * # القرار
 *
 * **«نعم، المنصة تعوّضه — وبفتح نزاع مع المتجر لحلّ القصة.»** (المالك،
 * ٢٠٢٦-٠٨-٠٣)
 *
 * السائقُ يُعوَّض **لحظتَها** فلا ينتظر نزاعاً ليُقبض له، **والمطالبةُ تُحسم
 * هنا.**
 *
 * # ولماذا لا تُخصم آلياً
 *
 * **الخصمُ قرارُ إنسانٍ بعد أن يسمع المتجر.** وقد يكون العذرُ حقّاً: انقطعت
 * الكهرباء، أو جاءه الطلبُ ولم يُبلَّغ. **ومالٌ يخرج من محفظةِ متجرٍ قبل أن
 * يُسأل نزاعٌ خُسر قبل أن يُفتح** — يبقى المالُ ويذهب المتجر.
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDateTime } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  Input,
  Modal,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  StatGrid,
  StatCard,
  useLiveData,
  IconStore,
  IconWallet,
  IconStatus,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const C = m.admin.claims;
const REASONS: Record<string, string> = m.merchant.warnings.reasons;

interface Claim {
  id: string;
  merchant_id: string;
  merchant_name: string;
  reason: string;
  claim_amount: number;
  order_number: number | null;
  created_at: string;
}

export default function ClaimsPage() {
  const { data, reload } = useLiveData<{ claims: Claim[]; total: number }>(
    () => api("/api/v1/admin/claims"),
    ["merchant", "wallet"],
  );
  const [target, setTarget] = useState<{ c: Claim; how: "charged" | "waived" } | null>(null);

  if (!data) return <LoadingState />;
  const list = data.claims ?? [];

  return (
    <PageContainer>
      <PageHeader icon={IconStore} title={C.title} subtitle={C.hint} />

      <StatGrid>
        <StatCard
          label={C.total}
          value={fmtNum(data.total)}
          icon={IconWallet}
          tone={data.total > 0 ? "accent" : "default"}
        />
        <StatCard label={C.count} value={fmtNum(list.length)} icon={IconStore} />
      </StatGrid>

      {list.length === 0 ? (
        <EmptyState icon={IconStatus} title={C.empty} />
      ) : (
        <div className="space-y-2">
          {list.map((c) => (
            <div
              key={c.id}
              className="flex flex-wrap items-center gap-3 rounded-card border border-line bg-surface p-4"
            >
              <div className="min-w-0 flex-1">
                <p className="font-bold">{c.merchant_name}</p>
                <p className="text-xs text-ink-muted">
                  {REASONS[c.reason] ?? c.reason}
                  {c.order_number !== null && (
                    <>
                      {" · "}
                      {m.terms.order} <span dir="ltr">#{fmtNum(c.order_number)}</span>
                    </>
                  )}
                  {" · "}
                  <span dir="ltr">{fmtDateTime(c.created_at)}</span>
                </p>
              </div>
              <Badge variant="warning">
                <span dir="ltr">{fmtNum(c.claim_amount)}</span> {m.common.currency}
              </Badge>
              <Button variant="secondary" onClick={() => setTarget({ c, how: "charged" })}>
                {C.charge}
              </Button>
              <Button variant="ghost" onClick={() => setTarget({ c, how: "waived" })}>
                {C.waive}
              </Button>
            </div>
          ))}
        </div>
      )}

      {target && (
        <SettleModal
          claim={target.c}
          how={target.how}
          onClose={() => setTarget(null)}
          onDone={() => {
            setTarget(null);
            reload();
          }}
        />
      )}
    </PageContainer>
  );
}

function SettleModal({
  claim,
  how,
  onClose,
  onDone,
}: {
  claim: Claim;
  how: "charged" | "waived";
  onClose: () => void;
  onDone: () => void;
}) {
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit() {
    // **والسببُ إلزاميٌّ في الحالين.** إعفاءٌ بلا كلمةٍ لا يُراجَع، **وخصمٌ بلا
    // كلمةٍ يجده المتجرُ في محفظته ولا يعرف عمّاذا.**
    if (!note.trim()) return setError(C.noteRequired);
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/claims/${claim.id}/settle`, {
        method: "POST",
        body: JSON.stringify({ settlement: how, note: note.trim() }),
      });
      onDone();
    } catch (e) {
      const key = e instanceof ApiError ? (e.body.message_key.split(".").pop() ?? "") : "";
      setError((m.errors as Record<string, string>)[key] ?? m.errors.internal);
      setBusy(false);
    }
  }

  return (
    <Modal
      open
      title={how === "charged" ? C.chargeTitle : C.waiveTitle}
      onClose={onClose}
    >
      <div className="space-y-3">
        <p className="text-sm text-ink-muted">
          {claim.merchant_name} — <span dir="ltr">{fmtNum(claim.claim_amount)}</span>{" "}
          {m.common.currency}
        </p>
        <p className="text-xs text-ink-muted">
          {how === "charged" ? C.chargeHint : C.waiveHint}
        </p>
        <Input label={C.note} value={note} onChange={(e) => setNote(e.target.value)} />
        {error && <p className="text-sm text-danger">{error}</p>}
        <div className="flex gap-2">
          <Button variant={how === "charged" ? "primary" : "secondary"} disabled={busy} onClick={submit}>
            {how === "charged" ? C.chargeConfirm : C.waiveConfirm}
          </Button>
          <Button variant="ghost" onClick={onClose}>
            {m.common.cancel}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
