"use client";

/**
 * الرنين المستمر لطلب جديد — نغمة مولّدة بـ WebAudio (بلا ملفات صوتية):
 * جرس ثنائي النغمة يتكرر ما دام هناك طلب بانتظار القبول والصوت مفعّلاً.
 * سياسة المتصفحات تتطلب تفاعلاً قبل الصوت — زر التفعيل يستأنف السياق.
 */

import { useEffect, useRef, useState } from "react";

const SOUND_KEY = "rahalgo_merchant_sound";

export function useRinger(active: boolean) {
  const [enabled, setEnabledState] = useState(true);
  const ctxRef = useRef<AudioContext | null>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    setEnabledState(localStorage.getItem(SOUND_KEY) !== "off");
  }, []);

  function setEnabled(on: boolean) {
    setEnabledState(on);
    localStorage.setItem(SOUND_KEY, on ? "on" : "off");
    if (on) {
      // استدعاء من نقرة المستخدم — اللحظة الصحيحة لإنشاء/استئناف السياق
      ctxRef.current ??= new AudioContext();
      void ctxRef.current.resume();
    }
  }

  useEffect(() => {
    if (!active || !enabled) {
      if (timerRef.current) {
        clearInterval(timerRef.current);
        timerRef.current = null;
      }
      return;
    }

    ctxRef.current ??= new AudioContext();
    const ctx = ctxRef.current;
    void ctx.resume().catch(() => undefined);

    function chime(freq: number, at: number) {
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
      if (ctx.state !== "running") return;
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
  }, [active, enabled]);

  return { enabled, setEnabled };
}
