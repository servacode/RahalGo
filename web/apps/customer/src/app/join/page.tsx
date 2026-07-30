"use client";

/** صفحة انضمام المتجر عبر رابط/باركود المندوب — عامة، مقيّدة بكود مندوب صالح.
 *  تلتقط طلباً معلّقاً (تصنيف، موقع، كلمة مرور صاحب المتجر) لا متجراً — الإنشاء
 *  الفعلي يتم عند موافقة الإدارة. */

import { Suspense, useEffect, useState } from "react";
import dynamic from "next/dynamic";
import { useSearchParams } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Button,
  Input,
  IconStore,
  IconPhone,
  IconUser,
  IconLocation,
  IconLock,
  IconSuccess,
  IconBlock,
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
  const [repName, setRepName] = useState<string | null>(null);
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

  // التسجيل حصراً عبر مندوب — نتحقق من صحة الكود قبل عرض النموذج.
  useEffect(() => {
    if (!ref) {
      setChecking(false);
      return;
    }
    api<{ name: string }>(`/api/v1/public/rep/${encodeURIComponent(ref)}`)
      .then((r) => setRepName(r.name))
      .catch(() => setRepName(null))
      .finally(() => setChecking(false));
  }, [ref]);

  // تصنيفات المتاجر لاختيار نوع المتجر.
  useEffect(() => {
    api<{ categories: Category[] }>("/api/v1/public/home")
      .then((d) => setCategories(d.categories ?? []))
      .catch(() => undefined);
  }, []);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (!categoryId) {
      setError(J.pickCategory);
      return;
    }
    if (password.length < 8) {
      setError(J.weakPassword);
      return;
    }
    if (password !== confirm) {
      setError(J.passwordMismatch);
      return;
    }
    setBusy(true);
    try {
      await api("/api/v1/public/join", {
        method: "POST",
        body: JSON.stringify({
          ref,
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

  // كود غائب أو غير صالح → لا نموذج؛ التسجيل يكون عبر مندوب المنصة فقط.
  if (!repName) {
    return (
      <div className="mx-auto max-w-md py-16 text-center">
        <span className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-card bg-danger/10 text-danger">
          <IconBlock size={30} />
        </span>
        <h1 className="text-xl font-bold">{J.invalidTitle}</h1>
        <p className="mx-auto mt-2 max-w-sm text-sm text-ink-muted">{J.invalidHint}</p>
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
        <p className="mt-2 inline-block rounded-badge bg-accent/15 px-3 py-1 text-sm text-accent-dark">
          {J.byRepName.replace("{name}", repName)}
        </p>
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

        {/* اختيار تصنيف المتجر */}
        <div>
          <label className="mb-1.5 block text-sm font-medium">{J.category}</label>
          <div className="grid grid-cols-3 gap-2 sm:grid-cols-5">
            {categories.map((c) => {
              const active = categoryId === c.id;
              return (
                <button
                  key={c.id}
                  type="button"
                  onClick={() => setCategoryId(c.id)}
                  className={`flex flex-col items-center gap-1 rounded-control border p-2 text-xs transition-colors ${
                    active
                      ? "border-primary bg-primary-light font-medium text-primary-dark"
                      : "border-line text-ink-muted hover:bg-page"
                  }`}
                >
                  <span className="text-xl">{c.icon}</span>
                  {c.name}
                </button>
              );
            })}
          </div>
        </div>

        <Input
          id="owner"
          label={J.ownerName}
          icon={<IconUser />}
          value={ownerName}
          onChange={(e) => setOwnerName(e.target.value)}
        />
        <div>
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
          <p className="mt-1 text-xs text-ink-muted">{J.phoneHint}</p>
        </div>

        <Input
          id="area"
          label={J.area}
          icon={<IconLocation />}
          value={area}
          onChange={(e) => setArea(e.target.value)}
        />

        {/* تحديد الموقع على الخريطة */}
        <div>
          <label className="mb-1.5 block text-sm font-medium">{J.pickLocation}</label>
          <div className="overflow-hidden rounded-control border border-line">
            <PickMap lat={lat} lng={lng} onPick={(la, ln) => { setLat(la); setLng(ln); }} />
          </div>
          <p className="mt-1 text-xs text-ink-muted">
            {lat != null ? J.locationSet : J.locationHint}
          </p>
        </div>

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
