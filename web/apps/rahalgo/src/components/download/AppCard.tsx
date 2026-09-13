/**
 * ══════════════════════════════════════════════════════════════════════
 * **بطاقةُ تطبيقٍ — وصفحةُ تطبيقٍ** (`DLC`، ٢٠٢٦-٠٩-١٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ومكوّنٌ واحدٌ لأربعةِ تطبيقاتٍ** — **وأربعُ نسخٍ تفترق في الرابعة.**
 *
 * # وحالٌ مُسمّاةٌ لا استنتاجٌ من نصّ
 *
 * **والزرُّ يظهر بالحال** (`status`) **لا بفحصِ «هل النصُّ غيرُ فارغ؟»**
 * — **ومن فحص النصَّ خلط الغيابَ بالعطب.**
 *
 * # ولا زرَّ ميّتاً يبدو حيّاً
 *
 * **وغيرُ المتوفّر يُقال بصراحةٍ** — **وزرٌّ رماديٌّ يُضغط ولا يفعل
 * يُقرأ عطباً في المنصّة.**
 */

import Link from "next/link";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Badge, ButtonLink } from "@rahalgo/ui";
import type { Release } from "@/lib/releases";
import { megabytes } from "@/lib/releases";

const m = getMessages(defaultLocale);
const D = m.site.download;

/** stateText **الحالُ بلفظها** — ولا لفظَ يُخترَع في الصفحة. */
export function stateText(r: Release): string {
  switch (r.status) {
    case "play":
      return D.state.play;
    case "direct":
      return D.state.direct;
    case "play_and_direct":
      return D.state.playAndDirect;
    default:
      return D.state.unavailable;
  }
}

/** appText نصوصُ تطبيقٍ من المعجم المركزيّ. */
export function appText(key: Release["key"]) {
  return D.apps[key];
}

export function AppCard({ release }: { release: Release }) {
  const t = appText(release.key);
  const live = release.status !== "unavailable";
  const size = megabytes(release.size_bytes);
  return (
    <article className="surface flex flex-col gap-3 p-5 elev-1">
      <div className="flex items-start justify-between gap-3">
        <h2 className="heading-card text-ink">{t.name}</h2>
        <Badge variant={live ? "success" : "neutral"}>{stateText(release)}</Badge>
      </div>
      <p className="text-sm leading-relaxed text-ink-muted">{t.short}</p>
      <dl className="text-xs text-ink-muted">
        <div className="flex gap-2">
          <dt>{D.platform}</dt>
          {release.version ? (
            <>
              <dd aria-hidden="true">·</dd>
              <dt>{D.versionLabel}</dt>
              <dd>{release.version}</dd>
            </>
          ) : null}
          {size ? (
            <>
              <dd aria-hidden="true">·</dd>
              <dd>
                {size} {D.sizeUnit}
              </dd>
            </>
          ) : null}
        </div>
      </dl>
      <div className="mt-auto pt-2">
        <ButtonLink href={`/download/${release.key}`} variant="secondary">
          {D.open}
        </ButtonLink>
      </div>
    </article>
  );
}

/**
 * AppPage **صفحةُ تطبيقٍ بعينه** — **وواحدةٌ للأربعة.**
 *
 * **والمعروضُ يصف الملفَّ الذي يُنزَّل بعينه**: النسخةُ والحجمُ والبصمةُ
 * — **والبصمةُ يحسبها المحرّكُ من الملفّ لا تُكتب بيد.**
 */
export function AppPage({
  release,
  downloadUrl,
}: {
  release: Release;
  downloadUrl: string;
}) {
  const t = appText(release.key);
  const size = megabytes(release.size_bytes);
  const showPlay = release.play_url !== "";
  const showDirect = downloadUrl !== "";
  return (
    <main className="mx-auto max-w-2xl px-4 py-10">
      <Link href="/download" className="text-sm text-muted underline">
        {D.back}
      </Link>

      <div className="mt-4 flex items-start justify-between gap-3">
        <h1 className="heading-page">{t.name}</h1>
        <Badge variant={release.status === "unavailable" ? "neutral" : "success"}>
          {stateText(release)}
        </Badge>
      </div>

      <p className="mt-3 leading-relaxed text-ink-muted">{t.long}</p>

      <div className="mt-6 flex flex-col gap-3">
        {showPlay ? (
          <ButtonLink href={release.play_url}>{D.playCta}</ButtonLink>
        ) : release.channel === "play" ? (
          /* **ولا زرَّ متجرٍ يُخترَع** — (قرارُ المالك ٢٠٢٦-٠٩-١٣). */
          <p className="text-sm text-ink-muted">{D.soonPlay}</p>
        ) : null}

        {showDirect ? (
          <>
            <ButtonLink href={downloadUrl} variant={showPlay ? "secondary" : "primary"}>
              {D.directCta}
            </ButtonLink>
            <p className="text-sm text-ink-muted">{D.officialNote}</p>
            <p className="text-xs text-ink-muted">{D.installNote}</p>
          </>
        ) : release.channel === "direct" ? (
          <p className="text-sm text-ink-muted">{D.soonDirect}</p>
        ) : null}
      </div>

      {/* **والتفاصيلُ التقنيّةُ مطويّةٌ** — **ولا تُزحم تجربةَ من يريد
          زرّاً واحداً.** */}
      {release.package_id || release.version || size || release.sha256 ? (
        <details className="mt-8 text-sm">
          <summary className="cursor-pointer text-muted">{D.details}</summary>
          <dl className="mt-3 grid gap-2">
            {release.package_id ? (
              <div className="flex flex-wrap gap-2">
                <dt className="text-ink-muted">{D.packageLabel}</dt>
                <dd className="font-mono text-xs" dir="ltr">
                  {release.package_id}
                </dd>
              </div>
            ) : null}
            {release.version ? (
              <div className="flex flex-wrap gap-2">
                <dt className="text-ink-muted">{D.versionLabel}</dt>
                <dd className="font-mono text-xs" dir="ltr">
                  {release.version}
                </dd>
              </div>
            ) : null}
            {size ? (
              <div className="flex flex-wrap gap-2">
                <dt className="text-ink-muted">{D.sizeLabel}</dt>
                <dd>
                  {size} {D.sizeUnit}
                </dd>
              </div>
            ) : null}
            {release.sha256 ? (
              <div className="flex flex-col gap-1">
                <dt className="text-ink-muted">{D.hashLabel}</dt>
                <dd className="break-all font-mono text-xs" dir="ltr">
                  {release.sha256}
                </dd>
                <dd className="text-xs text-ink-muted">{D.hashHint}</dd>
              </div>
            ) : null}
          </dl>
        </details>
      ) : null}
    </main>
  );
}
