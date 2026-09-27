"use client";

/**
 * **الالتزامات المالية — قراءة فقط.**
 *
 * الدَّينُ على المتاجر والمناديب (`financial_obligations`): على من، وكم،
 * ومن أين، وكم بقي. **ولا فعلَ هنا** — لا عفوَ ولا تسويةَ ولا تعديل.
 * التسويةُ من بابها (استرداد أو نزاع)، وهذا الباب يُري الحال.
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDateTime } from "@rahalgo/i18n";
import {
  Money,
  Badge,
  Button,
  Select,
  PageContainer,
  PageHeader,
  Pagination,
  EmptyState,
  LoadingState,
  StatGrid,
  StatCard,
  useLiveData,
  IconBalance,
  IconWallet,
  IconStore,
  IconUser,
  IconOrder,
  IconStatus,
  IconDate,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const O = m.admin.obligations;

type Kind = "" | "merchant" | "rep";
type State = "" | "open" | "closed";

interface Settlement {
  amount: number;
  remaining: number;
  order_number: number | null;
  created_at: string;
}

interface Obligation {
  id: string;
  party_kind: "merchant" | "rep";
  party_id: string;
  party_name: string;
  amount: number;
  outstanding: number;
  cause: string;
  order_number: number | null;
  created_at: string;
  state: "open" | "closed";
  settlements: Settlement[];
}

interface ObligationsPage {
  obligations: Obligation[];
  total: number;
  page: number;
  per_page: number;
  /** **مجموعُ المتبقّي على المفتوح** — لا عددُ الصفوف. */
  outstanding_total: number;
}

/** **سببٌ مصنَّفٌ يُترجَم، ومجهولُه يُعرض كما جاء.** */
const causeText = (c: string): string =>
  (O.causes as Record<string, string>)[c] ?? c;

export default function ObligationsPage() {
  const [kind, setKind] = useState<Kind>("");
  const [state, setState] = useState<State>("");
  const [page, setPage] = useState(1);
  /** **الالتزامُ المفتوحُ سطورُه** — واحدٌ في المرّة، فلا تزدحم الشاشة. */
  const [open, setOpen] = useState<string | null>(null);

  const { data, loading } = useLiveData<ObligationsPage>(
    () =>
      api(
        `/api/v1/admin/obligations?page=${page}` +
          (kind ? `&party_kind=${kind}` : "") +
          (state ? `&state=${state}` : ""),
      ),
    ["wallet"],
    [kind, state, page],
  );

  if (loading) return <LoadingState />;
  const rows = data?.obligations ?? [];

  /** **وتبديلُ الترشيح يعود إلى الأولى** — فلا يقع على صفحةٍ لا توجد. */
  function pick(next: () => void) {
    setPage(1);
    setOpen(null);
    next();
  }

  return (
    <PageContainer width="full">
      <PageHeader icon={IconBalance} title={O.title} subtitle={O.hint} />

      {/* **وكلُّ شاشةِ مالٍ تقول مجموعها** — «كم لنا عند الناس الآن». */}
      <StatGrid>
        <StatCard
          label={`${O.outstandingTotal} (${m.common.currency})`}
          value={fmtNum(data?.outstanding_total ?? 0)}
          icon={IconWallet}
          tone={(data?.outstanding_total ?? 0) > 0 ? "accent" : "default"}
        />
      </StatGrid>

      <div className="mb-4 mt-4 flex flex-wrap gap-3">
        <div className="w-44">
          <Select value={kind} onChange={(e) => pick(() => setKind(e.target.value as Kind))}>
            <option value="">{O.kinds.all}</option>
            <option value="merchant">{O.kinds.merchant}</option>
            <option value="rep">{O.kinds.rep}</option>
          </Select>
        </div>
        <div className="w-44">
          <Select value={state} onChange={(e) => pick(() => setState(e.target.value as State))}>
            <option value="">{O.states.all}</option>
            <option value="open">{O.states.open}</option>
            <option value="closed">{O.states.closed}</option>
          </Select>
        </div>
      </div>

      {rows.length === 0 ? (
        <EmptyState icon={IconBalance} title={O.empty} />
      ) : (
        <ul className="space-y-2">
          {rows.map((o) => (
            <li key={o.id} className="surface p-3">
              <div className="flex flex-wrap items-center gap-3">
                <span className="min-w-0 flex-1">
                  <span className="flex items-center gap-1.5 font-bold">
                    {o.party_kind === "merchant" ? (
                      <IconStore size={15} />
                    ) : (
                      <IconUser size={15} />
                    )}
                    {o.party_name || "—"}
                    <Badge variant="neutral">{O.kinds[o.party_kind]}</Badge>
                  </span>
                  <span className="block text-sm text-ink-muted">{causeText(o.cause)}</span>
                  <span dir="ltr" className="flex items-center gap-1 text-xs text-ink-muted">
                    <IconDate size={13} />
                    {fmtDateTime(o.created_at)}
                  </span>
                </span>

                {o.order_number != null && (
                  <span
                    dir="ltr"
                    className="flex shrink-0 items-center gap-1 text-sm text-ink-muted"
                  >
                    <IconOrder size={14} />#{fmtRef(o.order_number)}
                  </span>
                )}

                {/* **المبلغُ الأصليُّ والمتبقّي** — الأصلُ باهتٌ والباقي بارز. */}
                <span className="shrink-0 text-end">
                  <span className="block text-2xs text-ink-muted">{O.amount}</span>
                  <Money value={o.amount} className="text-ink-muted tabular-nums" />
                </span>
                <span className="shrink-0 text-end">
                  <span className="block text-2xs text-ink-muted">{O.outstanding}</span>
                  <Money
                    value={o.outstanding}
                    className={`font-bold tabular-nums ${
                      o.outstanding > 0 ? "text-warning" : "text-success"
                    }`}
                  />
                </span>

                <Badge variant={o.state === "open" ? "warning" : "neutral"}>
                  {O.states[o.state]}
                </Badge>

                {o.settlements.length > 0 && (
                  <Button
                    variant="secondary"
                    onClick={() => setOpen((cur) => (cur === o.id ? null : o.id))}
                  >
                    {O.history} ({fmtNum(o.settlements.length)})
                  </Button>
                )}
              </div>

              {/* **سطورُ التسوية** — كم اقتُطع، ومن أيّ طلب، وكم بقي بعده. */}
              {open === o.id && (
                <ul className="surface-inset mt-3 divide-y divide-line-soft">
                  {o.settlements.map((st, i) => (
                    <li
                      key={i}
                      className="flex flex-wrap items-center justify-between gap-2 px-3 py-2 text-sm"
                    >
                      <span className="flex items-center gap-2">
                        <IconStatus size={13} />
                        <span dir="ltr">{fmtDateTime(st.created_at)}</span>
                        {st.order_number != null && (
                          <span dir="ltr" className="flex items-center gap-1 text-ink-muted">
                            <IconOrder size={13} />#{fmtRef(st.order_number)}
                          </span>
                        )}
                      </span>
                      <span className="flex items-center gap-4">
                        <span>
                          <span className="text-2xs text-ink-muted">{O.settledAmount} </span>
                          <Money value={st.amount} className="tabular-nums text-success" />
                        </span>
                        <span>
                          <span className="text-2xs text-ink-muted">{O.remaining} </span>
                          <Money value={st.remaining} className="tabular-nums text-ink-muted" />
                        </span>
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </li>
          ))}
        </ul>
      )}

      {data && data.total > data.per_page && (
        <div className="mt-4 flex justify-center">
          <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
        </div>
      )}
    </PageContainer>
  );
}
