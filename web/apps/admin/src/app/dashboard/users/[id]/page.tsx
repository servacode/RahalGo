"use client";

/** صفحة تفاصيل الحساب: الملف، مؤشرات حسب أدواره، سجل المحفظة الكامل،
 *  وأدوات الأدمن (عمليات المحفظة، إعادة تعيين كلمة المرور). */

import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  Input,
  Modal,
  FormSection,
  IconUser,
  IconPhone,
  IconDate,
  IconWallet,
  IconOrder,
  IconStore,
  IconDriver,
  IconLock,
  IconPrev,
  IconLogout,
  IconStatus,
  IconStar,
  IconSupport,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import WalletModal from "@/components/WalletModal";
import { MediaThumb } from "@/components/ImageUpload";
import RoleBadge from "@/components/RoleBadge";

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");
const P = m.admin.users.profile;
const KINDS: Record<string, string> = m.admin.users.txKinds;
const ACTIONS: Record<string, string> = m.admin.users.auditActions;

interface Profile {
  id: string;
  phone: string;
  full_name: string;
  status: string;
  invite_code: string | null;
  avatar_thumb_url?: string | null;
  status_reason: string;
  active_sessions: number;
  roles: string[];
  created_at: string;
  balance: number;
  orders_count: number;
  orders_spent: number;
  merchants: string[];
  rep_stores: number;
  commissions: number;
  driver_cash: number;
  deliveries: number;
}

interface Activity {
  action: string;
  entity: string;
  entity_id: string;
  ip: string;
  details: string;
  by_name: string | null;
  by_self: boolean;
  created_at: string;
}

interface Feedback {
  tickets: { number: number; subject: string; status: string; compensation: number; created_at: string }[];
  ratings_given: { order_number: number; merchant_name: string; merchant_stars: number; driver_stars: number | null; comment: string; created_at: string }[];
  ratings_received: { order_number: number; merchant_name: string; stars: number; comment: string; created_at: string; as: string }[];
  avg_received: number | null;
}

interface Tx {
  id: string;
  kind: string;
  amount: number;
  note: string;
  by_name: string | null;
  order_number: number | null;
  ticket_number: number | null;
  created_at: string;
}

function errText(err: unknown): string {
  if (err instanceof ApiError) {
    const key = err.body.message_key.split(".").pop() ?? "";
    const known = (m.errors as Record<string, string>)[key];
    if (known) return known;
  }
  return m.errors.internal;
}

export default function UserProfilePage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { user: me } = useAuth();
  const isAdmin = !!me?.roles.includes("admin");
  const canWallet = !!me?.roles.some((r) => r === "admin" || r === "finance");

  const [p, setP] = useState<Profile | null>(null);
  const [txs, setTxs] = useState<Tx[]>([]);
  const [activity, setActivity] = useState<Activity[]>([]);
  const [feedback, setFeedback] = useState<Feedback | null>(null);
  const [phoneOpen, setPhoneOpen] = useState(false);
  const [tab, setTab] = useState<"overview" | "wallet" | "feedback" | "activity">("overview");
  const [notice, setNotice] = useState("");
  const [walletOpen, setWalletOpen] = useState(false);
  const [resetOpen, setResetOpen] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      setP(await api<Profile>(`/api/v1/admin/users/${id}`));
      const st = await api<{ transactions: Tx[] }>(`/api/v1/admin/users/${id}/wallet`);
      setTxs(st.transactions);
      setActivity(await api<Activity[]>(`/api/v1/admin/users/${id}/activity`));
      setFeedback(await api<Feedback>(`/api/v1/admin/users/${id}/feedback`));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [id]);

  useEffect(() => {
    void load();
  }, [load]);

  if (error) return <p className="py-10 text-center text-danger">{error}</p>;
  if (!p) return <p className="py-10 text-center text-ink-muted">{m.common.loading}</p>;

  const has = (r: string) => p.roles.includes(r);

  const stats: { label: string; value: string; icon: React.ReactNode }[] = [
    {
      label: `${m.admin.customers.balance} (${m.common.currency})`,
      value: fmt.format(p.balance),
      icon: <IconWallet className="text-primary" />,
    },
  ];
  if (has("customer") || p.orders_count > 0) {
    stats.push(
      { label: P.ordersCount, value: fmt.format(p.orders_count), icon: <IconOrder /> },
      {
        label: `${P.spent} (${m.common.currency})`,
        value: fmt.format(p.orders_spent),
        icon: <IconOrder className="text-success" />,
      }
    );
  }
  if (has("sales")) {
    stats.push(
      { label: P.repStores, value: fmt.format(p.rep_stores), icon: <IconStore /> },
      {
        label: `${P.commissions} (${m.common.currency})`,
        value: fmt.format(p.commissions),
        icon: <IconWallet className="text-success" />,
      }
    );
  }
  if (feedback?.avg_received != null) {
    stats.push({
      label: P.avgRating,
      value: `${feedback.avg_received.toFixed(1)} ★`,
      icon: <IconStar className="text-accent" />,
    });
  }
  if (has("driver")) {
    stats.push(
      { label: P.deliveries, value: fmt.format(p.deliveries), icon: <IconDriver /> },
      {
        label: `${P.driverCash} (${m.common.currency})`,
        value: fmt.format(p.driver_cash),
        icon: <IconWallet className="text-accent-dark" />,
      }
    );
  }

  return (
    <div>
      <button
        onClick={() => router.push("/dashboard/users")}
        className="mb-3 flex items-center gap-1 text-sm text-ink-muted hover:text-primary"
      >
        <IconPrev size={15} />
        {m.admin.users.title}
      </button>

      {/* الترويسة: البيانات في الصدارة والأزرار سطر واحد (ملاحظة مراجعة) */}
      <div className="mb-3 rounded-card border border-line bg-surface p-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <MediaThumb url={p.avatar_thumb_url} alt="" fallback={p.full_name || "؟"} size={56} />
            <div className="min-w-0">
              <h1 className="truncate text-lg font-bold">{p.full_name || "—"}</h1>
              <p className="flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-ink-muted">
                <span className="inline-flex items-center gap-1">
                  <IconPhone size={12} />
                  <span dir="ltr">{p.phone}</span>
                </span>
                <span className="inline-flex items-center gap-1">
                  <IconDate size={12} />
                  {new Date(p.created_at).toLocaleDateString("ar-SY")}
                </span>
                {p.invite_code && (
                  <span dir="ltr" className="rounded-badge bg-accent/15 px-1.5 font-mono text-accent-dark">
                    {p.invite_code}
                  </span>
                )}
              </p>
              <div className="mt-1 flex flex-wrap items-center gap-1">
                {p.roles.map((r) => (
                  <RoleBadge key={r} role={r} />
                ))}
                <Badge
                  variant={p.status === "active" ? "success" : p.status === "suspended" ? "warning" : "danger"}
                >
                  {p.status === "active"
                    ? m.admin.users.active
                    : p.status === "suspended"
                      ? m.admin.users.suspended
                      : m.admin.users.blocked}
                </Badge>
                {p.status !== "active" && p.status_reason && (
                  <span className="text-xs text-danger">{p.status_reason}</span>
                )}
              </div>
            </div>
          </div>

          <div className="flex flex-nowrap items-center gap-1.5 whitespace-nowrap">
            {canWallet && (
              <Button onClick={() => setWalletOpen(true)} className="flex items-center gap-1.5 !px-2.5">
                <IconWallet size={15} />
                {m.admin.users.wallet}
              </Button>
            )}
            {isAdmin && (
              <>
                <Button
                  variant="secondary"
                  onClick={() => setResetOpen(true)}
                  className="flex items-center gap-1.5 !px-2.5"
                >
                  <IconLock size={15} />
                  {P.passwordBtn}
                </Button>
                <Button
                  variant="secondary"
                  onClick={() => setPhoneOpen(true)}
                  className="flex items-center gap-1.5 !px-2.5"
                >
                  <IconPhone size={15} />
                  {P.changePhone}
                </Button>
                <Button
                  variant="danger"
                  onClick={async () => {
                    const res = await api<{ revoked_sessions: number }>(
                      `/api/v1/admin/users/${p.id}/logout-all`,
                      { method: "POST" }
                    );
                    setNotice(P.logoutAllDone.replace("{n}", String(res.revoked_sessions)));
                    await load();
                  }}
                  className="flex items-center gap-1.5 !px-2.5"
                >
                  <IconLogout size={15} />
                  {P.logoutAllShort} ({fmt.format(p.active_sessions)})
                </Button>
              </>
            )}
          </div>
        </div>
      </div>

      {notice && (
        <p className="mb-4 rounded-control bg-success/10 px-3 py-2 text-sm text-success">{notice}</p>
      )}

      {/* التبويبات — كل قسم في تبويبه (ملاحظة مراجعة) */}
      <div className="mb-3 flex gap-1 border-b border-line">
        {(
          [
            { key: "overview", label: P.tabs.overview, icon: <IconUser size={15} /> },
            { key: "wallet", label: P.tabs.wallet, icon: <IconWallet size={15} /> },
            { key: "feedback", label: P.tabs.feedback, icon: <IconStar size={15} /> },
            { key: "activity", label: P.tabs.activity, icon: <IconStatus size={15} /> },
          ] as const
        ).map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm transition-colors ${
              tab === t.key
                ? "border-primary font-bold text-primary-dark"
                : "border-transparent text-ink-muted hover:text-ink"
            }`}
          >
            {t.icon}
            {t.label}
          </button>
        ))}
      </div>

      {tab === "overview" && (
      <>
      {/* المؤشرات حسب الأدوار */}
      <div className="mb-3 grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-6">
        {stats.map((s) => (
          <div key={s.label} className="rounded-card border border-line bg-surface p-3">
            <div className="mb-1">{s.icon}</div>
            <p className="text-lg font-bold">{s.value}</p>
            <p className="text-xs text-ink-muted">{s.label}</p>
          </div>
        ))}
      </div>

      {p.merchants.length > 0 && (
        <div className="mb-3 rounded-card border border-line bg-surface p-3">
          <p className="mb-2 flex items-center gap-1.5 text-sm font-bold">
            <IconStore size={15} className="text-primary" />
            {P.merchantsOwned}
          </p>
          <div className="flex flex-wrap gap-2">
            {p.merchants.map((name) => (
              <Badge key={name} variant="primary">
                {name}
              </Badge>
            ))}
          </div>
        </div>
      )}

      </>
      )}

      {tab === "wallet" && (
      <FormSection title={P.statement} icon={<IconWallet />}>
        {txs.length === 0 ? (
          <p className="py-6 text-center text-sm text-ink-muted">{P.statementEmpty}</p>
        ) : (
          <ul className="space-y-1.5">
            {txs.map((t) => (
              <li
                key={t.id}
                className="flex items-center justify-between gap-3 rounded-control border border-line px-3 py-2 text-sm"
              >
                <div className="min-w-0 flex-1">
                  <span className="flex flex-wrap items-center gap-2">
                    <Badge variant={t.amount > 0 ? "success" : "danger"}>
                      {KINDS[t.kind] ?? t.kind}
                    </Badge>
                    {t.order_number != null && (
                      <button
                        type="button"
                        onClick={() => router.push(`/dashboard/orders?q=${t.order_number}`)}
                        className="text-xs font-medium text-primary hover:underline"
                      >
                        {P.orderRef} #{fmt.format(t.order_number)}
                      </button>
                    )}
                    {t.ticket_number != null && (
                      <span className="text-xs font-medium text-primary">
                        {P.ticketRef} #{fmt.format(t.ticket_number)}
                      </span>
                    )}
                    {t.note && <span className="truncate text-xs text-ink-muted">{t.note}</span>}
                  </span>
                  <p className="mt-0.5 text-xs text-ink-muted">
                    {P.by}: {t.by_name ?? P.system}
                  </p>
                </div>
                <span className="flex shrink-0 items-center gap-3">
                  <span className={`font-bold ${t.amount > 0 ? "text-success" : "text-danger"}`} dir="ltr">
                    {t.amount > 0 ? "+" : ""}
                    {fmt.format(t.amount)}
                  </span>
                  <span className="text-xs text-ink-muted" dir="ltr">
                    {new Date(t.created_at).toLocaleString("ar-SY", { dateStyle: "short", timeStyle: "short" })}
                  </span>
                </span>
              </li>
            ))}
          </ul>
        )}
      </FormSection>
      )}

      {tab === "feedback" && feedback && (
        <div className="space-y-4">
          <FormSection title={P.ticketsSection} icon={<IconSupport />}>
            {feedback.tickets.length === 0 ? (
              <p className="py-4 text-center text-sm text-ink-muted">{P.ticketsEmpty}</p>
            ) : (
              <ul className="space-y-1.5">
                {feedback.tickets.map((t) => (
                  <li
                    key={t.number}
                    onClick={() => router.push("/dashboard/tickets")}
                    className="flex cursor-pointer flex-wrap items-center justify-between gap-2 rounded-control border border-line px-3 py-2 text-sm hover:bg-page"
                  >
                    <span className="flex items-center gap-2">
                      <span className="font-bold">#{fmt.format(t.number)}</span>
                      <span className="truncate">{t.subject}</span>
                    </span>
                    <span className="flex items-center gap-2">
                      {t.compensation > 0 && (
                        <span className="text-xs text-success">
                          +{fmt.format(t.compensation)} {m.common.currency}
                        </span>
                      )}
                      <Badge
                        variant={t.status === "resolved" ? "success" : t.status === "open" ? "warning" : "primary"}
                      >
                        {(m.admin.tickets.status as Record<string, string>)[t.status] ?? t.status}
                      </Badge>
                      <span className="text-xs text-ink-muted" dir="ltr">
                        {new Date(t.created_at).toLocaleDateString("ar-SY")}
                      </span>
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </FormSection>

          <FormSection title={P.ratingsGiven} icon={<IconStar />}>
            {feedback.ratings_given.length === 0 ? (
              <p className="py-4 text-center text-sm text-ink-muted">{P.ratingsGivenEmpty}</p>
            ) : (
              <ul className="space-y-1.5">
                {feedback.ratings_given.map((rt, i) => (
                  <li
                    key={i}
                    className="flex flex-wrap items-center justify-between gap-2 rounded-control border border-line px-3 py-2 text-sm"
                  >
                    <span className="flex min-w-0 flex-wrap items-center gap-2">
                      <button
                        type="button"
                        onClick={() => router.push(`/dashboard/orders?q=${rt.order_number}`)}
                        className="font-medium text-primary hover:underline"
                      >
                        #{fmt.format(rt.order_number)}
                      </button>
                      <span className="text-ink-muted">{rt.merchant_name}</span>
                      <span className="text-accent-dark">★ {rt.merchant_stars}</span>
                      {rt.driver_stars != null && (
                        <span className="text-xs text-ink-muted">
                          ({m.admin.ordersPage.rating.driver}: ★ {rt.driver_stars})
                        </span>
                      )}
                      {rt.comment && (
                        <span className="truncate text-xs text-ink-muted">"{rt.comment}"</span>
                      )}
                    </span>
                    <span className="text-xs text-ink-muted" dir="ltr">
                      {new Date(rt.created_at).toLocaleDateString("ar-SY")}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </FormSection>

          <FormSection title={P.ratingsRecv} icon={<IconStar />}>
            {feedback.ratings_received.length === 0 ? (
              <p className="py-4 text-center text-sm text-ink-muted">{P.ratingsRecvEmpty}</p>
            ) : (
              <ul className="space-y-1.5">
                {feedback.ratings_received.map((rt, i) => (
                  <li
                    key={i}
                    className="flex flex-wrap items-center justify-between gap-2 rounded-control border border-line px-3 py-2 text-sm"
                  >
                    <span className="flex min-w-0 flex-wrap items-center gap-2">
                      <Badge variant={rt.as === "driver" ? "primary" : "warning"}>
                        {rt.as === "driver" ? P.asDriver : P.asMerchant}
                      </Badge>
                      <span className="text-accent-dark">★ {rt.stars}</span>
                      <button
                        type="button"
                        onClick={() => router.push(`/dashboard/orders?q=${rt.order_number}`)}
                        className="font-medium text-primary hover:underline"
                      >
                        #{fmt.format(rt.order_number)}
                      </button>
                      <span className="text-xs text-ink-muted">{rt.merchant_name}</span>
                      {rt.comment && (
                        <span className="truncate text-xs text-ink-muted">"{rt.comment}"</span>
                      )}
                    </span>
                    <span className="text-xs text-ink-muted" dir="ltr">
                      {new Date(rt.created_at).toLocaleDateString("ar-SY")}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </FormSection>
        </div>
      )}

      {tab === "activity" && (
      <div>
        <FormSection title={P.activity} icon={<IconStatus />}>
          {activity.length === 0 ? (
            <p className="py-6 text-center text-sm text-ink-muted">{P.activityEmpty}</p>
          ) : (
            <ul className="space-y-1.5">
              {activity.map((a, i) => (
                <li
                  key={i}
                  className="flex flex-wrap items-center justify-between gap-2 rounded-control border border-line px-3 py-2 text-sm"
                >
                  <span className="flex min-w-0 flex-wrap items-center gap-2">
                    <span className="font-medium">{ACTIONS[a.action] ?? a.action}</span>
                    <span className="text-xs text-ink-muted">
                      {P.by}: {a.by_self ? P.bySelf : (a.by_name ?? P.system)}
                    </span>
                    {a.ip && (
                      <span className="text-xs text-ink-muted" dir="ltr">
                        {a.ip}
                      </span>
                    )}
                  </span>
                  <span className="shrink-0 text-xs text-ink-muted" dir="ltr">
                    {new Date(a.created_at).toLocaleString("ar-SY", { dateStyle: "short", timeStyle: "short" })}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </FormSection>
      </div>
      )}

      {phoneOpen && (
        <ChangePhoneModal
          userID={p.id}
          current={p.phone}
          onClose={() => setPhoneOpen(false)}
          onDone={() => {
            setPhoneOpen(false);
            void load();
          }}
        />
      )}
      {walletOpen && (
        <WalletModal
          user={{ id: p.id, phone: p.phone, full_name: p.full_name }}
          isAdmin={canWallet}
          onClose={() => {
            setWalletOpen(false);
            void load();
          }}
        />
      )}
      {resetOpen && (
        <ResetPasswordModal userID={p.id} onClose={() => setResetOpen(false)} />
      )}
    </div>
  );
}

function ResetPasswordModal({ userID, onClose }: { userID: string; onClose: () => void }) {
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/users/${userID}/password`, {
        method: "POST",
        body: JSON.stringify({ password }),
      });
      setDone(true);
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={P.resetPassword}>
      {done ? (
        <div className="space-y-4 text-center">
          <p className="rounded-control bg-success/10 px-3 py-3 text-success">{P.resetDone}</p>
          <Button onClick={onClose}>{m.common.confirm}</Button>
        </div>
      ) : (
        <form onSubmit={submit} className="space-y-4">
          <Input
            id="new-pw"
            label={P.newPassword}
            dir="ltr"
            required
            minLength={8}
            autoFocus
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="font-mono"
          />
          {error && (
            <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="secondary" onClick={onClose}>
              {m.common.cancel}
            </Button>
            <Button type="submit" disabled={busy}>
              {m.common.save}
            </Button>
          </div>
        </form>
      )}
    </Modal>
  );
}

function ChangePhoneModal({
  userID,
  current,
  onClose,
  onDone,
}: {
  userID: string;
  current: string;
  onClose: () => void;
  onDone: () => void;
}) {
  const [phone, setPhone] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/users/${userID}`, {
        method: "PATCH",
        body: JSON.stringify({ phone }),
      });
      onDone();
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={P.changePhone}>
      <form onSubmit={submit} className="space-y-4">
        <p className="text-sm text-ink-muted" dir="ltr">{current}</p>
        <Input
          id="new-phone"
          label={P.newPhone}
          icon={<IconPhone />}
          dir="ltr"
          required
          autoFocus
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          className="text-end"
          placeholder="09xxxxxxxx"
        />
        <p className="rounded-control bg-page px-3 py-2 text-xs leading-relaxed text-ink-muted">
          {P.phoneHint}
        </p>
        {error && (
          <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
        )}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button type="submit" disabled={busy}>
            {m.common.save}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
