"use client";

/** شكاوى على متاجري — بلاغات الزبائن على طلبات متاجر المندوب. */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { ReputationComplaints } from "@rahalgo/ui";
import { api } from "@/lib/api";

const R = getMessages(defaultLocale).rep.reputation;

export default function ComplaintsPage() {
  return (
    <ReputationComplaints
      api={api}
      labels={{
        complaintsTitle: R.complaintsTitle,
        complaintsHint: R.complaintsHint,
        complaintsEmpty: R.complaintsEmpty,
      }}
    />
  );
}
