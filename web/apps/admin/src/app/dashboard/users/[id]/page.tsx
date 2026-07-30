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
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import WalletModal from "@/components/WalletModal";
import ImageUpload, { MediaThumb } from "@/components/ImageUpload";
import RoleBadge from "@/components/RoleBadge";

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");
const P = m.admin.users.profile;
const KINDS: Record<string, string> = m.admin.users.txKinds;

interface Profile {
  id: string;
  phone: string;
  full_name: string;
  status: string;
  invite_code: string | null;
  avatar_thumb_url?: string | null;
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

interface Tx {
  id: string;
  kind: string;
  amount: number;
  note: string;
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
  const [walletOpen, setWalletOpen] = useState(false);
  const [resetOpen, setResetOpen] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      setP(await api<Profile>(`/api/v1/admin/users/${id}`));
      const st = await api<{ transactions: Tx[] }>(`/api/v1/admin/users/${id}/wallet`);
      setTxs(st.transactions);
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
    <div className="mx-auto max-w-4xl">
      <button
        onClick={() => router.push("/dashboard/users")}
        className="mb-4 flex items-center gap-1 text-sm text-ink-muted hover:text-primary"
      >
        <IconPrev size={15} />
        {m.admin.users.title}
      </button>

      {/* الترويسة */}
      <div className="mb-5 flex flex-wrap items-center gap-4 rounded-card border border-line bg-surface p-5">
        <MediaThumb url={p.avatar_thumb_url} alt="" fallback={p.full_name || "؟"} size={64} />
        <div className="min-w-0 flex-1">
          <h1 className="text-xl font-bold">{p.full_name || "—"}</h1>
          <p className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-ink-muted">
            <span className="inline-flex items-center gap-1">
              <IconPhone size={13} />
              <span dir="ltr">{p.phone}</span>
            </span>
            <span className="inline-flex items-center gap-1">
              <IconDate size={13} />
              {P.joined}: {new Date(p.created_at).toLocaleDateString("ar-SY")}
            </span>
            {p.invite_code && (
              <span dir="ltr" className="rounded-badge bg-accent/15 px-2 font-mono text-accent-dark">
                {p.invite_code}
              </span>
            )}
          </p>
          <div className="mt-2 flex flex-wrap gap-1">
            {p.roles.map((r) => (
              <RoleBadge key={r} role={r} />
            ))}
            <Badge variant={p.status === "active" ? "success" : "danger"}>
              {p.status === "active" ? m.admin.users.active : m.admin.users.blocked}
            </Badge>
          </div>
        </div>
        <div className="flex flex-col gap-2">
          {canWallet && (
            <Button onClick={() => setWalletOpen(true)} className="flex items-center gap-1.5">
              <IconWallet size={15} />
              {P.walletOps}
            </Button>
          )}
          {isAdmin && (
            <Button
              variant="secondary"
              onClick={() => setResetOpen(true)}
              className="flex items-center gap-1.5"
            >
              <IconLock size={15} />
              {P.resetPassword}
            </Button>
          )}
        </div>
      </div>

      {isAdmin && (
        <div className="mb-5 rounded-card border border-line bg-surface p-4">
          <ImageUpload
            kind="avatar"
            label={P.avatar}
            initialUrl={p.avatar_thumb_url}
            onChange={async (mediaID) => {
              await api(`/api/v1/admin/users/${p.id}`, {
                method: "PATCH",
                body: JSON.stringify({ avatar_media_id: mediaID }),
              });
              await load();
            }}
          />
        </div>
      )}

      {/* المؤشرات حسب الأدوار */}
      <div className="mb-5 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
        {stats.map((s) => (
          <div key={s.label} className="rounded-card border border-line bg-surface p-3">
            <div className="mb-1">{s.icon}</div>
            <p className="text-lg font-bold">{s.value}</p>
            <p className="text-xs text-ink-muted">{s.label}</p>
          </div>
        ))}
      </div>

      {p.merchants.length > 0 && (
        <div className="mb-5 rounded-card border border-line bg-surface p-4">
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

      {/* سجل المحفظة الكامل — هنا لا في النافذة (ملاحظة المراجعة) */}
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
                <span className="flex min-w-0 items-center gap-2">
                  <Badge variant={t.amount > 0 ? "success" : "danger"}>
                    {KINDS[t.kind] ?? t.kind}
                  </Badge>
                  {t.note && <span className="truncate text-xs text-ink-muted">{t.note}</span>}
                </span>
                <span className="flex shrink-0 items-center gap-3">
                  <span className={`font-bold ${t.amount > 0 ? "text-success" : "text-danger"}`} dir="ltr">
                    {t.amount > 0 ? "+" : ""}
                    {fmt.format(t.amount)}
                  </span>
                  <span className="text-xs text-ink-muted" dir="ltr">
                    {new Date(t.created_at).toLocaleDateString("ar-SY")}
                  </span>
                </span>
              </li>
            ))}
          </ul>
        )}
      </FormSection>

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
