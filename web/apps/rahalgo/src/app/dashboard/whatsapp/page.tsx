"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/dashboard/whatsapp — «طلبات الواتساب»** (قرارُ المالك ٢٠٢٦-١١-٠٤ ثمّ ٢٠٢٦-١١-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «عندي طلبات خارجية على الواتس اب، بدي خيار باللوحة اسمه طلبات على الواتس اب… وينزل طلب
 * نظامي ع التطبيق، كأنه طلب جاي من تطبيق». **للأدمن والموظفين** (`orders.whatsapp`)، **وينزل
 * للسائقين مباشرةً بلا موافقة.**
 *
 * **النموذجُ ثمّ آخرُ الطلبات** — والطلبُ بعدها طلبٌ نظاميٌّ في لوحة الطلبات وسجلّها وتقاريرها
 * ومراقبة السائقين.
 */

import { useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { AdminPage, Alert, IconWhatsApp } from "@rahalgo/ui";
import { fill } from "@/lib/customer/texts";
import { OpenOrderLink, WhatsAppOrderForm } from "@/components/admin/whatsapp/WhatsAppOrderForm";
import { RecentWhatsAppOrders } from "@/components/admin/whatsapp/RecentWhatsAppOrders";

const m = getMessages(defaultLocale);
const W = m.admin.whatsappOrders;

export default function WhatsAppOrdersPage() {
  const [sent, setSent] = useState<{ id: string; number: number } | null>(null);
  const [refresh, setRefresh] = useState(0);

  return (
    <AdminPage
      title={W.title}
      icon={IconWhatsApp}
      notices={
        sent && (
          <Alert tone="success">
            <span className="flex flex-wrap items-center gap-2">
              {fill(W.sent, { n: sent.number })}
              <OpenOrderLink id={sent.id} />
            </span>
          </Alert>
        )
      }
    >
      <div className="grid gap-4 xl:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
        <WhatsAppOrderForm
          onSent={(o) => {
            setSent(o);
            setRefresh((n) => n + 1);
            if (typeof window !== "undefined") window.scrollTo({ top: 0, behavior: "smooth" });
          }}
        />
        <div>
          <RecentWhatsAppOrders refresh={refresh} />
        </div>
      </div>
    </AdminPage>
  );
}
