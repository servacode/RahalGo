"use client";

/** تقييمات متاجري — المصدر تقييمات الزبائن لمتاجر جلبها المندوب، لا تقييماً له هو. */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { ReputationReviews } from "@rahalgo/ui";
import { api } from "@/lib/api";

const R = getMessages(defaultLocale).rep.reputation;

export default function ReviewsPage() {
  return (
    <ReputationReviews
      api={api}
      labels={{
        reviewsTitle: R.reviewsTitle,
        reviewsHint: R.reviewsHint,
        reviewsEmpty: R.reviewsEmpty,
        avgLabel: R.avg,
      }}
    />
  );
}
