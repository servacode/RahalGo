"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الرسومُ البيانيّة للوحة المتجر — بلا مكتبة رسم**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (التصميمُ المعتمد ٢٠٢٦-١١-٠٨: رسمٌ مصغّرٌ في بطاقة المؤشّر، ورسمُ مبيعاتٍ بخطّ الفترة السابقة المتقطّع وتلميحٍ عائم.)
 *
 * **ملفٌّ مستثنى في حارس المركزيّة بالاسم** (`SOURCES.icons`) كالأيقونات: الرسمُ هنا مولَّدٌ من أرقامٍ لا
 * أيقونةٌ تُرسم باليد. **والألوانُ بأصنافِ التوكنز** (`stroke-primary` · `fill-surface`) لا بقيمٍ خام،
 * **والزمنُ يمضي من اليمين إلى اليسار كالعربيّة** — فاليومُ الأخيرُ على اليسار.
 */

import { useEffect, useId, useRef, useState } from "react";

/** عنصرٌ بفهرسٍ مضمونٍ (الفحصُ الصارمُ للفهارس لا يعرف أنّ الطولَ مفحوص). */
const at = <T,>(a: readonly T[], i: number): T => a[i] as T;

/** مسارٌ ناعمٌ بين نقاطٍ (انحناءٌ أفقيٌّ كما في التصميم). */
function smooth(pts: [number, number][]): string {
  if (pts.length === 0) return "";
  let d = `M${at(pts, 0)[0]},${at(pts, 0)[1]}`;
  for (let i = 0; i < pts.length - 1; i++) {
    const [x0, y0] = at(pts, i);
    const [x1, y1] = at(pts, i + 1);
    const cx = (x0 + x1) / 2;
    d += ` C${cx},${y0} ${cx},${y1} ${x1},${y1}`;
  }
  return d;
}

/** **رسمٌ مصغّرٌ** — خطٌّ ومساحةٌ باهتةٌ ونقطةٌ للقيمة الأخيرة. اللونُ من `currentColor` (يُحدَّد بصنف نصٍّ). */
export function Sparkline({
  data,
  width = 96,
  height = 34,
  className = "text-primary",
}: {
  /** من الأقدم إلى الأحدث. */
  data: readonly number[];
  width?: number;
  height?: number;
  className?: string;
}) {
  const id = useId();
  if (data.length < 2) return null;
  const mn = Math.min(...data);
  const mx = Math.max(...data);
  const pad = 3;
  const step = (width - pad * 2) / (data.length - 1);
  const pts: [number, number][] = data.map((v, i) => [
    width - pad - i * step,
    height - pad - ((v - mn) / (mx - mn || 1)) * (height - pad * 2),
  ]);
  const line = smooth(pts);
  const last = at(pts, pts.length - 1);
  const area = `${line} L${last[0]},${height} L${at(pts, 0)[0]},${height}Z`;
  return (
    <svg className={className} width={width} height={height} viewBox={`0 0 ${width} ${height}`} aria-hidden="true">
      <defs>
        <linearGradient id={id} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="currentColor" stopOpacity=".22" />
          <stop offset="1" stopColor="currentColor" stopOpacity="0" />
        </linearGradient>
      </defs>
      <path d={area} fill={`url(#${id})`} />
      <path d={line} fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
      <circle cx={last[0]} cy={last[1]} r="2.2" fill="currentColor" />
    </svg>
  );
}

/** حدٌّ أعلى «لطيف» للمحور: ١ · ٢ · ٢٫٥ · ٥ · ١٠ × قوّةُ عشرة، بخمس خانات. */
function niceMax(v: number): number {
  if (v <= 0) return 5000;
  const raw = v / 5;
  const pow = 10 ** Math.floor(Math.log10(raw));
  const f = raw / pow;
  const nice = f <= 1 ? 1 : f <= 2 ? 2 : f <= 2.5 ? 2.5 : f <= 5 ? 5 : 10;
  return nice * pow * 5;
}

export interface SalesPoint {
  /** تسميةُ المحور (اسمُ اليوم أو تاريخُه). */
  label: string;
  value: number;
  /** سطرٌ أوّلُ في التلميح (اليومُ كاملاً). */
  title: string;
  /** سطرٌ ثانٍ في التلميح (المبلغُ وعدد الطلبات). */
  detail: string;
}

/**
 * **رسمُ المبيعات** — خطٌّ ومساحةٌ ونقاطٌ، وخطٌّ متقطّعٌ للفترة السابقة، وتلميحٌ يتبع المؤشّر (أو التركيز بلوحة المفاتيح).
 * `prev` اختياريّ — يُحذف الخطُّ إن لم تتوفّر بياناتُه.
 */
export function SalesChart({
  points,
  prev,
  tickLabel,
  ariaLabel,
}: {
  /** من الأقدم إلى الأحدث. */
  points: readonly SalesPoint[];
  prev?: readonly number[];
  tickLabel: (v: number) => string;
  ariaLabel: string;
}) {
  const gid = useId();
  const [hover, setHover] = useState<number | null>(null);
  /** عرضُ الحاوية الفعليّ: على الحاسوب يُرسم بمقياس ١:١ فلا يتضخّم الخطّ، وجوّالاً يبقى المقياسُ القديم. */
  const box = useRef<HTMLDivElement>(null);
  const [wide, setWide] = useState(0);
  useEffect(() => {
    const el = box.current;
    if (!el || typeof ResizeObserver === "undefined") return;
    const ro = new ResizeObserver(() => setWide(Math.round(el.clientWidth)));
    ro.observe(el);
    return () => ro.disconnect();
  }, []);
  const n = points.length;
  if (n < 2) return null;
  const W = wide >= 600 ? wide : 730;
  const H = 214;
  const pl = 6;
  const pr = 52;
  const pt = 12;
  const pb = 28;
  const iw = W - pl - pr;
  const ih = H - pt - pb;
  const max = niceMax(Math.max(...points.map((p) => p.value), ...(prev ?? [0])));
  const x = (i: number) => W - pr - 22 - i * ((iw - 44) / (n - 1));
  const y = (v: number) => pt + ih - (v / max) * ih;
  const P = (a: readonly number[]): [number, number][] => a.map((v, i) => [x(i), y(v)]);
  const cur = P(points.map((p) => p.value));
  const line = smooth(cur);
  const area = `${line} L${at(cur, n - 1)[0]},${pt + ih} L${at(cur, 0)[0]},${pt + ih}Z`;
  const prevLine = prev && prev.length === n ? smooth(P(prev)) : null;
  const hi = hover ?? n - 1;
  const hx = x(hi);
  const hy = at(cur, hi)[1];
  const ticks = [0, 1, 2, 3, 4, 5].map((k) => (max / 5) * k);
  /** تسميةٌ كلَّ بضع نقاطٍ كي لا تتزاحم في ٣٠ و٩٠ يوماً. */
  const every = Math.max(1, Math.ceil(n / 8));
  return (
    <div className="relative" ref={box}>
      <svg
        viewBox={`0 0 ${W} ${H}`}
        width="100%"
        role="img"
        aria-label={ariaLabel}
        className="block text-primary"
        onPointerLeave={() => setHover(null)}
      >
        <defs>
          <linearGradient id={gid} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stopColor="currentColor" stopOpacity=".24" />
            <stop offset="1" stopColor="currentColor" stopOpacity="0" />
          </linearGradient>
        </defs>
        {ticks.map((t, k) => (
          <g key={k}>
            <line
              x1={pl}
              x2={W - pr + 4}
              y1={y(t)}
              y2={y(t)}
              className="stroke-line-soft"
              strokeWidth="1"
              strokeDasharray={t === 0 ? undefined : "3 4"}
            />
            <text x={W - pr + 14} y={y(t) + 4} className="fill-ink-dim" fontSize="12">
              {tickLabel(t)}
            </text>
          </g>
        ))}
        <line x1={hx} x2={hx} y1={pt} y2={pt + ih} className="stroke-line" strokeDasharray="3 3" />
        {prevLine && (
          <path d={prevLine} fill="none" className="stroke-line" strokeWidth="1.6" strokeDasharray="4 4" strokeLinecap="round" />
        )}
        <path d={area} fill={`url(#${gid})`} />
        <path d={line} fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" />
        {n <= 14 &&
          cur.map((p, i) =>
            i === hi ? null : <circle key={i} cx={p[0]} cy={p[1]} r="3" className="fill-surface" stroke="currentColor" strokeWidth="1.6" />,
          )}
        <circle cx={hx} cy={hy} r="9" fill="currentColor" opacity=".14" />
        <circle cx={hx} cy={hy} r="4.2" className="fill-surface" stroke="currentColor" strokeWidth="2.2" />
        {points.map((p, i) =>
          (n - 1 - i) % every === 0 || i === n - 1 ? (
            <text
              key={i}
              x={x(i)}
              y={H - 7}
              textAnchor="middle"
              fontSize="12"
              className={i === n - 1 ? "fill-ink" : "fill-ink-dim"}
              fontWeight={i === n - 1 ? 700 : 500}
            >
              {p.label}
            </text>
          ) : null,
        )}
        {points.map((p, i) => (
          <rect
            key={`h${i}`}
            x={x(i) - (iw - 44) / (n - 1) / 2}
            y={0}
            width={(iw - 44) / (n - 1)}
            height={H}
            fill="transparent"
            tabIndex={0}
            aria-label={`${p.title} — ${p.detail}`}
            onPointerEnter={() => setHover(i)}
            onFocus={() => setHover(i)}
            onBlur={() => setHover(null)}
          />
        ))}
      </svg>
      <div className="chart-tip" style={{ left: `clamp(5rem, ${(hx / W) * 100}%, calc(100% - 5rem))`, top: `${(hy / H) * 100 - 9}%` }}>
        <b className="block text-sm text-ink">{at(points, hi).title}</b>
        {at(points, hi).detail}
      </div>
    </div>
  );
}
