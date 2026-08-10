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
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Alert, EmptyState, IconOrder, IconDriver, useLiveRefresh, useChime } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { IncomingCard, type DriverOrder } from "@/components/driver/incoming";

const m = getMessages(defaultLocale);
const D = m.driver;

function errText(e: unknown): string {
  const key = e instanceof ApiError ? ((e.body.message_key ?? "").split(".").pop() ?? "") : "";
  return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
}

export default function IncomingPage() {
  const router = useRouter();
  const [rows, setRows] = useState<DriverOrder[]>([]);
  const [onShift, setOnShift] = useState<boolean | null>(null);
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");

  /** **والنغمةُ من الحزمة المشتركة** — تلزم في «مهامّي» أيضاً بعد الإسناد
      المباشر، **ونسخةٌ ثانيةٌ منها تفترق يوماً.** */
  const known = useRef<Set<string>>(new Set());
  const first = useRef(true);
  const chime = useChime();
  const [speed, setSpeed] = useState(20);

  const load = useCallback(() => {
    api<{ on_shift: boolean; avg_speed_kmh: number }>("/api/v1/driver/me")
      .then((me) => {
        setOnShift(me.on_shift);
        // **والسرعةُ منها يُحسب الوقتُ** — تُقرأ مع الدوام في النداء نفسِه،
        // **ونداءٌ ثانٍ لرقمٍ واحدٍ رحلةٌ زائدةٌ على شبكةِ درّاجة.**
        setSpeed(me.avg_speed_kmh || 20);
      })
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
      /**
       * **ومن أخذ طلباً يُنقَل إليه — لا يُترك أمام قائمةٍ نقص منها.**
       *
       * كان يبقى في «طلبات قادمة» ويختفي الطلبُ من تحته. **فيُقرأ الأخذُ
       * فشلاً**: ضغط فذهب ما ضغط عليه ولم يظهر شيء. ثمّ يبحث عنه في
       * التبويبات، **وثوانٍ يقضيها في البحث ثوانٍ ينتظرها الزبون.**
       *
       * (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «بعد الضغط على زرّ خذ الطلب يجب أن ينتقل
       * بشكلٍ تلقائيّ إلى مهامّي ليكمل مهامّ الطلب».)
       *
       * **والانتقالُ قبل التحميل**: لا معنى لتحديث شاشةٍ نغادرها.
       */
      router.push("/driver");
    } catch (e) {
      // **وفشلُ الأخذ يبقيه هنا** — «سبقك غيرُك» خبرٌ يُقرأ في مكانه،
      // **ونقلٌ إلى مهامَّ لم تزد شيئاً يجعل الخبرَ يضيع في الطريق.**
      setError(errText(e));
      load();
    } finally {
      setBusy("");
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="heading-section">{D.queue.title}</h1>

      {error && (
        <Alert>{error}</Alert>
      )}

      {onShift === false ? (
        <EmptyState icon={IconDriver} title={D.queue.offEmpty} />
      ) : rows.length === 0 ? (
        <EmptyState icon={IconOrder} title={D.queue.empty} />
      ) : (
        <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
          {rows.map((o) => (
            <IncomingCard
              key={o.id}
              o={o}
              busy={busy === o.id}
              onAccept={() => void accept(o)}
              speedKmh={speed}
            />
          ))}
        </div>
      )}
    </div>
  );
}
