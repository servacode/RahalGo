"use client";

/** **هدفي ومكافآتي** — الشاشةُ مشتركةٌ والمسارُ يختلف بالدور. */

import { MyIncentives } from "@rahalgo/ui";
import { api } from "@/lib/api";

export default function IncentivesPage() {
  return <MyIncentives api={api} path="/api/v1/driver/incentives" />;
}
