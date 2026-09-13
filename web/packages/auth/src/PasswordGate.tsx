"use client";

/**
 * بوابة تبديل كلمة المرور المؤقتة — مركزية لكل اللوحات.
 *
 * من يعرف كلمة مرورك ليس أنت: المندوب يسلّم صاحب المتجر كلمة مرور، والإدارة
 * تعيد تعيين كلمة مرور حساب. في الحالتين تُعلَّم الكلمة **مؤقتة** في الخادم،
 * وهذه البوابة تحجب كل شيء حتى يضبط صاحب الحساب كلمة مروره بنفسه.
 *
 * تُلفّ حول محتوى أي بوابة: <PasswordGate>{children}</PasswordGate>
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Alert, Button, Input, PasswordMeter, IconTile, IconLock, IconWarning, IconCheck, usePlatform } from "@rahalgo/ui";
import { useAuth } from "./provider";
import {
  api,
  authApi,
  clearPasswordChangeRequired,
  isPasswordChangeRequired,
  onPasswordChangeRequired,
} from "./client";
import { errText } from "./LoginCard";

const m = getMessages(defaultLocale);
const A = m.auth;
const G = A.mustChange;

export function PasswordGate({ children }: { children: React.ReactNode }) {
  const { user, loading, setUser } = useAuth();
  // **والطولُ من الإعدادات لا من رقمٍ مكتوب** — (قرارُ المالك ٢٠٢٦-٠٨-٠٩:
  // «موحّدةً بكلّ البرنامج»). **ورقمٌ هنا أقصرُ ممّا يقبله المحرّك يُمرّر
  // كلمةً تُرَدّ**، وأطولُ يمنع كلمةً كان يقبلها.
  const { passwordMinLength: minLen } = usePlatform();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  // ══════════════════════════════════════════════════════════════════
  // **وردُّ المحرّك سببٌ ثانٍ لفتح البوّابة** (`WEBA`، ٢٠٢٦-٠٩-١٣)
  // ══════════════════════════════════════════════════════════════════
  //
  // **والعلَمُ في `‎/auth/me` يُقنَّع حين يكون إعدادُ الإلحاح مُطفأً**
  // (قرارُ المالك ٢٠٢٦-٠٨-٠٩، **وهو مُطفأٌ في الإنتاج**) — **فكان
  // صاحبُ الكلمة المؤقّتة يُمنَع بـ٤٠٣ ولا يرى بوّابةً تقول له ما
  // يفعل.**
  //
  // **والمحرّكُ لا يُقنّع منعَه**: `403 password_change_required` —
  // **فتُفتَح به البوّابةُ ولو أخفى الردُّ العلَم.**
  const [blocked, setBlocked] = useState(false);
  useEffect(() => {
    setBlocked(isPasswordChangeRequired());
    return onPasswordChangeRequired(() => setBlocked(isPasswordChangeRequired()));
  }, []);

  if (loading || !(user?.must_change_password || blocked)) return <>{children}</>;

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (next.length < minLen) return setError(m.errors.weak_password);
    if (next !== confirm) return setError(m.errors.password_mismatch);
    setBusy(true);
    try {
      await api("/api/v1/auth/password", {
        method: "POST",
        body: JSON.stringify({ password: next, current_password: current }),
      });
      // نعيد قراءة الحساب: الخادم رفع علامة الإجبار فتُفتح اللوحة تلقائياً
      //
      // **وتُنسى الإشارةُ معها** — **وإلّا بقيت البوّابةُ مفتوحةً بعد
      // أن زال سببُها**، ولا مخرجَ إلّا بإعادة تحميل.
      clearPasswordChangeRequired();
      setBlocked(false);
      setUser(await authApi.me());
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  return (
    <main className="relative flex flex-1 items-center justify-center overflow-hidden p-4">
      {/* **ولا تدرّجَ محلّيّ**: كان هنا تدرّجٌ يرسمه هذا المكوّنُ لنفسه من يومَ
          كانت خلفيّةُ المشروع عارية. **وصار للموقع خلفيّةٌ واحدةٌ في
          `theme.css`** (طلبُ المالك: «خلفيّةٌ واحدةٌ… في كلّ مكان») —
          **وطبقةٌ فوقها تحجبها وتُقرأ شاشةً غريبةً عن أخواتها.** */}
      <div className="w-full max-w-md surface p-7 elev-1">
        <div className="mb-5 flex items-start gap-3">
          <IconTile tone="warning">
            <IconWarning size={22} />
          </IconTile>
          <div>
            <h1 className="heading-card text-ink">{G.title}</h1>
            <p className="mt-1 text-sm leading-relaxed text-ink-muted">{G.hint}</p>
          </div>
        </div>

        <form onSubmit={submit} className="space-y-4">
          <Input
            id="cur-pw"
            label={G.temporary}
            icon={<IconLock />}
            type="password"
            autoComplete="current-password"
            required
            value={current}
            onChange={(e) => setCurrent(e.target.value)}
          />
          <div className="space-y-2">
            <Input
              id="new-pw"
              label={A.newPassword}
              icon={<IconLock />}
              type="password"
              autoComplete="new-password"
              required
              value={next}
              onChange={(e) => setNext(e.target.value)}
            />
            <PasswordMeter
              value={next}
              labels={[A.strength.weak, A.strength.fair, A.strength.strong]}
            />
          </div>
          <Input
            id="conf-pw"
            label={m.shared.account.confirmPassword}
            icon={<IconLock />}
            type="password"
            autoComplete="new-password"
            required
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
          />
          <p className="text-xs text-ink-muted">{A.passwordHint}</p>

          {error && (
            <Alert>{error}</Alert>
          )}

          <Button type="submit" disabled={busy} className="flex w-full items-center justify-center gap-2 py-3 text-base">
            <IconCheck size={17} strokeWidth={3} />
            {busy ? m.common.loading : G.submit}
          </Button>
        </form>
      </div>
    </main>
  );
}
