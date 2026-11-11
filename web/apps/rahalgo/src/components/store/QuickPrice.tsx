"use client";

/**
 * **تعديلُ السعر السريع** (قرارُ المالك ٢٠٢٦-١١-١١) — لمسةٌ على سعر الصنف تفتح نافذةً صغيرةً فيها
 * **حقلُ السعر وحدَه** وبعملة الصنف (دولارٌ أو ليرة)، ويُحفظ بنداء تحديث الصنف الموجود.
 * مرآةُ `QuickPriceDialog` في `ui/menu/QuickPrice.kt` بالتطبيق.
 */

import { useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, Button, Input, Modal } from "@rahalgo/ui";
import { api } from "@/lib/api";
import type { MenuItem, MenuItemsPage } from "@/lib/store/types";
import { itemUsd, parseUsd, usdInput, usdText } from "@/lib/store/usd";

const m = getMessages(defaultLocale);
const M = m.store.menu;

export function QuickPriceModal({
  item,
  storeId,
  onClose,
  onSaved,
}: {
  item: MenuItem;
  storeId: string;
  onClose: () => void;
  /** **الصنفُ كما يردّه الخادمُ بعد الحفظ** — ليتحدّث معه سعرُه بالليرة. */
  onSaved: (fresh: MenuItem) => void;
}) {
  const usd = itemUsd(item) != null || item.price_currency === "USD";
  const [text, setText] = useState(usd ? usdText(item.merchant_price_usd) : String(item.price));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const valid = usd ? parseUsd(text) != null : text !== "" && Number(text) >= 0;

  async function save() {
    if (!valid || busy) return;
    setBusy(true);
    setError("");
    try {
      const body = usd
        ? { price_currency: "USD", price_usd: parseUsd(text) }
        : { price_currency: "SYP", price: Number(text) };
      await api(`/api/v1/merchant/menu/items/${item.id}`, { method: "PATCH", body: JSON.stringify(body) });
      const pg = await api<MenuItemsPage>(`/api/v1/merchant/stores/${storeId}/menu/items?item=${item.id}`);
      const fresh = pg.items[0];
      if (fresh) onSaved(fresh);
      onClose();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={() => !busy && onClose()} title={M.editPrice}>
      <form
        className="flex flex-col gap-rg-md"
        onSubmit={(e) => {
          e.preventDefault();
          void save();
        }}
      >
        <p className="line-clamp-2 text-caption text-ink-muted">{item.name}</p>
        <Input
          id="quick-price"
          label={usd ? M.priceUsd : M.priceSyp}
          inputMode={usd ? "decimal" : "numeric"}
          dir="ltr"
          autoFocus
          value={text}
          onChange={(e) => setText(usd ? usdInput(e.target.value) : e.target.value.replace(/\D/g, "").slice(0, 12))}
          data-testid="quick-price-input"
        />
        {error && <Alert tone="error">{error}</Alert>}
        <Button type="submit" disabled={!valid || busy} data-testid="quick-price-save">
          {m.common.save}
        </Button>
      </form>
    </Modal>
  );
}
