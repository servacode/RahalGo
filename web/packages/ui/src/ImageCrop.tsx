"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **معاينةُ الصورة وقصُّها قبل الرفع** — مرآةُ `ImageCrop.kt` (طلبُ المالك ٢٠٢٦-١٠-٢٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ما يبين داخل الإطار هو بالضبط ما يصل السوق.** يكبّر بإصبعين أو بعجلة
 * الفأرة أو بالمنزلقة، ويسحب، ويدوّر ربعَ دورة — ولا يُرفع شيءٌ قبل «اعتمد».
 *
 * **وصورةُ الصنف لها ثلاثةُ إطارات** (قرارُ المالك ٢٠٢٦-١٠-٢٨/٢٩: «صاحب المتجر
 * يقدر يخلّي الصورة كاملة»): **كاملة** (الافتراض — بلا قصٍّ، تُرفع بأبعادها) ·
 * مربّعة · طوليّة ٣:٤. **والشعارُ مربّعٌ وحدَه.**
 *
 * **والحسابُ نفسُه في التطبيق** (`frameSize` · `coverScale` · `clampOffset` ·
 * `cropFrame`): الصورةُ تغطّي الإطارَ دائماً فلا فراغَ فيه، **والناتجُ JPEG بجودة
 * ٨٥ وأطولُ ضلعٍ ١٦٠٠** كـ`readScaledImage`.
 *
 * **والإحداثيّاتُ فيزيائيّةٌ (يسار←يمين)** بغضّ النظر عن اتّجاه الصفحة —
 * وإلّا انعكس السحبُ بالعربيّ.
 */

import { useCallback, useEffect, useLayoutEffect, useRef, useState, type PointerEvent, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Modal } from "./components";
import { FormActions } from "./FormActions";
import { themeColor } from "./cssvar";
import { IconCheck, IconClose, IconFrameFull, IconFramePortrait, IconFrameSquare, IconReset, IconRotate, IconCamera } from "./icons";

const m = getMessages(defaultLocale);
const C = m.common.media.crop;

/** **أطولُ ضلعٍ يُعمل عليه** — `MAX_EDGE` في `ImagePick.kt`. */
const MAX_EDGE = 1600;
const JPEG_QUALITY = 0.85;
const MAX_ZOOM = 6;

/** **إطارُ القصّ** — `CropFrame` في التطبيق. */
export type CropFrame = "full" | "square" | "portrait";
/** إطاراتُ صورة الصنف — أوّلُها الافتراض. */
export const ITEM_CROP_FRAMES: CropFrame[] = ["full", "square", "portrait"];
/** إطارُ الشعار. */
export const LOGO_CROP_FRAMES: CropFrame[] = ["square"];

const RATIO: Record<CropFrame, number> = { full: 0, square: 1, portrait: 0.75 };
const FRAME_UI: Record<CropFrame, { icon: ReactNode; label: string }> = {
  full: { icon: <IconFrameFull size={22} />, label: C.frameFull },
  square: { icon: <IconFrameSquare size={22} />, label: C.frameSquare },
  portrait: { icon: <IconFramePortrait size={22} />, label: C.framePortrait },
};

interface Pt {
  x: number;
  y: number;
}

/** أبعادُ الإطار داخل مساحة العرض. «كاملة» = الصورةُ كلُّها محتواةً فيها. */
function frameSize(w: number, h: number, mode: CropFrame, bw: number, bh: number): { fw: number; fh: number } {
  if (w <= 0 || h <= 0 || bw <= 0 || bh <= 0) return { fw: 0, fh: 0 };
  if (mode === "full") {
    const s = Math.min(w / bw, h / bh);
    return { fw: bw * s, fh: bh * s };
  }
  const r = RATIO[mode];
  const fw = Math.min(w, h * r);
  return { fw, fh: fw / r };
}

/** المقياسُ الذي يجعل الصورةَ تغطّي الإطارَ كلَّه. */
function coverScale(w: number, h: number, fw: number, fh: number): number {
  return fw <= 0 || fh <= 0 ? 1 : Math.max(fw / w, fh / h);
}

/** لا فراغَ داخل الإطار — الصورةُ تغطّيه دائماً. */
function clampOffset(w: number, h: number, fw: number, fh: number, zoom: number, o: Pt): Pt {
  const k = coverScale(w, h, fw, fh) * zoom;
  const mx = Math.max(0, (w * k - fw) / 2);
  const my = Math.max(0, (h * k - fh) / 2);
  return { x: Math.min(mx, Math.max(-mx, o.x)), y: Math.min(my, Math.max(-my, o.y)) };
}

/** **يقرأ الملفَّ مصغّراً ومُداراً كما صُوّر** — `createImageBitmap` يحترم ترويسةَ الاتّجاه. */
async function decode(file: Blob): Promise<HTMLCanvasElement> {
  const bmp = await createImageBitmap(file, { imageOrientation: "from-image" });
  const scale = Math.min(1, MAX_EDGE / Math.max(bmp.width, bmp.height));
  const c = document.createElement("canvas");
  c.width = Math.max(1, Math.round(bmp.width * scale));
  c.height = Math.max(1, Math.round(bmp.height * scale));
  c.getContext("2d")?.drawImage(bmp, 0, 0, c.width, c.height);
  bmp.close();
  return c;
}

function rotate90(src: HTMLCanvasElement): HTMLCanvasElement {
  const c = document.createElement("canvas");
  c.width = src.height;
  c.height = src.width;
  const ctx = c.getContext("2d");
  if (!ctx) return src;
  ctx.translate(c.width, 0);
  ctx.rotate(Math.PI / 2);
  ctx.drawImage(src, 0, 0);
  return c;
}

function toJpeg(c: HTMLCanvasElement): Promise<Blob | null> {
  return new Promise((res) => c.toBlob(res, "image/jpeg", JPEG_QUALITY));
}

/** **ما داخل الإطار بإحداثيّات الصورة نفسها** — `cropFrame` حرفاً. */
function cropFrame(src: HTMLCanvasElement, fw: number, fh: number, zoom: number, o: Pt): Promise<Blob | null> {
  const w = src.width;
  const h = src.height;
  const k = coverScale(w, h, fw, fh) * zoom;
  const c = clampOffset(w, h, fw, fh, zoom, o);
  const cw = Math.min(w, Math.max(1, Math.round(fw / k)));
  const ch = Math.min(h, Math.max(1, Math.round(fh / k)));
  const left = ((w * k - fw) / 2 - c.x) / k;
  const top = ((h * k - fh) / 2 - c.y) / k;
  const x = Math.min(w - cw, Math.max(0, Math.round(left)));
  const y = Math.min(h - ch, Math.max(0, Math.round(top)));
  const out = document.createElement("canvas");
  out.width = cw;
  out.height = ch;
  out.getContext("2d")?.drawImage(src, x, y, cw, ch, 0, 0, cw, ch);
  return toJpeg(out);
}

function Tool({
  icon,
  label,
  selected = false,
  disabled = false,
  onClick,
}: {
  icon: ReactNode;
  label: string;
  selected?: boolean;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-pressed={selected || undefined}
      className={`flex min-h-14 min-w-14 flex-col items-center justify-center gap-0.5 rounded-control px-2 py-1.5 text-xs font-medium transition-colors disabled:opacity-50 ${
        selected ? "bg-primary-tint text-primary" : "text-ink-muted hover:bg-field hover:text-ink"
      }`}
    >
      {icon}
      <span>{label}</span>
    </button>
  );
}

export function ImageCropDialog({
  file,
  frames = LOGO_CROP_FRAMES,
  onDone,
  onAgain,
  onCancel,
}: {
  file: Blob;
  /** **الإطاراتُ المتاحة** — أوّلُها الافتراض. */
  frames?: CropFrame[];
  /** **الصورةُ الناتجة** — JPEG جاهزٌ للرفع. */
  onDone: (blob: Blob) => void;
  /** **«صورة أخرى»** — يُغلق ويفتح المنتقي من جديد. */
  onAgain: () => void;
  onCancel: () => void;
}) {
  const first = frames[0] ?? "square";
  const [orig, setOrig] = useState<HTMLCanvasElement | null>(null);
  const [bmp, setBmp] = useState<HTMLCanvasElement | null>(null);
  const [failed, setFailed] = useState(false);
  const [mode, setMode] = useState<CropFrame>(first);
  const [zoom, setZoom] = useState(1);
  const [off, setOff] = useState<Pt>({ x: 0, y: 0 });
  const [box, setBox] = useState({ w: 0, h: 0 });
  const [busy, setBusy] = useState(false);
  const boxRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const pointers = useRef(new Map<number, Pt>());
  const pinch = useRef<{ dist: number; mid: Pt } | null>(null);

  useEffect(() => {
    let live = true;
    setBmp(null);
    setOrig(null);
    setFailed(false);
    decode(file)
      .then((c) => {
        if (!live) return;
        setOrig(c);
        setBmp(c);
      })
      .catch(() => live && setFailed(true));
    return () => {
      live = false;
    };
  }, [file]);

  // **وصورةٌ جديدةٌ أو مُدارةٌ أو إطارٌ آخرُ يبدأ من أوّله** — كـ`remember(bmp, mode)` في التطبيق.
  useEffect(() => {
    setZoom(1);
    setOff({ x: 0, y: 0 });
  }, [bmp, mode]);

  useLayoutEffect(() => {
    const el = boxRef.current;
    if (!el) return;
    const measure = () => setBox({ w: el.clientWidth, h: el.clientHeight });
    measure();
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  }, [bmp]);

  const { fw, fh } = bmp ? frameSize(box.w, box.h, mode, bmp.width, bmp.height) : { fw: 0, fh: 0 };
  const movable = mode !== "full";

  // **الرسم** — الصورةُ بمقياس التغطية × التكبير، ومركزُها مزاحٌ بالسحب، وما خارج الإطار معتَّم.
  useEffect(() => {
    const cv = canvasRef.current;
    if (!cv || !bmp || box.w <= 0 || box.h <= 0) return;
    const { w, h } = box;
    const dpr = window.devicePixelRatio || 1;
    cv.width = Math.round(w * dpr);
    cv.height = Math.round(h * dpr);
    const ctx = cv.getContext("2d");
    if (!ctx) return;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, w, h);
    const k = coverScale(bmp.width, bmp.height, fw, fh) * zoom;
    const dw = bmp.width * k;
    const dh = bmp.height * k;
    ctx.drawImage(bmp, w / 2 + off.x - dw / 2, h / 2 + off.y - dh / 2, dw, dh);
    const fx = (w - fw) / 2;
    const fy = (h - fh) / 2;
    ctx.fillStyle = themeColor("scrim");
    ctx.beginPath();
    ctx.rect(0, 0, w, h);
    ctx.rect(fx, fy, fw, fh);
    ctx.fill("evenodd");
    ctx.strokeStyle = themeColor("on-solid");
    ctx.lineWidth = 2;
    ctx.strokeRect(fx, fy, fw, fh);
  }, [bmp, box, fw, fh, zoom, off]);

  const applyZoom = useCallback(
    (next: number, pan: Pt = { x: 0, y: 0 }) => {
      if (!bmp || !movable) return;
      const z = Math.min(MAX_ZOOM, Math.max(1, next));
      setZoom(z);
      setOff((o) => clampOffset(bmp.width, bmp.height, fw, fh, z, { x: o.x + pan.x, y: o.y + pan.y }));
    },
    [bmp, fw, fh, movable],
  );

  // **العجلةُ تكبّر** — ومستمعٌ غيرُ سلبيٍّ لأنّ الصفحةَ لا يجب أن تتمرّر تحتها.
  useEffect(() => {
    const cv = canvasRef.current;
    if (!cv) return;
    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      applyZoom(zoom * Math.exp(-e.deltaY * 0.0015));
    };
    cv.addEventListener("wheel", onWheel, { passive: false });
    return () => cv.removeEventListener("wheel", onWheel);
  }, [applyZoom, zoom]);

  function onPointerDown(e: PointerEvent<HTMLCanvasElement>) {
    e.currentTarget.setPointerCapture(e.pointerId);
    pointers.current.set(e.pointerId, { x: e.clientX, y: e.clientY });
    pinch.current = null;
  }

  function onPointerMove(e: PointerEvent<HTMLCanvasElement>) {
    const prev = pointers.current.get(e.pointerId);
    if (!prev || !bmp || !movable) return;
    pointers.current.set(e.pointerId, { x: e.clientX, y: e.clientY });
    const pts = [...pointers.current.values()];
    if (pts.length >= 2) {
      const [a, b] = pts as [Pt, Pt];
      const dist = Math.hypot(a.x - b.x, a.y - b.y);
      const mid = { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 };
      const last = pinch.current;
      pinch.current = { dist, mid };
      if (last && last.dist > 0) applyZoom(zoom * (dist / last.dist), { x: mid.x - last.mid.x, y: mid.y - last.mid.y });
      return;
    }
    const pan = { x: e.clientX - prev.x, y: e.clientY - prev.y };
    setOff((o) => clampOffset(bmp.width, bmp.height, fw, fh, zoom, { x: o.x + pan.x, y: o.y + pan.y }));
  }

  function onPointerUp(e: PointerEvent<HTMLCanvasElement>) {
    pointers.current.delete(e.pointerId);
    pinch.current = null;
  }

  function reset() {
    if (orig) setBmp(orig);
    setMode(first);
    setZoom(1);
    setOff({ x: 0, y: 0 });
  }

  async function use() {
    if (!bmp || busy) return;
    setBusy(true);
    const blob = mode === "full" ? await toJpeg(bmp) : await cropFrame(bmp, fw, fh, zoom, off);
    setBusy(false);
    if (blob) onDone(blob);
  }

  return (
    <Modal open onClose={onCancel} title={C.title}>
      <div className="flex flex-col gap-3">
        <div ref={boxRef} dir="ltr" className="relative mx-auto aspect-square w-full overflow-hidden rounded-control bg-shell">
          {bmp && (
            <canvas
              ref={canvasRef}
              className={`h-full w-full touch-none ${movable ? "cursor-grab active:cursor-grabbing" : ""}`}
              onPointerDown={onPointerDown}
              onPointerMove={onPointerMove}
              onPointerUp={onPointerUp}
              onPointerCancel={onPointerUp}
            />
          )}
          {failed && <p className="absolute inset-0 flex items-center justify-center p-4 text-center text-sm text-danger">{m.errors.internal}</p>}
        </div>
        {movable && (
          <label className="flex items-center gap-3 text-sm text-ink-muted">
            <span className="shrink-0">{C.zoom}</span>
            <input
              type="range"
              min={1}
              max={MAX_ZOOM}
              step={0.01}
              value={zoom}
              disabled={!bmp}
              onChange={(e) => applyZoom(Number(e.target.value))}
              className="w-full accent-primary"
              dir="ltr"
            />
          </label>
        )}
        <div className="flex flex-wrap items-center justify-center gap-1 border-y border-line-soft py-2">
          {frames.length > 1 && (
            <>
              {frames.map((f) => (
                <Tool key={f} icon={FRAME_UI[f].icon} label={FRAME_UI[f].label} selected={mode === f} disabled={!bmp} onClick={() => setMode(f)} />
              ))}
              <span className="mx-1 h-9 w-px bg-line" aria-hidden />
            </>
          )}
          <Tool icon={<IconRotate size={22} />} label={C.rotate} disabled={!bmp || busy} onClick={() => bmp && setBmp(rotate90(bmp))} />
          <Tool icon={<IconReset size={22} />} label={C.reset} disabled={!bmp || busy} onClick={reset} />
          <Tool icon={<IconCamera size={22} />} label={C.again} disabled={busy} onClick={onAgain} />
        </div>
        <FormActions
          onSave={() => void use()}
          onCancel={onCancel}
          busy={busy || !bmp}
          saveLabel={C.use}
          saveIcon={<IconCheck size={18} />}
          cancelIcon={<IconClose size={18} />}
        />
      </div>
    </Modal>
  );
}
