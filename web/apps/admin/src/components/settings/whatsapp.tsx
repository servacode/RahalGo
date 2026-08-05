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
        <Button variant="secondary" onClick={load}>
          {m.admin.whatsappPage.refresh}
        </Button>
      </div>

      {status?.provider === "dev" ? (
        <div className="rounded-card border border-line bg-surface p-6 text-ink-muted">
          {m.admin.whatsappPage.devMode}
        </div>
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          <div className="space-y-4 rounded-card border border-line bg-surface p-6">
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
            <div className="flex flex-col items-center rounded-card border border-line bg-surface p-6">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={qrDataURL} alt="WhatsApp QR" className="rounded-control" />
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
