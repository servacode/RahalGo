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
 * وملاحظةٍ إلزاميّة، **ببابه نفسِه** (`{amount, note}`): فوق السقف المالكُ وحدَه، وكلمةُ السرّ
 * يطلبها العميلُ وحدَه حين يردّ المحرّكُ بالتحدّي. **والمقترحُ صفراً «بلا مبلغ بعد»** — لا
 * يدخل المجموع.
 */

import { useState, type ComponentType } from "react";
import Link from "next/link";
import { fmtMoney, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Confirm,
  FormActions,
  Input,
  LoadingState,
  Modal,
  ReloadState,
  IconCheck,
  IconWallet,
  IconWarning,
  IconBlock,
  IconBalance,
  IconHandshake,
  IconReceipt,
  IconNote,
  IconStar,
  IconPromos,
  IconShieldX,
  Chips,
  useLiveData,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanCall } from "@/lib/policy";
import { T, sectionLabel, EmptyCard } from "./shared";
import { Tile } from "@/components/admin/DashParts";


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

/** **لونُ كلّ قسمٍ في رأس البطاقة** — يُعرف القسمُ بنظرة. */
const SECTION_TONE: Record<string, string> = {
  walletRequests: "bg-accent text-on-bright",
  cashCloses: "bg-success-solid text-on-solid",
  cashShortfalls: "bg-danger-solid text-on-solid",
  compensations: "bg-warning-solid text-on-solid",
  expenses: "bg-danger-solid text-on-solid",
  obligations: "bg-warning-solid text-on-solid",
  incentives: "bg-success-solid text-on-solid",
  promos: "bg-accent text-on-bright",
  disputes: "bg-danger-solid text-on-solid",
};

/** **أيقونةُ كلّ قسم** (تدقيقُ الخزينة ٢٠٢٦-١٠-٢٧) — كانت المحفظةُ على كلّ البطاقات. */
const SECTION_ICON: Record<string, ComponentType<{ size?: number; className?: string }>> = {
  walletRequests: IconWallet,
  cashCloses: IconBalance,
  cashShortfalls: IconWarning,
  compensations: IconHandshake,
  expenses: IconReceipt,
  obligations: IconNote,
  incentives: IconStar,
  promos: IconPromos,
  disputes: IconShieldX,
};

/** **بابُ القسم بنمط جدول القدرات** — `/api/v1/admin/x/{id}/reject` ⇒ `/x/{id}/reject`. */
function policyPath(path: string): string {
  return path.replace(/^\/api\/v1\/admin/, "");
}

type Quick = "" | "can" | "undecided";

export function ApprovalsTab() {
  const { user: me } = useAuth();
  const canCall = useCanCall();
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
  const [section, setSection] = useState("");
  const [quick, setQuick] = useState<Quick>("");
  const [rejectFail, setRejectFail] = useState("");

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

  /**
   * **النافذةُ تبقى مفتوحةً حتّى ينجح الطلب** (تدقيقُ الخزينة ٢٠٢٦-١٠-٢٧) — كانت تُغلق قبل الردّ
   * فيظهر خطأ «السبب لازم» أعلى الصفحة وربّما خارج الشاشة. والخطأُ الآن داخلها.
   */
  async function decide(it: ApprovalItem, ok: boolean) {
    setBusy(true);
    try {
      await api(ok ? it.approve_path : it.reject_path, {
        method: "POST",
        body: JSON.stringify({ note: ok ? "" : note.trim() }),
      });
      setApprove(null);
      setReject(null);
      setNote("");
      setFail("");
      setRejectFail("");
      reload();
    } catch (err) {
      if (ok) setFail(errorText(err));
      else setRejectFail(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  if (error && !data) return <ReloadState onRetry={reload} />;
  if (!data) return <LoadingState variant="text" />;
  const items = data.items ?? [];
  const now = Date.now();

  // **الأقسامُ بعدّها** — للفلترة بضغطة.
  const bySection = new Map<string, number>();
  for (const it of items) bySection.set(it.section, (bySection.get(it.section) ?? 0) + 1);
  const isDue = (it: ApprovalItem) => !(it.due_at !== null && new Date(it.due_at).getTime() > now);
  const shown = items.filter(
    (it) =>
      (!section || it.section === section) &&
      (quick !== "can" || (it.can_approve && isDue(it))) &&
      (quick !== "undecided" || !!it.amount_undecided),
  );
  const urgent = items.filter((it) => it.can_approve && isDue(it)).length;
  const filtered = section !== "" || quick !== "";

  return (
    <div className="space-y-4">
      {fail && <Alert onDismiss={() => setFail("")}>{fail}</Alert>}

      {/* **طابورُ الموافقات** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «نفهمها ونعيد تصميمها») — كلُّ مالٍ يتحرّك بيد
          موظّفٍ يمرّ بموافقة شخصٍ ثانٍ: حركةُ محفظة، سلفة، إغلاقُ صندوق، نقص، تعويض، مصروف، شطبُ دين… */}
      <section className="surface space-y-3 p-4">
        <p className="text-sm text-ink-muted">{T.approvals.what}</p>
        <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">
          <Tile
            icon={IconCheck}
            label={T.approvals.count}
            value={String(data.count)}
            tone={data.count > 0 ? "warning" : "muted"}
            onClick={() => {
              setSection("");
              setQuick("");
            }}
          />
          <Tile icon={IconWallet} label={T.approvals.total} value={fmtMoney(data.total)} tone={data.total > 0 ? "default" : "muted"} />
          <Tile
            icon={IconCheck}
            label={T.approvals.mine}
            value={String(urgent)}
            tone={urgent > 0 ? "accent" : "muted"}
            onClick={() => setQuick(quick === "can" ? "" : "can")}
          />
          <Tile
            icon={IconWarning}
            label={T.approvals.undecided}
            value={String(data.undecided_count ?? 0)}
            tone={(data.undecided_count ?? 0) > 0 ? "warning" : "muted"}
            onClick={() => setQuick(quick === "undecided" ? "" : "undecided")}
          />
        </div>
        {quick !== "" && (
          <Chips
            wrap
            items={[
              { id: "" as Quick, label: T.approvals.all },
              { id: "can" as Quick, label: T.approvals.filterCan },
              { id: "undecided" as Quick, label: T.approvals.filterUndecided },
            ]}
            value={quick}
            onChange={setQuick}
          />
        )}
        {bySection.size > 1 && (
          <Chips
            wrap
            items={[
              { id: "", label: T.approvals.all, count: items.length },
              ...[...bySection.entries()].map(([k, n]) => ({ id: k, label: sectionLabel(k), count: n })),
            ]}
            value={section}
            onChange={setSection}
          />
        )}
      </section>

      {shown.length === 0 ? (
        <EmptyCard
          icon={IconCheck}
          title={filtered ? T.approvals.emptyFiltered : T.approvals.empty}
          hint={filtered ? undefined : T.approvals.emptyHint}
        />
      ) : (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2 2xl:grid-cols-3">
          {shown.map((it) => {
            const notDue = it.due_at !== null && new Date(it.due_at).getTime() > now;
            const tone = SECTION_TONE[it.section] ?? "bg-accent text-on-bright";
            const initial = (it.party_name || it.proposer_name || "?").trim().charAt(0);
            const Icon = SECTION_ICON[it.section] ?? IconWallet;
            // **الرفضُ بقدرة بابه** (تدقيقُ الخزينة ٢٠٢٦-١٠-٢٧) — كان يظهر لمن لا يملكها فيُردّ ٤٠٣.
            const canReject = !!it.reject_path && canCall("POST", policyPath(it.reject_path));
            const mine = !!me && it.proposed_by === me.id;
            return (
              <article key={`${it.key}-${it.id}`} className="surface flex flex-col overflow-hidden transition-shadow hover:elev-2">
                {/* **رأسُ البطاقة**: القسمُ بلونه وأيقونته، ومتى اقتُرح. */}
                <header className="flex items-center gap-2.5 border-b border-line-soft bg-field px-4 py-2.5">
                  <span className={`inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full ${tone}`}>
                    <Icon size={15} />
                  </span>
                  <span className="min-w-0 flex-1 truncate text-sm font-bold">{sectionLabel(it.section)}</span>
                  <span className="shrink-0 text-2xs text-ink-muted" dir="ltr">{fmtDateTime(it.created_at)}</span>
                </header>

                <div className="flex flex-1 flex-col gap-3 p-4">
                  {/* **الطرفُ والمبلغ** — من يخصّه المال، والرقمُ كبيرٌ واضح. */}
                  <div className="flex items-center gap-3">
                    <span className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-field text-base font-bold text-ink-muted">
                      {initial}
                    </span>
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-bold">{it.party_name || T.approvals.noParty}</p>
                      <p className="truncate text-xs text-ink-muted">
                        {T.approvals.proposedBy}: {it.proposer_name || T.approvals.system}
                      </p>
                    </div>
                    <div className="shrink-0 text-end">
                      <span className="figure block leading-none">
                        <ApprovalAmount it={it} />
                      </span>
                    </div>
                  </div>

                  {it.note && (
                    <p className="border-s-2 border-accent ps-3 text-sm leading-relaxed text-ink">{it.note}</p>
                  )}

                  {(notDue || (it.self_approval && it.can_approve)) && (
                    <div className="space-y-1">
                      {notDue && it.due_at && (
                        <p className="text-xs text-ink-muted">
                          {T.approvals.notDue} <span dir="ltr">{fmtDateTime(it.due_at)}</span>
                        </p>
                      )}
                      {it.self_approval && it.can_approve && (
                        <p className="rounded-control bg-warning-solid px-2.5 py-1 text-xs font-bold text-on-solid">{T.approvals.selfFlag}</p>
                      )}
                    </div>
                  )}
                </div>

                {/* **قرارٌ بضغطة** — موافقةٌ بالأخضر ورفضٌ بإطارٍ أحمر، عرضَ البطاقة. ومن لا يملك القرارَ
                    يقرأ السببَ الصحيح: «اقتراحك» لصاحبه، و«ما معك صلاحية» لغيره. */}
                <footer className="grid grid-cols-2 gap-2 border-t border-line-soft p-3">
                  {it.approve_path ? (
                    <>
                      {it.can_approve ? (
                        <button
                          type="button"
                          disabled={busy || notDue}
                          onClick={() => (it.amount_editable ? openComp(it) : setApprove(it))}
                          className={`inline-flex items-center justify-center gap-1.5 rounded-control bg-success-solid py-2.5 text-sm font-bold text-on-solid transition-opacity hover:opacity-90 disabled:opacity-50 ${
                            canReject ? "" : "col-span-2"
                          }`}
                        >
                          <IconCheck size={16} />
                          {T.approvals.approve}
                        </button>
                      ) : (
                        <span
                          className={`flex items-center justify-center rounded-control bg-field px-2 py-2.5 text-center text-2xs text-ink-muted ${
                            canReject ? "" : "col-span-2"
                          }`}
                        >
                          {mine ? T.approvals.own : T.approvals.noPermission}
                        </span>
                      )}
                      {canReject && (
                        <button
                          type="button"
                          disabled={busy || (it.key === "cash_shortfalls" && notDue)}
                          onClick={() => {
                            setNote("");
                            setRejectFail("");
                            setReject(it);
                          }}
                          className="inline-flex items-center justify-center gap-1.5 rounded-control border border-danger py-2.5 text-sm font-bold text-danger transition-colors hover:bg-danger-tint disabled:opacity-50"
                        >
                          <IconBlock size={16} />
                          {T.approvals.reject}
                        </button>
                      )}
                    </>
                  ) : (
                    <Link
                      href={it.href}
                      className="col-span-2 inline-flex items-center justify-center rounded-control border border-line py-2.5 text-sm font-bold hover:bg-row-hover"
                    >
                      {T.approvals.open} ↗
                    </Link>
                  )}
                </footer>
              </article>
            );
          })}
        </div>
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
              {fail && <Alert>{fail}</Alert>}
            </div>
          )
        }
        confirmLabel={T.approvals.approve}
        busy={busy}
        onConfirm={() => approve && void decide(approve, true)}
        onCancel={() => {
          setApprove(null);
          setFail("");
        }}
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
      {/* **الرفضُ بسببٍ لازم** — الزرُّ معطّلٌ والسببُ فارغ، والخطأُ داخل النافذة. */}
      <Modal open={reject !== null} onClose={() => setReject(null)} title={T.approvals.rejectTitle}>
        {reject && (
          <div className="space-y-3">
            <p className="text-sm">
              {sectionLabel(reject.section)}
              {reject.party_name ? ` · ${T.approvals.party.replace("{name}", reject.party_name)}` : ""}
            </p>
            {rejectFail && <Alert>{rejectFail}</Alert>}
            <Input
              id="ap-reject-note"
              label={T.approvals.rejectNote}
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
            {note.trim() === "" && <p className="text-2xs text-ink-muted">{T.approvals.reasonNeeded}</p>}
            <FormActions
              busy={busy || note.trim() === ""}
              tone="danger"
              saveLabel={T.approvals.reject}
              onCancel={() => setReject(null)}
              onSave={() => void decide(reject, false)}
            />
          </div>
        )}
      </Modal>
      <p className="text-xs text-ink-muted">{T.statement.tz}</p>
    </div>
  );
}
