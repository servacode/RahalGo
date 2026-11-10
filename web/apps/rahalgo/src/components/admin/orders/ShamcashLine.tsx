"use client";

import { getMessages, defaultLocale, fmtMoney, fmtUsd } from "@rahalgo/i18n";

const m = getMessages(defaultLocale);
const S = m.admin.shamcash;

/** **حقولُ شام كاش في معلومات اللوحة** (`BoardInfo` بالخادم) — والدولارُ بالسنتات. */
export interface ShamcashBoard {
  shamcash_currency?: string;
  shamcash_amount?: number;
  shamcash_status?: string;
}

/** **«انقبض شام كاش: X»** نصّاً — وفارغٌ حين قبضه السائقُ نقداً كلَّه (قرارُ المالك ٢٠٢٦-١١-١٠). */
export function shamcashText(b: ShamcashBoard | null | undefined): string {
  if (!b?.shamcash_amount || !b.shamcash_currency) return "";
  const amount = b.shamcash_currency === "USD" ? fmtUsd(b.shamcash_amount) : fmtMoney(b.shamcash_amount);
  return b.shamcash_status === "not_arrived" ? `${amount} (${S.orderLineNotArrived})` : amount;
}

export const SHAMCASH_LABEL = S.orderLine;
