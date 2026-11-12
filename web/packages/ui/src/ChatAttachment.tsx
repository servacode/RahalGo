"use client";

/**
 * **مرفقُ رسالة الحديث — صورةٌ أو رسالةٌ صوتيّة** (قرارُ المالك ٢٠٢٦-١١-١٢).
 *
 * مكوّنٌ واحدٌ يعرضه حديثُ الزبون وسجلُّ الإدارة، فلا يفترق الشكلان.
 * الروابطُ نسبيّةٌ موقَّعة (`/media/...?exp&sig`) تُحلّ بعنوان المحرّك.
 */

import { useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconPause, IconPlay } from "./icons";
import { ImageViewer } from "./ImageViewer";
import { apiBase } from "./runtimeconfig";

const m = getMessages(defaultLocale);
const C = m.chat;

export interface ChatAttachment {
  kind: "photo" | "audio";
  url: string;
  thumb_url?: string;
  duration_ms?: number;
  caption?: string;
}

function abs(path: string | undefined): string {
  if (!path) return "";
  return path.startsWith("http") ? path : `${apiBase()}${path}`;
}

/** **m:ss** — بأرقامٍ لاتينيّة كباقي المنصّة. */
export function fmtDuration(ms: number): string {
  const s = Math.max(0, Math.round(ms / 1000));
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
}

/** **صوتٌ واحدٌ يعمل في كلّ مرّة** — الجديدُ يوقف ما قبله. */
let playing: HTMLAudioElement | null = null;

function AudioPlayer({ url, durationMs, onDark }: { url: string; durationMs?: number; onDark: boolean }) {
  const ref = useRef<HTMLAudioElement | null>(null);
  const [on, setOn] = useState(false);
  const [pos, setPos] = useState(0);

  useEffect(
    () => () => {
      const a = ref.current;
      if (a && playing === a) playing = null;
    },
    [],
  );

  function toggle() {
    const a = ref.current;
    if (!a) return;
    if (a.paused) {
      if (playing && playing !== a) playing.pause();
      playing = a;
      void a.play().catch(() => setOn(false));
    } else {
      a.pause();
    }
  }

  const total = durationMs ?? 0;
  const pct = total > 0 ? Math.min(100, (pos / total) * 100) : 0;

  return (
    <div className="flex min-w-40 items-center gap-2.5">
      <button
        type="button"
        onClick={toggle}
        aria-label={on ? C.pause : C.play}
        className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-full ${
          onDark ? "bg-on-bright text-accent" : "bg-accent text-on-bright"
        }`}
      >
        {on ? <IconPause size={16} /> : <IconPlay size={16} className="rtl:-scale-x-100" />}
      </button>
      <div className="flex flex-1 flex-col gap-1">
        <div className={`h-1 overflow-hidden rounded-full ${onDark ? "bg-accent-soft" : "bg-field"}`}>
          <div className={`h-full ${onDark ? "bg-on-bright" : "bg-accent"}`} style={{ width: `${pct}%` }} />
        </div>
        <span dir="ltr" className="text-2xs opacity-80">
          {fmtDuration(on || pos > 0 ? pos : total)}
        </span>
      </div>
      <audio
        ref={ref}
        src={url}
        preload="none"
        onPlay={() => setOn(true)}
        onPause={() => setOn(false)}
        onEnded={() => {
          setOn(false);
          setPos(0);
        }}
        onTimeUpdate={(e) => setPos(e.currentTarget.currentTime * 1000)}
      />
    </div>
  );
}

/**
 * @param onDark داخلَ فقّاعةٍ مصمتةٍ بلون النبرة (رسالتي) — تُعكس ألوانُ المشغّل.
 * @param native مشغّلُ المتصفّح الكامل (`controls`) — لسجلّ الإدارة.
 */
export function ChatAttachmentView({
  attachment,
  onDark = false,
  native = false,
}: {
  attachment: ChatAttachment;
  onDark?: boolean;
  native?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const full = abs(attachment.url);

  if (attachment.kind === "photo") {
    const thumb = abs(attachment.thumb_url) || full;
    return (
      <>
        <button
          type="button"
          onClick={() => setOpen(true)}
          aria-label={C.photoAlt}
          className="block overflow-hidden rounded-control"
        >
          {/* صور الوسائط ديناميكية من خادمنا — لا تمر بمحسّن Next */}
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={thumb} alt={C.photoAlt} loading="lazy" className="max-h-60 w-full max-w-60 object-cover" />
        </button>
        {open && <ImageViewer src={full} alt={C.photoAlt} onClose={() => setOpen(false)} />}
      </>
    );
  }

  if (native) {
    return <audio controls preload="none" src={full} className="h-9 max-w-full" />;
  }
  return <AudioPlayer url={full} durationMs={attachment.duration_ms} onDark={onDark} />;
}
