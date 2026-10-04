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
  Input,
  LoadingState,
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
  created_at: string;
  due_at: string | null;
  approve_path: string;
  reject_path: string;
  href: string;
  can_approve: boolean;
  self_approval: boolean;
}

export function ApprovalsTab() {
  const { data, error, reload } = useLiveData<{ items: ApprovalItem[]; count: number; total: number }>(
    () => api(`/api/v1/admin/approvals`),
    ["wallet"],
  );
  const [approve, setApprove] = useState<ApprovalItem | null>(null);
  const [reject, setReject] = useState<ApprovalItem | null>(null);
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [fail, setFail] = useState("");

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
                <span className="font-bold" dir="ltr">
                  {fmtMoney(it.amount)}
                </span>
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
                        <Button className="!px-2.5" disabled={busy || notDue} onClick={() => setApprove(it)}>
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
                  .replace("{amount}", fmtMoney(approve.amount))
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
