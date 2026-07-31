"use client";
import { ReputationReviews } from "@rahalgo/ui";
import { api } from "@/lib/api";
export default function ReviewsPage() {
  return <ReputationReviews api={api} />;
}
