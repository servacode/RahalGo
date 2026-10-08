"use client";

/**
 * الرنين المستمر لطلب جديد — نغمة مولّدة بـ WebAudio (بلا ملفات صوتية):
 * جرس ثنائي النغمة يتكرر ما دام هناك ما ينتظر والصوت مفعّلاً.
 * سياسة المتصفحات تتطلب تفاعلاً قبل الصوت — **وأوّلُ لمسةٍ في اللوحة تفتحه**
 * (السياقُ المشترك في `@rahalgo/ui/chime`)، وزرُّ التفعيل احتياطٌ لا شرط.
 *
 * **ويستعمله لوحُ الطلبات** (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٥): رنينٌ متكرّرٌ
 * للطلب الجديد حتّى يضغط موظّفٌ «استلمتها». **و`unlocked` يقول أيسمع المتصفّحُ
 * أصلاً** — فإن لم يسمع ظهر زرُّ «شغّل صوت التنبيه» بدل صمتٍ لا يُرى.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { sharedAudio, audioRunning, onAudioState, installAudioUnlock } from "@rahalgo/ui";

const DEFAULT_KEY = "rahalgo_merchant_sound";

/** **قراءةُ المتصفّح قد تُرمى** (نافذةٌ خاصّة · تخزينٌ محجوب) — فلا تُسقط الشاشة. */
function readPref(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function writePref(key: string, v: string) {
  try {
    localStorage.setItem(key, v);
  } catch {
    // **تفضيلٌ لا يُحفظ لا يُسقط شيئاً** — يعود الافتراضُ في الفتحة التالية.
  }
}

export function useRinger(active: boolean, storageKey: string = DEFAULT_KEY) {
  const [enabled, setEnabledState] = useState(true);
  /** **أيسمح المتصفّحُ بالصوت الآن؟** — يصير صادقاً بعد أوّل لمسةٍ في اللوحة. */
  const [unlocked, setUnlocked] = useState(false);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    setEnabledState(readPref(storageKey) !== "off");
  }, [storageKey]);

  // ══════════════════════════════════════════════════════════════════
  // **سياقٌ واحدٌ مع جرس التنبيهات** (قرارُ المالك ٢٠٢٦-١٠-٠٨)
  // ══════════════════════════════════════════════════════════════════
  //
  // كان لهذا الرنين سياقُه الخاصّ وزرُّه الخاصّ — **فيُضغط «فعّل صوت الطلبات»
  // كلَّ دوام.** والآن يتبع السياقَ المشترك الذي تفتحه أوّلُ لمسةٍ في اللوحة.
  useEffect(() => {
    installAudioUnlock();
    setUnlocked(audioRunning());
    return onAudioState(setUnlocked);
  }, []);

  const context = useCallback((): AudioContext | null => sharedAudio(), []);

  /** **يُنادى من ضغطة المستخدم** — اللحظةُ التي يسمح فيها المتصفّحُ بالصوت. */
  const unlock = useCallback(() => {
    const ctx = context();
    if (!ctx) return;
    void ctx
      .resume()
      .then(() => setUnlocked(ctx.state === "running"))
      .catch(() => undefined);
  }, [context]);

  function setEnabled(on: boolean) {
    setEnabledState(on);
    writePref(storageKey, on ? "on" : "off");
    if (on) unlock();
  }

  useEffect(() => {
    if (!active || !enabled) {
      if (timerRef.current) {
        clearInterval(timerRef.current);
        timerRef.current = null;
      }
      return;
    }

    const ctx = context();
    if (!ctx) return;
    void ctx.resume().catch(() => undefined);

    function chime(freq: number, at: number) {
      if (!ctx) return;
      const osc = ctx.createOscillator();
      const gain = ctx.createGain();
      osc.type = "sine";
      osc.frequency.value = freq;
      gain.gain.setValueAtTime(0.001, at);
      gain.gain.exponentialRampToValueAtTime(0.4, at + 0.02);
      gain.gain.exponentialRampToValueAtTime(0.001, at + 0.45);
      osc.connect(gain).connect(ctx.destination);
      osc.start(at);
      osc.stop(at + 0.5);
    }

    function ring() {
      if (!ctx || ctx.state !== "running") return;
      const t = ctx.currentTime;
      chime(880, t);
      chime(660, t + 0.28);
    }

    ring();
    timerRef.current = setInterval(ring, 2200);
    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current);
        timerRef.current = null;
      }
    };
  }, [active, enabled, context]);

  return { enabled, setEnabled, unlocked, unlock };
}
