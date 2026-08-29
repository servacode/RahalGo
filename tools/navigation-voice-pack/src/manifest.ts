/**
 * ══════════════════════════════════════════════════════════════════════
 * **الفهرس — سجلٌّ لكلّ ملفّ، ولا مفتاحَ فيه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مواصفةُ المالك ٢٠٢٦-٠٨-٢٣، البند ١٨.)
 *
 * **وهو الذي يُقرأ في التطبيق لاحقاً** — فيربط مناورةَ OSRM بملفّ
 * صوت. **ولذلك يحمل `maneuverType` و`maneuverModifier`** ولو لم
 * نستعملهما اليوم.
 *
 * **ويحمل البصمةَ** — فيُعرف في التشغيل التالي ما يُتخطّى.
 */

import type { Instruction } from './corpus.js';
import type { VoiceSettings } from './config.js';

export interface ManifestEntry {
  readonly id: string;
  readonly category: string;
  readonly displayText: string;
  readonly ttsText: string;
  readonly file: string;
  readonly distanceMeters: number | null;
  readonly roundaboutExit: number | null;
  readonly maneuverType: string | null;
  readonly maneuverModifier: string | null;
  readonly voiceId: string;
  readonly modelId: string;
  readonly outputFormat: string;
  readonly voiceSettings: VoiceSettings;
  readonly sourceHash: string;
  readonly generatedAt: string;
  readonly characterCount: number;
}

export interface Manifest {
  readonly schemaVersion: number;
  readonly pack: string;
  readonly language: string;
  readonly entries: readonly ManifestEntry[];
}

export const SCHEMA_VERSION = 1;

export function entryFor(opts: {
  readonly instruction: Instruction;
  readonly voiceId: string;
  readonly modelId: string;
  readonly outputFormat: string;
  readonly voiceSettings: VoiceSettings;
  readonly sourceHash: string;
  readonly generatedAt: string;
}): ManifestEntry {
  const i = opts.instruction;
  return {
    id: i.id,
    category: i.category,
    displayText: i.displayText,
    ttsText: i.ttsText,
    file: i.file,
    distanceMeters: i.distanceMeters ?? null,
    roundaboutExit: i.roundaboutExit ?? null,
    maneuverType: i.maneuverType,
    maneuverModifier: i.maneuverModifier,
    voiceId: opts.voiceId,
    modelId: opts.modelId,
    outputFormat: opts.outputFormat,
    voiceSettings: opts.voiceSettings,
    sourceHash: opts.sourceHash,
    generatedAt: opts.generatedAt,
    characterCount: [...i.ttsText].length,
  };
}

/**
 * **وجدولٌ للمراجعة البشريّة** — البند ١٨.
 *
 * **والفاصلةُ تُهرَّب بالاقتباس** — **ونصُّ «عِند الدَّوّار، خُذ…» فيه
 * فاصلةٌ تكسر كلَّ عمودٍ بعدها.**
 */
export function toCsv(m: Manifest): string {
  const cols = [
    'id',
    'category',
    'file',
    'distanceMeters',
    'roundaboutExit',
    'maneuverType',
    'maneuverModifier',
    'characterCount',
    'displayText',
    'ttsText',
  ] as const;
  const esc = (v: unknown): string => {
    const s = v === null || v === undefined ? '' : String(v);
    return `"${s.replace(/"/g, '""')}"`;
  };
  const lines = [cols.join(',')];
  for (const e of m.entries) {
    lines.push(cols.map((c) => esc((e as unknown as Record<string, unknown>)[c])).join(','));
  }
  // **وعلامةُ ترتيبِ البايتات في أوّله** — **وإكسل يقرأ العربيّةَ
  // حروفاً مقطّعةً بلاها.**
  return `﻿${lines.join('\n')}\n`;
}

export class ManifestError extends Error {}

export function validateManifest(m: Manifest): void {
  if (m.schemaVersion !== SCHEMA_VERSION) throw new ManifestError('نسخةُ صيغةٍ غيرُ مدعومة');
  const ids = new Set<string>();
  const files = new Set<string>();
  for (const e of m.entries) {
    if (ids.has(e.id)) throw new ManifestError(`معرّفٌ مكرّر: ${e.id}`);
    if (files.has(e.file)) throw new ManifestError(`ملفٌّ مكرّر: ${e.file}`);
    ids.add(e.id);
    files.add(e.file);
    if (e.sourceHash.length !== 64) throw new ManifestError(`بصمةٌ غيرُ صالحة: ${e.id}`);
    if (e.characterCount <= 0) throw new ManifestError(`عدُّ أحرفٍ غيرُ صالح: ${e.id}`);
  }
  // **ولا مفتاحَ في الفهرس** — البند ١٨، ويُفحص لا يُفترض.
  const raw = JSON.stringify(m);
  if (/sk_[A-Za-z0-9]{8,}/.test(raw)) throw new ManifestError('مفتاحٌ تسرّب إلى الفهرس');
}
