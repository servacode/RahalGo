"use client";

import { useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);

interface WhatsAppStatus {
  provider: string;
  connected?: boolean;
  logged_in?: boolean;
  paired_as?: string;
  qr?: string;
  last_error?: string;
}

export default function DashboardPage() {
  const { user } = useAuth();
  const [wa, setWa] = useState<WhatsAppStatus | null>(null);

  useEffect(() => {
    api<WhatsAppStatus>("/api/v1/admin/whatsapp")
      .then(setWa)
      .catch(() => setWa(null));
  }, []);

  return (
    <div>
      <h1 className="text-2xl font-bold">
        {m.admin.dashboard.welcome}
        {user?.full_name ? `، ${user.full_name}` : ""} 👋
      </h1>

      <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <section className="rounded-card border border-line bg-surface p-5">
          <h2 className="mb-3 font-bold">{m.admin.dashboard.whatsappStatus}</h2>
          {wa === null ? (
            <p className="text-sm text-ink-muted">{m.common.loading}</p>
          ) : wa.provider === "dev" ? (
            <p className="text-sm text-ink-muted">{m.admin.dashboard.waDevProvider}</p>
          ) : (
            <div className="space-y-2 text-sm">
              <p className="flex items-center gap-2">
                <span
                  className={`inline-block h-2.5 w-2.5 rounded-badge ${wa.connected ? "bg-success" : "bg-danger"}`}
                />
                {wa.connected ? m.admin.dashboard.waConnected : m.admin.dashboard.waDisconnected}
              </p>
              {wa.logged_in ? (
                <p className="text-ink-muted">
                  {m.admin.dashboard.waPaired}: <span dir="ltr">+{wa.paired_as}</span>
                </p>
              ) : (
                <p className="text-warning">{m.admin.dashboard.waNotPaired}</p>
              )}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
