"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«شحن محفظة» من الخزينة** — قرارُ المالك ٢٠٢٦-١٠-٠٥
 * ══════════════════════════════════════════════════════════════════════
 *
 * **الماليّةُ لا ترى الحسابات** — واقتراحُ حركةٍ يدويّةٍ كان من ملفّ الحساب وحدَه.
 * **فبحثٌ ضيّق** (`GET /treasury/wallet-lookup` بقدرة `finance.manage`) يردّ المعرّفَ
 * والاسمَ والهاتفَ والأدوارَ والرصيد لا غير، **ثمّ نافذةُ المحفظة نفسُها** (`WalletModal`):
 * الطلبُ من بابه القائم، وسقفُه `finance.manual_wallet_max`، **ولا يوافق عليه مقترحُه.**
 */

import { useEffect, useState } from "react";
import { fmtMoney, errorText } from "@rahalgo/i18n";
import { Alert, Badge, Button, EmptyState, Input, LoadingState, IconWallet } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { roleLabelByCode } from "@/lib/rolemeta";
import WalletModal from "@/components/admin/WalletModal";
import { T } from "./shared";

interface Account {
  id: string;
  name: string;
  phone?: string;
  roles: string[];
  balance: number;
}

export function TopupTab() {
  const P = T.topup;
  const [q, setQ] = useState("");
  const [rows, setRows] = useState<Account[] | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [target, setTarget] = useState<Account | null>(null);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    const term = q.trim();
    if (term.length < 2) {
      setRows(null);
      return;
    }
    let alive = true;
    // **وبعد توقّف الكتابة** — لا نداءٌ لكلّ حرف.
    const t = window.setTimeout(() => {
      setBusy(true);
      api<{ accounts: Account[] }>(`/api/v1/admin/treasury/wallet-lookup?q=${encodeURIComponent(term)}`)
        .then((r) => {
          if (!alive) return;
          setRows(r.accounts ?? []);
          setError("");
        })
        .catch((e) => alive && setError(errorText(e)))
        .finally(() => alive && setBusy(false));
    }, 300);
    return () => {
      alive = false;
      window.clearTimeout(t);
    };
  }, [q, tick]);

  return (
    <div className="space-y-3">
      <p className="text-sm text-ink-muted">{P.hint}</p>
      <Input
        id="topup-search"
        label={P.search}
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder={P.minChars}
      />
      {error && <Alert>{error}</Alert>}
      {busy && rows === null ? (
        <LoadingState />
      ) : rows !== null && rows.length === 0 ? (
        <EmptyState icon={IconWallet} title={P.none} />
      ) : (
        <ul className="space-y-2">
          {(rows ?? []).map((a) => (
            <li key={a.id} className="surface flex flex-wrap items-center gap-3 p-3">
              <span className="min-w-0 flex-1">
                <span className="block font-medium">{a.name || a.phone || "—"}</span>
                {/* **والرقمُ لمن يملك قدرةَ الاتّصال** — يحذفه المحرّكُ عن غيره (`XG-42`). */}
                {a.phone && (
                  <span className="block text-xs text-ink-muted" dir="ltr">
                    {a.phone}
                  </span>
                )}
              </span>
              <span className="flex flex-wrap gap-1">
                {a.roles.map((r) => (
                  <Badge key={r} variant="neutral">
                    {roleLabelByCode(r)}
                  </Badge>
                ))}
              </span>
              <span className="text-sm">
                {P.balance}: <span className="figure">{fmtMoney(a.balance)}</span>
              </span>
              <Button variant="secondary" onClick={() => setTarget(a)}>
                {P.propose}
              </Button>
            </li>
          ))}
        </ul>
      )}
      {target && (
        <WalletModal
          user={{ id: target.id, phone: target.phone ?? "", full_name: target.name }}
          isAdmin
          onClose={() => {
            setTarget(null);
            setTick((n) => n + 1);
          }}
        />
      )}
    </div>
  );
}
