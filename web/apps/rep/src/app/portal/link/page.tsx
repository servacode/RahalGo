"use client";

/** رابط وباركود التسجيل — يرسله المندوب للمتاجر ليسجّلوا عبره تلقائياً. */

import { useEffect, useState } from "react";
import QRCode from "qrcode";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, IconLink, IconQr } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const SITE_URL = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3003";

interface Me {
  invite_code: string | null;
}

export default function LinkPage() {
  const [code, setCode] = useState<string | null>(null);
  const [qr, setQr] = useState<string>("");
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    api<Me>("/api/v1/rep/me")
      .then((me) => setCode(me.invite_code))
      .catch(() => undefined);
  }, []);

  const link = code ? `${SITE_URL}/join?ref=${encodeURIComponent(code)}` : "";

  useEffect(() => {
    if (!link) return;
    QRCode.toDataURL(link, { width: 512, margin: 2, errorCorrectionLevel: "M" })
      .then(setQr)
      .catch(() => undefined);
  }, [link]);

  if (!code) {
    return <p className="py-12 text-center text-ink-muted">{m.common.loading}</p>;
  }

  const shareText = encodeURIComponent(`${m.rep.shareText.replace("{code}", code)}\n${link}`);

  return (
    <div className="mx-auto max-w-lg space-y-6">
      <div className="text-center">
        <h1 className="flex items-center justify-center gap-2 text-lg font-bold">
          <IconQr size={20} className="text-primary" />
          {m.rep.linkTitle}
        </h1>
        <p className="mx-auto mt-1 max-w-md text-sm text-ink-muted">{m.rep.linkHint}</p>
      </div>

      {/* الباركود */}
      <div className="flex flex-col items-center gap-4 rounded-card border border-line bg-surface p-6">
        {qr ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={qr}
            alt={m.rep.linkTitle}
            className="h-56 w-56 rounded-card border border-line bg-white p-2"
          />
        ) : (
          <div className="flex h-56 w-56 items-center justify-center rounded-card bg-page text-ink-muted">
            {m.common.loading}
          </div>
        )}

        {/* الرابط */}
        <div className="flex w-full items-center gap-2 rounded-control border border-line bg-page px-3 py-2">
          <IconLink size={16} className="shrink-0 text-ink-muted" />
          <span dir="ltr" className="min-w-0 flex-1 truncate text-sm text-ink" title={link}>
            {link}
          </span>
        </div>

        <div className="flex w-full flex-wrap justify-center gap-2">
          <Button
            variant="secondary"
            onClick={() => {
              void navigator.clipboard.writeText(link);
              setCopied(true);
              setTimeout(() => setCopied(false), 1500);
            }}
          >
            {copied ? m.rep.linkCopied : m.rep.copyLink}
          </Button>
          <a
            href={`https://wa.me/?text=${shareText}`}
            target="_blank"
            rel="noreferrer"
            className="rounded-control bg-primary px-4 py-2 text-sm font-medium text-white hover:bg-primary-dark"
          >
            {m.rep.share}
          </a>
          {qr && (
            <a
              href={qr}
              download={`rahalgo-${code}.png`}
              className="rounded-control border border-line px-4 py-2 text-sm font-medium text-ink hover:bg-page"
            >
              {m.rep.downloadQr}
            </a>
          )}
        </div>
      </div>
    </div>
  );
}
