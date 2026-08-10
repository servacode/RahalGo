"use client";

import { useCallback, useEffect, useState } from "react";
import QRCode from "qrcode";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Alert,
  PageHeader, Button, Badge, IconWhatsApp,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);

interface WAStatus {
  provider: string;
  connected?: boolean;
  logged_in?: boolean;
  paired_as?: string;
  qr?: string;
  last_error?: string;
}

export default function WhatsAppPanel() {
  const [status, setStatus] = useState<WAStatus | null>(null);
  const [qrDataURL, setQrDataURL] = useState("");
  const [busy, setBusy] = useState(false);
  /** **وما وقع يُقال** — وزرٌّ يُضغط بلا أثرٍ ظاهرٍ يُضغط مرّتين. */
  const [notice, setNotice] = useState("");

  const load = useCallback(async () => {
    try {
      const st = await api<WAStatus>("/api/v1/admin/whatsapp");
      setStatus(st);
      if (st.qr) {
        setQrDataURL(await QRCode.toDataURL(st.qr, { width: 280, margin: 1 }));
      } else {
        setQrDataURL("");
      }
    } catch {
      setStatus(null);
    }
  }, []);

  useEffect(() => {
    void load();
    const t = setInterval(load, 10000); // تحديث تلقائي كل 10 ثوانٍ (رمز QR يتغير)
    return () => clearInterval(t);
  }, [load]);

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <PageHeader icon={IconWhatsApp} title={m.admin.whatsappPage.title} />
        <div className="flex gap-2">
          <Button variant="secondary" onClick={load}>
            {m.admin.whatsappPage.refresh}
          </Button>
          {/* ══════════════════════════════════════════════════════════
              **وزرُّ إعادة الربط — ولا رمزَ بلا فكّ**
              ══════════════════════════════════════════════════════════

              (قرارُ المالك ٢٠٢٦-٠٨-١٠: «إذا تمّ فصلُ الاقتران لا يوجد زرٌّ
               لإعادة ربط الجهاز».)

              **ورمزُ الربط لا يُولَّد إلّا لجهازٍ بلا هويّة** — ومن فُصل من
              هاتفه تبقى هويّتُه مخزَّنة، **فتقول اللوحةُ «غير مقترن» ولا
              سبيلَ إلى الاقتران.**

              **ولا يظهر لمزوّد التطوير** — لا اقترانَ فيه أصلاً. */}
          {status?.provider === "whatsapp" && (
            <Button
              variant="danger"
              disabled={busy}
              onClick={async () => {
                if (!confirm(m.admin.whatsappPage.unpairConfirm)) return;
                setBusy(true);
                try {
                  await api("/api/v1/admin/whatsapp/unpair", { method: "POST" });
                  setNotice(m.admin.whatsappPage.unpairDone);
                  await load();
                } finally {
                  setBusy(false);
                }
              }}
            >
              {m.admin.whatsappPage.unpair}
            </Button>
          )}
        </div>
      </div>

      {notice && <Alert tone="success" className="mb-4">{notice}</Alert>}

      {status?.provider === "dev" ? (
        <div className="surface p-6 text-ink-muted">
          {m.admin.whatsappPage.devMode}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <div className="space-y-4 surface p-6">
            <div className="flex items-center justify-between">
              <span className="font-medium">{m.admin.whatsappPage.connection}</span>
              <Badge variant={status?.connected ? "success" : "danger"}>
                {status?.connected
                  ? m.admin.whatsappPage.connected
                  : m.admin.whatsappPage.disconnected}
              </Badge>
            </div>
            <div className="flex items-center justify-between">
              <span className="font-medium">{m.admin.whatsappPage.pairing}</span>
              {status?.logged_in ? (
                <Badge variant="success">
                  {m.admin.whatsappPage.paired}: <span dir="ltr">+{status.paired_as}</span>
                </Badge>
              ) : (
                <Badge variant="warning">{m.admin.whatsappPage.notPaired}</Badge>
              )}
            </div>
            {status?.last_error && (
              <Alert>
                {m.admin.whatsappPage.lastError}: <span dir="ltr">{status.last_error}</span>
              </Alert>
            )}
          </div>

          {qrDataURL && !status?.logged_in && (
            <div className="flex flex-col items-center surface p-6">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={qrDataURL} alt="WhatsApp QR" loading="lazy" className="rounded-control" />
              <p className="mt-3 max-w-xs text-center text-sm text-ink-muted">
                {m.admin.whatsappPage.scanHint}
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
