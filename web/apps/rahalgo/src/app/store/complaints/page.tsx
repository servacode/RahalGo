"use client";
import { ReputationComplaints } from "@rahalgo/ui";
import { api } from "@/lib/api";
export default function ComplaintsPage() {
  return <ReputationComplaints api={api} />;
}
