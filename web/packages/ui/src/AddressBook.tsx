"use client";

/**
 * دفتر العناوين — مركزي لأنه يُستعمل في مكانين: صفحة الحساب والسلّة.
 *
 * وبناؤه مرّتين كان سيُنتج نسختين تنحرفان — وقد رأينا ذلك ثلاث مرّات في هذا
 * المشروع (المحفظة، الشريط العلوي، محرّر الأصناف).
 *
 * ويعرض الخريطة **دائماً** لا عند الطلب: العنوان عندنا دبّوسٌ لا نصّ، ودقّتُه هي
 * ما يبلغ به السائق الباب. فإخفاء الخريطة خلف زرٍّ يجعل نصف العناوين بلا دبّوس دقيق.
 */

import { useCallback, useEffect, useState, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, Input, Badge } from "./components";
import { Alert } from "./feedback";
import { EmptyState } from "./layout";
import { IconLocation, IconAdd, IconDelete, IconCheck } from "./icons";

const m = getMessages(defaultLocale);
const A = m.site.addresses;

type ApiFn = <T>(path: string, init?: RequestInit) => Promise<T>;

export interface SavedAddress {
  id: string;
  label: string;
  address_text: string;
  lat: number;
  lng: number;
  is_default: boolean;
}

export function AddressBook({
  api,
  /** الخريطة تُحقن لأنها ثقيلة وتُحمَّل ديناميكياً في كل تطبيق على حدة */
  picker,
  onPick,
  selectedID,
}: {
  api: ApiFn;
  picker?: (
    value: { lat: number; lng: number; address: string } | null,
    onChange: (v: { lat: number; lng: number; address: string }) => void,
  ) => ReactNode;
  /** عند تمريرها تصير القائمة **قابلة للاختيار** — وضع السلّة */
  onPick?: (a: SavedAddress) => void;
  selectedID?: string;
}) {
  const [list, setList] = useState<SavedAddress[] | null>(null);
  const [adding, setAdding] = useState(false);
  const [label, setLabel] = useState("");
  const [pin, setPin] = useState<{ lat: number; lng: number; address: string } | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  /**
   * **وفشلُ القراءة لا يُعرض «لا عناوينَ محفوظة».**
   *
   * كان `.catch(() => setList([]))` — **فيرسم الزبونُ دبّوسَه من جديد** وهو
   * قد حفظ بيتَه من قبل، **ثمّ يجد عنوانين لبيتٍ واحد.**
   */
  const load = useCallback(() => {
    setError("");
    api<SavedAddress[]>("/api/v1/my/addresses")
      .then(setList)
      .catch(() => {
        setList([]);
        setError(m.errors.offline);
      });
  }, [api]);

  useEffect(load, [load]);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    if (!pin) return setError(A.needPin);
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/my/addresses", {
        method: "POST",
        body: JSON.stringify({
          label,
          address_text: pin.address,
          lat: pin.lat,
          lng: pin.lng,
        }),
      });
      setAdding(false);
      setLabel("");
      setPin(null);
      load();
    } catch (err) {
      const key =
        typeof err === "object" && err && "body" in err
          ? ((err as { body?: { message_key?: string } }).body?.message_key ?? "").split(".").pop() ?? ""
          : "";
      setError((m.errors as Record<string, string>)[key] ?? m.errors.internal);
    } finally {
      setBusy(false);
    }
  }

  if (list === null) return null;

  return (
    <div className="space-y-3">
      {list.length === 0 && !adding ? (
        <EmptyState icon={IconLocation} title={A.empty} />
      ) : (
        <ul className="space-y-2">
          {list.map((a) => {
            const chosen = onPick && (selectedID === a.id || (!selectedID && a.is_default));
            return (
              <li
                key={a.id}
                onClick={onPick ? () => onPick(a) : undefined}
                className={`flex items-center gap-3 rounded-card border p-3 ${
                  chosen ? "border-primary bg-primary-light" : "border-line bg-surface"
                } ${onPick ? "cursor-pointer" : ""}`}
              >
                <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-control bg-page">
                  {chosen ? (
                    <IconCheck size={17} strokeWidth={3} className="text-primary-dark" />
                  ) : (
                    <IconLocation size={17} className="text-ink-muted" />
                  )}
                </span>
                <div className="min-w-0 flex-1">
                  <p className="flex items-center gap-2 font-medium">
                    {a.label}
                    {a.is_default && <Badge variant="neutral">{A.default}</Badge>}
                  </p>
                  <p className="truncate text-xs text-ink-muted">{a.address_text}</p>
                </div>
                <div className="flex shrink-0 gap-1" onClick={(e) => e.stopPropagation()}>
                  {!a.is_default && (
                    <Button
                      variant="ghost"
                      onClick={async () => {
                        await api(`/api/v1/my/addresses/${a.id}/default`, { method: "POST" });
                        load();
                      }}
                      title={A.makeDefault}
                    >
                      <IconCheck size={15} />
                    </Button>
                  )}
                  <Button
                    variant="ghost"
                    onClick={async () => {
                      await api(`/api/v1/my/addresses/${a.id}`, { method: "DELETE" });
                      load();
                    }}
                  >
                    <IconDelete size={15} className="text-danger" />
                  </Button>
                </div>
              </li>
            );
          })}
        </ul>
      )}

      {adding ? (
        <form onSubmit={save} className="space-y-3 rounded-card border border-line bg-surface p-4">
          <div>
            <Input
              id="addr-label"
              label={A.label}
              required
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              placeholder={A.labelPlaceholder}
            />
            <p className="mt-1 text-xs text-ink-muted">{A.labelHint}</p>
          </div>
          {picker?.(pin, setPin)}
          {error && (
            <Alert>{error}</Alert>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="secondary" onClick={() => setAdding(false)}>
              {m.common.cancel}
            </Button>
            <Button type="submit" disabled={busy}>
              {m.common.save}
            </Button>
          </div>
        </form>
      ) : (
        <Button variant="secondary" onClick={() => setAdding(true)} className="flex items-center gap-1.5">
          <IconAdd size={16} />
          {A.add}
        </Button>
      )}
    </div>
  );
}
