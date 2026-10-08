"use client";

/**
 * نغمةُ تنبيهٍ تُولَّد في المتصفّح.
 *
 * **لا ملفَّ صوتٍ يُنتظر من الشبكة** — والسائقُ على شبكةٍ ضعيفةٍ في حيٍّ بعيد،
 * **وملفٌّ لم يصل تنبيهٌ لم يُسمَع.** ولا ملفَّ يضيع في نشرةٍ قادمة.
 *
 * # ولماذا في الحزمة المشتركة
 *
 * كانت مكتوبةً في «طلبات قادمة» وحدَها. ثمّ لزمت في «مهامّي» بعد الإسناد
 * المباشر — **ونسخةٌ ثانيةٌ من ستّةَ عشرَ سطرَ صوتٍ تفترق يوماً**: تُضبط نغمةٌ
 * في موضعٍ وتبقى الأخرى، فيسمع السائقُ نغمتين لشيءٍ واحد ولا يعرف الفرق.
 */

import { useCallback, useEffect, useRef } from "react";

/**
 * **سياقُ صوتٍ واحدٌ للصفحة كلِّها — لا واحدٌ لكلّ رنّة.**
 *
 * (وُجد في فحص التهنيج ٢٠٢٦-٠٨-٠٧.)
 *
 * **كان `new AudioContext()` في كلّ نداءٍ ولا يُغلق أبداً.** والرنينُ المتكرّر
 * ينادي كلَّ أربع ثوانٍ عشرين ثانية — **خمسةُ سياقاتٍ لتنبيهٍ واحد**، وكلُّ
 * سياقٍ خيطُ صوتٍ حيٌّ في النظام.
 *
 * **وكروم يقف عند نحو ستّة** ثمّ يرمي — فيُبتلع الخطأُ هنا صامتاً **فلا
 * يُسمع التنبيهُ أصلاً بعد أوّل دقيقة.** فالتسريبُ يكسر الوظيفةَ لا الأداءَ
 * وحدَه.
 *
 * **والواحدُ يُستأنف إن علّقه المتصفّح**: كروم يوقف السياقَ حتّى أوّلِ لمسة.
 */
let shared: AudioContext | null = null;
const stateListeners = new Set<(running: boolean) => void>();

/**
 * **السياقُ المشترك** — يُنشأ مرّةً ويُستأنف إن علّقه المتصفّح.
 *
 * **ويستعمله رنينُ الطلبات نفسُه** (`ringer.ts`) — فما فتحته ضغطةٌ واحدةٌ فُتح
 * للجميع، **ولا زرَّ «فعّل الصوت» لكلّ شاشة.**
 */
export function sharedAudio(): AudioContext | null {
  if (typeof window === "undefined") return null;
  const Ctx =
    window.AudioContext ??
    (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
  if (!Ctx) return null;
  if (!shared) {
    try {
      shared = new Ctx();
    } catch {
      return null;
    }
    shared.addEventListener("statechange", emitAudioState);
    emitAudioState();
  }
  if (shared.state === "suspended") void shared.resume().then(emitAudioState).catch(() => undefined);
  return shared;
}

/**
 * **يُبلَّغ المستمعون بالحال الحاضر** — بعد الإنشاء وبعد كلّ استئناف أيضاً:
 * سياقٌ وُلد «يعمل» داخل ضغطةٍ لا يُطلق `statechange` أبداً.
 */
function emitAudioState() {
  const running = shared?.state === "running";
  stateListeners.forEach((fn) => {
    try {
      fn(running);
    } catch {
      /* تجاهل */
    }
  });
}

/** **أيسمع المتصفّحُ الآن؟** */
export function audioRunning(): boolean {
  return shared?.state === "running";
}

/** يُبلَّغ كلّما تبدّل حالُ السياق — يُعيد دالّةَ الإلغاء. */
export function onAudioState(fn: (running: boolean) => void): () => void {
  stateListeners.add(fn);
  return () => {
    stateListeners.delete(fn);
  };
}

/**
 * **فتحُ الصوت عند أوّل لمسةٍ في أيّ مكان** (قرارُ المالك ٢٠٢٦-١٠-٠٨).
 *
 * المتصفّحُ يُبقي السياقَ معلّقاً حتّى يلمس المستخدمُ الصفحة — **وكان السياقُ
 * يُنشأ لحظةَ وصول الإشعار**، أي بلا لمسة، **فيبقى صامتاً.** فيُنشأ ويُستأنف
 * عند أوّل ضغطةٍ أو مفتاح، ثمّ تُرفع المستمعات.
 */
let unlockInstalled = false;
export function installAudioUnlock(): void {
  if (unlockInstalled || typeof window === "undefined") return;
  unlockInstalled = true;
  const events = ["pointerdown", "keydown", "touchstart"] as const;
  const handler = () => {
    const ctx = sharedAudio();
    if (!ctx) return;
    void ctx
      .resume()
      .then(() => {
        emitAudioState();
        if (ctx.state === "running") {
          events.forEach((e) => window.removeEventListener(e, handler, true));
        }
      })
      .catch(() => undefined);
  };
  events.forEach((e) => window.addEventListener(e, handler, { capture: true, passive: true }));
}

/** **لا رنّتان في ثانيةٍ ونصف** — دفعةُ إشعاراتٍ معاً تُسمع رنّةً واحدة. */
const CHIME_GAP_MS = 1500;
let lastChime = 0;

/**
 * نغمتان صاعدتان — قصيرتان تُسمعان تحت خوذة.
 *
 * @param force يتخطّى منعَ التكرار — لزرّ «جرّب الصوت».
 */
export function playChime(force = false): void {
  try {
    const now = Date.now();
    if (!force && now - lastChime < CHIME_GAP_MS) return;
    const ctx = sharedAudio();
    if (!ctx) return;
    lastChime = now;
    [880, 1175].forEach((hz, i) => {
      const osc = ctx.createOscillator();
      const gain = ctx.createGain();
      osc.frequency.value = hz;
      osc.connect(gain);
      gain.connect(ctx.destination);
      const t = ctx.currentTime + i * 0.18;
      gain.gain.setValueAtTime(0.0001, t);
      gain.gain.exponentialRampToValueAtTime(0.25, t + 0.02);
      gain.gain.exponentialRampToValueAtTime(0.0001, t + 0.16);
      osc.start(t);
      osc.stop(t + 0.18);
    });
  } catch {
    // صوتٌ لا يعمل لا يُسقط الشاشة — **والبطاقةُ تظهر على أيّ حال.**
  }
}

/** نغمةُ التنبيه — بمنع التكرار المتقارب. */
export function useChime(): () => void {
  return useCallback(() => playChime(), []);
}

/**
 * تنبيهٌ متكرّرٌ لمدّةٍ محدودة — **لمهمّةٍ وقعت في يده بلا أن يطلبها.**
 *
 * # لماذا يتكرّر
 *
 * في العرض يضغط السائقُ «خذ الطلب» — **فهو ينظر أصلاً.** وفي الإسناد المباشر
 * يقع الطلبُ في مهامّه وهو لا ينظر، **ورنّةٌ واحدةٌ تضيع في ضجيج الشارع أو
 * تحت الخوذة.** (قرارُ المالك: «تنبيهُ مهمّةٍ جديدةٍ متكرّرٌ لمدّة ٢٠ ثانية».)
 *
 * # ولمدّةٍ لا إلى الأبد
 *
 * **صوتٌ لا يتوقّف يُكتم الجهازُ من أجله** — ثمّ لا يُسمع التنبيهُ التالي.
 * فيرنّ عشرين ثانيةً ثمّ يصمت، **والبطاقةُ تبقى.**
 *
 * @param on ارفعه حين تظهر مهمّةٌ جديدة، وأنزله حين يتحرّك.
 */
export function useRepeatingChime(on: boolean, seconds = 20, everyMs = 4000): void {
  const chime = useChime();
  const started = useRef(0);

  useEffect(() => {
    if (!on) {
      started.current = 0;
      return;
    }
    // **ولا يُعاد العدُّ ما دام مرفوعاً** — تحديثُ الشاشة لا يُطيل الرنين.
    if (started.current === 0) {
      started.current = Date.now();
      chime();
    }
    const id = setInterval(() => {
      if (Date.now() - started.current >= seconds * 1000) {
        clearInterval(id);
        return;
      }
      chime();
    }, everyMs);
    return () => clearInterval(id);
  }, [on, seconds, everyMs, chime]);
}
