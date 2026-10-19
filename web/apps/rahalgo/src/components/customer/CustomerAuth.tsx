"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **دخولُ الزبون وتسجيلُه واستعادتُه — كما في تطبيق أندرويد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-١٧: زبونُ آيفون يدخل ويطلب من المتصفّح.)
 *
 * **ثلاثُ شاشاتٍ بمساراتها** (`/login` · `/signup` · `/forgot`) — **ولا
 * تُمسّ بطاقةُ الدخول المشتركة** (`LoginCard`): هي بابُ الإدارة على
 * `/adminrahalgo`، ولها رمزُ الأدمن وتبويبُ الرمز.
 *
 * - **التسجيلُ يتبع المنصّة** (`signup_verify`): **رمزٌ أوّلاً إن طُلب**،
 *   وإلّا فالاسمُ وكلمةُ المرور مباشرة — **والتوثيقُ عند أوّل طلب.**
 *   **ويدخل حين ينتهي** — المحرّكُ يفتح الجلسةَ مع الإنشاء.
 * - **الاستعادةُ مقلوبة** (قرارُ المالك ٢٠٢٦-٠٨-٢٥): **يفتح واتساب ويرسل
 *   هو** رسالةً جاهزة، **فيصله الرمزُ رداً** — وواتساب لا يقيّد الردّ.
 * - **والعميلُ «ويب»** — المتصفّحُ لا يرسل ترويسةَ نوعٍ، **والمحرّكُ يقرأ
 *   الغيابَ ويب** (`identity.ClientFrom`) كما يفعل مع اللوحة.
 */

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import {
  Alert,
  BrandMark,
  Button,
  ButtonLink,
  Checkbox,
  Chips,
  Input,
  LoadingState,
  OtpInput,
  IconLock,
  IconPhone,
  IconUser,
  IconPrev,
} from "@rahalgo/ui";
import { api, authApi, tokenStore, type AuthResult } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { routeByRole, safeNext } from "@rahalgo/auth";
import { useCustomer } from "@/lib/customer/context";
import { fill } from "@/lib/customer/texts";

const m = getMessages(defaultLocale);
const T = m.shop.auth;

export type AuthMode = "login" | "signup" | "forgot";

function Frame({ title, hint, children }: { title: string; hint?: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-1 items-center justify-center p-4">
      <div className="w-full max-w-md surface p-6 sm:p-8">
        <div className="mb-6 text-center">
          <BrandMark size={96} disc className="mx-auto mb-4" />
          <h1 className="heading-section">{title}</h1>
          {hint && <p className="mt-1.5 text-sm text-ink-muted">{hint}</p>}
        </div>
        {children}
      </div>
    </div>
  );
}

function PhoneField({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  return (
    <Input
      id="phone"
      label={T.phone}
      icon={<IconPhone />}
      dir="ltr"
      inputMode="tel"
      autoComplete="tel"
      required
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="text-end"
      placeholder={T.phoneHint}
    />
  );
}

function PasswordField({
  id,
  label,
  value,
  onChange,
  auto,
}: {
  id: string;
  label: string;
  value: string;
  onChange: (v: string) => void;
  auto: string;
}) {
  return (
    <Input
      id={id}
      label={label}
      icon={<IconLock />}
      type="password"
      autoComplete={auto}
      required
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  );
}

/**
 * **بعد الدخول تُحفظ الجلسةُ ويُعلَن صاحبُها** — **والتوجيهُ في موضعٍ واحد**
 * (`CustomerAuth` أدناه): إلى حيث كان (`next`)، وإلّا بابُ دوره — والزبونُ
 * إلى السوق. **وتوجيهان يتسابقان يجلبان القدراتِ مرّتين.**
 */
function useEnter() {
  const { enter } = useAuth();
  return async (r: AuthResult & { pin_required?: boolean }) => {
    if (r.pin_required) throw new Error("pin");
    tokenStore.set(r.tokens, true);
    enter(r.user);
    // **وصاحبُ المتجر إلى لوحته فوراً** (قرارُ المالك ٢٠٢٦-١٠-١٩) — فالفورمُ المضمَّنُ في
    // «حسابي» يختفي لحظةَ الدخول وتُرسم صفحةُ الزبون قبل أن يوجِّه أحد.
    if (r.user.roles?.includes("merchant")) window.location.href = "/store";
  };
}

/**
 * **الدخولُ برمزٍ على واتساب** — إن فتحته اللوحة (`auth.otp_login`)، كما في
 * التطبيق. **يفتح الجلسةَ مباشرةً** (`/auth/otp/verify`).
 */
function OtpLogin() {
  const doEnter = useEnter();
  const [phone, setPhone] = useState("");
  const [sent, setSent] = useState(false);
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const run = (fn: () => Promise<void>) => async (e?: React.FormEvent) => {
    e?.preventDefault();
    setBusy(true);
    setError("");
    try {
      await fn();
    } catch (err) {
      setError(err instanceof Error && err.message === "pin" ? T.staffNote : errorText(err));
    } finally {
      setBusy(false);
    }
  };
  const verify = (c: string) => run(async () => doEnter((await authApi.verifyOtp(phone.trim(), c)) as AuthResult))();

  if (!sent) {
    return (
      <form
        className="space-y-4"
        onSubmit={run(async () => {
          await authApi.requestOtp(phone.trim());
          setSent(true);
          setCode("");
        })}
      >
        <p className="text-sm text-ink-muted">{T.codeHint}</p>
        <PhoneField value={phone} onChange={setPhone} />
        {error && <Alert>{error}</Alert>}
        <Button type="submit" disabled={busy} className="w-full py-3 text-base">
          {busy ? T.busy : T.send}
        </Button>
      </form>
    );
  }
  return (
    <form
      className="space-y-4"
      onSubmit={(e) => {
        e.preventDefault();
        void verify(code);
      }}
    >
      <p className="text-center text-sm text-ink-muted">
        {T.codeSentTo}{" "}
        <span dir="ltr" className="font-bold text-ink">
          {phone.trim()}
        </span>
      </p>
      <OtpInput value={code} onChange={setCode} autoFocus boxLabel={m.auth.otpBoxLabel} onComplete={(v) => void verify(v)} />
      {error && <Alert>{error}</Alert>}
      <Button type="submit" disabled={busy || !code.trim()} className="w-full py-3">
        {busy ? T.busy : T.verify}
      </Button>
      <button type="button" onClick={() => setSent(false)} className="w-full text-sm text-accent-text">
        {T.changeNumber}
      </button>
    </form>
  );
}

function Login({ next }: { next?: string }) {
  const doEnter = useEnter();
  const params = useSearchParams();
  const { platform } = useCustomer();
  const [how, setHow] = useState<"password" | "otp">("password");
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const q = next ? `next=${encodeURIComponent(next)}` : params.toString();
  const otp = platform?.otp_login === true;
  return (
    <Frame title={T.loginTitle}>
      {otp && (
        <div className="mb-4 flex justify-center">
          <Chips<"password" | "otp">
            items={[
              { id: "password", label: T.byPassword },
              { id: "otp", label: T.byCode },
            ]}
            value={how}
            onChange={setHow}
          />
        </div>
      )}
      {otp && how === "otp" ? (
        <OtpLogin />
      ) : (
      <form
        className="space-y-4"
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          setError("");
          try {
            await doEnter((await authApi.loginPassword(phone.trim(), password)) as AuthResult);
          } catch (err) {
            setError(err instanceof Error && err.message === "pin" ? T.staffNote : errorText(err));
            setBusy(false);
          }
        }}
      >
        <PhoneField value={phone} onChange={setPhone} />
        <PasswordField id="password" label={T.password} value={password} onChange={setPassword} auto="current-password" />
        <div className="text-end">
          <Link href={`/forgot${q ? `?${q}` : ""}`} className="text-sm text-accent-text hover:underline">
            {T.forgot}
          </Link>
        </div>
        {error && <Alert>{error}</Alert>}
        <Button type="submit" disabled={busy} className="w-full py-3 text-base">
          {busy ? T.busy : T.submit}
        </Button>
      </form>
      )}
      <div className="mt-6 flex flex-col items-center gap-3 border-t border-line-soft pt-4 text-sm">
        <p className="text-ink-muted">
          {T.noAccount}{" "}
          <Link href={`/signup${q ? `?${q}` : ""}`} className="font-bold text-accent-text hover:underline">
            {T.signupOpen}
          </Link>
        </p>
        <Link href="/shop" className="text-ink-muted hover:text-ink">
          {T.guest}
        </Link>
      </div>
    </Frame>
  );
}

function Signup() {
  const doEnter = useEnter();
  const { platform } = useCustomer();
  const params = useSearchParams();
  const ref = params.get("ref") ?? "";
  const [step, setStep] = useState<"phone" | "code" | "details" | "taken">("phone");
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [password2, setPassword2] = useState("");
  const [agreed, setAgreed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const min = platform?.password_min_length ?? 8;

  if (!platform) return <LoadingState variant="inline" />;
  if (!platform.launch.customer_signup) {
    return (
      <Frame title={T.signupTitle}>
        <Alert tone="warning">{platform.launch.notice || T.signupClosed}</Alert>
        <BackToLogin />
      </Frame>
    );
  }

  const run = (fn: () => Promise<void>) => async (e?: React.FormEvent) => {
    e?.preventDefault();
    setBusy(true);
    setError("");
    try {
      await fn();
    } catch (err) {
      const c = (err as { body?: { code?: string } })?.body?.code;
      if (c === "phone_taken") setStep("taken");
      else setError(errorText(err));
    } finally {
      setBusy(false);
    }
  };

  const verify = (c: string) =>
    run(async () => {
      await authApi.verifySignup(phone.trim(), c);
      setCode(c);
      setStep("details");
    })();

  if (step === "taken") {
    return (
      <Frame title={T.takenTitle} hint={T.takenHint}>
        <div className="flex flex-col gap-2">
          <ButtonLink href="/login" className="w-full py-3">
            {T.loginLink}
          </ButtonLink>
          <ButtonLink href="/forgot" variant="secondary" className="w-full py-3">
            {T.forgot}
          </ButtonLink>
        </div>
      </Frame>
    );
  }

  return (
    <Frame
      title={T.signupTitle}
      hint={step === "phone" ? T.signupHintPhone : step === "code" ? T.signupHintCode : T.signupHintDetails}
    >
      {step === "phone" && (
        <form
          className="space-y-4"
          onSubmit={run(async () => {
            if (!platform.signup_verify) {
              setCode("");
              setStep("details");
              return;
            }
            await authApi.requestSignup(phone.trim());
            setStep("code");
          })}
        >
          <PhoneField value={phone} onChange={setPhone} />
          {error && <Alert>{error}</Alert>}
          <Button type="submit" disabled={busy} className="w-full py-3">
            {busy ? T.busy : platform.signup_verify ? T.send : T.next}
          </Button>
        </form>
      )}
      {step === "code" && (
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault();
            void verify(code);
          }}
        >
          <p className="text-center text-sm text-ink-muted">
            {T.codeSentTo}{" "}
            <span dir="ltr" className="font-bold text-ink">
              {phone}
            </span>
          </p>
          <OtpInput value={code} onChange={setCode} autoFocus boxLabel={m.auth.otpBoxLabel} onComplete={(v) => void verify(v)} />
          {error && <Alert>{error}</Alert>}
          <Button type="submit" disabled={busy} className="w-full py-3">
            {T.verify}
          </Button>
          <button type="button" onClick={() => setStep("phone")} className="w-full text-sm text-accent-text">
            {T.changeNumber}
          </button>
        </form>
      )}
      {step === "details" && (
        <form
          className="space-y-4"
          onSubmit={run(async () => {
            if (password !== password2) throw Object.assign(new Error("mismatch"), { body: { message_key: "errors.password_mismatch" } });
            if (!agreed) throw Object.assign(new Error("agree"), { body: { message_key: "auth.mustAgree" } });
            const r = await authApi.confirmSignup(phone.trim(), code, name.trim(), password, ref);
            await doEnter(r);
          })}
        >
          <Input
            id="full-name"
            label={T.name}
            icon={<IconUser />}
            autoComplete="name"
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <PasswordField id="new-password" label={T.password} value={password} onChange={setPassword} auto="new-password" />
          <PasswordField id="new-password2" label={T.password2} value={password2} onChange={setPassword2} auto="new-password" />
          <p className="text-xs text-ink-muted">{fill(T.passwordHint, { n: min })}</p>
          <Checkbox
            id="agree"
            checked={agreed}
            onChange={(e) => setAgreed(e.target.checked)}
            label={
              <span>
                {m.auth.agreePrefix}{" "}
                <a href="/terms" target="_blank" rel="noreferrer" className="text-accent-text underline">
                  {m.auth.agreeTerms}
                </a>{" "}
                {m.auth.agreeAnd}{" "}
                <a href="/privacy" target="_blank" rel="noreferrer" className="text-accent-text underline">
                  {m.auth.agreePrivacy}
                </a>
              </span>
            }
          />
          {error && <Alert>{error}</Alert>}
          <Button type="submit" disabled={busy} className="w-full py-3">
            {busy ? T.busy : T.create}
          </Button>
        </form>
      )}
      <div className="mt-6 border-t border-line-soft pt-4 text-center text-sm text-ink-muted">
        {T.haveAccount}{" "}
        <Link href="/login" className="font-bold text-accent-text hover:underline">
          {T.loginLink}
        </Link>
      </div>
    </Frame>
  );
}

function BackToLogin() {
  return (
    <div className="mt-6 border-t border-line-soft pt-4 text-center">
      <Link href="/login" className="inline-flex items-center gap-1 text-sm text-accent-text hover:underline">
        <IconPrev size={15} />
        {T.back}
      </Link>
    </div>
  );
}

function Forgot() {
  const doEnter = useEnter();
  const [step, setStep] = useState<"phone" | "code" | "password">("phone");
  const [phone, setPhone] = useState("");
  const [waUrl, setWaUrl] = useState("");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [password2, setPassword2] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const run = (fn: () => Promise<void>) => async (e?: React.FormEvent) => {
    e?.preventDefault();
    setBusy(true);
    setError("");
    try {
      await fn();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  };

  const check = (c: string) =>
    run(async () => {
      await authApi.verifyReset(phone.trim(), c);
      setCode(c);
      setStep("password");
    })();

  return (
    <Frame
      title={T.resetTitle}
      hint={step === "phone" ? T.resetHintPhone : step === "code" ? T.resetHintCode : undefined}
    >
      {step === "phone" && (
        <form
          className="space-y-4"
          onSubmit={run(async () => {
            const t = await api<{ wa_url: string }>("/api/v1/auth/wa/ticket", {
              method: "POST",
              body: JSON.stringify({ phone: phone.trim(), purpose: "reset" }),
            });
            setWaUrl(t.wa_url);
            setStep("code");
            /* **ويُفتح واتساب في الحال إن سمح المتصفّح** — وسفاري يمنع
               نافذةً بعد انتظار، **فيبقى الرابطُ ظاهراً يُضغط بيده.** */
            try {
              window.open(t.wa_url, "_blank", "noopener");
            } catch {
              /* الرابطُ تحت يده */
            }
          })}
        >
          <PhoneField value={phone} onChange={setPhone} />
          {error && <Alert>{error}</Alert>}
          <Button type="submit" disabled={busy} className="w-full py-3">
            {busy ? T.busy : T.resetSend}
          </Button>
        </form>
      )}
      {step === "code" && (
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault();
            void check(code);
          }}
        >
          {waUrl && (
            <p className="text-center text-sm text-ink-muted">
              {T.resetWaHint}{" "}
              <a href={waUrl} target="_blank" rel="noreferrer" className="font-bold text-accent-text underline">
                {T.resetWaLink}
              </a>
            </p>
          )}
          <OtpInput value={code} onChange={setCode} autoFocus boxLabel={m.auth.otpBoxLabel} onComplete={(v) => void check(v)} />
          {error && <Alert>{error}</Alert>}
          <Button type="submit" disabled={busy} className="w-full py-3">
            {T.verify}
          </Button>
          <button type="button" onClick={() => setStep("phone")} className="w-full text-sm text-accent-text">
            {T.changeNumber}
          </button>
        </form>
      )}
      {step === "password" && (
        <form
          className="space-y-4"
          onSubmit={run(async () => {
            if (password !== password2) throw Object.assign(new Error("mismatch"), { body: { message_key: "errors.password_mismatch" } });
            const r = await authApi.confirmReset(phone.trim(), code, password);
            await doEnter(r);
          })}
        >
          <PasswordField id="reset-password" label={T.resetNew} value={password} onChange={setPassword} auto="new-password" />
          <PasswordField id="reset-password2" label={T.password2} value={password2} onChange={setPassword2} auto="new-password" />
          {error && <Alert>{error}</Alert>}
          <Button type="submit" disabled={busy} className="w-full py-3">
            {busy ? T.busy : T.resetSave}
          </Button>
        </form>
      )}
      <BackToLogin />
    </Frame>
  );
}

/** **من دخل يُساق إلى حيث كان** — ولا يرى شاشةَ دخولٍ وهو داخل. */
export default function CustomerAuth({ mode, next }: { mode: AuthMode; next?: string }) {
  const { user, loading } = useAuth();
  const params = useSearchParams();
  const sent = useRef(false);
  useEffect(() => {
    if (loading || !user || sent.current) return;
    sent.current = true;
    void routeByRole(user, safeNext(next ?? params.get("next")) ?? undefined);
  }, [user, loading, params, next]);
  if (loading || user) return <LoadingState variant="inline" />;
  if (mode === "signup") return <Signup />;
  if (mode === "forgot") return <Forgot />;
  return <Login next={next} />;
}
