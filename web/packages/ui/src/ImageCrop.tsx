"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **معاينةُ الصورة وقصُّها قبل الرفع** — مرآةُ `ImageCrop.kt` (طلبُ المالك ٢٠٢٦-١٠-٢٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **الزبونُ يرى صورةَ الصنف والشعارَ مربّعين** — فالإطارُ هنا مربّع: ما يبين
 * داخله هو بالضبط ما يصل السوق. **يكبّر بإصبعين أو بعجلة الفأرة أو بالمنزلقة،
 * ويسحب، ويدوّر ربعَ دورة** — ولا يُرفع شيءٌ قبل «اعتمد».
 *
 * **والحسابُ نفسُه في التطبيق** (`coverScale` · `clampOffset` · `cropSquare`):
 * الصورةُ تغطّي الإطارَ دائماً فلا فراغَ فيه، **والناتجُ مربّعٌ JPEG بجودة ٨٥
 * وأطولُ ضلعٍ ١٦٠٠** كـ`readScaledImage`.
 *
 * **والإحداثيّاتُ فيزيائيّةٌ (يسار←يمين)** بغضّ النظر عن اتّجاه الصفحة —
 * وإلّا انعكس السحبُ بالعربيّ.
 */

import { useCallback, useEffect, useLayoutEffect, useRef, useState, type PointerEvent } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, Modal } from "./components";
import { FormActions } from "./FormActions";

const m = getMessages(defaultLocale);
const C = m.common.media.crop;

/** **أطولُ ضلعٍ يُعمل عليه** — `MAX_EDGE` في `ImagePick.kt`. */
const MAX_EDGE = 1600;
const JPEG_QUALITY = 0.85;
const MAX_ZOOM = 6;

interface Pt {
  x: number;
  y: number;
}

/** المقياسُ الذي يجعل الصورةَ تغطّي الإطارَ كلَّه. */
function coverScale(w: number, h: number, side: number): number {
  return side <= 0 ? 1 : side / Math.min(w, h);
}

/** لا فراغَ داخل الإطار — الصورةُ تغطّيه دائماً. */
function clampOffset(w: number, h: number, side: number, zoom: number, o: Pt): Pt {
  const k = coverScale(w, h, side) * zoom;
  const mx = Math.max(0, (w * k - side) / 2);
  const my = Math.max(0, (h * k - side) / 2);
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

/** **ما داخل الإطار بإحداثيّات الصورة نفسها** — `cropSquare` حرفاً. */
function cropSquare(src: HTMLCanvasElement, side: number, zoom: number, o: Pt): Promise<Blob | null> {
  const w = src.width;
  const h = src.height;
  const k = coverScale(w, h, side) * zoom;
  const c = clampOffset(w, h, side, zoom, o);
  const n = Math.min(Math.min(w, h), Math.max(1, Math.round(side / k)));
  const left = ((w * k - side) / 2 - c.x) / k;
  const top = ((h * k - side) / 2 - c.y) / k;
  const x = Math.min(w - n, Math.max(0, Math.round(left)));
  const y = Math.min(h - n, Math.max(0, Math.round(top)));
  const out = document.createElement("canvas");
  out.width = n;
  out.height = n;
  out.getContext("2d")?.drawImage(src, x, y, n, n, 0, 0, n, n);
  return new Promise((res) => out.toBlob(res, "image/jpeg", JPEG_QUALITY));
}

export function ImageCropDialog({
  file,
  onDone,
  onAgain,
  onCancel,
}: {
  file: Blob;
  /** **المربّعُ المقصوص** — JPEG جاهزٌ للرفع. */
  onDone: (blob: Blob) => void;
  /** **«صورة تانية»** — يُغلق ويفتح المنتقي من جديد. */
  onAgain: () => void;
  onCancel: () => void;
}) {
  const [bmp, setBmp] = useState<HTMLCanvasElement | null>(null);
  const [failed, setFailed] = useState(false);
  const [zoom, setZoom] = useState(1);
  const [off, setOff] = useState<Pt>({ x: 0, y: 0 });
  const [side, setSide] = useState(0);
  const [busy, setBusy] = useState(false);
  const boxRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const pointers = useRef(new Map<number, Pt>());
  const pinch = useRef<{ dist: number; mid: Pt } | null>(null);

  useEffect(() => {
    let live = true;
    setBmp(null);
    setFailed(false);
    decode(file)
      .then((c) => live && setBmp(c))
      .catch(() => live && setFailed(true));
    return () => {
      live = false;
    };
  }, [file]);

  // **وصورةٌ جديدةٌ أو مُدارةٌ تبدأ من أوّلها** — كـ`remember(bmp)` في التطبيق.
  useEffect(() => {
    setZoom(1);
    setOff({ x: 0, y: 0 });
  }, [bmp]);

  useLayoutEffect(() => {
    const el = boxRef.current;
    if (!el) return;
    const measure = () => setSide(el.clientWidth);
    measure();
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  }, [bmp]);

  // **الرسم** — الصورةُ بمقياس التغطية × التكبير، ومركزُها مزاحٌ بالسحب.
  useEffect(() => {
    const cv = canvasRef.current;
    if (!cv || !bmp || side <= 0) return;
    const dpr = window.devicePixelRatio || 1;
    cv.width = Math.round(side * dpr);
    cv.height = Math.round(side * dpr);
    const ctx = cv.getContext("2d");
    if (!ctx) return;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, side, side);
    const k = coverScale(bmp.width, bmp.height, side) * zoom;
    const dw = bmp.width * k;
    const dh = bmp.height * k;
    ctx.drawImage(bmp, side / 2 + off.x - dw / 2, side / 2 + off.y - dh / 2, dw, dh);
  }, [bmp, side, zoom, off]);

  const applyZoom = useCallback(
    (next: number, pan: Pt = { x: 0, y: 0 }) => {
      if (!bmp) return;
      const z = Math.min(MAX_ZOOM, Math.max(1, next));
      setZoom(z);
      setOff((o) => clampOffset(bmp.width, bmp.height, side, z, { x: o.x + pan.x, y: o.y + pan.y }));
    },
    [bmp, side],
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
    if (!prev || !bmp) return;
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
    setOff((o) => clampOffset(bmp.width, bmp.height, side, zoom, { x: o.x + pan.x, y: o.y + pan.y }));
  }

  function onPointerUp(e: PointerEvent<HTMLCanvasElement>) {
    pointers.current.delete(e.pointerId);
    pinch.current = null;
  }

  async function use() {
    if (!bmp || busy) return;
    setBusy(true);
    const blob = await cropSquare(bmp, side, zoom, off);
    setBusy(false);
    if (blob) onDone(blob);
  }

  return (
    <Modal open onClose={onCancel} title={C.title}>
      <div className="flex flex-col gap-3">
        <p className="text-center text-sm text-ink-muted">{C.hint}</p>
        <div ref={boxRef} dir="ltr" className="relative mx-auto aspect-square w-full overflow-hidden rounded-control bg-field">
          {bmp && (
            <canvas
              ref={canvasRef}
              className="h-full w-full cursor-grab touch-none active:cursor-grabbing"
              onPointerDown={onPointerDown}
              onPointerMove={onPointerMove}
              onPointerUp={onPointerUp}
              onPointerCancel={onPointerUp}
            />
          )}
          {failed && <p className="absolute inset-0 flex items-center justify-center p-4 text-center text-sm text-danger">{m.errors.internal}</p>}
          <span className="pointer-events-none absolute inset-0 rounded-control border-2 border-primary" aria-hidden />
        </div>
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
        <div className="flex gap-2">
          <Button type="button" variant="secondary" className="flex-1" disabled={!bmp || busy} onClick={() => bmp && setBmp(rotate90(bmp))}>
            {C.rotate}
          </Button>
          <Button type="button" variant="ghost" className="flex-1" disabled={busy} onClick={onAgain}>
            {C.again}
          </Button>
        </div>
        <FormActions onSave={() => void use()} onCancel={onCancel} busy={busy || !bmp} saveLabel={C.use} />
      </div>
    </Modal>
  );
}
