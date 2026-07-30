"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth, canAccessPanel } from "@/lib/auth";

export default function IndexPage() {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (loading) return;
    router.replace(canAccessPanel(user) ? "/dashboard" : "/login");
  }, [user, loading, router]);

  return null;
}
