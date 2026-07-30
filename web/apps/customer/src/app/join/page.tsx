"use client";

/** صفحة انضمام المتجر عبر رابط/باركود المندوب — عامة، تلتقط الطلب منسوباً للكود. */

import { Suspense, useState } from "react";
import { useSearchParams } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, Input, IconStore, IconPhone, IconUser, IconLocation, IconSuccess } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const J = m.site.join;

export default function JoinPage() {
  return (
    <Suspense>
      <JoinForm />
    </Suspense>
  );
}

function JoinForm() {
  const ref = useSearchParams().get("ref") ?? "";
  const [storeName, setStoreName] = useState("");
  const [ownerName, setOwnerName] = useState("");
  const [phone, setPhone] = useState("");
  const [area, setArea] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/public/join", {
        method: "POST",
        body: JSON.stringify({
          ref,
          store_name: storeName,
          owner_name: ownerName,
          phone,
          area,
          note,
        }),
      });
      setDone(true);
    } catch (err) {
      setError(err instanceof ApiError ? m.errors.internal : m.errors.internal);
      setBusy(false);
    }
  }

  if (done) {
    return (
      <div className="mx-auto max-w-md py-16 text-center">
        <IconSuccess size={56} className="mx-auto mb-4 text-success" />
        <p className="text-lg font-bold">{J.done}</p>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-md">
      <div className="mb-5 text-center">
        <span className="mx-auto mb-3 flex h-14 w-14 items-center justify-center rounded-card bg-primary text-2xl font-bold text-white">
          ر
        </span>
        <h1 className="text-2xl font-bold">{J.title}</h1>
        <p className="mt-1 text-sm text-ink-muted">{J.subtitle}</p>
        {ref && (
          <p className="mt-2 inline-block rounded-badge bg-accent/15 px-3 py-1 font-mono text-sm text-accent-dark" dir="ltr">
            {J.byRep.replace("{code}", ref)}
          </p>
        )}
      </div>

      <form onSubmit={submit} className="space-y-4 rounded-card border border-line bg-surface p-5">
        <Input
          id="store"
          label={J.storeName}
          icon={<IconStore />}
          required
          value={storeName}
          onChange={(e) => setStoreName(e.target.value)}
        />
        <Input
          id="owner"
          label={J.ownerName}
          icon={<IconUser />}
          value={ownerName}
          onChange={(e) => setOwnerName(e.target.value)}
        />
        <Input
          id="phone"
          label={J.phone}
          icon={<IconPhone />}
          dir="ltr"
          inputMode="tel"
          required
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          className="text-end"
          placeholder="09xxxxxxxx"
        />
        <Input
          id="area"
          label={J.area}
          icon={<IconLocation />}
          value={area}
          onChange={(e) => setArea(e.target.value)}
        />
        <Input id="note" label={J.note} value={note} onChange={(e) => setNote(e.target.value)} />
        {error && (
          <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
        )}
        <Button type="submit" disabled={busy} className="w-full py-2.5">
          {busy ? J.sending : J.submit}
        </Button>
      </form>
    </div>
  );
}
