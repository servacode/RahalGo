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
import { getMessages, defaultLocale, fmtTime, fmtRelDay, errorText } from "@rahalgo/i18n";
import { Alert } from "./feedback";
import { IconChat, IconSend, IconCheck, IconCheckAll, IconPhoto, IconMic } from "./icons";
import { ChatAttachmentView, fmtDuration, type ChatAttachment } from "./ChatAttachment";
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
  /** صورةٌ أو رسالةٌ صوتيّة — وحين تحضر تُعرض هي وشرحُها بدل `body` (نصٌّ احتياطيٌّ للتطبيقات القديمة). */
  attachment?: ChatAttachment;
}

interface Thread {
  messages: ChatMessage[];
  peer_name: string;
  open: boolean;
  closes_at: string | null;
}

/** أقصى ما يرتفع حقلُ الكتابة قبل أن ينزلق داخلَ نفسِه — **أربعةُ أسطر.** */
const MAX_COMPOSER = 96;

/** أطولُ رسالةٍ صوتيّة — يوقفها الإرسالُ تلقائيّاً. */
const MAX_VOICE_MS = 60_000;
/** أقصرُ من هذا يُهمَل: لمسةٌ عابرةٌ لا رسالة. */
const MIN_VOICE_MS = 700;
/** إن ابتعد الإصبعُ هذا القدر عن موضع الضغط فالتسجيلُ يُلغى. */

/** أفضلُ صيغةٍ يسجّلها المتصفّح — سفاري `mp4` وغيره `webm`. */
function pickMime(): string {
  if (typeof MediaRecorder === "undefined") return "";
  for (const t of ["audio/mp4", "audio/webm;codecs=opus"]) {
    if (MediaRecorder.isTypeSupported(t)) return t;
  }
  return "";
}

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
  /**
   * **يُنادى حين يصل الحديثُ من المحرّك** — **ووصولُه هو وسمُ القراءة هناك.** يقرأ به صاحبُ
   * الشارة حقيقةَ العدّاد بعد القراءة لا قبلها.
   */
  onLoaded,
}: {
  api: ApiFn;
  orderId: string;
  bare?: boolean;
  streamClass?: string;
  onLoaded?: () => void;
}) {
  const [thread, setThread] = useState<Thread | null>(null);
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const endRef = useRef<HTMLDivElement | null>(null);
  const boxRef = useRef<HTMLTextAreaElement | null>(null);
  const fileRef = useRef<HTMLInputElement | null>(null);
  const [canRecord, setCanRecord] = useState(false);
  const [recording, setRecording] = useState(false);
  const [recMs, setRecMs] = useState(0);
  const rec = useRef<{
    recorder: MediaRecorder | null;
    stream: MediaStream | null;
    chunks: Blob[];
    startedAt: number;
    timer: ReturnType<typeof setInterval> | null;
    pressing: boolean;
    cancel: boolean;
    x: number;
    y: number;
  }>({ recorder: null, stream: null, chunks: [], startedAt: 0, timer: null, pressing: false, cancel: false, x: 0, y: 0 });

  // **ودالّةُ «وصل الحديث» في مرجعٍ** — كي لا يُعاد الجلبُ كلَّما تبدّلت هويّتُها.
  const loadedRef = useRef(onLoaded);
  loadedRef.current = onLoaded;

  const load = useCallback(async () => {
    try {
      const t = await api<Thread>(`/api/v1/orders/${orderId}/messages`);
      setThread(t);
      return true;
    } catch {
      // **وقناةٌ لا تُفتح لا تُسقط الشاشة** — الطلبُ يُقرأ بلا حديثه.
      setThread(null);
      return false;
    }
  }, [api, orderId]);

  useEffect(() => {
    // **عند الفتح والتبديل وحدَهما** — لا مع كلّ نبضة: النبضةُ تُنعش العدّادَ بنفسها.
    void load().then((ok) => {
      if (ok) loadedRef.current?.();
    });
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

  // **المايك يظهر لمن يملكه** — يُفحص بعد التركيب كي لا يختلف الرسمُ بين الخادم والمتصفّح.
  useEffect(() => {
    setCanRecord(
      typeof MediaRecorder !== "undefined" &&
        typeof navigator !== "undefined" &&
        !!navigator.mediaDevices?.getUserMedia,
    );
    const r = rec.current;
    return () => {
      r.pressing = false;
      r.cancel = true;
      if (r.timer) clearInterval(r.timer);
      if (r.recorder && r.recorder.state !== "inactive") r.recorder.stop();
      r.stream?.getTracks().forEach((t) => t.stop());
    };
  }, []);

  /** يرفع مرفقاً (صورة أو صوت) بالمسار الجديد. */
  async function upload(file: Blob, kind: "photo" | "audio", name: string, durationMs?: number) {
    setBusy(true);
    setError("");
    try {
      const fd = new FormData();
      fd.append("file", file, name);
      fd.append("kind", kind);
      if (durationMs !== undefined) fd.append("duration_ms", String(Math.round(durationMs)));
      await api(`/api/v1/orders/${orderId}/messages/attachment`, { method: "POST", body: fd });
      await load();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  function onPhoto(file: File | undefined) {
    if (fileRef.current) fileRef.current.value = "";
    if (!file || busy) return;
    void upload(file, "photo", file.name || "photo.jpg");
  }

  function finishRec() {
    const r = rec.current;
    if (r.timer) clearInterval(r.timer);
    r.timer = null;
    r.stream?.getTracks().forEach((t) => t.stop());
    r.stream = null;
    const ms = Date.now() - r.startedAt;
    const type = r.recorder?.mimeType || "audio/webm";
    const chunks = r.chunks;
    const cancelled = r.cancel;
    r.recorder = null;
    r.chunks = [];
    setRecording(false);
    setRecMs(0);
    if (cancelled || ms < MIN_VOICE_MS || chunks.length === 0) return;
    const ext = type.includes("mp4") ? "m4a" : type.includes("ogg") ? "ogg" : "webm";
    void upload(new Blob(chunks, { type }), "audio", `voice.${ext}`, Math.min(ms, MAX_VOICE_MS));
  }

  /** يوقف التسجيل — `send=false` يلغيه. */
  function stopRec(send: boolean) {
    const r = rec.current;
    r.pressing = false;
    r.cancel = !send;
    if (r.recorder && r.recorder.state !== "inactive") r.recorder.stop();
  }

  /* **كبسةٌ تبدأ وكبسةٌ ترسل** (قرارُ المالك ٢٠٢٦-١١-١٢: «كبسة تسجيل الصوت لازم تظل مستمرة حتى لو شال
     ايدو عن المايك») — مرآةُ زرّ التطبيق: أثناء التسجيل يصير الزرُّ «إرسال»، و«احذف» في شريط التسجيل. */
  async function startRec() {
    const r = rec.current;
    if (busy || r.recorder || r.pressing) return;
    r.pressing = true;
    r.cancel = false;
    setError("");
    let stream: MediaStream;
    try {
      // **الإذنُ يُطلب عند أوّل ضغطةٍ فقط** — لا عند فتح الصفحة.
      stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    } catch {
      r.pressing = false;
      setError(C.micDenied);
      return;
    }
    // **أُلغي أثناء نافذة الإذن** — فلا يُبدأ.
    if (!r.pressing) {
      stream.getTracks().forEach((t) => t.stop());
      return;
    }
    const mime = pickMime();
    let recorder: MediaRecorder;
    try {
      recorder = mime ? new MediaRecorder(stream, { mimeType: mime }) : new MediaRecorder(stream);
    } catch {
      stream.getTracks().forEach((t) => t.stop());
      r.pressing = false;
      setError(C.micDenied);
      return;
    }
    r.stream = stream;
    r.recorder = recorder;
    r.chunks = [];
    recorder.ondataavailable = (ev) => {
      if (ev.data.size > 0) r.chunks.push(ev.data);
    };
    recorder.onstop = finishRec;
    r.startedAt = Date.now();
    recorder.start();
    setRecording(true);
    setRecMs(0);
    r.timer = setInterval(() => {
      const el = Date.now() - r.startedAt;
      setRecMs(el);
      if (el >= MAX_VOICE_MS) stopRec(true);
    }, 200);
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
        thread.messages.map((x, i) => (
          <div key={x.id} className="space-y-3">
            {/* **وفاصلُ اليوم بين رسائل أيّامٍ مختلفة** (بتوقيت دمشق) — فلا تُقرأ ٣:٠٥ اليوم بعد ٣:٢٩ أمس
                كأنّها من يومٍ واحد. */}
            {(i === 0 || fmtRelDay(thread.messages[i - 1]!.created_at) !== fmtRelDay(x.created_at)) && (
              <p className="flex justify-center">
                <span className="rounded-control bg-field px-2.5 py-0.5 text-2xs text-ink-muted">
                  {fmtRelDay(x.created_at)}
                </span>
              </p>
            )}
          <div
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
              {x.attachment ? (
                <div className="space-y-1.5">
                  <ChatAttachmentView attachment={x.attachment} onDark={x.mine} />
                  {x.attachment.caption && (
                    <p className="whitespace-pre-wrap break-words">{x.attachment.caption}</p>
                  )}
                </div>
              ) : (
                <p className="whitespace-pre-wrap break-words">{x.body}</p>
              )}
            </div>
            {/* **والوقتُ تحت الفقّاعة لا فيها** — **رمادِيٌّ داخلَ لونٍ
                مصمتٍ يسقط تباينُه**، ويُقرأ الوقتُ جزءاً من الكلام. */}
            <p className="mt-1 flex items-center gap-1 px-1 text-2xs text-ink-muted">
              <span dir="ltr">{fmtTime(x.created_at)}</span>
              {/* ══════════════════════════════════════════════════════
                  **وعلامتان لا كلمة**
                  ══════════════════════════════════════════════════════

                  (قرارُ المالك ٢٠٢٦-٠٨-١٠: «إشارتين صح زرقاء مثل الواتساب».)

                  **كانت كلمةَ «قُرئت»** — تُقرأ فتُشغل موضعَ كلمة، **ومن
                  ينظر إلى عشرين سطراً لا يقرأ عشرين كلمة.** والعلامةُ
                  تُدرَك بلا قراءة.

                  **وحالتان لا ثلاث**: صحٌّ واحدٌ «وصلت الخادمَ»، وصحّان
                  زرقاوان «قرأها». **ولا حالةَ وسطى** — لا نعرف متى بلغت
                  جهازَه ولم يفتحها بعد، **وعلامةٌ نضعها بلا أن نعرف كذبٌ
                  يُبنى عليه**: من رآها ظنّ أنّ الرسالةَ في يده وهي في
                  الشبكة.

                  **ولمن أرسل وحدَه** — ولا معنى لها عند المتلقّي: يعرف
                  أنّه قرأ. */}
              {x.mine &&
                (x.read_at ? (
                  <IconCheckAll size={14} className="text-info" aria-label={C.seen} />
                ) : (
                  <IconCheck size={14} className="text-ink-muted" aria-label={C.sent} />
                ))}
            </p>
          </div>
          </div>
        ))
      )}
      <div ref={endRef} />
    </div>
  );

  const recordingBar = recording && (
    <div
      role="status"
      className="mt-2 flex items-center justify-between gap-2 rounded-control bg-field px-3 py-2 text-xs text-ink"
    >
      <span className="flex items-center gap-2">
        <span className="h-2 w-2 animate-pulse rounded-full bg-danger-solid" />
        {C.recording}
        <span dir="ltr">{fmtDuration(recMs)}</span>
      </span>
      <span className="flex items-center gap-3">
        <span className="text-ink-muted">{C.tapToSend}</span>
        <button type="button" onClick={() => stopRec(false)} className="font-bold text-danger">
          {C.discard}
        </button>
      </span>
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
    <>
    {recordingBar}
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
      <input
        ref={fileRef}
        type="file"
        accept="image/*"
        hidden
        onChange={(e) => onPhoto(e.target.files?.[0])}
      />
      <button
        type="button"
        disabled={busy || recording}
        onClick={() => fileRef.current?.click()}
        aria-label={C.photo}
        className="flex h-9 w-9 shrink-0 items-center justify-center rounded-badge text-ink-muted disabled:opacity-40"
      >
        <IconPhoto size={19} />
      </button>
      {canRecord && (
        <button
          type="button"
          disabled={busy && !recording}
          onClick={() => (recording ? stopRec(true) : void startRec())}
          aria-label={recording ? C.sendVoice : C.voice}
          title={recording ? C.sendVoice : C.voice}
          className={`flex h-9 w-9 shrink-0 touch-none select-none items-center justify-center rounded-badge disabled:opacity-40 ${
            recording ? "bg-danger-solid text-on-bright" : "text-ink-muted"
          }`}
        >
          <IconMic size={19} />
        </button>
      )}
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
    </>
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
