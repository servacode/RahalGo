# Merchant Settlement Method (cash / wallet) — Design for Review (Rev. 4)

> **Status: REVISED. Awaiting FINAL line-by-line financial approval. No code, no migration
> written or executed.** Rev. 1 → approved with 7 corrections + Q1–Q5. Rev. 2 → folded in.
> Rev. 3 → added two proofs; forced a unified per-(order,merchant) identity. Rev. 4 (this) →
> closes the four blocking items: legacy cutover safety, FI-12 physical-cash payout, cash-pay
> vs refund concurrency, and zero-net entitlement. Every file:line measured against the tree.
>
> **Review this file line by line. On approval I implement; not before.**

## What changed in Rev. 4 (the four blocking items)

- **Item 1 — Legacy cutover / double-settle safety (§C.1, §J.2).** Added a legacy guard: if an
  order has **no** `merchant_settlements` rows **and** a historical `merchant_earning` exists for
  it, treat it as LEGACY ALREADY SETTLED and return — no new credit, no cash accrual. Fixed
  Class-A detection to be **order-scoped** (any historical `merchant_earning` for the order),
  **independent of current `merchants.owner_user_id`**, so ownership changes can't misclassify a
  historically wallet-settled source as cash.
- **Item 2 — FI-12 counts physical cash payout (§E.2, §F).** `merchant_cash_paid` is now
  explicitly a **global external money-out to a real party**; per-order `toParties` still excludes
  it. New numeric conservation table for 100/90/10 across all stages; new FI-14.f payout
  reconciliation. Without this, FI-12 would wrongly expect treasury=100 after payment.
- **Item 3 — Cash-pay vs refund/return concurrency (§H.2).** Both the admin cash-paid confirm and
  any pre-pay reversal `SELECT … FROM merchant_settlements WHERE id=$1 FOR UPDATE` the **same**
  row; deterministic outcomes defined; race test added (payment-wins / refund-wins only).
- **Item 4 — Zero-net entitlement (§J.2, §Z).** Measured: `commission_percent BETWEEN 0 AND 100`
  ⇒ 100% commission ⇒ `due=0` is a valid config. Design: `due<=0` is **skipped** (no row, no
  ledger, no cash_due), exactly-once trivially, no CHECK failure — mirroring transitions.go:983.

## What changed in Rev. 3 (the two required proofs)

- **Point 1 — Per-source exactly-once, symmetrical identity (§J).** Measured: the current
  wallet guard is **order-level** (`EXISTS(merchant_earning WHERE ref=order)`,
  transitions.go:866-875), not per (order, merchant). It cannot express per-source identity and
  a same-owner-two-merchants order is indistinguishable in the ledger by `(ref,kind,user_id)`.
  **Fix (authorized):** a unified `merchant_settlements` identity table, `UNIQUE(order_id,
  merchant_id)`, that gates **both** wallet and cash with `INSERT … ON CONFLICT DO NOTHING
  RETURNING`. Exactly-once-per-source and XOR both become structural.
- **Point 2 — Signed cash-accrual reversal correlation (§K).** Exact ref/idempotency identity,
  coexistence with the positive accrual, duplicate prevention, and treasury attribution — with
  a deterministic +100 / −30 / −70 worked example and the exact conservation result.
- **FI-12 post-paid-refund (§E, §F, §I):** SET-25 now **numerically** proves FI-12 using the
  open merchant obligation as a receivable counterbalance — not "similar to the wallet case."

---

## §0. The one hard problem (why this touches sacred accounting)

Cash settlement must NOT credit the merchant wallet and must record a separate cash payable. Two
measured facts:

1. **`financial_obligations` cannot represent a cash payable.** `CHECK(debt >= 0)`
   (`0071_merchant_debt.sql:33`), merchant→platform only. Right for reversals (merchant owes
   platform back), wrong for "platform owes merchant."
2. **Treasury is computed by difference** (`treasury.go:117` `creditTreasury`):
   ```
   paid      = wallet_paid + (cash_due if delivered/refunded)
   toParties = Σ amount WHERE kind IN ('merchant_earning','driver_earning','commission')  -- ref=order
   refunded  = Σ amount WHERE kind = 'refund'
   posted    = Σ amount WHERE kind = 'platform_profit'
   delta     = (paid - refunded - toParties) - posted    -- credited to treasury as platform_profit
   ```
   The merchant's net leaves via `merchant_earning` ∈ `toParties`, so treasury doesn't keep it.
   **If a cash order simply omits `merchant_earning`, treasury books the merchant's net as
   profit.** So the cash payable must be recognized as a *party outflow*.

**Two independent concepts** (never inferred one from the other; all 4 combos valid, SET-07..10):

| Concept | Field | Values | Owner |
|---|---|---|---|
| Customer **payment** (existing) | `orders.payment_method` | cash \| wallet | customer, per order |
| Merchant **settlement** (NEW) | `merchants.settlement_method` | cash \| wallet | admin, per merchant |

## Owner decisions locked (Q1–Q5)

- **Q1 = PER-SOURCE snapshot authoritative** (`order_items.merchant_settlement_method`); any
  order-level field is informational/single-source only, never overriding a per-source value.
- **Q2 = dedicated non-spendable system holding wallet**; `merchant_cash_accrued` ∈ `toParties`,
  `merchant_cash_paid` not; balance reconciles exactly to outstanding cash liability.
- **Q3 = cash-paid = holding settlement + audit only; no Cashbox module** this phase.
- **Q4 = Merchant APK read-only** "طريقة استلام المستحقات" (+ unpaid cash if safe), current
  merchant context, merchant cannot change.
- **Q5 = AppMerchant-only cash-paid notification**, idempotent.

---

## §J. Per-source settlement identity — exactly-once for BOTH paths (proof for point 1)

### J.1 Measured current behavior (the gap)
`settleMerchant` (transitions.go:856) guards the **whole function** once, at the top:
```go
EXISTS(SELECT 1 FROM wallet_transactions WHERE ref = orderID AND kind = 'merchant_earning')
```
It then loops per source-merchant (grouped from `order_items.merchant_id`, line 915-923) and
credits each owner (line 987). This is exactly-once **only** because all sources settle in one
atomic tx and the guard blocks whole-function re-entry. It is **not** a per-(order,merchant)
identity, and:
- **All-cash order:** no `merchant_earning` ever ⇒ the guard never trips ⇒ cannot gate re-entry.
- **Same owner owns A and B:** two shares credit the same `user_id`; in the ledger the two
  entitlements differ only by amount — `(ref,kind,user_id)` cannot tell them apart, so any
  per-owner idempotency would **collapse** them into one.

### J.2 Fix — unified identity table (structural exactly-once + structural XOR)
One record per settled (order, merchant), for **either** method:
```sql
CREATE TABLE merchant_settlements (
  id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id         uuid NOT NULL REFERENCES orders(id),
  merchant_id      uuid NOT NULL REFERENCES merchants(id),   -- AUTHORITATIVE liability party (§party-identity)
  owner_user_id    uuid NOT NULL REFERENCES users(id),       -- accrual-time audit/notify snapshot ONLY
  method           text NOT NULL CHECK (method IN ('cash','wallet')),  -- copied from the per-source snapshot
  amount           bigint NOT NULL CHECK (amount > 0),        -- authoritative accrued merchant net (§amount)
  reversed_amount  bigint NOT NULL DEFAULT 0
                    CHECK (reversed_amount >= 0 AND reversed_amount <= amount),
  state            text NOT NULL
                    CHECK (state IN ('wallet_credited','wallet_reversed',
                                     'cash_due','cash_paid','cash_reversed')),
  earning_tx_id    bigint REFERENCES wallet_transactions(id), -- wallet: the merchant_earning entry
  accrued_tx_id    bigint REFERENCES wallet_transactions(id), -- cash:   the +merchant_cash_accrued entry
  paid_tx_id       bigint REFERENCES wallet_transactions(id), -- cash:   the −merchant_cash_paid entry
  paid_by          uuid REFERENCES users(id),                 -- cash:   admin actor at payment
  paid_owner_user_id uuid REFERENCES users(id),               -- cash:   owner resolved AT payment
  paid_at          timestamptz,
  note             text NOT NULL DEFAULT '',
  created_at       timestamptz NOT NULL DEFAULT now(),
  UNIQUE (order_id, merchant_id),         -- THE per-source entitlement identity, symmetrical for both methods
  -- method↔state coherence, enforced by the DB (item 1 / Rev.4):
  CHECK (   (method='wallet' AND state IN ('wallet_credited','wallet_reversed'))
         OR (method='cash'   AND state IN ('cash_due','cash_paid','cash_reversed')) )
);
```
`settleMerchant` becomes (legacy cutover first, then the per-source gate):
```
-- LEGACY CUTOVER (pre-migration history only) — item 1:
if NOT EXISTS(merchant_settlements WHERE order_id=$1)
   AND EXISTS(wallet_transactions WHERE ref=$1 AND kind='merchant_earning'):
     return nil                          -- LEGACY ALREADY SETTLED → no new credit, no cash accrual

for each source-merchant sh:
   if sh.owner is NULL or sh.due <= 0:  continue   -- ZERO/NEGATIVE NET (item 4 / §Z): no row, no ledger
   row := INSERT INTO merchant_settlements (order_id, merchant_id, owner_user_id, method, amount, state)
            VALUES (o, sh.merchant, sh.owner, sh.method, sh.due, <initial-for-method>)
            ON CONFLICT (order_id, merchant_id) DO NOTHING
            RETURNING id;
   if row IS NULL:                       -- CONFLICT: a settlement already exists → VALIDATE, never blind-skip (item 1)
       ex := SELECT * FROM merchant_settlements WHERE order_id=o AND merchant_id=sh.merchant FOR UPDATE;
       assert ex.merchant_id == sh.merchant                         else FAIL_CLOSED
       assert ex.method     == sh.method (per-source snapshot)      else FAIL_CLOSED
       assert ex.amount     == recomputed authoritative net (§amount) else FAIL_CLOSED
       assert (ex.method,ex.state) satisfies the method↔state CHECK  else FAIL_CLOSED
       assert the expected financial tx identity exists AND matches state:
              wallet_credited → ex.earning_tx_id present AND a matching merchant_earning row
              cash_due        → ex.accrued_tx_id present AND net cash accrual == amount−reversed_amount
              cash_paid       → ex.accrued_tx_id + ex.paid_tx_id present AND −paid == amount−reversed_amount
              cash_reversed   → ex.accrued_tx_id present AND reversed_amount == amount, holding contribution 0
       -- everything matches ⇒ harmless validated no-op (a true retry); ANY mismatch ⇒ invariant error, abort tx
       continue
   -- fresh settlement of this source:
   if method='wallet':  merchant_earning(+due)→owner ; store earning_tx_id ; state='wallet_credited' ; offsetMerchantDebt(wallet)
   if method='cash':    merchant_cash_accrued(+due)→holding ; store accrued_tx_id ; state='cash_due' ; offsetMerchantDebt(cash)
```
**offsetMerchantDebt is method-aware (item 2).** Wallet branch: existing negative `merchant_earning`
offset (transitions.go:1032). Cash branch: the analogous negative `merchant_cash_accrued` offset
(reduces the fresh `cash_due` and holding by `take=min(debt,due)`), calling the **same**
`obligations.Settle` with the cash offset tx — no change to the obligations engine. If `due−take==0`
the source is zero-net (§Z: no residual row/ledger). This is how a cash merchant's case-D obligation
is recovered — symmetrical to wallet, still a reduced payout, never invented money-in (§E.2).
The legacy guard fires **only** for pre-migration orders (no `merchant_settlements` rows yet but a
historical `merchant_earning`). All post-migration settlements use `merchant_settlements` as the
sole identity; the old order-level `merchant_earning`-EXISTS guard is otherwise gone.
- The `ON CONFLICT (order_id, merchant_id)` is the **single exactly-once gate for both methods**,
  but a conflict is **validated, never blindly skipped** (item 1): the existing row is locked
  `FOR UPDATE` and its merchant_id / method / amount / method-state / financial-tx identity are
  checked against the recomputed truth. A full match is a harmless validated no-op (a real retry);
  **any mismatch fails closed with an invariant error** — corruption is never mistaken for a
  successful retry. The order-level `merchant_earning`-EXISTS guard is **removed** (kept only, if
  desired, as a cheap early-out `EXISTS(merchant_settlements WHERE order_id=$1)`).
- **XOR is structural:** one row per (order, merchant); `method` is the branch; a source cannot
  be both.
- The defensive double-call (StPickedUp then StDelivered) is handled: the second call's INSERT
  conflicts for every source ⇒ skipped, no double credit/accrual.

### J.3 Proof for the three required cases
| Case | Rows created | Ledger effect | Retry |
|---|---|---|---|
| A + B, **different owners** | 2 rows (distinct merchant_id) | 2 credits, 2 owners | both INSERTs conflict → no double |
| A + B, **same owner** | 2 rows (distinct merchant_id, same owner_user_id) | 2 credits to the SAME wallet (2 ledger rows) — **not collapsed**, because identity is `merchant_id`, not owner | both conflict → no double |
| mixed A=wallet, B=cash | 2 rows (method differs) | A: merchant_earning; B: cash accrual | both conflict → no double; XOR holds |

**Cash and wallet now share the identity `(order_id, merchant_id)`** — the symmetry the owner
required. This is a deliberate, correctness-required change to the wallet path's idempotency; it
does not change *amounts* (still §amount) or *who* is paid, only the exactly-once identity.

### J.4 Migration impact of J.2 (no historical rewrite)
Historical orders are **not** backfilled into `merchant_settlements` — they are terminal and
never re-enter `settleMerchant` (class A/C, §C). Only future settlements (class B, open
pre-accrual orders) create rows, under the new gate. Removing the old order-level guard is
forward-only and safe because a class-B order has no `merchant_earning` yet.

---

## §K. Signed cash-accrual reversal correlation (proof for point 2)

**Accrual identity** = the `merchant_settlements` row (order_id, merchant_id). **Reversal
history** = an append-only child table + one signed-negative holding ledger entry per reversal:
```sql
CREATE TABLE merchant_settlement_reversals (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  settlement_id  uuid NOT NULL REFERENCES merchant_settlements(id),
  tx_id          bigint NOT NULL REFERENCES wallet_transactions(id), -- the −merchant_cash_accrued entry
  amount         bigint NOT NULL CHECK (amount > 0),
  cause          text NOT NULL,          -- 'refund_full' | 'returned_goods'
  event_ref      text NOT NULL,          -- identifies the triggering refund/return event
  created_at     timestamptz NOT NULL DEFAULT now(),
  UNIQUE (settlement_id, event_ref)       -- duplicate-reversal prevention
);
```

Answering each sub-question exactly:

- **What ref/idempotency identity the reversal uses.** Domain idempotency:
  `UNIQUE(settlement_id, event_ref)` — the same refund/return event can post at most one reversal
  per settlement. Ledger `ref` of the negative entry = **`order_id`** (identical to the accrual),
  so the treasury sees it. The reversal is additionally bounded by
  `CHECK(reversed_amount <= amount)` and by the holding wallet `CHECK(balance >= 0)`.
- **How it coexists with the original positive accrual.** Two separate append-only ledger rows,
  **same `ref=order_id`, same `kind='merchant_cash_accrued'`, opposite signs**. The accrual row
  is never edited or deleted. `reversed_amount` on the parent caches Σ of reversal amounts.
- **How duplicate reversal is prevented.** `UNIQUE(settlement_id, event_ref)` (a re-fired event
  is a no-op), plus the outstanding bound (`reversed_amount <= amount`), plus the DB
  `CHECK(balance>=0)` on the holding (cannot reverse more than is held), plus the driving
  refund/return transition being itself once-only.
- **How `treasury.toParties` attributes both to the SAME order/merchant entitlement.** Both the
  +accrual and every −reversal use `kind='merchant_cash_accrued'` and `ref=order_id`, so
  `toParties = Σ amount WHERE ref=order AND kind IN (...,'merchant_cash_accrued')` **includes and
  nets both** within that order. It cannot disappear (same ref as everything else for the order)
  and cannot be double-counted (each is one immutable row). Per-(order,merchant) audit attribution
  is provided by `merchant_settlements` + `merchant_settlement_reversals` (both carry
  merchant_id/settlement_id); treasury math is per-order, which `ref=order_id` satisfies exactly.

### K.1 Deterministic worked example (+100, −30, −70)
Single cash source, accrued net = 100, then two partial reversals 30 then 70. Ledger (holding,
`ref=order`, `kind=merchant_cash_accrued`): `+100`, `−30`, `−70`.

| Step | holding entries (net) | merchant_settlements | toParties (cash accrual component, this order) |
|---|---|---|---|
| accrual +100 | +100 | amount=100, reversed=0, state=cash_due, outstanding=100 | +100 |
| reversal −30 | +70 | reversed=30, state=cash_due, outstanding=70 | +70 |
| reversal −70 | 0 | reversed=100, state=**cash_reversed**, outstanding=0 | **0** |

Results asserted by tests: **holding outstanding = 0**; **net cash accrual in `toParties` = 0**;
therefore the order's treasury `delta = paid − refunded − toParties` equals the exact
no-cash-accrual baseline (whatever driver/commission remain), i.e. **conservation is exact and
the merchant's cash net was correctly neither profit nor a dangling liability.** A third
identical `event_ref` reversal is a UNIQUE no-op; a `−1` beyond outstanding is refused by
`CHECK(balance>=0)` / `reversed_amount<=amount`.

---

## §Z. Zero-net entitlement (proof + safe behavior for point 4)

**Measured:** `merchants.commission_percent` is `CHECK(commission_percent BETWEEN 0 AND 100)`
(`0013_commissions.sql:5`); the settings default key can also be 0..100. A merchant on **100%
commission** yields `due = cost − 100%·cost = 0`. So `amount > 0` is **NOT** an enforced invariant
— a valid config can produce a zero merchant net (a negative is impossible here: commission ≤ 100%
so `due ≥ 0`).

**Current behavior:** `settleMerchant` already skips it — transitions.go:983
`if sh.ownerID == nil || sh.due <= 0 { continue }` — no `merchant_earning`, no ledger movement.

**Design (mirror it, both methods):** in the per-source loop (§J.2) `if sh.due <= 0: continue`
**before** the INSERT. Therefore a zero-net source:
- creates **no** `merchant_settlements` row (so `CHECK(amount > 0)` is never reached — no tx
  failure);
- posts **no** wallet credit and **no** cash accrual, hence **no** `cash_due`;
- is **exactly-once trivially** — re-entry recomputes `due=0` and skips again (idempotent no-op);
- for an all-zero-net order there are simply no settlement rows (the early-out `EXISTS(
  merchant_settlements)` doesn't fire; the loop harmlessly recomputes-and-skips each call).

`CHECK(amount > 0)` is kept as belt-and-suspenders (a positive row is always a real payable); it is
never triggered because the skip precedes the INSERT. Test **SET-28**: 100%-commission source (and
a mixed order with one zero-net + one positive source) ⇒ zero-net source has no row/ledger/cash_due,
positive source settles normally, retry no double, all FI green.

---

## §A. Settlement + refund state machine (per (order, merchant))

**Snapshot (per source, at CreateTx, under merchant-row lock §H, immutable):**
`order_items.merchant_settlement_method ∈ {cash, wallet}`.

**Settlement (pickup, `settleMerchant`, order FOR UPDATE, per-source ON CONFLICT gate §J):**
```
per source: INSERT merchant_settlements ON CONFLICT DO NOTHING RETURNING → if skipped, continue
  method=wallet → merchant_earning(+net)→merchant wallet ; state=wallet_credited   (EXISTING amounts)
  method=cash   → merchant_cash_accrued(+net)→HOLDING ; state=cash_due
```
**Cash lifecycle (append-only; row never deleted):**
```
   cash_due ──(admin confirms; pays outstanding = amount−reversed_amount)──► cash_paid
      │                                                                          │
      │ pre-pay refund/return (proportional): −merchant_cash_accrued→holding,    │ post-pay refund/return:
      │ +reversal child row; reversed==amount ⇒ cash_reversed else cash_due      │ obligations.Create(merchant→platform,
      ▼                                                                          ▼   exact revAmt); row stays cash_paid
   cash_reversed (terminal, fully reversed pre-payment)
```
**Wallet lifecycle:** `wallet_credited`; on refund/return → existing `reverseCommissions` /
returned-goods reverses the actual credited net (state may be recorded `wallet_reversed`),
remainder→obligation (existing). Cancellation before accrual: no row (SET-20).

---

## §B. Schema (one migration, additive, reversible)

1. `merchants.settlement_method text NOT NULL DEFAULT 'cash' CHECK IN (cash,wallet)` — admin-only.
2. `order_items.merchant_settlement_method text CHECK IN (cash,wallet)` — **authoritative
   per-source snapshot** (Q1). Order-level field omitted (recommended) or informational only.
3. `merchant_settlements` (§J.2) — unified per-(order,merchant) identity, `UNIQUE(order_id,
   merchant_id)`.
4. `merchant_settlement_reversals` (§K) — append-only reversal history.
5. `wallets.is_cash_holding boolean NOT NULL DEFAULT false` + seeded system user +
   `EnsureCashHolding()` mirroring `EnsureTreasury`. **Not** treasury, so `CHECK(balance>=0)`
   stays → liability can never go negative. Excluded from every spend/withdraw/list/payout path
   exactly as `is_treasury` is.
6. New `wallet_transactions.kind`: `merchant_cash_accrued` (±, holding), `merchant_cash_paid`
   (−, holding).

### §party-identity (amendment 6)
`merchant_id` is the authoritative party owed. `owner_user_id` = accrual-time audit snapshot
only. On ownership change between `cash_due` and `cash_paid`: liability stays with the store; at
payment we re-resolve the **current** owner into `paid_owner_user_id`, pay/notify that operator,
keep `owner_user_id` for audit, record both. Never silently pay/notify a stale owner.

---

## §C. Migration / backfill classification (amendment 3 — no historical falsification)

Existing merchants → `settlement_method='cash'` (default). Existing orders/sources classified;
**no** `merchant_settlements` rows created by the migration for any historical order (§J.4).

| Class | Detection (**order-scoped**, item 1) | Snapshot | Effect |
|---|---|---|---|
| **A** already accrued (historically wallet) | the **order** has ANY `merchant_earning` | `order_items…='wallet'` (truthful) | none |
| **B** open, pre-accrual | order NOT terminal AND order has no `merchant_earning` | `'cash'` (launch default) | settles cash at real pickup |
| **C** terminal, never accrued | terminal AND order has no `merchant_earning` | `'cash'` (record only) | none, ever; inert; implies no payment |

**Class-A is detected from historical `merchant_earning` truth alone — NOT from current
`merchants.owner_user_id`** (item 1). Pre-feature settlement was wallet-only and **atomic for all
sources of an order**, so "the order has any `merchant_earning`" ⇒ every source of that order was
wallet-settled. This does not read ownership, so an ownership change after a historical wallet
settlement can never misclassify a source as cash:
```sql
-- A: any historical merchant_earning on the order ⇒ ALL its sources were wallet-settled
UPDATE order_items oi SET merchant_settlement_method = 'wallet'
WHERE EXISTS (SELECT 1 FROM wallet_transactions wt
              WHERE wt.ref = oi.order_id::text AND wt.kind = 'merchant_earning');
-- B and C: everything else → cash
UPDATE order_items SET merchant_settlement_method = 'cash' WHERE merchant_settlement_method IS NULL;
```
No retroactive cash dues, no retroactive wallet credits, no historical ledger rewrite. Down = drop
columns/tables/kinds/flag.

### §C.1 Legacy cutover guard (double-settlement safety, item 1)
Historical orders create **no** `merchant_settlements` rows (§J.4). An order that already accrued
`merchant_earning` before migration but is still **in-flight** could re-enter `settleMerchant`
later (e.g. the defensive StDelivered call, or an admin data-fix). With the old order-level guard
removed, re-entry would wrongly credit again. The legacy guard in `settleMerchant` (§J.2) prevents
this: **no `merchant_settlements` rows for the order AND a historical `merchant_earning` exists ⇒
return without any new wallet credit or cash accrual.** This fallback exists solely for
pre-migration history; post-migration, `merchant_settlements` is the sole identity.
Test **SET-26**: historical order `picked_up`→`merchant_earning` posted → cutover → later
`delivered` re-enters `settleMerchant` ⇒ NO second merchant credit, NO cash row, financial state
byte-identical.

---

## §D. Holding-ledger entry matrix

Holding = pooled system wallet (`is_cash_holding`). All entries `ref=order_id`.

| Event | Ledger entry | In `toParties`? | Table effect |
|---|---|---|---|
| Accrual (pickup, cash) | `merchant_cash_accrued` **+net** → holding | YES | `merchant_settlements` state=cash_due, `accrued_tx_id` |
| Pre-pay reversal | `merchant_cash_accrued` **−revAmt** → holding | YES (nets) | `+merchant_settlement_reversals` row; `reversed_amount+=revAmt`; ==amount⇒cash_reversed |
| Payment | `merchant_cash_paid` **−(amount−reversed)** → holding | NO | state=cash_paid, `paid_tx_id/by/owner/at` |
| Post-pay refund/return | *(none)* → `obligations.Create(merchant→platform, revAmt)` | n/a | row stays cash_paid |
| Wallet path | `merchant_earning` **+net** → merchant wallet (EXISTING) | YES | state=wallet_credited, `earning_tx_id` |

Signed-negative same kind is the established idiom (goods.go:178-183). Holding `CHECK(balance>=0)`
forbids over-reversal/over-pay.

---

## §E. Treasury formula impact + FI-12 (numeric)

**One change** in `creditTreasury` (`treasury.go:151-160`):
```
toParties = Σ amount WHERE kind IN ('merchant_earning','driver_earning','commission','merchant_cash_accrued')
```
`merchant_cash_paid` excluded. Per-order conservation:

| Scenario | toParties (cash part) | delta = paid−refunded−toParties | Result |
|---|---|---|---|
| accrued, unpaid | +net | commission share | ✅ net excluded from profit |
| fully reversed pre-pay | +net−net=0 | paid−paid−0=0 | ✅ = wallet-refund |
| paid | +net (payment excluded) | commission share | ✅ payment doesn't disturb treasury |
| paid then refunded (case D) | +net | paid−paid−net = **−net** | offset by obligation +net (below) |

### §E.2 FI-12 (global) — physical cash payout is money-OUT (item 2)

Per-order treasury is computed **by difference** at settlement and is **not** re-run at cash-paid.
So the payment step must be captured in the GLOBAL conservation, or FI-12 would wrongly expect
treasury to still hold the merchant's cash after we physically paid it out. Distinguish sharply:

- **`merchant_cash_accrued`** — included in per-order `toParties` (excludes merchant net from
  platform profit). **NOT** counted again below.
- **`merchant_cash_paid`** — **NOT** in per-order `toParties`; **IS** counted in FI-12 global
  `money_out_to_real_parties` as actual external cash handed to the merchant.

```
treasury_net ==  money_in
              −  wallet_credited_net        (Σ merchant_earning, net of reversals)
              −  driver_earning_net
              −  cash_paid_out              (= −Σ merchant_cash_paid; physical cash handed to merchants)   ◄ item 2
              −  holding_balance            (is_cash_holding: cash accrued, not yet paid = still owed)
              −  refunds_to_customers       (Σ refund)
              +  obligations_open           (receivable: merchant→platform, e.g. post-paid refund)
```

**Numeric proof — 100 gross / 90 merchant net / 10 platform (cash-settled merchant, driver 0):**

| Stage | money_in | cash_paid_out | holding | refunds | oblig | treasury (identity) | ✓ |
|---|---|---|---|---|---|---|---|
| accrual | 100 | 0 | 90 | 0 | 0 | 100−0−90−0+0 = **10** | ✅ |
| **cash payment** | 100 | **90** | **0** | 0 | 0 | 100−90−0−0+0 = **10** | ✅ (was the bug: without cash_paid_out it'd expect 100) |
| paid, then **full** refund | 100 | 90 | 0 | 100 | 90 | 100−90−0−100+90 = **0** (commission zeroed, transitions.go:1310) | ✅ |

Order X ends: treasury 0, an **open obligation of 90**. The repayment is a **separate future**
event, and its accounting is the measured truth below — not money-in on X.

### §E.3 Obligation repayment — measured `offsetMerchantDebt` accounting (item 2)

**Measured:** `obligations.Settle` has exactly two callers — `offsetMerchantDebt`
(transitions.go:1039) and `offsetRepDebt` (1440). `offsetMerchantDebt` (1015-1048) posts
`ApplyTxID(owner, −take, "merchant_earning", …)` then `Settle(take)`. **There is NO direct
cash-repayment path.** So distinguish:

- **A) DIRECT merchant repayment — NOT present in the engine.** No measured path takes external
  cash from a merchant to reduce debt. FI-12 therefore **does not** assume any `money_in` for
  repayment. (If such a path is ever added, it would be `+money_in` with `−obligation`; SET-25
  asserts that shape only if/when the code exists — today it does not.)
- **B) OFFSET against a future entitlement — the real mechanism (reduced payout, no money_in).**
  On the merchant's next settled order **Y**, `offsetMerchantDebt` posts `−take` of the party
  payout (wallet: `−take merchant_earning`; **cash: `−take merchant_cash_accrued`** per the
  method-aware offset, §J.2). `toParties(Y)` drops by `take`, so **treasury(Y) rises by `take`** —
  the platform recovers the debt purely by keeping `take` more of order Y's by-difference profit.
  Obligation → 0. **No external money_in is invented.**

**Numeric (repay the 90 via offset on order Y; Y's own gross 100, merchant net 90, platform 10):**

| On order Y | money_in(Y) | wallet_credited(Y) net | cash_paid_out | holding | oblig | treasury(Y) | ✓ |
|---|---|---|---|---|---|---|---|
| Y settled, debt offset | 100 | 90 − **90**(offset) = 0 | 0 | 0 | 90→**0** | 100 − 0 − 0 − 0 + 0 = **100** = Y's 10 profit + **90 recovered** | ✅ |

Cross-order sum (X refunded + Y offset): the platform's 90 cash paid then refunded is recovered by
withholding 90 from Y's merchant payout; obligations net to 0; conservation exact — **without any
phantom money-in**. FI-12 is adapted to this measured accounting, not the other way round.

Partial stages assert the **same identity** with the real returned-goods amount (`sh.earned`,
goods.go:173-197), which the tests compute exactly:
- **partial pre-pay reversal** (return 30 before payment): holding 90→60, refunds 30, cash_paid_out 0
  ⇒ identity holds; outstanding payable = 60.
- **paid then partial return** (return 30 after payment): cash_paid_out 90, obligation +30
  (returned portion), refunds 30 ⇒ identity holds; row stays cash_paid.

SET-25 asserts every stage numerically; conservation is exact throughout.

---

## §F. FI invariants (FI-14.a..f), revised

- **FI-14.a** `is_cash_holding` balance == `Σ (amount − reversed_amount)` over `state='cash_due'`
  (current outstanding — reflects partial reversals; excludes paid/reversed).
- **FI-14.b** XOR: no (order, merchant) has both `method='wallet'` credit and a `method='cash'`
  row — structural (one row), plus a checker over ledger vs `merchant_settlements`.
- **FI-14.c** exactly-once: one `merchant_settlements` row per (order, merchant); net
  `merchant_cash_accrued` per settlement ∈ [0, amount] (no over-reversal). Mirrors FI-05.c.
- **FI-14.d** fidelity: `amount` == authoritative net at accrual (§amount); `reversed_amount` ==
  Σ `merchant_settlement_reversals.amount` == Σ negative `merchant_cash_accrued` for the
  settlement == the authoritative entitlement-reversal amount (full net for full refund; returned
  portion for partial — §G).
- **FI-14.e** holding never a value source: balance ≥ 0 (DB CHECK re-asserted); never a
  payout/withdraw source.
- **FI-14.f — payout reconciliation (item 2).** `−Σ merchant_cash_paid` (physical cash out) ==
  `Σ (amount − reversed_amount)` over `state='cash_paid'`. Every dinar that left the holding via
  `merchant_cash_paid` corresponds to a settled `cash_paid` row's outstanding — nothing vanishes,
  nothing double-pays. Together with FI-14.a (holding == open `cash_due` outstanding), the holding
  and the paid-out cash fully reconcile to the settlement table.
- **FI-06 / FI-12** honest after accrual, payment, pre-pay reversal (full+partial), and post-paid
  obligation — proven numerically in §E.2 with `merchant_cash_paid` counted as global money-out.
  FI-06.d unchanged (cash accrual only AT pickup). FI-13.* unchanged.

---

## §G. Full / partial refund matrix (amendment 1)

"revAmt" = exact existing entitlement-reversal amount. Full order refund →
`reverseCommissions` reverses actual credited net (transitions.go:1304), remainder→obligation.
Partial return → `goods.go:173-197` reverses returned portion `sh.earned`, remainder→obligation.

| # | Snapshot | Timing | revAmt | Action | Engine |
|---|---|---|---|---|---|
| A | wallet | not accrued | — | nothing | — |
| B | wallet | accrued, **full** refund | actual credited net | existing `reverseCommissions` | existing |
| B′ | wallet | accrued, **partial** return | returned `sh.earned` | existing returned-goods | existing |
| C | cash | cash_due, **full** pre-pay | full accrued net | −net→holding; reversed=amount; state=cash_reversed | new |
| C′ | cash | cash_due, **partial** pre-pay | returned portion | −portion→holding; reversed+=portion; stays cash_due | new |
| D | cash | **paid**, full refund | full net | row stays cash_paid; obligation +net (CauseRefundMerchant) | existing |
| D′ | cash | **paid**, partial return | returned portion | row stays cash_paid; obligation +portion (CauseReturnedGoods) | existing |

Pre-pay cash reversal (C/C′) needs no wallet cap / no remainder (money still in holding).
History never rewritten (paid stays cash_paid; full reversal → cash_reversed, a new state, not a
deletion).

---

## §H. Concurrency / locking (amendments 4 + 5)

- **Admin change:** `SELECT … FROM merchants WHERE id=$1 FOR UPDATE` then UPDATE + audit, one tx.
- **CreateTx snapshot:** lock all source merchants **sorted-ID order** (`… WHERE id = ANY($ids)
  ORDER BY id FOR UPDATE`), read method under lock, write per-source snapshots. Sorted order
  prevents deadlock between overlapping multi-source creates.
- **SET-06:** concurrent change + create serialize on the merchant row → snapshot is exactly old
  (create won the lock) or exactly new (change committed first); never null/mixed.
- **Fail-closed cross-path guards** in `settleMerchant`, under order FOR UPDATE: the unified
  `ON CONFLICT (order_id, merchant_id)` gate already makes both-paths impossible (one row);
  additionally, before crediting/accruing, assert the row's `method` matches the branch → abort
  otherwise. A bug fails closed (no settlement), never both.
- **Optional DB trigger** (described, not silently added): a `BEFORE INSERT` cross-table guard is
  possible but would be a surprising trigger on the hottest financial table; per repo convention
  it needs its own approval. Recommendation: rely on the unified UNIQUE identity + snapshot + order
  FOR UPDATE + in-tx guards + FI-14.b (defense in depth already).

### §H.2 Cash-paid vs pre-pay reversal concurrency (item 3)

Both the admin **cash-paid confirm** and any **pre-pay refund/return reversal** begin their tx with
`SELECT … FROM merchant_settlements WHERE id = $1 FOR UPDATE` on the **same** settlement row (the
per-(order,merchant) identity). This serializes them; each then reads `outstanding = amount −
reversed_amount` **under the lock** and acts on the current value.

- **Payment** pays exactly the current `outstanding` (`merchant_cash_paid −outstanding`), sets
  `state=cash_paid`. If a reversal committed first, it pays only the reduced outstanding.
- **Reversal** on a still-`cash_due` row reduces `reversed_amount`; if it finds the row already
  `cash_paid` (payment won), it takes the **post-pay** branch → `obligations.Create(merchant→
  platform, revAmt)`.

**Race test SET-27 — `cash_due=100`, concurrent A) admin pays, B) return reverses 30. Only two
serialized outcomes:**

| Winner | Result |
|---|---|
| Payment first | `cash_paid=100`; then B sees `cash_paid` → obligation 30 (post-pay) |
| Reversal first | `reversed_amount=30`, outstanding=70; then A pays only **70** |

Impossible by construction (row lock + outstanding-under-lock + holding `CHECK(balance>=0)` +
reversal UNIQUE): overpay 100 while treating 30 as unpaid, negative holding, double reversal, or a
missing obligation. FI-14.a/f re-assert the reconciliation after either interleaving.

---

## §I. SET-01..25 mapping (updated)

- SET-01 default cash. SET-02/03 admin switch + audit + step-up. SET-04/05 order keeps/gets
  per-source snapshot.
- **SET-06** concurrent change+create → exactly one old-or-new snapshot (§H).
- SET-07..10 four payment×settlement combos.
- **SET-11a** wallet, A+B **different owners** → 2 rows, 2 credits once (§J.3).
  **SET-11b** wallet, A+B **same owner** → 2 rows (per-merchant identity), 2 credits once, **not
  collapsed**; retry no double. **SET-11c** all-cash order retry → no re-accrual (gate, not the old
  merchant_earning guard).
- SET-12 cash: wallet credit 0, cash_due = exact net, holding +net.
- SET-13 duplicate lifecycle event → no double-settle (both paths, via ON CONFLICT gate).
- SET-14 concurrency + fail-closed → never wallet+cash for same (order, merchant).
- SET-15 admin cash-paid once; SET-16 duplicate cash-paid idempotent (no 2nd ledger entry, no 2nd push).
- **SET-17a** cash full pre-pay refund → cash_reversed, holding nets 0, no obligation, history kept.
  **SET-17b** cash partial pre-pay return → proportional, stays cash_due, outstanding shrinks.
  **SET-17c** the **+100/−30/−70** determinism (§K.1): outstanding 0, toParties net 0, exact conservation.
- **SET-18a** cash paid then full refund → obligation=net, row stays cash_paid.
  **SET-18b** cash paid then partial return → obligation=returned portion.
- SET-19 wallet accrued then refund (B/B′) → existing path unchanged.
- SET-20 cancel before accrual → no row.
- SET-21 merchant A cash / B wallet under one owner → isolated.
- SET-22 unauthorized change/confirm blocked (IDOR/API). SET-23 admin without capability / step-up blocked.
- SET-24 audit: change + cash-paid carry actor/amount/merchant/source/from→to/timestamp/reference;
  ownership change records both owners.
- **SET-25** all FI green incl. FI-14.a..f, FI-06.a/d, and the **§E.2/§E.3 numeric conservation**
  for 100/90/10 at every stage (accrual, cash **payment** with `merchant_cash_paid` as money-out,
  pre-pay reversal full+partial, paid-then-full/partial refund obligation as receivable
  counterbalance, and **repayment via the measured offset — reduced future payout, no money-in**;
  the direct-cash-repayment path is proven absent so FI-12 assumes no repayment money-in),
  FI-05.c, FI-13.*.
- **SET-26** legacy cutover (item 1): historical `merchant_earning`, no settlement rows → re-enter
  `settleMerchant` ⇒ no second credit, no cash row, state byte-identical.
- **SET-27** cash-paid vs pre-pay reversal race (item 3, §H.2): only payment-wins (100 + oblig 30)
  or reversal-wins (reversed 30, pay 70); never overpay/negative-holding/double-reversal/missing-oblig.
- **SET-28** zero-net entitlement (item 4, §Z): 100%-commission source ⇒ no row/ledger/cash_due;
  mixed order settles the positive source only; retry no double; FI green.
- **SET-29** conflict-path validation / fail-closed (item 1, §J.2): a re-entry with a **matching**
  existing settlement → validated no-op (no double); a re-entry where method / amount / method-state
  / expected tx identity **differs** from the row (simulated corruption) → **invariant error, tx
  aborts**, never a silent success. Also the method↔state DB CHECK rejects any illegal combination.
- **SET-30** cash-merchant debt offset (item 2, §E.3): a cash merchant with an open obligation,
  next cash order ⇒ `merchant_cash_accrued` reduced by `take` via the method-aware offset, cash_due
  = due−take, obligation→0, treasury recovers `take`, no money-in; `due−take==0` ⇒ no residual row.

---

## Admin UX / Merchant APK / Notifications / Security / Files / Staging

- **Admin:** "طريقة تسوية مستحقات المتجر" [نقدي/المحفظة] default نقدي; change needs financial
  capability + step-up (new `authz/sensitive.go` entry — PATCH /merchants/{id} is not step-up'd
  today) + confirm dialog + audit; verbatim line **"يُطبّق التغيير على الطلبات الجديدة فقط، ولا
  يغيّر تسوية الطلبات السابقة."**; shows current method, outstanding cash total, cash history; safe
  "تأكيد دفع المستحق نقداً".
- **Merchant APK (Q4):** read-only "طريقة استلام المستحقات: نقدي/المحفظة"; cash → "المستحقات
  النقدية غير المسددة" (current-merchant outstanding) if safe; cannot change. Only APK change.
- **Notification (Q5):** on cash_paid, AppMerchant-only "تم تسديد مستحقاتك نقداً" + amount, to
  current owner; idempotent (tied to the state transition).
- **Security:** merchant/rep/driver/customer cannot change/confirm; IDOR + capability + step-up (SET-22/23).
- **Files:** `0146` migration; `service.go` CreateTx (sorted-ID lock + per-source snapshot),
  `economics_snapshot.go`; `transitions.go` settleMerchant (ON CONFLICT gate + branch),
  `cash_settlement.go` (accrue/reverse/pay); `treasury.go` (`toParties`); `wallet`
  (`is_cash_holding`, EnsureCashHolding, exclude from spend/withdraw); `fininv/checks.go`
  (FI-14.a..f + FI-12 holding/obligation); `server` admin handlers + `authz/sensitive.go` +
  `catalog`; `web` admin UI + i18n; `mobile/app-merchant`+`shared`; `docs/TRUTH.md` + apidoc/testtruth.
- **Staging witness:** two QA merchants (cash+wallet), real customer orders, prove the REAL
  `settleMerchant` path via QA bridge (no faked driver pickup) — **SETTLEMENT ENGINE
  LIVE/INTEGRATION WITNESS**; true end-to-end pickup stays **CROSS-APP DEPENDENCY — DRIVER**.

---

**Awaiting your FINAL line-by-line approval of Rev. 4.** No code, migration, or execution until
then. Production untouched · R1 not started · no Driver/Admin phase begun.
