"use client";

/** **دردشاتي السابقة** — المكوّنُ المركزيّ المشترك (@rahalgo/ui). */

import { ChatArchive } from "@rahalgo/ui";
import { api } from "@/lib/api";

export default function Page() {
  return <ChatArchive api={api} />;
}
