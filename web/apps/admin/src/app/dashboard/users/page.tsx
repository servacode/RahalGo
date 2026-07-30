"use client";

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Button,
  Input,
  Select,
  Badge,
  Modal,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  IconUser,
  IconPhone,
  IconRoles,
  IconStatus,
  IconSearch,
  IconAdd,
  IconBlock,
  IconUnblock,
  IconLock,
  IconWallet,
} from "@rahalgo/ui";
import { api, ApiError, type AuthUser } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);

const ROLE_LABELS: Record<string, string> = m.roles;
const ALL_ROLES = Object.keys(ROLE_LABELS);

interface UserPage {
  users: AuthUser[];
  total: number;
  page: number;
  per_page: number;
}

function translateKey(key: string): string {
  let node: unknown = m;
  for (const part of key.split(".")) {
    if (typeof node !== "object" || node === null) return m.errors.internal;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" ? node : m.errors.internal;
}

function errText(err: unknown): string {
  return err instanceof ApiError ? translateKey(err.body.message_key) : m.errors.internal;
}

export default function UsersPage() {
  const { user: me } = useAuth();
  const isAdmin = !!me?.roles.includes("admin");

  const [data, setData] = useState<UserPage | null>(null);
  const [query, setQuery] = useState("");
  const [role, setRole] = useState("");
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [rolesUser, setRolesUser] = useState<AuthUser | null>(null);
  const [walletUser, setWalletUser] = useState<AuthUser | null>(null);
  const [view, setView] = useViewMode("users");

  const load = useCallback(async () => {
    try {
      const params = new URLSearchParams({ query, role, page: String(page), per_page: "10" });
      setData(await api<UserPage>(`/api/v1/admin/users?${params}`));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [query, role, page]);

  useEffect(() => {
    const t = setTimeout(load, 250); // تهدئة البحث
    return () => clearTimeout(t);
  }, [load]);

  async function toggleBlock(u: AuthUser) {
    try {
      await api(`/api/v1/admin/users/${u.id}`, {
        method: "PATCH",
        body: JSON.stringify({ status: u.status === "active" ? "blocked" : "active" }),
      });
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  const totalPages = data ? Math.max(1, Math.ceil(data.total / data.per_page)) : 1;

  const columns: DataColumn<AuthUser>[] = [
    {
      id: "name",
      header: m.admin.users.table.name,
      icon: <IconUser />,
      primary: true,
      cell: (u) => u.full_name || "—",
    },
    {
      id: "phone",
      header: m.admin.users.table.phone,
      icon: <IconPhone />,
      primary: true,
      cell: (u) => (
        <span dir="ltr" className="font-medium">
          {u.phone}
        </span>
      ),
    },
    {
      id: "roles",
      header: m.admin.users.table.roles,
      icon: <IconRoles />,
      cell: (u) => (
        <div className="flex flex-wrap justify-end gap-1 sm:justify-start">
          {u.roles.map((r) => (
            <Badge key={r} variant={r === "admin" ? "primary" : "neutral"}>
              {ROLE_LABELS[r] ?? r}
            </Badge>
          ))}
          {u.invite_code && (
            <Badge variant="warning">
              <span dir="ltr" className="font-mono">{u.invite_code}</span>
            </Badge>
          )}
        </div>
      ),
    },
    {
      id: "status",
      header: m.admin.users.table.status,
      icon: <IconStatus />,
      cell: (u) => (
        <Badge variant={u.status === "active" ? "success" : "danger"}>
          {u.status === "active" ? m.admin.users.active : m.admin.users.blocked}
        </Badge>
      ),
    },
  ];

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold">{m.admin.users.title}</h1>
        {isAdmin && (
          <Button onClick={() => setCreateOpen(true)} className="flex items-center gap-1.5">
            <IconAdd size={16} />
            {m.admin.users.create}
          </Button>
        )}
      </div>

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="w-64">
          <Input
            icon={<IconSearch />}
            placeholder={m.admin.users.searchPlaceholder}
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setPage(1);
            }}
          />
        </div>
        <div className="w-44">
          <Select
            value={role}
            onChange={(e) => {
              setRole(e.target.value);
              setPage(1);
            }}
          >
            <option value="">{m.admin.users.allRoles}</option>
            {ALL_ROLES.map((r) => (
              <option key={r} value={r}>
                {ROLE_LABELS[r]}
              </option>
            ))}
          </Select>
        </div>
        <div className="ms-auto">
          <ViewToggle
            view={view}
            onChange={setView}
            tableLabel={m.common.viewTable}
            cardsLabel={m.common.viewCards}
          />
        </div>
      </div>

      {error && (
        <p className="mb-4 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      <DataView
        items={data?.users ?? []}
        getKey={(u) => u.id}
        columns={columns}
        view={view}
        empty={m.admin.users.noResults}
        actions={
          isAdmin
            ? (u) => (
                <>
                  <Button
                    variant="ghost"
                    onClick={() => setWalletUser(u)}
                    className="flex items-center gap-1.5"
                  >
                    <IconWallet size={15} />
                    {m.admin.users.wallet}
                  </Button>
                  <Button
                    variant="ghost"
                    onClick={() => setRolesUser(u)}
                    className="flex items-center gap-1.5"
                  >
                    <IconRoles size={15} />
                    {m.admin.users.manageRoles}
                  </Button>
                  {u.id !== me?.id && (
                    <Button
                      variant={u.status === "active" ? "danger" : "secondary"}
                      onClick={() => toggleBlock(u)}
                      className="flex items-center gap-1.5"
                    >
                      {u.status === "active" ? <IconBlock size={15} /> : <IconUnblock size={15} />}
                      {u.status === "active" ? m.admin.users.block : m.admin.users.unblock}
                    </Button>
                  )}
                </>
              )
            : undefined
        }
      />

      {data && (
        <div className="mt-4 flex items-center justify-between text-sm text-ink-muted">
          <span>{m.admin.users.totalCount.replace("{count}", String(data.total))}</span>
          <div className="flex items-center gap-2">
            <Button variant="secondary" disabled={page <= 1} onClick={() => setPage(page - 1)}>
              {m.admin.users.prev}
            </Button>
            <span>
              {page} / {totalPages}
            </span>
            <Button
              variant="secondary"
              disabled={page >= totalPages}
              onClick={() => setPage(page + 1)}
            >
              {m.admin.users.next}
            </Button>
          </div>
        </div>
      )}

      <CreateUserModal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onCreated={() => {
          setCreateOpen(false);
          void load();
        }}
      />
      <ManageRolesModal user={rolesUser} onClose={() => setRolesUser(null)} onChanged={load} />
      {walletUser && <WalletModal user={walletUser} onClose={() => setWalletUser(null)} isAdmin={isAdmin || !!me?.roles.includes("finance")} />}
    </div>
  );
}

function CreateUserModal({
  open,
  onClose,
  onCreated,
}: {
  open: boolean;
  onClose: () => void;
  onCreated: () => void;
}) {
  const [phone, setPhone] = useState("");
  const [fullName, setFullName] = useState("");
  const [roles, setRoles] = useState<string[]>(["driver"]);
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  function toggleRole(r: string) {
    setRoles((prev) => (prev.includes(r) ? prev.filter((x) => x !== r) : [...prev, r]));
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/admin/users", {
        method: "POST",
        body: JSON.stringify({ phone, full_name: fullName, roles, password }),
      });
      setPhone("");
      setFullName("");
      setRoles(["driver"]);
      setPassword("");
      onCreated();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title={m.admin.users.createTitle}>
      <form onSubmit={submit} className="space-y-4">
        <Input
          id="new-phone"
          label={m.auth.phone}
          icon={<IconPhone />}
          dir="ltr"
          required
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          placeholder="09xxxxxxxx"
          className="text-end"
        />
        <Input
          id="new-name"
          label={m.admin.users.fullName}
          icon={<IconUser />}
          value={fullName}
          onChange={(e) => setFullName(e.target.value)}
        />
        <div>
          <span className="mb-1 block text-sm font-medium">{m.admin.users.rolesLabel}</span>
          <div className="flex flex-wrap gap-2">
            {ALL_ROLES.map((r) => (
              <button
                type="button"
                key={r}
                onClick={() => toggleRole(r)}
                className={`rounded-badge border px-3 py-1 text-xs transition-colors ${
                  roles.includes(r)
                    ? "border-primary bg-primary-light text-primary-dark"
                    : "border-line text-ink-muted hover:border-primary/50"
                }`}
              >
                {ROLE_LABELS[r]}
              </button>
            ))}
          </div>
        </div>
        <Input
          id="new-password"
          label={m.admin.users.passwordOptional}
          icon={<IconLock />}
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        {error && (
          <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
        )}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button type="submit" disabled={busy || roles.length === 0}>
            {m.common.save}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

function ManageRolesModal({
  user,
  onClose,
  onChanged,
}: {
  user: AuthUser | null;
  onClose: () => void;
  onChanged: () => Promise<void> | void;
}) {
  const [error, setError] = useState("");
  const [current, setCurrent] = useState<string[]>([]);

  useEffect(() => {
    setCurrent(user?.roles ?? []);
    setError("");
  }, [user]);

  if (!user) return null;

  async function toggle(role: string) {
    if (!user) return;
    setError("");
    try {
      if (current.includes(role)) {
        await api(`/api/v1/admin/users/${user.id}/roles/${role}`, { method: "DELETE" });
        setCurrent((p) => p.filter((r) => r !== role));
      } else {
        await api(`/api/v1/admin/users/${user.id}/roles`, {
          method: "POST",
          body: JSON.stringify({ role }),
        });
        setCurrent((p) => [...p, role]);
      }
      await onChanged();
    } catch (err) {
      setError(errText(err));
    }
  }

  return (
    <Modal open onClose={onClose} title={`${m.admin.users.rolesFor}: ${user.full_name || user.phone}`}>
      <div className="flex flex-wrap gap-2">
        {ALL_ROLES.map((r) => (
          <button
            type="button"
            key={r}
            onClick={() => toggle(r)}
            className={`rounded-badge border px-3 py-1.5 text-sm transition-colors ${
              current.includes(r)
                ? "border-primary bg-primary-light text-primary-dark"
                : "border-line text-ink-muted hover:border-primary/50"
            }`}
          >
            {ROLE_LABELS[r]}
          </button>
        ))}
      </div>
      {error && (
        <p className="mt-3 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}
      <div className="mt-5 flex justify-end">
        <Button variant="secondary" onClick={onClose}>
          {m.common.back}
        </Button>
      </div>
    </Modal>
  );
}


const fmtNum = new Intl.NumberFormat("ar-SY");

interface WalletTx {
  id: number;
  amount: number;
  kind: string;
  ref: string;
  note: string;
  created_at: string;
}

function WalletModal({
  user,
  onClose,
  isAdmin,
}: {
  user: AuthUser;
  onClose: () => void;
  isAdmin: boolean;
}) {
  const KINDS: Record<string, string> = m.admin.users.txKinds;
  const [balance, setBalance] = useState<number | null>(null);
  const [txs, setTxs] = useState<WalletTx[]>([]);
  const [amount, setAmount] = useState("");
  const [kind, setKind] = useState("topup");
  const [debit, setDebit] = useState(false);
  const [note, setNote] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const loadWallet = useCallback(async () => {
    try {
      const st = await api<{ balance: number; transactions: WalletTx[] }>(
        `/api/v1/admin/users/${user.id}/wallet`,
      );
      setBalance(st.balance);
      setTxs(st.transactions);
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [user.id]);

  useEffect(() => {
    void loadWallet();
  }, [loadWallet]);

  async function apply(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    const sign = kind === "payout" || (kind === "adjustment" && debit) ? -1 : 1;
    try {
      await api(`/api/v1/admin/users/${user.id}/wallet`, {
        method: "POST",
        body: JSON.stringify({ amount: sign * (Number(amount) || 0), kind, note }),
      });
      setAmount("");
      setNote("");
      await loadWallet();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      size="lg"
      title={`${m.admin.users.walletTitle}: ${user.full_name || user.phone}`}
    >
      <div className="mb-4 flex items-center justify-between rounded-card bg-primary-light p-4">
        <span className="flex items-center gap-2 font-medium text-primary-dark">
          <IconWallet size={18} />
          {m.admin.users.balance}
        </span>
        <span className="text-2xl font-bold text-primary-dark">
          {balance === null ? "…" : `${fmtNum.format(balance)} ${m.common.currency}`}
        </span>
      </div>

      {isAdmin && (
        <form onSubmit={apply} className="mb-4 space-y-3 rounded-card border border-line p-4">
          <div className="grid grid-cols-2 gap-3">
            <Input
              id="w-amount"
              label={`${m.admin.users.amount} (${m.common.currency})`}
              type="number"
              min="1"
              required
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
            />
            <Select
              id="w-kind"
              label={m.admin.users.movementKind}
              value={kind}
              onChange={(e) => setKind(e.target.value)}
            >
              <option value="topup">{KINDS.topup}</option>
              <option value="compensation">{KINDS.compensation}</option>
              <option value="adjustment">{KINDS.adjustment}</option>
              <option value="payout">{KINDS.payout}</option>
            </Select>
          </div>
          {kind === "adjustment" && (
            <label className="flex cursor-pointer items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={debit}
                onChange={(e) => setDebit(e.target.checked)}
                className="h-4 w-4 accent-danger"
              />
              {m.admin.users.isDebit}
            </label>
          )}
          <Input
            id="w-note"
            label={m.admin.users.noteField}
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
          {error && (
            <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
          )}
          <Button type="submit" disabled={busy} className="w-full">
            {m.admin.users.applyMovement}
          </Button>
        </form>
      )}

      <h3 className="mb-2 text-sm font-bold">{m.admin.users.txHistory}</h3>
      {txs.length === 0 ? (
        <p className="rounded-control bg-page p-4 text-center text-sm text-ink-muted">
          {m.admin.users.noTx}
        </p>
      ) : (
        <ul className="max-h-60 space-y-1.5 overflow-y-auto">
          {txs.map((t) => (
            <li
              key={t.id}
              className="flex items-center justify-between rounded-control border border-line px-3 py-2 text-sm"
            >
              <span className="flex items-center gap-2">
                <Badge variant={t.amount > 0 ? "success" : "danger"}>
                  {KINDS[t.kind] ?? t.kind}
                </Badge>
                {t.note && <span className="text-xs text-ink-muted">{t.note}</span>}
              </span>
              <span className={`font-bold ${t.amount > 0 ? "text-success" : "text-danger"}`}>
                {t.amount > 0 ? "+" : ""}
                {fmtNum.format(t.amount)}
              </span>
            </li>
          ))}
        </ul>
      )}
    </Modal>
  );
}
