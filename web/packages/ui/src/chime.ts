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

function audio(): AudioContext | null {
  const Ctx =
    window.AudioContext ??
    (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
  if (!Ctx) return null;
  shared ??= new Ctx();
  if (shared.state === "suspended") void shared.resume();
  return shared;
}

/** نغمتان صاعدتان — قصيرتان تُسمعان تحت خوذة. */
export function useChime(): () => void {
  return useCallback(() => {
    try {
      const ctx = audio();
      if (!ctx) return;
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
  }, []);
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
