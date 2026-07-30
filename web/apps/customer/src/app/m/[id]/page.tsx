/** صفحة المتجر — تُقدَّم من الخادم: القائمة في HTML الأولي وعنوان الصفحة باسم المتجر. */

import type { Metadata } from "next";
import { notFound } from "next/navigation";
import MerchantClient, { type Merchant, type Section } from "./MerchantClient";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export const dynamic = "force-dynamic";

async function load(id: string): Promise<{ merchant: Merchant; menu: Section[] } | null> {
  try {
    const res = await fetch(`${API}/api/v1/public/merchants/${id}`, { cache: "no-store" });
    if (!res.ok) return null;
    const json = (await res.json()) as { data?: { merchant: Merchant; menu: Section[] } };
    return json.data ?? null;
  } catch {
    return null;
  }
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string }>;
}): Promise<Metadata> {
  const { id } = await params;
  const data = await load(id);
  if (!data) return {};
  return {
    title: data.merchant.name,
    description: data.merchant.description || undefined,
  };
}

export default async function MerchantPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const data = await load(id);
  if (!data) notFound();
  return <MerchantClient merchant={data.merchant} menu={data.menu} />;
}
