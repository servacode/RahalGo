"use client";

/**
 * **أهذا الحسابُ موظّف؟** — مرآةُ `TeamAccess` في تطبيق المتجر.
 *
 * يُعرف من ردّ المحرّك لا من الشاشة: بابُ الموظّفين يردّ الموظّفَ
 * `staff_not_allowed`. **ويُسأل مرّةً في الجلسة** — وتقرؤه شارةُ الدور في
 * «حسابي» والقائمةُ الجانبيّة.
 */

import { useEffect, useState } from "react";
import { api, ApiError, onLogout } from "@/lib/api";

export type TeamRole = "" | "owner" | "staff";

let cached: TeamRole = "";
let inflight: Promise<TeamRole> | null = null;

// **والخروجُ يمحو الجواب** — فمن دخل بعده بحسابٍ آخرَ يُسأل له من جديد.
if (typeof window !== "undefined") {
  onLogout(() => {
    cached = "";
  });
}

function probe(): Promise<TeamRole> {
  if (cached) return Promise.resolve(cached);
  if (inflight) return inflight;
  const p: Promise<TeamRole> = api("/api/v1/merchant/staff")
    .then((): TeamRole => (cached = "owner"))
    .catch((e: unknown): TeamRole => {
      if (e instanceof ApiError && e.body.code === "staff_not_allowed") cached = "staff";
      return cached;
    })
    .finally(() => {
      inflight = null;
    });
  inflight = p;
  return p;
}

/** **`""` قبل أن يُعرف الجواب** — فلا تُرسم الشارةُ ولا يُخفى بندٌ سلفاً. */
export function useTeamRole(): TeamRole {
  const [role, setRole] = useState<TeamRole>(cached);
  useEffect(() => {
    let live = true;
    void probe().then((r) => {
      if (live) setRole(r);
    });
    return () => {
      live = false;
    };
  }, []);
  return role;
}
