"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شريطُ المتجر — اسمُه وحالُه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `StoreSwitcher.kt`.) **ولا مفتاحَ «مفتوح / مغلق» يدويّ** (ملاحظةُ المالك ٦،
 * ٢٠٢٦-١٠-٢٨): الحالُ تتبع ساعاتِ الدوام، **والإغلاقُ الطارئُ في شريط الحال في
 * «متجري» وحدَه.** وهنا الاسمُ (أو مبدّلُ الفروع) وشارةُ الحال للقراءة.
 *
 * **والحالُ أربعٌ من الخادم** — موقوفٌ من الإدارة · مغلقٌ مؤقتاً · مغلقٌ حسب
 * ساعات العمل · مفتوح.
 */

import Link from "next/link";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Alert, IconStore, Select } from "@rahalgo/ui";
import { useStore } from "@/lib/store/context";

const m = getMessages(defaultLocale);
const S = m.store;

export default function StoreBar() {
  const { store, stores, select } = useStore();
  if (!store) return null;

  const suspended = store.status === "suspended";
  const label = suspended
    ? S.state.suspended
    : store.emergency_closed
      ? S.state.tempClosed
      : !store.open_now
        ? S.state.closedHours
        : S.home.openNow;
  // **أرضٌ فاتحةٌ من لون الحال وحرفٌ بلونه** — كشريط الحال في «متجري» (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩).
  const tone = suspended
    ? "bg-danger-tint text-danger"
    : store.emergency_closed
      ? "bg-warning-tint text-warning"
      : !store.open_now
        ? "bg-field text-ink-muted"
        : "bg-success-tint text-success";

  return (
    <section className="surface flex flex-col gap-2 p-3" aria-label={S.nav.store}>
      <div className="flex items-center gap-2">
        <IconStore size={18} className="shrink-0 text-primary" />
        {stores.length > 1 ? (
          <Select
            aria-label={S.pickStore}
            value={store.id}
            onChange={(e) => select(e.target.value)}
            className="min-w-0 flex-1 font-bold"
          >
            {stores.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </Select>
        ) : (
          <h1 className="min-w-0 flex-1 truncate text-base font-bold">{store.name}</h1>
        )}
        <Link href="/store" className={`shrink-0 rounded-badge px-3 py-1 text-xs font-bold ${tone}`} data-testid="store-state">
          {label}
        </Link>
      </div>
      {suspended && <Alert tone="error">{S.suspendedBanner}</Alert>}
    </section>
  );
}
