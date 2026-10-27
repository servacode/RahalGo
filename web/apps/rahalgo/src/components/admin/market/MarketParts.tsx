"use client";

/**
 * **قطعُ السوق المشتركة** (تدقيقُ السوق ٢٠٢٦-١٠-٢٧) — حالةُ الخطأ وحالةُ الفراغ بشكلٍ واحدٍ في كلّ
 * تبويب: دائرةٌ ملوّنة وعنوانٌ وجملةٌ تقول شو يعمل الموظّف، وزرٌّ إن لزم.
 *
 * **وكانت التبويباتُ لا تقرأ الخطأ أصلاً** — فينقطع النتُّ أو يُردّ ٤٠٣ والدولابُ يلفّ للأبد، وتبويبُ
 * التصنيفات يقول «ما في تصنيفات» فيُضيفها الموظّفُ مرّةً ثانية.
 */

import { useEffect, useState, type ComponentType, type ReactNode } from "react";
import { getMessages, defaultLocale, fmtStoreNo } from "@rahalgo/i18n";
import { Badge, Button, Input, LoadingState, Modal, IconSearch, IconStore, IconWarning } from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { storeStatusVariant } from "@/components/admin/StoreActions";

const m = getMessages(defaultLocale);
const U = m.admin.market.ui;

type IconT = ComponentType<{ size?: number; className?: string }>;

const TONE = {
  success: "bg-success-solid text-on-solid",
  warning: "bg-warning-solid text-on-solid",
  danger: "bg-danger-solid text-on-solid",
  accent: "bg-accent text-on-bright",
  muted: "bg-field text-ink-muted",
} as const;

/** **حالةٌ في وسط بطاقة** — دائرةٌ بلونها، وعنوان، وجملةٌ تساعد، وزرٌّ اختياريّ. */
export function StateBox({
  icon: Icon,
  tone = "muted",
  title,
  hint,
  action,
}: {
  icon: IconT;
  tone?: keyof typeof TONE;
  title: string;
  hint?: string;
  action?: ReactNode;
}) {
  return (
    <section className="surface flex flex-col items-center gap-3 p-8 text-center">
      <span className={`inline-flex h-16 w-16 items-center justify-center rounded-full ${TONE[tone]}`}>
        <Icon size={28} />
      </span>
      <h2 className="heading-section">{title}</h2>
      {hint && <p className="max-w-md text-sm text-ink-muted">{hint}</p>}
      {action}
    </section>
  );
}

/** **ما قدرنا نجيب البيانات** — دائرةٌ حمراء وزرُّ «جرّب مرة تانية» ينادي `reload`. */
export function LoadError({ onRetry }: { onRetry: () => void }) {
  return (
    <StateBox
      icon={IconWarning}
      tone="danger"
      title={U.loadErrorTitle}
      hint={U.loadErrorHint}
      action={
        <Button variant="secondary" onClick={onRetry}>
          {U.retry}
        </Button>
      }
    />
  );
}

export interface PickedStore {
  id: string;
  name: string;
  number: number;
}

/**
 * **اختيارُ متجرٍ بالبحث** (تدقيقُ السوق ٢٠٢٦-١٠-٢٧) — بدل قائمةٍ منسدلةٍ ثابتةٍ تنقص بعد مئة متجرٍ بلا
 * كلمة. **بالاسم أو بالرقم** («M-1045» أو «1045»)، من باب السوق نفسِه (`/admin/market/stores`).
 */
export function StorePickerModal({
  title,
  hint,
  onPick,
  onClose,
}: {
  title: string;
  hint?: string;
  onPick: (s: PickedStore) => void;
  onClose: () => void;
}) {
  const [q, setQ] = useState("");
  const [rows, setRows] = useState<(PickedStore & { status: string; logo_url: string | null })[] | null>(null);
  const [failed, setFailed] = useState(false);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let alive = true;
    const t = setTimeout(() => {
      const qs = new URLSearchParams({ q: q.trim(), per_page: "20" });
      api<{ stores: (PickedStore & { status: string; logo_url: string | null })[] }>(`/api/v1/admin/market/stores?${qs}`)
        .then((r) => {
          if (!alive) return;
          setRows(r.stores ?? []);
          setFailed(false);
        })
        .catch(() => alive && setFailed(true));
    }, 250);
    return () => {
      alive = false;
      clearTimeout(t);
    };
  }, [q, tick]);

  return (
    <Modal open title={title} onClose={onClose}>
      <div className="space-y-3">
        {hint && <p className="text-sm text-ink-muted">{hint}</p>}
        <Input
          id="mkt-store-q"
          icon={<IconSearch />}
          placeholder={U.searchStore}
          value={q}
          autoFocus
          onChange={(e) => setQ(e.target.value)}
        />
        {failed ? (
          <div className="flex flex-col items-center gap-2 py-4 text-center">
            <p className="text-sm text-danger">{U.loadErrorTitle}</p>
            <Button variant="secondary" onClick={() => setTick((n) => n + 1)}>
              {U.retry}
            </Button>
          </div>
        ) : rows === null ? (
          <LoadingState variant="inline" />
        ) : rows.length === 0 ? (
          <p className="py-4 text-center text-sm text-ink-muted">{U.noStoresFound}</p>
        ) : (
          <ul className="max-h-80 space-y-1.5 overflow-y-auto">
            {rows.map((s) => (
              <li key={s.id}>
                <button
                  type="button"
                  onClick={() => onPick({ id: s.id, name: s.name, number: s.number })}
                  className="flex w-full items-center gap-3 rounded-card border border-line-soft p-2.5 text-start transition-colors hover:bg-row-hover"
                >
                  <span className="flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-full bg-field">
                    {s.logo_url ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={mediaUrl(s.logo_url) ?? ""} alt={s.name} className="h-full w-full object-cover" />
                    ) : (
                      <IconStore size={18} className="text-ink-muted" />
                    )}
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="block truncate font-bold">{s.name}</span>
                    <span className="block text-xs text-ink-muted" dir="ltr">
                      {fmtStoreNo(s.number)}
                    </span>
                  </span>
                  {s.status !== "active" && <Badge variant={storeStatusVariant(s.status)}>{storeStatusLabel(s.status)}</Badge>}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </Modal>
  );
}

/** **اسمُ حال المتجر بالعربي دائماً** — وحالٌ جديدةٌ لا يعرفها المعجمُ تُقال «حالة تانية» لا رمزُها. */
export function storeStatusLabel(status: string): string {
  return (m.admin.merchants as unknown as Record<string, string>)[status] ?? U.statusOther;
}
