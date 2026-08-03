"use client";

/**
 * **قسمُ «طلبات قادمة»** — قائمٌ بذاته، لا ذيلاً لمهامّي.
 *
 * # ما يُعرض
 *
 * **كلُّ طلبٍ لم يُسنَد إلى سائقٍ بعد** — ومجرّدُ أخذه ينتقل إلى «مهامّي».
 *
 * وفي نمط «بالترتيب» **لا يرى السائقُ إلّا ما عُرض عليه في دوره**: ذاك قرارُ
 * التوزيع لا قرارُ هذه الشاشة. **وشاشةٌ تعرض ما لا يستطيع أخذَه تُعلّمه ألّا
 * يثق بما يرى.**
 *
 * # والتنبيهُ الصوتيّ
 *
 * السائقُ على درّاجته لا أمام شاشته. **وطلبٌ يظهر صامتاً يُقرأ بعد أن ينقضي
 * دورُه** — فيراه ذاهباً لا قادماً، ويظنّ أنّ المنصة لا تعطيه شيئاً.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { EmptyState, IconOrder, IconDriver, useLiveRefresh } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { IncomingCard, type DriverOrder } from "@/components/incoming";

const m = getMessages(defaultLocale);
const D = m.driver;

function errText(e: unknown): string {
  const key = e instanceof ApiError ? ((e.body.message_key ?? "").split(".").pop() ?? "") : "";
  return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
}

export default function IncomingPage() {
  const [rows, setRows] = useState<DriverOrder[]>([]);
  const [onShift, setOnShift] = useState<boolean | null>(null);
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");

  /**
   * **نغمةٌ تُولَّد في المتصفّح** — لا ملفَّ صوتٍ يُنتظر من الشبكة، ولا ملفَّ
   * يضيع في نشرةٍ قادمة.
   */
  const known = useRef<Set<string>>(new Set());
  const first = useRef(true);
  const chime = useCallback(() => {
    try {
      const Ctx =
        window.AudioContext ??
        (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
      if (!Ctx) return;
      const ctx = new Ctx();
      [880, 1175].forEach((hz, i) => {
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();
        osc.frequency.value = hz;
        osc.connect(gain);
        gain.connect(ctx.destination);
        const t = ctx.currentTime + i * 0.18;
        gain.gain.setValueAtTime(0.0001, t);
        gain.gain.exponentialRampToValueAtTime(0.25, t + 0.02);
        gain.gain.exponentialRampToValueAtTime(0.0001, t + 0.16);
        osc.start(t);
        osc.stop(t + 0.18);
      });
    } catch {
      // صوتٌ لا يعمل لا يُسقط الشاشة — **والبطاقةُ تظهر على أيّ حال.**
    }
  }, []);

  const load = useCallback(() => {
    api<{ on_shift: boolean }>("/api/v1/driver/me")
      .then((me) => setOnShift(me.on_shift))
      .catch(() => undefined);
    api<DriverOrder[]>("/api/v1/driver/queue")
      .then((list) => {
        // **يرنّ للجديد وحدَه** — ولا يرنّ لأوّل تحميل، **وإلّا رنّ كلَّما فتح
        // الشاشةَ فصار الصوتُ خبراً كاذباً.**
        const fresh = list.some((o) => !known.current.has(o.id));
        known.current = new Set(list.map((o) => o.id));
        if (fresh && !first.current) chime();
        first.current = false;
        setRows(list);
      })
      .catch(() => undefined);
  }, [chime]);

  useEffect(load, [load]);
  useLiveRefresh(["order"], load);

  async function accept(o: DriverOrder) {
    setBusy(o.id);
    setError("");
    try {
      await api(`/api/v1/driver/orders/${o.id}/accept`, { method: "POST" });
      // **ومجرّدُ الأخذ ينتقل إلى «مهامّي»** — فيختفي من هنا في التحميل التالي.
      load();
    } catch (e) {
      setError(errText(e));
      load();
    } finally {
      setBusy("");
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="text-xl font-bold">{D.queue.title}</h1>

      {error && (
        <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      {onShift === false ? (
        <EmptyState icon={IconDriver} title={D.queue.offEmpty} />
      ) : rows.length === 0 ? (
        <EmptyState icon={IconOrder} title={D.queue.empty} />
      ) : (
        <div className="grid gap-3 xl:grid-cols-2">
          {rows.map((o) => (
            <IncomingCard key={o.id} o={o} busy={busy === o.id} onAccept={() => void accept(o)} />
          ))}
        </div>
      )}
    </div>
  );
}
