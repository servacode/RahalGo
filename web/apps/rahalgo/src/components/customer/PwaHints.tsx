"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«أضف للشاشة الرئيسية» — لزبون آيفون، ومرّةً واحدة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-١٧: «زبونُ آيفون ما عنده تطبيق — يطلب من المتصفّح
 *  ويضيفه للشاشة فيفتح كأنّه تطبيق».)
 *
 * **سفاري لا يعرض زرَّ تثبيتٍ بنفسه** — ولا واجهةَ برمجيّةً تفتحه.
 * **فيُقال للزبون أين يضغط**: زرُّ المشاركة ثمّ «إضافة إلى الشاشة
 * الرئيسية». **ويُرسم زرُّ المشاركة كما يراه** فيُعرف بلا قراءة.
 *
 * **ومرّةً واحدة** — من أغلقها لا تعود (تخزينُ المتصفّح، محميّ). **ومن
 * فتح الموقعَ من الشاشة الرئيسيّة لا يراها أصلاً** (`standalone`).
 *
 * **وأندرويد له تطبيقُه** — فيُقترح عليه بلافتةٍ صغيرةٍ تُغلق.
 */

import { useEffect, useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, withPlatform } from "@rahalgo/i18n";
import {
  Button,
  Modal,
  IconShare,
  IconAddToHome,
  IconSmartphone,
  IconClose,
  usePlatform,
} from "@rahalgo/ui";
import { readJSON, writeJSON } from "@/lib/customer/storage";

const m = getMessages(defaultLocale);
const P = m.shop.pwa;

const IOS_SEEN = "rahalgo_pwa_ios_seen";
const ANDROID_OFF = "rahalgo_pwa_android_off";

type Kind = "" | "ios" | "android";

function detect(): Kind {
  if (typeof window === "undefined") return "";
  const nav = window.navigator as Navigator & { standalone?: boolean };
  const standalone =
    nav.standalone === true || window.matchMedia?.("(display-mode: standalone)").matches;
  if (standalone) return "";
  const ua = nav.userAgent || "";
  // **وآيبادُ اليوم يقول عن نفسه «ماك»** — يُعرف باللمس.
  const ios = /iPhone|iPad|iPod/i.test(ua) || (nav.platform === "MacIntel" && nav.maxTouchPoints > 1);
  if (ios) return "ios";
  if (/Android/i.test(ua)) return "android";
  return "";
}

/**
 * **ولوحةُ المتجر تستعملها أيضاً** (قرارُ المالك ٢٠٢٦-١٠-١٧) — **بلافتةٍ
 * تقترح تطبيقَ المتجر لا تطبيقَ الزبون**، ومفتاحِ إغلاقٍ خاصٍّ بها: من أغلق
 * لافتةَ الزبون لم يُغلق لافتةَ متجره.
 */
export interface AndroidHint {
  href: string;
  title: string;
  body: string;
  cta: string;
  offKey: string;
}

export default function PwaHints({ android }: { android?: AndroidHint } = {}) {
  const { name } = usePlatform();
  const [kind, setKind] = useState<Kind>("");
  const offKey = android?.offKey ?? ANDROID_OFF;

  useEffect(() => {
    const k = detect();
    if (k === "ios" && !readJSON<boolean>(IOS_SEEN, false)) setKind("ios");
    if (k === "android" && !readJSON<boolean>(offKey, false)) setKind("android");
  }, [offKey]);

  if (kind === "ios") {
    const close = () => {
      writeJSON(IOS_SEEN, true);
      setKind("");
    };
    return (
      <Modal open onClose={close} title={withPlatform(P.iosTitle, name)}>
        <p className="mb-3 text-sm text-ink-muted">{P.iosLead}</p>
        <ol className="space-y-3 text-sm">
          <li className="flex items-center gap-3">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-control bg-field text-accent-text">
              <IconShare size={18} />
            </span>
            {P.iosStep1}
          </li>
          <li className="flex items-center gap-3">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-control bg-field text-accent-text">
              <IconAddToHome size={18} />
            </span>
            {P.iosStep2}
          </li>
          <li className="flex items-center gap-3">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-control bg-field text-accent-text">
              <IconSmartphone size={18} />
            </span>
            {P.iosStep3}
          </li>
        </ol>
        <Button className="mt-5 w-full py-3" onClick={close}>
          {P.iosOk}
        </Button>
      </Modal>
    );
  }

  if (kind === "android") {
    const off = () => {
      writeJSON(offKey, true);
      setKind("");
    };
    return (
      <div className="surface flex items-center gap-3 p-3">
        <IconSmartphone size={22} className="shrink-0 text-accent-text" />
        <div className="min-w-0 flex-1">
          <p className="text-sm font-bold">{android?.title ?? P.androidTitle}</p>
          <p className="text-xs text-ink-muted">{android?.body ?? P.androidBody}</p>
        </div>
        <Link
          href={android?.href ?? "/download/customer"}
          /* **و`/get/…` تنزيلٌ لا صفحة** — لا يُجلب مسبقاً فيُعدّ تنزيلاً لم يقع. */
          prefetch={android ? false : undefined}
          className="shrink-0 rounded-control bg-accent px-3 py-2 text-xs font-bold text-on-bright"
        >
          {android?.cta ?? P.androidCta}
        </Link>
        <button
          type="button"
          onClick={off}
          aria-label={P.close}
          className="taparea flex h-8 w-8 shrink-0 items-center justify-center rounded-control text-ink-muted hover:bg-row-hover"
        >
          <IconClose size={16} />
        </button>
      </div>
    );
  }

  return null;
}
