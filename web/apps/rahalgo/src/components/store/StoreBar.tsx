"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شريطُ المتجر — اسمُه وحالُه ومفتاحُ «مفتوح / مغلق»**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `StoreSwitcher.kt` وبطاقةِ الحال في `StoreScreen.kt`.)
 *
 * **«مفتوح / مغلق» لا «يستقبل الطلبات»** — تصحيحُ المالك ٢٠٢٦-٠٨-٢٣ في
 * التطبيق: «المتجر إذا كان مفتوحاً يستقبل وإذا كان مغلقاً لا يستقبل».
 * **ولفظان لشيءٍ واحدٍ يجعلانه يظنّهما مفتاحين.**
 *
 * **والحالُ أربعٌ من الخادم** — موقوفٌ من الإدارة · مغلقٌ مؤقتاً (المفتاح) ·
 * مغلقٌ حسب ساعات العمل · مفتوح. **والمفتاحُ يحكم الإغلاقَ الطارئ وحدَه**
 * (`emergency_closed`): من فتحه خارجَ ساعاته يبقى «مغلق حسب ساعات العمل»
 * ويُقال له متى يفتح — **وهو سببُ «لماذا لا يفتح» كما يقوله المحرّك.**
 */

import { useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, Confirm, IconStore, Select, Switch } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { backAt, fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const S = m.store;

export default function StoreBar() {
  const { store, stores, select, patch, reload } = useStore();
  const toast = useShopToast();
  const [saving, setSaving] = useState(false);
  // **والفتحُ بلا سؤالٍ والإغلاقُ بسؤال** (مرآةُ `StoreScreen.kt`، تدقيقُ ٢٠٢٦-١٠-٢٧) —
  // لمسةٌ بالغلط كانت تُسكّر المتجرَ عن الزبائن بلا سؤال.
  const [asking, setAsking] = useState(false);
  if (!store) return null;

  const suspended = store.status === "suspended";
  const label = suspended
    ? S.state.suspended
    : store.emergency_closed
      ? S.state.tempClosed
      : !store.open_now
        ? S.state.closedHours
        : S.state.open;
  const next = !suspended && !store.emergency_closed && !store.open_now ? backAt(store.next_open) : "";

  async function setOpen(on: boolean) {
    if (!store) return;
    setSaving(true);
    try {
      const r = await api<{ emergency_closed: boolean }>(
        `/api/v1/merchant/stores/${store.id}/emergency`,
        { method: "POST", body: JSON.stringify({ closed: !on }) },
      );
      patch({ emergency_closed: r.emergency_closed });
      // **والحالُ الفعليُّ من الخادم** — «مفتوح» قد يبقى «مغلق حسب الساعات».
      void reload();
    } catch (e) {
      toast(errorText(e), "error");
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="surface flex flex-col gap-3 p-3" aria-label={S.nav.store}>
      <div className="flex items-center gap-2">
        <IconStore size={18} className="shrink-0 text-accent-text" />
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
      </div>
      <div
        className={`rounded-control border px-3 py-2 ${
          suspended
            ? "border-danger-edge bg-danger-tint"
            : label === S.state.open
              ? "border-success-edge bg-success-tint"
              : "border-line bg-field"
        }`}
      >
        <Switch
          checked={!store.emergency_closed}
          disabled={saving || suspended}
          onChange={(on) => (on ? void setOpen(true) : setAsking(true))}
          label={label}
          hint={next ? fill(S.state.nextOpen, { when: next }) : S.state.switchLabel}
        />
      </div>
      {suspended && <Alert tone="error">{S.suspendedBanner}</Alert>}
      <Confirm
        open={asking}
        title={S.settings.emergencyQ}
        body={S.settings.emergencyHint}
        confirmLabel={S.settings.emergencyDo}
        busy={saving}
        onConfirm={() => {
          setAsking(false);
          void setOpen(false);
        }}
        onCancel={() => setAsking(false)}
      />
    </section>
  );
}
