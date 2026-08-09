"use client";

/**
 * **حديثُ الطلب — مكوّنٌ واحدٌ لطرفيه.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «لازم الاثنان لا يقدران يوصلان لبعض إلّا عن طريق
 *  المنصّة فقط».)
 *
 * # ولماذا واحدٌ لا اثنان
 *
 * **الشاشتان تعرضان الحديثَ نفسَه** — ونسختان منه تفترقان: يُصلَح شيءٌ في
 * لوحة السائق ويبقى عند الزبون، **فيرى أحدُهما ما لا يراه الآخر من محادثةٍ
 * بينهما.**
 *
 * **والخادمُ يعرف من هو من جلسته** — فلا يأخذ هذا المكوّنُ دوراً ولا معرّفاً،
 * **وكلُّ رسالةٍ تحمل `mine` من الخادم.**
 *
 * # ولا رقمَ ولا اسمَ شخصٍ يُطلب
 *
 * **يُعرَض اسمُ الطرف الآخر كما يردّه الخادم** (`peer_name`)، **ولا رقمَ معه
 * أبداً** — لا في الحمولة ولا في الشاشة.
 *
 * # والقناةُ تُقفل للكتابة لا للقراءة
 *
 * **حديثٌ يختفي بانتهاء الطلب يمحو ما يُحتجّ به** عند شكوى. فيبقى مقروءاً،
 * **ويُقال متى أُغلق** — ومن كتب فرُدّ بلا سببٍ يعيدها ثمّ يظنّ التطبيق معطوباً.
 *
 * # وشكلُه من لغة الأسطح لا من خارجها
 *
 * (شكوى المالك ٢٠٢٦-٠٨-١٠: «شكلُ شاشة الدردشة مختلفٌ عن الثيم، وشاشةٌ مو
 *  احترافيّة أبداً، بايخة بدائيّة».)
 *
 * **وكان اللوحُ أبيضَ صريحاً** (`bg-paper` = ‎#ffffff) **في منصّةٍ داكنة** —
 * لوحٌ من عالمٍ آخرَ يطفو فوقها. **وداخلَه بطاقةٌ ثانيةٌ بحدٍّ وحشوةٍ ثانية**
 * (`surface` داخل `bg-paper`)، **وعنوانان فوق بعضهما**: عنوانُ اللوح وعنوانُ
 * المكوّن.
 *
 * **والفقّاعتان كانتا صبغتين باهتتين** لا يُفرَّق بينهما إلّا بالجهة —
 * **ومن قرأ سطراً لا يعرف أقاله هو أم قيل له** إلّا بعد أن يتتبّع الحافّة.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale, fmtTime } from "@rahalgo/i18n";
import { Alert } from "./feedback";
import { IconChat, IconSend } from "./icons";
import { useLiveRefresh } from "./Notifications";

const m = getMessages(defaultLocale);
const C = m.chat;

type ApiFn = <T = unknown>(path: string, init?: RequestInit) => Promise<T>;

interface ChatMessage {
  id: string;
  body: string;
  role: "customer" | "driver";
  mine: boolean;
  created_at: string;
  read_at: string | null;
}

interface Thread {
  messages: ChatMessage[];
  peer_name: string;
  open: boolean;
  closes_at: string | null;
}

/** أقصى ما يرتفع حقلُ الكتابة قبل أن ينزلق داخلَ نفسِه — **أربعةُ أسطر.** */
const MAX_COMPOSER = 96;

export function OrderChat({
  api,
  orderId,
  /**
   * **عارياً — بلا سطحٍ ولا عنوان.**
   *
   * **حين يسكن لوحاً له عنوانُه** (الفقّاعةُ العائمة): **سطحٌ داخلَ سطحٍ
   * يضاعف الحدَّ والحشوة**، وعنوانان فوق بعضهما يُقرآن عطباً.
   */
  bare = false,
  /** ارتفاعُ منطقة الرسائل — **لوحُ الفقّاعة يعطيها ما بقي من طوله.** */
  streamClass = "max-h-72",
}: {
  api: ApiFn;
  orderId: string;
  bare?: boolean;
  streamClass?: string;
}) {
  const [thread, setThread] = useState<Thread | null>(null);
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const endRef = useRef<HTMLDivElement | null>(null);
  const boxRef = useRef<HTMLTextAreaElement | null>(null);

  const load = useCallback(async () => {
    try {
      const t = await api<Thread>(`/api/v1/orders/${orderId}/messages`);
      setThread(t);
    } catch {
      // **وقناةٌ لا تُفتح لا تُسقط الشاشة** — الطلبُ يُقرأ بلا حديثه.
      setThread(null);
    }
  }, [api, orderId]);

  useEffect(() => {
    void load();
  }, [load]);

  // **حيٌّ**: الرسالةُ تصل وصاحبُها ينظر — **وشاشةٌ لا تتحدّث تجعله يكتب
  // مرّتين** ظنّاً أنّ الأولى لم تصل.
  useLiveRefresh(["order"], load);

  // **وينزل إلى آخره** — الجديدُ في الأسفل، **ومن فتح فوجد قديماً ظنّ أنّ
  // شيئاً لم يصل.**
  useEffect(() => {
    endRef.current?.scrollIntoView({ block: "nearest" });
  }, [thread?.messages.length]);

  /**
   * **والحقلُ ينمو بما فيه ثمّ يقف.**
   *
   * **سطران محجوزان لمن يكتب كلمةً واحدة** يأكلان من الحديث بلا سبب،
   * **وحقلٌ لا ينمو يُخفي أوّلَ ما كُتب** عمّن يراجع جملتَه قبل الإرسال.
   */
  function grow(el: HTMLTextAreaElement | null) {
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${Math.min(el.scrollHeight, MAX_COMPOSER)}px`;
  }

  if (!thread) return null;

  async function submit() {
    const body = draft.trim();
    if (!body || busy) return;
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/orders/${orderId}/messages`, {
        method: "POST",
        body: JSON.stringify({ body }),
      });
      setDraft("");
      grow(boxRef.current);
      await load();
    } catch {
      setError(C.sendFailed);
    } finally {
      setBusy(false);
    }
  }

  const stream = (
    <div className={`flex-1 space-y-3 overflow-y-auto px-1 ${streamClass}`}>
      {thread.messages.length === 0 ? (
        <p className="py-8 text-center text-sm text-ink-muted">{C.empty}</p>
      ) : (
        thread.messages.map((x) => (
          <div
            key={x.id}
            className={`flex flex-col ${x.mine ? "items-end" : "items-start"}`}
          >
            {/* ══════════════════════════════════════════════════════════
                **ولي فقّاعةٌ مصمتةٌ وله واحدةٌ غائرة**
                ══════════════════════════════════════════════════════════

                **كانتا صبغتين باهتتين** (`accent-tint` و`field`) لا يُفرَّق
                بينهما إلّا بالجهة — **ومن قرأ سطراً لا يعرف أقاله هو أم قيل
                له** حتّى يتتبّع الحافّة.

                **والفرقُ في الوظيفة لا في الدرجة**: كلامي **يخرج منّي**
                فيأخذ لونَ النبرة مصمتاً، **وكلامُه يصل إليّ** فيجلس في
                غائرٍ من أرض اللوح.

                **والزاويةُ الرابعةُ ضيّقةٌ في جهة قائلها** — ذيلٌ بلا رسمٍ
                إضافيّ، **يقول من أين جاء السطرُ قبل أن يُقرأ.** */}
            <div
              className={`max-w-[80%] px-3 py-2 text-sm leading-relaxed ${
                x.mine
                  ? "rounded-card rounded-se-control bg-accent text-on-bright"
                  : "surface-inset rounded-card rounded-ss-control text-ink"
              }`}
            >
              <p className="whitespace-pre-wrap break-words">{x.body}</p>
            </div>
            {/* **والوقتُ تحت الفقّاعة لا فيها** — **رمادِيٌّ داخلَ لونٍ
                مصمتٍ يسقط تباينُه**، ويُقرأ الوقتُ جزءاً من الكلام. */}
            <p className="mt-1 px-1 text-2xs text-ink-muted">
              <span dir="ltr">{fmtTime(x.created_at)}</span>
              {/* **و«قُرئت» لمن أرسل وحدَه** — ولا معنى لها عند المتلقّي. */}
              {x.mine && x.read_at ? ` · ${C.seen}` : ""}
            </p>
          </div>
        ))
      )}
      <div ref={endRef} />
    </div>
  );

  const composer = thread.open ? (
    /* ══════════════════════════════════════════════════════════════════
       **وشريطٌ سفليٌّ واحدٌ — لا صندوقٌ وزرٌّ بجانبه**
       ══════════════════════════════════════════════════════════════════

       (قرارُ المالك ٢٠٢٦-٠٨-١٠: «خلّي شريطاً سفليّاً للكتابة، وزرّ الإنتر
        يقوم بالإرسال، وأيضاً أيقونة إرسالٍ جانبَ شريط الكتابة».)

       **كان صندوقاً بسطرين وزرّاً مربّعاً بجانبه** — جسمان منفصلان
       بارتفاعين، **فيُقرآن حقلاً وزرّاً لا شريطَ كتابة.**

       **وصارا جسماً واحداً**: الحقلُ والأيقونةُ في غائرٍ واحدٍ مستديرِ
       الطرفين — **وهو الشكلُ الذي تعرفه كلُّ يدٍ أمسكت هاتفاً.** */
    <form
      onSubmit={(e) => {
        e.preventDefault();
        void submit();
      }}
      className="mt-2 flex items-end gap-1.5 surface-inset rounded-badge py-1 pe-1 ps-3"
    >
      <textarea
        ref={boxRef}
        rows={1}
        value={draft}
        onChange={(e) => {
          setDraft(e.target.value);
          grow(e.currentTarget);
        }}
        /* **والإنتر يُرسل — والسطرُ الجديد بـ`Shift`.**

           **ومن كتب سطراً ثمّ بحث عن زرٍّ يضغطه توقّف**: العادةُ في كلّ
           تطبيق حديثٍ أنّ الإنتر يُرسل، **وشاشةٌ تخالف العادةَ تُقرأ
           معطوبةً لا مختلفة.**

           **ولا يُرسل وهو يؤلّف الحرف**: لوحاتُ المفاتيح العربيّةُ
           والصينيّةُ تفتح نافذةَ تأليفٍ يُنهيها الإنتر — **فيُرسل نصفُ
           كلمةٍ لمن ضغطه ليُتمّها.** */
        onKeyDown={(e) => {
          if (e.key !== "Enter" || e.shiftKey || e.nativeEvent.isComposing) return;
          e.preventDefault();
          void submit();
        }}
        placeholder={C.placeholder}
        maxLength={500}
        className="max-h-24 flex-1 resize-none bg-transparent py-1.5 text-sm text-ink outline-none placeholder:text-ink-muted"
      />
      {/* **والأيقونةُ داخلَ الشريط لا بجانبه** — **وزرٌّ خارجَه يزيد جسماً
          ثالثاً في سطرٍ فيه اثنان.**

          **وتخفت حتّى يُكتب شيء**: قرصٌ مضيءٌ فوق حقلٍ فارغٍ يدعو إلى ضغطةٍ
          لا تفعل. */}
      <button
        type="submit"
        disabled={busy || !draft.trim()}
        aria-label={C.send}
        className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-badge transition-colors ${
          draft.trim() && !busy
            ? "bg-accent text-on-bright"
            : "bg-field text-ink-muted"
        }`}
      >
        {/* **ورأسُ السهم إلى جهة القراءة** — **سهمٌ يشير إلى الخلف في
            العربيّة يُقرأ رجوعاً لا إرسالاً.** */}
        <IconSend size={17} className="rtl:-scale-x-100" />
      </button>
    </form>
  ) : (
    /* **ويُقال إنّها أُغلقت** — لا يُخفى الحقلُ بلا كلمة. */
    <p className="mt-2 rounded-control bg-field px-3 py-2 text-center text-xs text-ink-muted">
      {C.closed}
    </p>
  );

  const body = (
    <div className="flex min-h-0 flex-1 flex-col">
      {stream}
      {error && <Alert className="mt-2">{error}</Alert>}
      {composer}
    </div>
  );

  if (bare) return body;
  return (
    <section className="surface flex flex-col p-4">
      <h3 className="mb-3 flex items-center gap-2 heading-card text-ink">
        <IconChat size={18} className="text-ink-muted" />
        {C.title.replace("{name}", thread.peer_name)}
      </h3>
      {body}
    </section>
  );
}
