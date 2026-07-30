"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
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
  IconView,
} from "@rahalgo/ui";
import { api, ApiError, tokenStore, type AuthUser } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import WalletModal from "@/components/WalletModal";
import RoleBadge, { ROLE_STYLES } from "@/components/RoleBadge";
import { MediaThumb } from "@/components/ImageUpload";

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
  const router = useRouter();
  const isAdmin = !!me?.roles.includes("admin");

  const [data, setData] = useState<UserPage | null>(null);
  const [query, setQuery] = useState("");
  const [role, setRole] = useState("");
  const [onlineOnly, setOnlineOnly] = useState(false);
  const [statusFilter, setStatusFilter] = useState("");
  const [roleCounts, setRoleCounts] = useState<{ total: number; roles: Record<string, number> } | null>(null);
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [rolesUser, setRolesUser] = useState<AuthUser | null>(null);
  const [walletUser, setWalletUser] = useState<AuthUser | null>(null);
  const [statusModal, setStatusModal] = useState<{ user: AuthUser; status: string } | null>(null);
  const [view, setView] = useViewMode("users");

  const load = useCallback(async () => {
    try {
      const params = new URLSearchParams({ query, role, status: statusFilter, online: onlineOnly ? "true" : "", page: String(page), per_page: "10" });
      api<{ total: number; roles: Record<string, number> }>("/api/v1/admin/users/stats")
        .then(setRoleCounts)
        .catch(() => undefined);
      setData(await api<UserPage>(`/api/v1/admin/users?${params}`));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [query, role, statusFilter, onlineOnly, page]);

  useEffect(() => {
    const t = setTimeout(load, 250); // تهدئة البحث
    return () => clearTimeout(t);
  }, [load]);

  async function exportCsv() {
    const params = new URLSearchParams({ query, role, status: statusFilter, online: onlineOnly ? "true" : "" });
    const base = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
    const res = await fetch(`${base}/api/v1/admin/users/export?${params}`, {
      headers: { Authorization: `Bearer ${tokenStore.access ?? ""}` },
    });
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "accounts.csv";
    a.click();
    URL.revokeObjectURL(url);
  }

  async function setStatus(u: AuthUser, status: string, reason = "") {
    try {
      await api(`/api/v1/admin/users/${u.id}`, {
        method: "PATCH",
        body: JSON.stringify({ status, status_reason: reason }),
      });
      setStatusModal(null);
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
      cell: (u) => (
        <span className="flex w-full items-center justify-between gap-2">
          <span className="inline-flex min-w-0 items-center gap-2">
            <MediaThumb url={u.avatar_thumb_url} alt="" fallback={u.full_name || "؟"} size={32} />
            <span className="truncate">{u.full_name || "—"}</span>
          </span>
          <button
            type="button"
            title={m.admin.users.viewProfile}
            onClick={(e) => {
              e.stopPropagation();
              router.push(`/dashboard/users/${u.id}`);
            }}
            className="shrink-0 rounded-control p-1 text-ink-muted transition-colors hover:bg-primary-light hover:text-primary"
          >
            <IconView size={17} />
          </button>
        </span>
      ),
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
            <RoleBadge key={r} role={r} />
          ))}
        </div>
      ),
    },
    {
      id: "status",
      header: m.admin.users.table.status,
      icon: <IconStatus />,
      cell: (u) => (
        <Badge
          variant={u.status === "active" ? "success" : u.status === "suspended" ? "warning" : "danger"}
        >
          {u.status === "active"
            ? m.admin.users.active
            : u.status === "suspended"
              ? m.admin.users.suspended
              : m.admin.users.blocked}
        </Badge>
      ),
    },
    {
      id: "seen",
      header: m.admin.users.lastSeen,
      icon: <IconStatus />,
      cell: (u) => <PresenceCell lastSeen={u.last_seen_at} />,
    },
  ];

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold">{m.admin.users.title}</h1>
        <div className="flex gap-2">
          <Button
            variant="secondary"
            onClick={exportCsv}
            className="flex items-center gap-1.5"
          >
            <IconView size={16} />
            {m.admin.users.export}
          </Button>
          {isAdmin && (
            <Button onClick={() => setCreateOpen(true)} className="flex items-center gap-1.5">
              <IconAdd size={16} />
              {m.admin.users.create}
            </Button>
          )}
        </div>
      </div>

      {roleCounts && (
        <div className="mb-4 grid grid-cols-2 gap-2 sm:grid-cols-4 lg:grid-cols-7">
          <button
            onClick={() => setRole("")}
            className={`rounded-card border p-2.5 text-center transition-colors ${role === "" ? "border-primary bg-primary-light" : "border-line bg-surface hover:border-primary/40"}`}
          >
            <p className="text-lg font-bold">{roleCounts.total}</p>
            <p className="text-xs text-ink-muted">{m.admin.users.allRoles}</p>
          </button>
          {([
            { key: "staff", label: m.admin.users.staffCard, style: ROLE_STYLES.ops },
            { key: "sales", label: ROLE_LABELS.sales, style: ROLE_STYLES.sales },
            { key: "driver", label: ROLE_LABELS.driver, style: ROLE_STYLES.driver },
            { key: "merchant", label: ROLE_LABELS.merchant, style: ROLE_STYLES.merchant },
            { key: "customer", label: ROLE_LABELS.customer, style: ROLE_STYLES.customer },
          ] as const).map(({ key, label, style }) => (
            <button
              key={key}
              onClick={() => { setRole(role === key ? "" : key); setPage(1); }}
              className={`rounded-card border p-2.5 text-center transition-colors ${role === key ? "border-primary bg-primary-light" : "border-line bg-surface hover:border-primary/40"}`}
            >
              <p className={`inline-flex items-center gap-1 text-lg font-bold ${style ? style.cls.split(" ").filter((c) => c.startsWith("text-")).join(" ") : ""}`}>
                {style && <style.Icon size={15} />}
                {roleCounts.roles[key] ?? 0}
              </p>
              <p className="text-xs text-ink-muted">{label}</p>
            </button>
          ))}
          <button
            onClick={() => { setOnlineOnly(!onlineOnly); setPage(1); }}
            className={`rounded-card border p-2.5 text-center transition-colors ${onlineOnly ? "border-success bg-success/10" : "border-line bg-surface hover:border-success/40"}`}
          >
            <p className="inline-flex items-center gap-1.5 text-lg font-bold text-success">
              <span className="h-2 w-2 animate-pulse rounded-badge bg-success" />
              {roleCounts.roles.online ?? 0}
            </p>
            <p className="text-xs text-ink-muted">{m.admin.users.onlineCard}</p>
          </button>
        </div>
      )}

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
        <div className="w-36">
          <Select
            value={statusFilter}
            onChange={(e) => {
              setStatusFilter(e.target.value);
              setPage(1);
            }}
          >
            <option value="">{m.admin.users.statusFilter.all}</option>
            <option value="active">{m.admin.users.statusFilter.active}</option>
            <option value="suspended">{m.admin.users.statusFilter.suspended}</option>
            <option value="blocked">{m.admin.users.statusFilter.blocked}</option>
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
        onRowClick={(u) => router.push(`/dashboard/users/${u.id}`)}
        actions={
          isAdmin
            ? (u) => (
                <>
                  <Button
                    variant="secondary"
                    onClick={() => setWalletUser(u)}
                    className="flex items-center gap-1.5"
                  >
                    <IconWallet size={15} />
                    {m.admin.users.wallet}
                  </Button>
                  <Button
                    variant="secondary"
                    onClick={() => setRolesUser(u)}
                    className="flex items-center gap-1.5"
                  >
                    <IconRoles size={15} />
                    {m.admin.users.manageRoles}
                  </Button>
                  {u.id !== me?.id &&
                    (u.status === "active" ? (
                      <>
                        <Button
                          variant="secondary"
                          onClick={() => setStatusModal({ user: u, status: "suspended" })}
                          className="flex items-center gap-1.5 !text-warning"
                        >
                          <IconBlock size={15} />
                          {m.admin.users.suspend}
                        </Button>
                        <Button
                          variant="danger"
                          onClick={() => setStatusModal({ user: u, status: "blocked" })}
                          className="flex items-center gap-1.5"
                        >
                          <IconBlock size={15} />
                          {m.admin.users.block}
                        </Button>
                      </>
                    ) : (
                      <Button
                        variant="secondary"
                        onClick={() => setStatus(u, "active")}
                        className="flex items-center gap-1.5"
                      >
                        <IconUnblock size={15} />
                        {m.admin.users.activate}
                      </Button>
                    ))}
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
  const [password2, setPassword2] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  function toggleRole(r: string) {
    setRoles((prev) => (prev.includes(r) ? prev.filter((x) => x !== r) : [...prev, r]));
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (password !== password2) {
      setError(m.errors.password_mismatch);
      return;
    }
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
        <div className="grid grid-cols-2 gap-3">
          <Input
            id="new-password"
            label={m.admin.users.passwordRequired}
            icon={<IconLock />}
            type="password"
            required
            minLength={8}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <Input
            id="new-password2"
            label={m.admin.users.confirmPassword}
            icon={<IconLock />}
            type="password"
            required
            minLength={8}
            value={password2}
            onChange={(e) => setPassword2(e.target.value)}
          />
        </div>
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
  const [pending, setPending] = useState<{ role: string; adding: boolean } | null>(null);
  const [reason, setReason] = useState("");

  useEffect(() => {
    setCurrent(user?.roles ?? []);
    setError("");
    setPending(null);
    setReason("");
  }, [user]);

  if (!user) return null;

  async function apply() {
    if (!user || !pending) return;
    setError("");
    try {
      if (!pending.adding) {
        await api(
          `/api/v1/admin/users/${user.id}/roles/${pending.role}?reason=${encodeURIComponent(reason)}`,
          { method: "DELETE" }
        );
        setCurrent((p) => p.filter((r) => r !== pending.role));
      } else {
        await api(`/api/v1/admin/users/${user.id}/roles`, {
          method: "POST",
          body: JSON.stringify({ role: pending.role, reason }),
        });
        setCurrent((p) => [...p, pending.role]);
      }
      setPending(null);
      setReason("");
      await onChanged();
    } catch (err) {
      setError(errText(err));
    }
  }
  function toggle(role: string) {
    setPending({ role, adding: !current.includes(role) });
    setReason("");
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
      {pending && (
        <div className="mt-4 rounded-control border border-primary/30 bg-primary-light/40 p-3">
          <p className="mb-2 text-sm font-medium">
            {m.admin.users.roleReasonTitle}: {ROLE_LABELS[pending.role]}
          </p>
          <Input
            id="role-reason"
            label={m.admin.users.roleReasonLabel}
            required
            autoFocus
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />
          <div className="mt-3 flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setPending(null)}>
              {m.common.cancel}
            </Button>
            <Button disabled={!reason.trim()} onClick={apply}>
              {m.common.confirm}
            </Button>
          </div>
        </div>
      )}
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

// خلية الحضور: نقطة خضراء نابضة إن كان نشطاً خلال دقيقتين، وإلا آخر ظهور نسبي.
function PresenceCell({ lastSeen }: { lastSeen: string | null }) {
  if (!lastSeen) return <span className="text-xs text-ink-muted">{m.admin.users.neverSeen}</span>;
  const diffMin = Math.floor((Date.now() - new Date(lastSeen).getTime()) / 60000);
  if (diffMin < 2) {
    return (
      <span className="inline-flex items-center gap-1.5 text-xs font-medium text-success">
        <span className="h-2 w-2 animate-pulse rounded-badge bg-success" />
        {m.admin.users.online}
      </span>
    );
  }
  const rtf = new Intl.RelativeTimeFormat("ar", { numeric: "auto" });
  const label =
    diffMin < 60
      ? rtf.format(-diffMin, "minute")
      : diffMin < 1440
        ? rtf.format(-Math.floor(diffMin / 60), "hour")
        : rtf.format(-Math.floor(diffMin / 1440), "day");
  return <span className="text-xs text-ink-muted">{label}</span>;
}

function StatusReasonModal({
  target,
  onSubmit,
  onClose,
}: {
  target: { user: AuthUser; status: string };
  onSubmit: (reason: string) => void;
  onClose: () => void;
}) {
  const [reason, setReason] = useState("");
  const actionLabel =
    target.status === "suspended" ? m.admin.users.suspend : m.admin.users.block;
  return (
    <Modal open onClose={onClose} title={m.admin.users.statusReasonTitle.replace("{action}", actionLabel)}>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          onSubmit(reason);
        }}
        className="space-y-4"
      >
        <Input
          id="status-reason"
          label={m.admin.users.statusReasonLabel}
          required
          autoFocus
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button type="submit" variant={target.status === "blocked" ? "danger" : "primary"}>
            {actionLabel}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
