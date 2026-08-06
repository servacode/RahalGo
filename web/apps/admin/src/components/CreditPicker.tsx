"use client";

/**
 * **اختيارُ من يُشحن له** — قبل نافذة المحفظة.
 *
 * `WalletModal` تعرف ماذا تفعل بمستخدمٍ **مُعطى**، ولا تعرف كيف يُختار. وكانت
 * تُفتح من جداولَ فيها المستخدمُ أمام عينك — **وفي قسم المال لا جدولَ يُنقر**،
 * فيلزم بحثٌ بالاسم أو الهاتف.
 *
 * **والبحثُ في الخادم لا في الشاشة**: قائمةُ المستخدمين مُصفَّحة، **وتحميلُها
 * كلَّها لتُرشَّح في المتصفّح يعمل اليومَ ويسقط بعد ألف حساب.**
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Modal, Input, IconSearch, IconUser } from "@rahalgo/ui";
import { api, type AuthUser } from "@/lib/api";
import RoleBadge from "@/components/RoleBadge";

const m = getMessages(defaultLocale);
const P = m.shared.payout;

export default function CreditPicker({
  onPick,
  onClose,
}: {
  onPick: (u: AuthUser) => void;
  onClose: () => void;
}) {
  const [query, setQuery] = useState("");
  const [rows, setRows] = useState<AuthUser[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const t = setTimeout(async () => {
      setLoading(true);
      try {
        const params = new URLSearchParams({ query, per_page: "10" });
        const res = await api<{ users: AuthUser[] }>(`/api/v1/admin/users?${params}`);
        setRows(res.users ?? []);
      } catch {
        setRows([]);
      }
      setLoading(false);
    }, 250);
    return () => clearTimeout(t);
  }, [query]);

  return (
    <Modal open onClose={onClose} title={P.creditPick}>
      <Input
        icon={<IconSearch />}
        placeholder={m.admin.users.searchPlaceholder}
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      <div className="mt-3 max-h-80 overflow-y-auto">
        {loading && <p className="py-6 text-center text-sm text-ink-muted">{m.common.loading}</p>}
        {!loading && rows.length === 0 && (
          <p className="py-6 text-center text-sm text-ink-muted">{P.creditNoResults}</p>
        )}
        <ul className="divide-y divide-line">
          {rows.map((u) => (
            <li key={u.id}>
              <button
                onClick={() => onPick(u)}
                className="flex w-full items-center gap-3 py-2.5 text-start hover:bg-row-hover"
              >
                <IconUser size={16} className="shrink-0 text-ink-muted" />
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-medium">{u.full_name || "—"}</span>
                  <span dir="ltr" className="block text-xs text-ink-muted">
                    {u.phone}
                  </span>
                </span>
                <span className="flex shrink-0 flex-wrap justify-end gap-1">
                  {u.roles.map((r) => (
                    <RoleBadge key={r} role={r} />
                  ))}
                </span>
              </button>
            </li>
          ))}
        </ul>
      </div>
    </Modal>
  );
}
