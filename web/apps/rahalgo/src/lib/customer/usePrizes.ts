"use client";

/**
 * **حالُ جوائز الزبون** — نداءٌ واحدٌ لصاحب حسابٍ وحدَه، وفشلُه صمتٌ (الشارةُ لا تُظهر خطأً).
 * والمنطقُ في `prizes.ts`؛ هذا ربطٌ بالشبكة فقط.
 */

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import type { PrizesMe } from "./types";

export function usePrizesMe(): { me: PrizesMe | null; error: boolean; loading: boolean; reload: () => void } {
  const { user } = useAuth();
  const [me, setMe] = useState<PrizesMe | null>(null);
  const [error, setError] = useState(false);
  const [loading, setLoading] = useState(false);
  const [tick, setTick] = useState(0);
  const reload = useCallback(() => setTick((n) => n + 1), []);

  useEffect(() => {
    if (!user) {
      setMe(null);
      return;
    }
    let alive = true;
    setLoading(true);
    setError(false);
    api<PrizesMe>("/api/v1/prizes/me")
      .then((x) => alive && setMe(x ?? {}))
      .catch(() => alive && setError(true))
      .finally(() => alive && setLoading(false));
    return () => {
      alive = false;
    };
  }, [user, tick]);

  return { me, error, loading, reload };
}
