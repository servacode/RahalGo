"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الأدوارُ والصلاحيّات** — قرارُ المالك ٢٠٢٦-١٠-٠٤
 * ══════════════════════════════════════════════════════════════════════
 *
 * **تبويبان**: الأدوارُ بأقسامها (إدارةٌ عليا · موظّفون · مخصّصة · أنواعُ
 * الحسابات للقراءة · قديمة)، **ومصفوفةٌ واحدةٌ** — الأدوارُ أعمدةٌ والقدراتُ
 * صفوفٌ مجمّعةٌ في سبع مجموعات، **وما يحرّك مالاً أو يعطي سلطةً بالأحمر.**
 *
 * **وقبل المنح**: أثرُه («سيقدر ٣ موظفين على…») وسببٌ مكتوبٌ إلزاميٌّ يُحفظ في
 * السجلّ، **وكلمةُ السرّ تطلبها بوّابةُ التأكيد المركزيّة** (`StepUpGate`).
 * **وقبل النزع**: سؤالُ تأكيدٍ يقول ما يُنزَع وممّن.
 *
 * # ولا حقيقةَ ثانيةً في الواجهة
 *
 * **كلُّ ما يُعرَض مقروءٌ من المحرّك** (`lib/rbac.ts`) — والصنفُ والقابليّةُ
 * للتعديل والحذف يقولها المحرّك. **والحراسةُ في المحرّك** — وإخفاءُ زرٍّ هنا
 * لطفٌ بالعين لا حراسة.
 */

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtNum, fmtDate } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Confirm,
  EmptyState,
  Input,
  LoadingState,
  Modal,
  Select,
  PageContainer,
  PageHeader,
  FormActions,
  Tabs,
  Textarea,
  IconRoles,
  IconCheck,
} from "@rahalgo/ui";
import {
  CAPABILITY_GROUPS,
  createRole,
  deleteRole,
  grantCapability,
  listCapabilities,
  listRoles,
  revokeCapability,
  roleImpact,
  roleMembers,
  type Capability,
  type Role,
  type RoleImpact,
  type RoleMember,
} from "@/lib/rbac";
import {
  CAPABILITY_GROUP_LABELS,
  capabilityDescription,
  capabilityLabel,
  roleDescription,
  roleLabel,
} from "@/lib/rolemeta";

const m = getMessages(defaultLocale);
const t = m.admin.roles;

type TabKey = "roles" | "matrix";

/** **أقسامُ الأدوار بترتيب المالك.** */
const SECTIONS: { key: string; title: string; classes: NonNullable<Role["class"]>[] }[] = [
  { key: "top", title: t.sectionTop, classes: ["protected", "elevated"] },
  { key: "staff", title: t.sectionStaff, classes: ["staff"] },
  { key: "custom", title: t.sectionCustom, classes: ["custom"] },
  { key: "account", title: t.sectionAccountType, classes: ["account_type"] },
  { key: "legacy", title: t.sectionLegacy, classes: ["legacy"] },
];

function RiskBadge({ risk }: { risk?: string }) {
  if (risk === "money") return <Badge variant="danger">{t.riskMoney}</Badge>;
  if (risk === "power") return <Badge variant="danger">{t.riskPower}</Badge>;
  return null;
}

export default function RolesPage() {
  const [roles, setRoles] = useState<Role[] | null>(null);
  const [caps, setCaps] = useState<Capability[]>([]);
  const [err, setErr] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [tab, setTab] = useState<TabKey>("roles");

  const [createOpen, setCreateOpen] = useState(false);
  const [code, setCode] = useState("");
  const [name, setName] = useState("");

  const [openRole, setOpenRole] = useState<string | null>(null);
  const [pick, setPick] = useState("");

  const [membersOf, setMembersOf] = useState<string | null>(null);
  const [members, setMembers] = useState<RoleMember[] | null>(null);

  const [grant, setGrant] = useState<{ role: string; cap: string } | null>(null);
  const [impact, setImpact] = useState<RoleImpact | null>(null);
  const [reason, setReason] = useState("");

  const [revoke, setRevoke] = useState<{ role: string; cap: string } | null>(null);
  const [removeRole, setRemoveRole] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const [r, c] = await Promise.all([listRoles(), listCapabilities()]);
      setRoles(r);
      setCaps(c);
      setErr("");
    } catch (e) {
      setErr(errorText(e));
      setRoles([]);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function act(fn: () => Promise<unknown>, done?: string) {
    setBusy(true);
    setErr("");
    setNote("");
    try {
      await fn();
      await load();
      if (done) setNote(done);
      return true;
    } catch (e) {
      // **وطلبُ التأكيد بكلمة السرّ تلتقطه بوّابةُ `StepUpGate` المركزيّة.**
      setErr(errorText(e));
      return false;
    } finally {
      setBusy(false);
    }
  }

  const capByCode = useMemo(() => new Map(caps.map((c) => [c.code, c])), [caps]);
  const capName = (c: string) => capabilityLabel(c, capByCode.get(c)?.description);
  const capDesc = (c: string) => capabilityDescription(c, capByCode.get(c)?.description);
  const capRisk = (c: string) => capByCode.get(c)?.risk ?? "";

  /** **القدراتُ مجمّعةً بترتيب المالك.** */
  const grouped = useMemo(
    () =>
      CAPABILITY_GROUPS.map((g) => ({
        group: g,
        caps: caps.filter((c) => (c.group ?? "security") === g),
      })).filter((x) => x.caps.length > 0),
    [caps],
  );

  /** **خطرُ الدور** — أشدُّ ما فيه من قدرات. */
  const roleRisk = (r: Role) => {
    if (r.capabilities.some((c) => capRisk(c) === "money")) return "money";
    if (r.capabilities.some((c) => capRisk(c) === "power")) return "power";
    return "";
  };

  const current = roles?.find((r) => r.code === openRole) ?? null;
  const roleByCode = (c: string) => roles?.find((r) => r.code === c);
  const matrixRoles = (roles ?? []).filter((r) => r.class !== "account_type");

  function openMembers(code: string) {
    setMembersOf(code);
    setMembers(null);
    roleMembers(code)
      .then(setMembers)
      .catch((e) => {
        setErr(errorText(e));
        setMembers([]);
      });
  }

  function askGrant(role: string, cap: string) {
    setGrant({ role, cap });
    setReason("");
    setImpact(null);
    roleImpact(role, cap)
      .then(setImpact)
      .catch(() => setImpact(null));
  }

  function toggleCell(r: Role, cap: string) {
    if (!r.editable) return;
    if (r.capabilities.includes(cap)) setRevoke({ role: r.code, cap });
    else askGrant(r.code, cap);
  }

  const impactLine = (() => {
    if (!grant || !impact) return "";
    if (impact.active_members === 0) return t.impactNone;
    return t.impact
      .replace("{n}", fmtNum(impact.would_gain))
      .replace("{cap}", capName(grant.cap));
  })();

  return (
    <PageContainer>
      <PageHeader
        title={t.title}
        subtitle={t.subtitle}
        actions={<Button onClick={() => setCreateOpen(true)}>{t.create}</Button>}
      />

      {err ? <Alert>{err}</Alert> : null}
      {note ? (
        <Alert tone="success" onDismiss={() => setNote("")}>
          {note}
        </Alert>
      ) : null}

      <Tabs<TabKey>
        items={[
          { key: "roles", label: t.tabRoles, count: roles?.length },
          { key: "matrix", label: t.tabMatrix },
        ]}
        value={tab}
        onChange={setTab}
      />

      {roles === null ? (
        <LoadingState variant="text" />
      ) : roles.length === 0 ? (
        <EmptyState icon={<IconRoles size={32} />} title={t.empty} />
      ) : tab === "roles" ? (
        <div className="grid gap-5">
          {SECTIONS.map((sec) => {
            const list = roles.filter((r) => sec.classes.includes(r.class ?? "custom"));
            if (list.length === 0) return null;
            return (
              <section key={sec.key} className="grid gap-2">
                <h2 className="heading-card">{sec.title}</h2>
                {list.map((r) => {
                  const risk = roleRisk(r);
                  return (
                    <div
                      key={r.code}
                      className="flex flex-wrap items-center justify-between gap-3 rounded-card border p-3"
                    >
                      <span className="flex min-w-0 flex-1 flex-col gap-0.5 text-start">
                        <span className="flex flex-wrap items-center gap-2 font-medium">
                          {roleLabel(r)}
                          <RiskBadge risk={risk} />
                        </span>
                        {roleDescription(r.code) ? (
                          <span className="text-xs text-ink-muted">{roleDescription(r.code)}</span>
                        ) : null}
                        {r.class === "account_type" ? (
                          <span className="text-xs text-ink-muted">{t.readOnly}</span>
                        ) : null}
                        {r.class === "protected" ? (
                          <span className="text-xs text-ink-muted">{t.protectedNote}</span>
                        ) : null}
                        {/* **والرمزُ سطرٌ باهتٌ لحاله** — لا يُخلط بالاسم العربيّ في سطرٍ واحد. */}
                        <span className="text-[11px] opacity-50" dir="ltr">
                          {r.code}
                        </span>
                      </span>
                      <span className="flex flex-wrap items-center gap-2">
                        <Badge>
                          {t.capabilities}: {fmtNum(r.capabilities.length)}
                        </Badge>
                        <Button variant="secondary" onClick={() => openMembers(r.code)}>
                          {t.holdersCount.replace("{n}", fmtNum(r.members))}
                        </Button>
                        {r.class !== "account_type" ? (
                          <Button
                            variant="secondary"
                            onClick={() => {
                              setOpenRole(r.code);
                              setPick("");
                            }}
                          >
                            {t.capabilities}
                          </Button>
                        ) : null}
                        {r.deletable ? (
                          <Button variant="secondary" onClick={() => setRemoveRole(r.code)}>
                            {t.deleteRole}
                          </Button>
                        ) : null}
                      </span>
                    </div>
                  );
                })}
              </section>
            );
          })}
        </div>
      ) : (
        <div className="grid gap-2">
          <p className="text-xs text-ink-muted">{t.matrixHint}</p>
          <div className="overflow-x-auto rounded-card border">
            <table className="w-full border-collapse text-sm">
              <thead>
                <tr>
                  <th className="sticky start-0 z-10 bg-surface p-2 text-start">{t.colCapability}</th>
                  {matrixRoles.map((r) => (
                    <th key={r.code} className="whitespace-nowrap p-2 text-center text-xs font-medium">
                      {roleLabel(r)}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {grouped.map((g) => [
                  <tr key={`g-${g.group}`} className="bg-ink-faint">
                    <td
                      colSpan={matrixRoles.length + 1}
                      className="sticky start-0 p-2 text-start text-xs font-bold"
                    >
                      {CAPABILITY_GROUP_LABELS[g.group] ?? g.group}
                    </td>
                  </tr>,
                  ...g.caps.map((c) => {
                    const danger = !!c.risk;
                    return (
                      <tr key={c.code} className="border-t border-line-soft">
                        <td className="sticky start-0 z-10 bg-surface p-2 text-start">
                          <span className={`block ${danger ? "font-medium text-danger" : ""}`}>
                            {capName(c.code)}
                          </span>
                          <span className="block text-xs text-ink-muted">{capDesc(c.code)}</span>
                        </td>
                        {matrixRoles.map((r) => {
                          const on = r.capabilities.includes(c.code);
                          return (
                            <td key={r.code} className="p-1 text-center">
                              <button
                                type="button"
                                disabled={!r.editable || busy}
                                onClick={() => toggleCell(r, c.code)}
                                aria-pressed={on}
                                aria-label={`${roleLabel(r)} — ${capName(c.code)}`}
                                className={`inline-flex h-7 w-7 items-center justify-center rounded-control border ${
                                  on
                                    ? danger
                                      ? "border-danger bg-danger-tint text-danger"
                                      : "border-primary bg-primary-tint text-primary"
                                    : "border-line-soft"
                                } disabled:opacity-60`}
                              >
                                {on ? <IconCheck size={16} /> : null}
                              </button>
                            </td>
                          );
                        })}
                      </tr>
                    );
                  }),
                ])}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* ── إنشاءُ دور ────────────────────────────────────────────── */}
      <Modal open={createOpen} onClose={() => setCreateOpen(false)} title={t.createTitle}>
        <div className="grid gap-3">
          <div className="grid gap-1">
            <Input label={t.code} value={code} onChange={(e) => setCode(e.target.value)} dir="ltr" />
            <p className="text-xs opacity-60">{t.codeHint}</p>
          </div>
          <div className="grid gap-1">
            <Input label={t.name} value={name} onChange={(e) => setName(e.target.value)} />
            <p className="text-xs opacity-60">{t.nameHint}</p>
          </div>
          <FormActions
            busy={busy || !code.trim() || !name.trim()}
            saveLabel={t.save}
            cancelLabel={t.cancel}
            onCancel={() => setCreateOpen(false)}
            onSave={async () => {
              if (await act(() => createRole(code.trim(), name.trim()), t.created)) {
                setCreateOpen(false);
                setCode("");
                setName("");
              }
            }}
          />
        </div>
      </Modal>

      {/* ── صلاحيّاتُ دور ─────────────────────────────────────────── */}
      <Modal
        open={openRole !== null}
        onClose={() => setOpenRole(null)}
        title={current ? roleLabel(current) : ""}
        size="lg"
      >
        {current ? (
          <div className="grid gap-3">
            {current.class === "protected" ? (
              <p className="text-xs text-ink-muted">{t.protectedNote}</p>
            ) : null}
            {current.capabilities.length === 0 ? (
              <p className="text-sm opacity-70">{t.noCapabilities}</p>
            ) : (
              <ul className="grid gap-2">
                {current.capabilities.map((c) => (
                  <li key={c} className="flex items-center justify-between gap-2">
                    <span className="flex min-w-0 flex-col gap-0.5">
                      <span className="flex flex-wrap items-center gap-2 text-sm">
                        {capName(c)}
                        <RiskBadge risk={capRisk(c)} />
                      </span>
                      <span className="text-xs text-ink-muted">{capDesc(c)}</span>
                      <code className="text-[11px] opacity-50" dir="ltr">
                        {c}
                      </code>
                    </span>
                    {current.editable ? (
                      <Button
                        variant="secondary"
                        disabled={busy}
                        onClick={() => setRevoke({ role: current.code, cap: c })}
                      >
                        {t.removeCapability}
                      </Button>
                    ) : null}
                  </li>
                ))}
              </ul>
            )}

            {current.editable ? (
              <div className="flex flex-wrap items-end gap-2">
                <Select
                  label={t.addCapability}
                  id="cap-pick"
                  className="min-w-0 flex-1"
                  value={pick}
                  onChange={(e) => setPick(e.target.value)}
                >
                  <option value="">{t.pickCapability}</option>
                  {grouped.map((g) => (
                    <optgroup key={g.group} label={CAPABILITY_GROUP_LABELS[g.group] ?? g.group}>
                      {g.caps
                        .filter((c) => !current.capabilities.includes(c.code))
                        .map((c) => (
                          <option key={c.code} value={c.code}>
                            {capName(c.code)}
                          </option>
                        ))}
                    </optgroup>
                  ))}
                </Select>
                <Button disabled={busy || !pick} onClick={() => askGrant(current.code, pick)}>
                  {t.addCapability}
                </Button>
              </div>
            ) : null}
          </div>
        ) : null}
      </Modal>

      {/* ── منحٌ بأثره وسببه ─────────────────────────────────────── */}
      <Modal open={grant !== null} onClose={() => setGrant(null)} title={t.grantTitle}>
        {grant ? (
          <div className="grid gap-3">
            <p className="text-sm">
              <span className="font-medium">{capName(grant.cap)}</span>
              {" — "}
              {roleLabel(roleByCode(grant.role) ?? { code: grant.role })}
            </p>
            <p className="text-xs text-ink-muted">{capDesc(grant.cap)}</p>
            {impactLine ? <Alert tone="info">{impactLine}</Alert> : null}
            {grant.cap === "roles.manage" ? (
              <Alert tone="warning">{t.impactRolesManage}</Alert>
            ) : capRisk(grant.cap) === "money" ? (
              <Alert tone="warning">{t.impactMoney}</Alert>
            ) : capRisk(grant.cap) === "power" ? (
              <Alert tone="warning">{t.impactPower}</Alert>
            ) : null}
            <Textarea
              label={t.reasonLabel}
              placeholder={t.reasonPlaceholder}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              autoGrow
            />
            <FormActions
              busy={busy || !reason.trim()}
              saveLabel={t.confirmGrant}
              cancelLabel={t.cancel}
              onCancel={() => setGrant(null)}
              onSave={async () => {
                const g = grant;
                let changed = true;
                const ok = await act(async () => {
                  const res = await grantCapability(g.role, g.cap, reason.trim());
                  changed = res?.changed !== false;
                });
                if (ok) {
                  setNote(changed ? t.granted : t.unchanged);
                  setGrant(null);
                  setPick("");
                }
              }}
            />
          </div>
        ) : null}
      </Modal>

      {/* ── نزعٌ بسؤال تأكيد ─────────────────────────────────────── */}
      <Confirm
        open={revoke !== null}
        title={t.confirmRevokeTitle}
        body={
          revoke
            ? t.confirmRevokeBody
                .replace("{cap}", capName(revoke.cap))
                .replace("{role}", roleLabel(roleByCode(revoke.role) ?? { code: revoke.role }))
                .replace("{n}", fmtNum(roleByCode(revoke.role)?.members ?? 0))
            : undefined
        }
        confirmLabel={t.confirmRevoke2}
        busy={busy}
        onCancel={() => setRevoke(null)}
        onConfirm={async () => {
          const r = revoke;
          if (!r) return;
          if (await act(() => revokeCapability(r.role, r.cap), t.revoked)) setRevoke(null);
        }}
      />

      {/* ── حذفُ دورٍ فارغ ───────────────────────────────────────── */}
      <Confirm
        open={removeRole !== null}
        title={t.confirmDeleteTitle}
        body={
          removeRole
            ? t.confirmDeleteBody.replace(
                "{role}",
                roleLabel(roleByCode(removeRole) ?? { code: removeRole }),
              )
            : undefined
        }
        confirmLabel={t.deleteRole}
        busy={busy}
        onCancel={() => setRemoveRole(null)}
        onConfirm={async () => {
          const r = removeRole;
          if (!r) return;
          if (await act(() => deleteRole(r), t.deleted)) setRemoveRole(null);
        }}
      />

      {/* ── حاملو الدور ──────────────────────────────────────────── */}
      <Modal
        open={membersOf !== null}
        onClose={() => setMembersOf(null)}
        title={`${t.membersTitle} — ${membersOf ? roleLabel(roleByCode(membersOf) ?? { code: membersOf }) : ""}`}
      >
        {members === null ? (
          <LoadingState variant="text" />
        ) : members.length === 0 ? (
          <p className="text-sm text-ink-muted">{t.noMembers}</p>
        ) : (
          <ul className="grid gap-2">
            {members.map((u) => (
              <li key={u.id} className="flex items-center justify-between gap-2">
                <Link href={`/dashboard/users/${u.id}`} className="text-sm font-medium text-primary">
                  {u.full_name || u.id}
                </Link>
                <span className="flex items-center gap-2 text-xs text-ink-muted">
                  {u.status !== "active" ? <Badge variant="warning">{t.memberBlocked}</Badge> : null}
                  {fmtDate(u.granted_at)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </Modal>
    </PageContainer>
  );
}
