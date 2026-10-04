"use client";

/**
 * **ما خسرناه وما نطالب به — في بابٍ واحد.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٨: «النزاعات تكون مع الخسائر لأنّها هي بسبب
 *  الخسائر».)
 *
 * # وهو صحيح
 *
 * **الواقعةُ واحدةٌ ووجهاها اثنان**: طلبٌ يفشل، فتعوّض المنصّةُ السائقَ
 * **لحظتَها** — وذاك قيدُ خسارة — **ثمّ تفتح نزاعاً مع المتجر لتسترجع**.
 * فمن قرأ الخسارةَ وحدَها رأى مالاً خرج ولم يعرف أيُطالَب به أحد، **ومن قرأ
 * النزاعَ وحدَه رأى مطالبةً لا يعرف من أين جاءت.**
 *
 * # والصلاحيّتان مختلفتان — فيُحفظ الفرق
 *
 * **الخسائرُ لـ`admin` و`finance`، والنزاعاتُ لهما ولـ`ops` معهما.** وجمعُهما
 * في بابٍ واحدٍ لا يوسّع صلاحيّةَ أحد: **الصفحةُ تُفتح لمن كان يفتح النزاعات،
 * وتبويبُ الخسائر لا يُرسَم إلّا لمن كان يراها.**
 *
 * **فموظّفُ العمليات يرى تبويباً واحداً** — وهو ما يراه اليومَ بعينه، لا
 * أقلَّ ولا أكثر. **ولا يُخفى عنه ما كان يراه، ولا يُكشف له ما لم يكن.**
 */

import { useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { TabCards } from "@rahalgo/ui";
import { useAuth } from "@/lib/auth";
import { useCanCall } from "@/lib/policy";
import { LossesView } from "@/components/admin/money/losses";
import { DisputesView } from "@/components/admin/money/disputes";

const m = getMessages(defaultLocale);

export default function MoneyLostPage() {
  useAuth();
  const canCall = useCanCall();
  /* **ومن لا يملك التبويبَ لا يُرسَم له** — لا يُعطَّل ولا يُخفى بعد ظهور.
     **والسؤالُ سؤالُ المحرّك** (`useCanCall`): أيُّ قدرةٍ تفتح هذا الباب؟
     (قرارُ المالك ٢٠٢٦-١٠-٠٤: الماليّةُ ترى النزاعات وتحسم، والدعمُ يرى ويفتح.) */
  const canSeeLosses = canCall("GET", "/reports/losses");
  const canSeeDisputes = canCall("GET", "/disputes");
  const tabs = [
    ...(canSeeLosses ? [{ key: "losses", label: m.admin.nav.losses }] : []),
    ...(canSeeDisputes ? [{ key: "disputes", label: m.admin.nav.claims }] : []),
  ];
  const [tab, setTab] = useState<string>("losses");
  /* **والصلاحيّةُ تصل بعد أوّل رسم** (`useAuth` تُحمّل): فتبويبٌ اختِيرَ قبل
     وصولها قد لا يوجد بعده — **فيُصحَّح إلى الموجود لا يُترك معلّقاً.** */
  const active = tabs.some((t) => t.key === tab) ? tab : (tabs[0]?.key ?? "");

  return (
    <div className="space-y-5">
      {/* **وتبويبٌ واحدٌ ليس تبويباً** — من لا خيارَ له لا يُعرض عليه صفٌّ
          فيه زرٌّ واحدٌ مضغوطٌ أبداً. */}
      {tabs.length > 1 && <TabCards items={tabs} active={active} onChange={setTab} />}
      {active === "losses" && <LossesView />}
      {active === "disputes" && <DisputesView />}
    </div>
  );
}
