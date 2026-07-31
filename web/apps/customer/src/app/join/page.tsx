"use client";

/** صفحة انضمام المتجر — التسجيل عبر كود دعوة (مندوب أو المنصة). طلب معلّق لا متجر؛
 *  الإنشاء الفعلي عند موافقة الإدارة. نموذج عريض ثنائي العمود. */

import { Suspense, useEffect, useState } from "react";
import dynamic from "next/dynamic";
import { useSearchParams } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Button,
  Input,
  Select,
  IconStore,
  IconPhone,
  IconUser,
  IconLocation,
  IconLock,
  IconSuccess,
  IconPromos,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const PickMap = dynamic(() => import("@/components/map/PickMap"), { ssr: false });

const m = getMessages(defaultLocale);
const J = m.site.join;

interface Category {
  id: string;
  name: string;
  icon: string;
}

export default function JoinPage() {
  return (
    <Suspense>
      <JoinForm />
    </Suspense>
  );
}

function JoinForm() {
  const ref = useSearchParams().get("ref") ?? "";
  const [inviteCode, setInviteCode] = useState<string>("");
  const [checking, setChecking] = useState(true);
  const [categories, setCategories] = useState<Category[]>([]);

  const [storeName, setStoreName] = useState("");
  const [ownerName, setOwnerName] = useState("");
  const [phone, setPhone] = useState("");
  const [area, setArea] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [lat, setLat] = useState<number | null>(null);
  const [lng, setLng] = useState<number | null>(null);

  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState("");

  // كود الدعوة المعروض: كود المندوب إن صحّ، وإلا كود المنصة (تسجيل مباشر).
  useEffect(() => {
    const q = ref ? `?ref=${encodeURIComponent(ref)}` : "";
    api<{ code: string }>(`/api/v1/public/invite${q}`)
      .then((r) => setInviteCode(r.code))
      .catch(() => setInviteCode(""))
      .finally(() => setChecking(false));
  }, [ref]);

  useEffect(() => {
    api<{ categories: Category[] }>("/api/v1/public/home")
      .then((d) => setCategories(d.categories ?? []))
      .catch(() => undefined);
  }, []);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (!categoryId) return setError(J.pickCategory);
    if (password.length < 8) return setError(J.weakPassword);
    if (password !== confirm) return setError(J.passwordMismatch);
    setBusy(true);
    try {
      await api("/api/v1/public/join", {
        method: "POST",
        body: JSON.stringify({
          ref: inviteCode,
          store_name: storeName,
          owner_name: ownerName,
          phone,
          area,
          category_id: categoryId,
          password,
          lat,
          lng,
        }),
      });
      setDone(true);
    } catch (err) {
      if (err instanceof ApiError) {
        const key = err.body.message_key.split(".").pop() ?? "";
        setError((m.errors as Record<string, string>)[key] ?? m.errors.internal);
      } else {
        setError(m.errors.internal);
      }
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

  if (checking) {
    return <div className="py-20 text-center text-ink-muted">{m.common.loading}</div>;
  }

  return (
    <div className="mx-auto max-w-2xl">
      <div className="mb-5 text-center">
        <span className="mx-auto mb-3 flex h-14 w-14 items-center justify-center rounded-card bg-primary text-2xl font-bold text-white">
          {m.terms.brandInitial}
        </span>
        <h1 className="text-2xl font-bold">{J.title}</h1>
        <p className="mt-1 text-sm text-ink-muted">{J.subtitle}</p>
      </div>

      <form onSubmit={submit} className="rounded-card border border-line bg-surface p-6">
        {/* كود الدعوة — للقراءة فقط */}
        <div className="mb-4">
          <label className="mb-1 flex items-center gap-1.5 text-sm font-medium">
            <span className="text-ink-muted [&>svg]:h-4 [&>svg]:w-4">
              <IconPromos />
            </span>
            {J.inviteCode}
          </label>
          <div
            dir="ltr"
            className="flex items-center justify-end rounded-control border border-dashed border-accent bg-accent/5 px-3 py-2 font-mono text-sm font-bold text-accent-dark"
          >
            {inviteCode || "—"}
          </div>
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Input
            id="store"
            label={J.storeName}
            icon={<IconStore />}
            required
            value={storeName}
            onChange={(e) => setStoreName(e.target.value)}
          />
          <Select
            id="category"
            label={J.category}
            value={categoryId}
            onChange={(e) => setCategoryId(e.target.value)}
          >
            <option value="">{J.pickCategory}</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </Select>

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
            id="password"
            label={J.password}
            icon={<IconLock />}
            type="password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <Input
            id="confirm"
            label={J.confirmPassword}
            icon={<IconLock />}
            type="password"
            required
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
          />

          <Input
            id="area"
            label={J.area}
            icon={<IconLocation />}
            value={area}
            onChange={(e) => setArea(e.target.value)}
            placeholder={J.areaPlaceholder}
            className="sm:col-span-2"
          />

          {/* الخريطة تمتد على العرض الكامل */}
          <div className="sm:col-span-2">
            <label className="mb-1.5 block text-sm font-medium">{J.pickLocation}</label>
            <div className="overflow-hidden rounded-control border border-line">
              <PickMap lat={lat} lng={lng} onPick={(la, ln) => { setLat(la); setLng(ln); }} />
            </div>
            <p className="mt-1 text-xs text-ink-muted">
              {lat != null ? J.locationSet : J.locationHint}
            </p>
          </div>
        </div>

        {error && (
          <p className="mt-4 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
        )}
        <Button type="submit" disabled={busy} className="mt-5 w-full py-2.5">
          {busy ? J.sending : J.submit}
        </Button>
      </form>
    </div>
  );
}
