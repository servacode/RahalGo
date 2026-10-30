"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عارضُ الصورة بملء الشاشة** — مرآةُ `ImageViewer.kt` (قرارُ المالك ٢٠٢٦-١٠-٢٨/٢٩)
 * ══════════════════════════════════════════════════════════════════════
 *
 * الصورةُ كلُّها على خلفيّةٍ داكنة، **تكبيرٌ بإصبعين أو بالعجلة، وسحبٌ،
 * ونقرتان للتكبير**، وزرُّ إغلاقٍ ظاهر. **و«رجوع» المتصفّح أو الهاتف يُغلقه**
 * ولا يغادر الصفحة — ومثلُه `Esc`.
 */

import { useEffect, useRef, useState, type PointerEvent } from "react";
import { createPortal } from "react-dom";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconClose, IconZoomIn, IconZoomOut } from "./icons";

const m = getMessages(defaultLocale);

const MAX_ZOOM = 5;
const DOUBLE_TAP_ZOOM = 2.5;
const DOUBLE_TAP_MS = 300;

interface Pt {
  x: number;
  y: number;
}

export function ImageViewer({ src, alt = "", onClose }: { src: string; alt?: string; onClose: () => void }) {
  const [scale, setScale] = useState(1);
  const [off, setOff] = useState<Pt>({ x: 0, y: 0 });
  const [mounted, setMounted] = useState(false);
  const boxRef = useRef<HTMLDivElement>(null);
  const pointers = useRef(new Map<number, Pt>());
  const pinch = useRef<{ dist: number; mid: Pt } | null>(null);
  const lastTap = useRef(0);
  const closeRef = useRef(onClose);
  closeRef.current = onClose;

  useEffect(() => setMounted(true), []);

  // **«رجوع» يُغلق العارض** — قيدٌ في التاريخ يحمل حالةَ الصفحة نفسَها فلا يرتبك الموجِّه.
  useEffect(() => {
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    window.history.pushState({ ...(window.history.state ?? {}), rgViewer: true }, "");
    let popped = false;
    const onPop = () => {
      popped = true;
      closeRef.current();
    };
    // **و`Esc` يُغلق العارضَ وحدَه** — لا النافذةَ التي تحته.
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return;
      e.stopPropagation();
      closeRef.current();
    };
    window.addEventListener("popstate", onPop);
    window.addEventListener("keydown", onKey, true);
    return () => {
      window.removeEventListener("popstate", onPop);
      window.removeEventListener("keydown", onKey, true);
      document.body.style.overflow = prev;
      if (!popped && window.history.state?.rgViewer) window.history.back();
    };
  }, []);

  function size(): { w: number; h: number; cx: number; cy: number } {
    const r = boxRef.current?.getBoundingClientRect();
    const w = r?.width ?? window.innerWidth;
    const h = r?.height ?? window.innerHeight;
    return { w, h, cx: (r?.left ?? 0) + w / 2, cy: (r?.top ?? 0) + h / 2 };
  }

  function clamp(o: Pt, s: number): Pt {
    const { w, h } = size();
    const mx = (w * (s - 1)) / 2;
    const my = (h * (s - 1)) / 2;
    return { x: Math.min(mx, Math.max(-mx, o.x)), y: Math.min(my, Math.max(-my, o.y)) };
  }

  /** **يكبّر حول نقطة** — ما تحت الإصبع يبقى تحته. */
  function zoomAt(next: number, at: Pt, pan: Pt = { x: 0, y: 0 }) {
    const ns = Math.min(MAX_ZOOM, Math.max(1, next));
    const z = ns / scale;
    const { cx, cy } = size();
    const o = {
      x: (at.x - cx) * (1 - z) + off.x * z + pan.x,
      y: (at.y - cy) * (1 - z) + off.y * z + pan.y,
    };
    setScale(ns);
    setOff(ns === 1 ? { x: 0, y: 0 } : clamp(o, ns));
  }

  useEffect(() => {
    const el = boxRef.current;
    if (!el) return;
    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      zoomAt(scale * Math.exp(-e.deltaY * 0.0015), { x: e.clientX, y: e.clientY });
    };
    el.addEventListener("wheel", onWheel, { passive: false });
    return () => el.removeEventListener("wheel", onWheel);
  });

  function onPointerDown(e: PointerEvent<HTMLDivElement>) {
    e.currentTarget.setPointerCapture(e.pointerId);
    pointers.current.set(e.pointerId, { x: e.clientX, y: e.clientY });
    pinch.current = null;
    if (pointers.current.size === 1) {
      const now = Date.now();
      if (now - lastTap.current < DOUBLE_TAP_MS) {
        lastTap.current = 0;
        if (scale > 1) {
          setScale(1);
          setOff({ x: 0, y: 0 });
        } else {
          zoomAt(DOUBLE_TAP_ZOOM, { x: e.clientX, y: e.clientY });
        }
      } else {
        lastTap.current = now;
      }
    }
  }

  function onPointerMove(e: PointerEvent<HTMLDivElement>) {
    const prev = pointers.current.get(e.pointerId);
    if (!prev) return;
    pointers.current.set(e.pointerId, { x: e.clientX, y: e.clientY });
    const pts = [...pointers.current.values()];
    if (pts.length >= 2) {
      const [a, b] = pts as [Pt, Pt];
      const dist = Math.hypot(a.x - b.x, a.y - b.y);
      const mid = { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 };
      const last = pinch.current;
      pinch.current = { dist, mid };
      if (last && last.dist > 0) zoomAt(scale * (dist / last.dist), mid, { x: mid.x - last.mid.x, y: mid.y - last.mid.y });
      return;
    }
    if (scale <= 1) return;
    setOff((o) => clamp({ x: o.x + e.clientX - prev.x, y: o.y + e.clientY - prev.y }, scale));
  }

  function onPointerUp(e: PointerEvent<HTMLDivElement>) {
    pointers.current.delete(e.pointerId);
    pinch.current = null;
  }

  if (!mounted) return null;
  return createPortal(
    <div className="fixed inset-0 z-[90] scrim" role="dialog" aria-modal="true" aria-label={alt || m.common.media.image}>
      <div
        ref={boxRef}
        dir="ltr"
        className="absolute inset-0 touch-none select-none overflow-hidden bg-shell"
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        onPointerCancel={onPointerUp}
      >
        {/* صور الوسائط ديناميكية من خادمنا — لا تمر بمحسّن Next */}
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={src}
          alt={alt}
          draggable={false}
          className="h-full w-full object-contain"
          style={{ transform: `translate(${off.x}px, ${off.y}px) scale(${scale})` }}
        />
      </div>
      <button
        type="button"
        onClick={onClose}
        aria-label={m.common.media.viewerClose}
        className="absolute end-3 top-3 flex h-11 w-11 items-center justify-center rounded-full bg-scrim text-on-solid"
        style={{ marginTop: "env(safe-area-inset-top)" }}
      >
        <IconClose size={24} />
      </button>
      {/* **وزرّا تكبيرٍ وتصغيرٍ لمن لا إصبعين ولا عجلةَ له** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — حول المركز. */}
      <div
        className="pointer-events-none absolute inset-x-0 bottom-4 flex justify-center gap-2"
        style={{ marginBottom: "env(safe-area-inset-bottom)" }}
      >
        {(
          [
            [m.common.media.zoomOut, IconZoomOut, 1 / 1.5, scale <= 1],
            [m.common.media.zoomIn, IconZoomIn, 1.5, scale >= MAX_ZOOM],
          ] as const
        ).map(([label, Icon, f, off]) => (
          <button
            key={label}
            type="button"
            aria-label={label}
            title={label}
            disabled={off}
            onClick={() => {
              const { cx, cy } = size();
              zoomAt(scale * f, { x: cx, y: cy });
            }}
            className="pointer-events-auto flex h-11 w-11 items-center justify-center rounded-full bg-scrim text-on-solid disabled:opacity-40"
          >
            <Icon size={22} />
          </button>
        ))}
      </div>
    </div>,
    document.body,
  );
}
