"use client";

/** **عرضُ الرئيسيّة** — يقرأ صورةَ الصفحة من الإعدادات ويرسمها. */

import { useEffect, useState } from "react";
import { Hero, type HeroContent } from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";

interface Wire {
  image: string | null;
  image_mobile: string | null;
}

const EMPTY: HeroContent = { imageUrl: null, imageMobileUrl: null };

export default function HeroSection() {
  const [content, setContent] = useState<HeroContent>(EMPTY);

  useEffect(() => {
    api<Wire>("/api/v1/public/hero")
      .then((r) =>
        setContent({
          imageUrl: mediaUrl(r.image) ?? null,
          imageMobileUrl: mediaUrl(r.image_mobile) ?? null,
        }),
      )
      /* @empty-ok — **ورئيسيّةٌ بلا صورةٍ تبقى تعمل**: فشلُ النداء يُبقي
         الصفحةَ فارغةً، ولا رسالةَ خطأٍ في وجه أوّل زائر. */
      .catch(() => undefined);
  }, []);

  return <Hero content={content} />;
}
