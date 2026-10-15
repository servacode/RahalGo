"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الموافقاتُ الموحّدة** — قرارُ المالك ٢٠٢٦-١٠-٠٤ (الخزينة)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كلُّ ما ينتظر قراراً من كلّ قسمٍ في قائمةٍ واحدة**، والزرُّ ينادي بابَ القسم
 * نفسَه — فحكمُ «المقترحُ غيرُ الموافق» والقيدُ بطرفين يبقيان في القسم.
 * **وصاحبُ الاقتراح لا يرى زرَّ الموافقة** (`can_approve` من المحرّك)، إلّا المالكُ
 * وحدَه حين لا يوجد غيرُه — ويُعلَّم.
 *
 * **والتعويضُ يُوافَق من هنا** (قرارُ المالك ٢٠٢٦-١٠-١٥) — بمبلغٍ يُعدَّل (افتراضُه المقترح)
 * وملاحظةٍ إلزاميّة، **ببابه نفسِه** (`{amount, note}`): فوق السقف مديرُ المنصّة، وكلمةُ السرّ
 * يطلبها العميلُ وحدَه حين يردّ المحرّكُ بالتحدّي. **والمقترحُ صفراً «بلا مبلغ بعد»** — لا
 * يدخل المجموع.
 */

import { useState } from "react";
import Link from "next/link";
import { fmtMoney, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Confirm,
  EmptyState,
  FormActions,
  Input,
  LoadingState,
  Modal,
  ReloadState,
  IconCheck,
  useLiveData,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { T, sectionLabel } from "./shared";


export interface ApprovalItem {
  key: string;
  section: string;
  id: string;
  amount: number;
  note: string;
  proposed_by: string;
  proposer_name: string;
  /** لمن يذهب المالُ أو على من يقع — فارغٌ حيث لا طرف. */
  party_name?: string;
  created_at: string;
  due_at: string | null;
  approve_path: string;
  reject_path: string;
  href: string;
  can_approve: boolean;
  self_approval: boolean;
  /** **المبلغُ مقترحٌ يُعدَّل عند الموافقة** — تُرسَل `{amount, note}`. */
  amount_editable?: boolean;
  /** **«بلا مبلغ بعد»** — مقترحٌ صفرٌ خارجَ المجموع. */
  amount_undecided?: boolean;
}

/** **مبلغُ السطر** — أو «بلا مبلغ بعد»، و«مقترح» لما يُعدَّل عند الموافقة. */
export function ApprovalAmount({ it }: { it: ApprovalItem }) {
  if (it.amount_undecided) return <Badge variant="warning">{T.approvals.undecided}</Badge>;
  return (
    <span className="inline-flex items-center gap-1">
      <span className="font-bold" dir="ltr">
        {fmtMoney(it.amount)}
      </span>
      {it.amount_editable && <span className="text-2xs text-ink-muted">{T.approvals.suggested}</span>}
    </span>
  );
}

export function ApprovalsTab() {
  const { data, error, reload } = useLiveData<{
    items: ApprovalItem[];
    count: number;
    total: number;
    undecided_count?: number;
  }>(
    () => api(`/api/v1/admin/approvals`),
    ["wallet"],
  );
  const [approve, setApprove] = useState<ApprovalItem | null>(null);
  const [reject, setReject] = useState<ApprovalItem | null>(null);
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [fail, setFail] = useState("");
  const [comp, setComp] = useState<ApprovalItem | null>(null);
  const [compAmount, setCompAmount] = useState("");
  const [compNote, setCompNote] = useState("");

  function openComp(it: ApprovalItem) {
    setComp(it);
    setCompAmount(it.amount > 0 ? String(it.amount) : "");
    setCompNote("");
  }

  async function approveComp() {
    if (!comp) return;
    setBusy(true);
    try {
      await api(comp.approve_path, {
        method: "POST",
        body: JSON.stringify({ amount: Math.round(Number(compAmount)), note: compNote.trim() }),
      });
      setComp(null);
      setFail("");
      reload();
    } catch (err) {
      setFail(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  async function decide(it: ApprovalItem, ok: boolean) {
    setApprove(null);
    setReject(null);
    setBusy(true);
    try {
      await api(ok ? it.approve_path : it.reject_path, {
        method: "POST",
        body: JSON.stringify({ note: ok ? "" : note.trim() }),
      });
      setNote("");
      setFail("");
      reload();
    } catch (err) {
      setFail(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  if (error && !data) return <ReloadState onRetry={reload} />;
  if (!data) return <LoadingState variant="text" />;
  const items = data.items ?? [];
  const now = Date.now();

  return (
    <div className="space-y-3">
      {fail && <Alert>{fail}</Alert>}
      <p className="text-sm text-ink-muted">
        {T.approvals.count}: <span dir="ltr">{data.count}</span> · {T.approvals.total}:{" "}
        <span dir="ltr" className="font-bold">
          {fmtMoney(data.total)}
        </span>
        {(data.undecided_count ?? 0) > 0 && (
          <> · {T.approvals.undecidedCount.replace("{n}", String(data.undecided_count))}</>
        )}
      </p>
      {items.length === 0 ? (
        <EmptyState icon={IconCheck} title={T.approvals.empty} />
      ) : (
        <ul className="divide-y divide-line surface">
          {items.map((it) => {
            const notDue = it.due_at !== null && new Date(it.due_at).getTime() > now;
            return (
              <li key={`${it.key}-${it.id}`} className="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2 text-sm">
                <Badge variant="neutral">{sectionLabel(it.section)}</Badge>
                <ApprovalAmount it={it} />
                {it.party_name && <span className="font-medium">{T.approvals.party.replace("{name}", it.party_name)}</span>}
                <span className="min-w-0 flex-1 truncate text-ink-muted">{it.note}</span>
                <span className="shrink-0 text-xs text-ink-muted">
                  {T.approvals.proposedBy}: {it.proposer_name || T.approvals.system} ·{" "}
                  <span dir="ltr">{fmtDateTime(it.created_at)}</span>
                </span>
                <span className="flex shrink-0 flex-wrap items-center gap-1.5">
                  {it.approve_path ? (
                    <>
                      {notDue && it.due_at && (
                        <span className="text-xs text-ink-muted">
                          {T.approvals.notDue} · <span dir="ltr">{fmtDateTime(it.due_at)}</span>
                        </span>
                      )}
                      {it.can_approve ? (
                        <Button
                          className="!px-2.5"
                          disabled={busy || notDue}
                          onClick={() => (it.amount_editable ? openComp(it) : setApprove(it))}
                        >
                          {T.approvals.approve}
                        </Button>
                      ) : (
                        <span className="text-xs text-ink-muted">{T.approvals.own}</span>
                      )}
                      <Button
                        variant="secondary"
                        className="!px-2.5"
                        disabled={busy || (it.key === "cash_shortfalls" && notDue)}
                        onClick={() => setReject(it)}
                      >
                        {T.approvals.reject}
                      </Button>
                    </>
                  ) : (
                    <Link href={it.href} className="text-primary underline">
                      {T.approvals.open}
                    </Link>
                  )}
                </span>
              </li>
            );
          })}
        </ul>
      )}

      <Confirm
        open={approve !== null}
        tone="primary"
        title={T.approvals.approveTitle}
        body={
          approve && (
            <div className="space-y-2">
              <p>
                {T.approvals.approveBody
                  .replace(
                    "{amount}",
                    approve.party_name
                      ? `${fmtMoney(approve.amount)} ${T.approvals.party.replace("{name}", approve.party_name)}`
                      : fmtMoney(approve.amount),
                  )
                  .replace("{section}", sectionLabel(approve.section))}
              </p>
              {approve.self_approval && <Alert tone="warning">{T.approvals.selfFlag}</Alert>}
            </div>
          )
        }
        confirmLabel={T.approvals.approve}
        busy={busy}
        onConfirm={() => approve && void decide(approve, true)}
        onCancel={() => setApprove(null)}
      />
      <Modal open={comp !== null} onClose={() => setComp(null)} title={T.approvals.compTitle}>
        {comp && (
          <div className="space-y-3">
            <p className="text-sm">
              {sectionLabel(comp.section)}
              {comp.party_name ? ` · ${T.approvals.party.replace("{name}", comp.party_name)}` : ""}
            </p>
            <p className="text-sm text-ink-muted">
              {comp.amount > 0
                ? T.approvals.compHint.replace("{amount}", fmtMoney(comp.amount))
                : T.approvals.compNoSuggestion}
            </p>
            {comp.self_approval && <Alert tone="warning">{T.approvals.selfFlag}</Alert>}
            {fail && <Alert>{fail}</Alert>}
            <Input
              id="ap-comp-amount"
              type="number"
              inputMode="numeric"
              dir="ltr"
              label={T.approvals.compAmount}
              value={compAmount}
              onChange={(e) => setCompAmount(e.target.value)}
            />
            <Input id="ap-comp-note" label={T.approvals.compNote} value={compNote} onChange={(e) => setCompNote(e.target.value)} />
            <FormActions
              busy={busy || !(Number(compAmount) > 0) || compNote.trim() === ""}
              saveLabel={T.approvals.approve}
              onCancel={() => setComp(null)}
              onSave={() => void approveComp()}
            />
          </div>
        )}
      </Modal>
      <Confirm
        open={reject !== null}
        title={T.approvals.rejectTitle}
        body={<Input id="ap-reject-note" label={T.approvals.rejectNote} value={note} onChange={(e) => setNote(e.target.value)} />}
        confirmLabel={T.approvals.reject}
        busy={busy}
        onConfirm={() => reject && void decide(reject, false)}
        onCancel={() => setReject(null)}
      />
      <p className="text-xs text-ink-muted">{T.statement.tz}</p>
    </div>
  );
}
