# RahalGo — Customer Application Final Acceptance Master

> **Owner acceptance contract** — the single authoritative acceptance plan for the
> RahalGo Customer Android application (written at the Owner's explicit request,
> 2026-09-19).
>
> **Status of this document:** READY FOR OWNER / ChatGPT REVIEW.
> **No acceptance case has been executed under this document yet.** Test execution
> starts only after the Owner and ChatGPT approve it.

---

## 0 · How to read this document

- Sections 1–10 are the Owner's contract, preserved as given.
- Sections 11–34 are the acceptance cases, one table per group. Every row carries the
  15 required fields (§9). Execution-time fields (Actual · Device/Build · Evidence ·
  Defect · Regression) stay `—` until the row is executed.
- Section 35–37 are the release-gate and project principles.
- Section 38 is the read-only **coverage audit** that checked this document against
  the real Customer source, backend contracts, tests and known defects — and lists
  every case added to the supplied contract and every row marked `NOT_APPLICABLE`.
- Section 39 is the count ledger. **Before anyone says "Customer acceptance is
  complete", §39 must show zero open mandatory rows.**

Column legend used in every table:

| Column | Meaning |
|---|---|
| ID | Stable test ID — never renumbered; retired rows stay with status `NOT_APPLICABLE` |
| Area | Short area tag |
| Scenario | What is being proven |
| Pre | Preconditions |
| Steps | Exact steps |
| Expected | Expected result (contract) |
| Actual | Filled at execution |
| Status | `NOT_TESTED` · `PASS` · `FAIL` · `BLOCKED` · `NOT_APPLICABLE` |
| Device/Build | Filled at execution: device · Android · package · versionCode · APK sha256 |
| Net | Network condition |
| SoT | Backend / source-of-truth check (read-only query or API read) |
| Evidence | Evidence file / commit reference |
| Defect | Defect ID if failed |
| Regression | Regression test reference if a defect was fixed |
| Notes | Contract references, prior evidence, cautions |

Abbreviations: **SoT** = source of truth (Staging backend); **ON/OFF** = launch flag
state; **#1050** = protected shared witness order (never progressed by Customer
acceptance); **UIA** = uiautomator text evidence (no screenshots — project rule).

---

## 1 · Purpose

We are no longer testing the Customer app casually.

The purpose of this document is to prove, before release, that the Customer
application behaves correctly from the first second after installation until the end
of the customer's complete usable lifecycle.

Testing must be performed with three simultaneous mindsets:

1. **NORMAL CUSTOMER** — a real customer who installs the app and simply wants to use
   RahalGo.
2. **CARELESS / UNPREDICTABLE CUSTOMER** — taps buttons repeatedly; goes backward
   unexpectedly; switches screens during loading; changes addresses midway; denies
   permissions; loses connectivity; backgrounds the app; closes it; reopens it;
   changes network; makes mistakes; enters bad values; does actions in an unexpected
   order.
3. **ADVERSARIAL ENGINEER** — actively trying to find: crashes; stale state; race
   conditions; duplicated operations; authorization bypasses; client/server
   inconsistencies; silent failures; fake success; infinite loading; corrupt local
   state; invalid order creation; cross-account data leakage; security gaps; incorrect
   recovery; backend/client contract mismatches.

**Passing the happy path is NOT sufficient.**

## 2 · Core acceptance principle

Every user action must end in exactly one of these categories:

- **A. CLEAR SUCCESS**
- **B. CLEAR SAFE DENIAL**
- **C. CLEAR RECOVERABLE FAILURE**

The following are **NEVER** acceptable:

- a button that silently does nothing;
- an indefinite spinner;
- a screen frozen without explanation;
- fake success;
- duplicate order creation;
- stale data presented as authoritative live data;
- client state contradicting backend state;
- hidden failure;
- user action accepted locally but rejected silently by the server;
- a network action remaining apparently usable while the application knows it is offline;
- data from one account appearing in another account;
- security depending only on the UI hiding a button.

**The backend remains the source of truth. The UI is not a security boundary.**

## 3 · Test governance — no random work

No test failure may be "fixed quickly" without being recorded. For every discovered
defect:

1. Record the failing acceptance case.
2. Preserve evidence.
3. Give the defect a stable ID.
4. Determine the root cause.
5. Run P-9 impact analysis.
6. Determine all affected applications/surfaces.
7. Add a permanent regression test where technically possible.
8. Apply the smallest correct centralized fix.
9. Run impacted automated suites.
10. Re-run the exact failed acceptance case.
11. Run required regression cases.
12. Only then mark the defect CLOSED.
13. Continue the acceptance sequence.

For a P0/P1, security, financial, data-corruption, duplicate-order, cross-account,
authorization or source-of-truth defect: **STOP progression until it is understood and
safely resolved.**

Do not bundle unrelated fixes. Do not redesign unrelated code during acceptance.

## 4 · Document memory rule

This document is the shared memory between the Owner, ChatGPT, Claude, the repository
and future testing sessions. Nobody should rely only on remembered conversation
context.

Before saying **"Customer acceptance is complete"**, Claude must re-read this document
and prove that every mandatory row is `PASS` or `NOT_APPLICABLE` with a documented
reason. No mandatory row may remain `NOT_TESTED`, `FAIL` or `BLOCKED`.

If the Owner or ChatGPT asks to move forward while a mandatory Customer acceptance item
is still open, Claude must explicitly point out the open item. If Claude attempts to
move forward and the document shows an open item, the Owner/ChatGPT will stop the
progression.

If a new scenario is discovered during testing: **ADD IT TO THE DOCUMENT FIRST, then
test it.** Do not keep important acceptance scenarios only in chat or memory.

## 5 · Current baseline

- Current testing is **DEBUG / STAGING** acceptance.
- Current Customer debug package: **`com.rahalgo.customer.debug`**.
- Current known device acceptance phone: **Samsung SM-A525F · Android 14**.
- Record the exact device/build details again at the beginning of every formal
  acceptance execution (CUST-00).
- **Production must receive zero mutations** during Customer acceptance unless the
  Owner explicitly authorizes a specific Production action.
- Do not build final release packages during this phase. Customer final release build
  happens only after final acceptance/freeze.

## 6 · Current known acceptance state

- Admin Staging ↔ Production functional parity is **CLOSED**. Do not reopen Admin
  parity merely because Customer testing is underway.
- **P8-DEF-001** original Android connectivity race / root cause: **FIXED**
  (`cc6fe128`). Physical device evidence: offline detection = PASS · network recovery =
  PASS (`P8-MASTER-MATRIX.md`, 2026-09-19).
- **L1-019 final acceptance remains OPEN.** Current L1 truth: **20 / 21 PASS**.
- Reason: the Owner strengthened the offline product contract. The implemented
  behaviour detects offline state and shows an offline banner, but normal Customer
  interaction is not yet blocked. The acceptance evidence proved the gap: during a
  contaminated earlier device run a person changed the cart while fully offline. That
  must not be possible under the final Customer contract.

## 7 · Offline contract — mandatory (CLOSED PRODUCT REQUIREMENT)

When the application becomes truly offline:

1. The Customer app enters an explicit application-level **OFFLINE** state.
2. The user sees a clear message **«لا يوجد اتصال بالإنترنت»** and a clear action
   **«أعد المحاولة»**.
3. Network-dependent Customer interaction must be blocked.
4. The Customer must not be able to continue interacting with market data as though it
   were live.
5. While offline the Customer must not be able to: add an item to the cart; remove an
   item from the cart; change cart quantities; submit an order; run network-dependent
   refreshes as if they succeeded; execute another network-dependent mutation.
6. No silent taps.
7. No fake success.
8. No infinite spinner.
9. Previously loaded state may remain preserved internally. It does NOT have to be
   destroyed.
10. The preserved data must not remain usable as authoritative live interactive data
    during offline state.
11. When connectivity returns: verify real connectivity; refresh authoritative state;
    remove the blocking offline state; restore normal interaction; preserve safe user
    state where appropriate; require no reinstall; require no data wipe; require no
    unnecessary login.
12. A network interface existing is NOT enough. Internet/API reachability must be
    handled correctly.

This contract must receive permanent regression coverage where feasible.

## 8 · Test result states

Every test case has exactly one status: `NOT_TESTED` · `PASS` · `FAIL` · `BLOCKED` ·
`NOT_APPLICABLE`.

`NOT_APPLICABLE` requires an explanation and evidence showing why the scenario does not
exist in the Customer product. `PASS` requires evidence. A verbal "looks fine" is not
evidence.

## 9 · Required test case fields

Stable Test ID · Area · Scenario · Preconditions · Exact steps · Expected result ·
Actual result · Status · Device/build · Network condition where relevant ·
Backend/source-of-truth check where relevant · Evidence reference · Defect ID if
failed · Regression reference if a defect was fixed · Notes.

## 10 · Acceptance sequence

Customer testing proceeds broadly in this order:

	CUST-00  Environment and build identity
	CUST-01  Installation / upgrade
	CUST-02  First launch
	CUST-03  Permissions
	CUST-04  Registration
	CUST-05  OTP / verification
	CUST-06  Login / sessions / logout / reset
	CUST-07  Location and addresses
	CUST-08  Coverage and service availability
	CUST-09  Home / catalog / sections
	CUST-10  Product interaction
	CUST-11  Cart
	CUST-12  Quote / checkout
	CUST-13  Order submission
	CUST-CUSTOM  Custom order (طلب خاص)
	CUST-14  Order lifecycle / history
	CUST-15  Realtime and notifications
	CUST-16  Offline / degraded network
	CUST-17  App lifecycle / Android interruption
	CUST-18  Remote operational changes
	CUST-19  Adversarial / security
	CUST-20  UI / UX / RTL
	CUST-21  Performance / resilience
	CUST-22  Final regression / release-candidate acceptance

Do not jump randomly between groups unless a defect requires an isolated reproduction.
Groups added by the coverage audit (§38) are slotted in the sequence where their
surface lives and are listed in §38.

**Execution guardrails carried from project rules (apply to every row):**

- Evidence is text (uiautomator XML / API JSON / read-only SQL) — no screenshots.
- Physical-device runs record device serial, Android version, package, versionCode and
  installed-APK sha256 (CUST-00) and are rejected if any non-scripted touch is found in
  the device input log during a scripted run (contamination rule, 2026-09-19).
- Staging mutations only through supported application/Admin paths, recorded before
  and after; #1050 is never progressed by a Customer case.
- Production: zero mutations; read-only only when a row says so.

---

## 11 · CUST-00 — Environment and build identity

#1050 is a protected existing witness. Do not progress or close it merely to satisfy an unrelated Customer case.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-00-001 | Env | Confirm physical device model, Android version and SDK | Device on ADB (USB preferred) | `getprop ro.product.model` · `ro.build.version.release` · `ro.build.version.sdk` · `ro.serialno` | Recorded exactly; matches the device named in the run header | — | `NOT_TESTED` | — | any | — | — | — | — | Serial and model recorded, never inferred from a name (project rule: package identity from source) |
| CUST-00-002 | Env | Confirm exact Customer package ID | APK built from the accepted commit | `aapt2 dump badging <apk>` · `pm list packages --user 0 \| grep rahalgo.customer` | `com.rahalgo.customer.debug` in the APK and on the device — nothing else | — | `NOT_TESTED` | — | any | — | — | — | — | — |
| CUST-00-003 | Env | Confirm versionName and versionCode | As 002 | `dumpsys package com.rahalgo.customer.debug \| grep version` | Recorded; equals the badging of the APK under test | — | `NOT_TESTED` | — | any | — | — | — | — | — |
| CUST-00-004 | Env | Confirm APK fingerprint matches the intended tested artifact | Built APK sha256 recorded at build time | `pm path` → pull base.apk → sha256; compare with the built artifact | Installed sha256 == built sha256 (byte-identical) | — | `NOT_TESTED` | — | any | — | — | — | — | Precedent: 2026-09-19 acceptance 831ec163… |
| CUST-00-005 | Env | Confirm application points ONLY to Staging | As 004 | Search classes*.dex for `https://staging-api.rahalgo.com`; observe first API call host in logcat/proxy-free check (identity endpoint response `environment=staging`) | Only staging-api host present/used | — | `NOT_TESTED` | — | online | `/api/v1/public/identity` → staging | — | — | — | Debug default comes from `app-customer/build.gradle.kts` (`rahalgo.apiBaseUrl` override else staging) |
| CUST-00-006 | Env | Production API address not embedded/used by the Staging debug app | As 004 | Count `https://api.rahalgo.com` in classes*.dex; confirm ProductionEndpointGuardTest passes | 0 occurrences; guard test green | — | `NOT_TESTED` | — | any | — | — | — | — | Automated: `ProductionEndpointGuardTest` (app-customer unit test) |
| CUST-00-007 | Env | No temporary diagnostic instrumentation remains | As 004 | Search dex for known diagnostic tags (e.g. `RGNET`) and debug-only logging added during investigations | 0 occurrences | — | `NOT_TESTED` | — | any | — | — | — | — | P8-DEF-001 diagnostic tag `RGNET` must stay absent |
| CUST-00-008 | Env | Test app exists once in the main Android profile | Device on ADB | `pm list packages --user 0 \| grep -c rahalgo.customer` | Exactly 1 | — | `NOT_TESTED` | — | any | — | — | — | — | — |
| CUST-00-009 | Env | No unintended Dual Messenger / work-profile duplicate remains | Device on ADB | `pm list users`; for every non-zero user `pm list packages --user <n> \| grep rahalgo` | 0 RahalGo packages outside user 0 | — | `NOT_TESTED` | — | any | — | — | — | — | 2026-09-19: the Owner manually removed the 4 RahalGo apps from Dual Messenger (user 95) — DONE; ADB verification pending until the phone is reachable (§40.13) |
| CUST-00-010 | Env | Confirm Staging baseline before testing | SSH read access to Staging | Read-only: identity, migration, launch flags, #1050, order count, wallet tx count, moneycheck | Recorded; moneycheck 51/51 | — | `NOT_TESTED` | — | online | read-only SQL + `moneycheck` | — | — | — | — |
| CUST-00-011 | Env | Confirm Production baseline and zero intended Production mutation | — | Read `https://api.rahalgo.com/api/v1/public/identity` only | Recorded (release, migration); no other Production access | — | `NOT_TESTED` | — | online | identity endpoint only | — | — | — | Production mutations must stay 0 |
| CUST-00-012 | Env | Record current Customer-related launch flags | SSH read access | Read-only `app_settings` where key LIKE 'launch.%' plus `site.show_*` | Recorded verbatim | — | `NOT_TESTED` | — | online | read-only SQL | — | — | — | — |
| CUST-00-013 | Env | Record active Staging geography/coverage relevant to tests | SSH read access | Read-only: active cities, delivery zones (+hours), operational areas used by the test addresses | Recorded | — | `NOT_TESTED` | — | online | read-only SQL | — | — | — | Zone referenced by #1050 must not be modified |
| CUST-00-014 | Env | Record current test Customer account(s) | — | List test accounts by phone tail, roles, status, verification state | Recorded; no Production identities | — | `NOT_TESTED` | — | online | read-only SQL | — | — | — | Never print passwords/OTP/tokens |
| CUST-00-015 | Env | Record protected shared witnesses | — | Record #1050 status/events/snapshot and any other protected fixtures | Recorded; each later case proves it unchanged | — | `NOT_TESTED` | — | online | read-only SQL | — | — | — | #1050 is a protected witness — never progressed or closed by a Customer case |

## 12 · CUST-01 — Installation / update

Do not perform destructive install cases against important unsaved evidence without preserving the evidence first.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-01-001 | Install | Fresh installation succeeds | Customer app absent from user 0 (or on a clean device/emulator) | `adb install --user 0 <apk>` | `Success`; exactly one package | — | `NOT_TESTED` | — | any | — | — | — | — | Destructive to local data — preserve evidence first; prefer emulator for clean-device variants |
| CUST-01-002 | Install | Correct application icon/name appear | Installed | Read launcher label (`dumpsys package` / aapt2 badging `application-label`); UIA of launcher | Arabic app name and RahalGo icon; debug suffix only where designed | — | `NOT_TESTED` | — | any | — | — | — | — | — |
| CUST-01-003 | Install | First launch after fresh installation does not crash | Fresh install | Launch; wait 15 s; check `dumpsys activity` + logcat for FATAL/ANR | Signed-out entry screen; no crash/ANR | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-01-004 | Install | Updating replaces the same package (no duplicate app) | Older debug build installed with data | `adb install -r --user 0 <new apk>`; list packages all users | Same package, 1 copy, firstInstallTime unchanged | — | `NOT_TESTED` | — | any | — | — | — | — | Always `--user 0` (adb default installs for every user — caused the Dual Messenger copies) |
| CUST-01-005 | Install | Normal update preserves expected application data | As 004 with signed-in user, saved address, cart | Update; relaunch | ceDataInode unchanged; address/cart/session preserved where designed | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-01-006 | Install | Update preserves valid session where contract allows | Signed in before update | Update; relaunch | Still signed in (refresh token valid); no forced login | — | `NOT_TESTED` | — | online | `auth.refresh` in audit, no new password_login | — | — | — | — |
| CUST-01-007 | Install | Usable after update without manual storage clearing | After 004 | Browse, open cart, open orders | All screens work; no crash | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-01-008 | Install | Unsupported downgrade behaviour is safe | Newer build installed | `adb install -r` of an older versionCode (expect refusal) ; `-d` only on emulator | Android refuses (INSTALL_FAILED_VERSION_DOWNGRADE) or, with -d on emulator, app starts without corrupt state | — | `NOT_TESTED` | — | any | — | — | — | — | Never use `-d` on the Owner's phone |
| CUST-01-009 | Install | Interrupted/failed installation does not leave two Customer apps | Emulator | Abort an install mid-stream; list packages | Old install intact or absent; never two | — | `NOT_TESTED` | — | any | — | — | — | — | Emulator only |
| CUST-01-010 | Install | Insufficient-storage failure does not corrupt the existing install | Emulator with filled storage | Attempt update | Android error; previous install still launches | — | `NOT_TESTED` | — | any | — | — | — | — | Emulator only |
| CUST-01-011 | Install | Uninstall/reinstall gives the expected clean-device state | Emulator or disposable test user | Uninstall; reinstall; launch | Signed-out; no previous cart/address/session | — | `NOT_TESTED` | — | online | — | — | — | — | Destructive — preserve evidence first |

## 13 · CUST-02 — First launch

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-02-001 | Launch | Cold first launch completes | Fresh install | Force-stop; launch; UIA at +5/+15 s | Entry screen rendered | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-02-002 | Launch | No white-screen permanent hang | As 001 | UIA at +5/+15/+30 s | Non-empty UI tree by +5 s | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-02-003 | Launch | No black-screen permanent hang | As 001 | As 002 | As 002 | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-02-004 | Launch | No infinite splash/loading state | As 001 | UIA at +30 s | Content, explicit error or explicit offline — never a spinner at +30 s | — | `NOT_TESTED` | — | online / slow | — | — | — | — | — |
| CUST-02-005 | Launch | Initial route correct for a signed-out user | Signed out | Launch | Signed-out entry per contract (browse or auth screen as designed) | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-02-006 | Launch | Initial route correct for an already authenticated user | Signed in | Launch | Market (تسوق) tab with authoritative data | — | `NOT_TESTED` | — | online | `auth.refresh` audit only | — | — | — | — |
| CUST-02-007 | Launch | Initial route when Customer browsing is remotely closed | `launch.customer_browse`=OFF (Staging, via Admin, restored after) | Launch | Owner's launch notice (`launch.notice`) — not an empty market | — | `NOT_TESTED` | — | online | flag value read-only before/after | — | — | — | Mutation of a Staging flag through Admin; restore and record |
| CUST-02-008 | Launch | Initial route when signup is remotely closed | Signed out · `launch.customer_signup`=OFF | Launch; open signup | Explicit closed state from `launch_closed`; login still reachable | — | `NOT_TESTED` | — | online | flag value before/after | — | — | — | — |
| CUST-02-009 | Launch | First launch while offline follows the offline contract | Fully offline (net=none) | Launch | OFFLINE state: «لا يوجد اتصال بالإنترنت» + «أعد المحاولة»; no empty market | — | `NOT_TESTED` | — | offline | — | — | — | — | L1-019 cold path previously PASS; re-verify under §7 |
| CUST-02-010 | Launch | Internet up but API unreachable → explicit recoverable failure | Wi-Fi valid; Staging API host unreachable (see §38 harness note) | Launch | Explicit recoverable failure with retry; never «نعمل حاليًا على إضافة المتاجر والمنتجات» | — | `NOT_TESTED` | — | API unreachable | — | — | — | — | Harness: Private-DNS/hosts block of staging-api only — to be approved before use |
| CUST-02-011 | Launch | Update-required gate (added) | Installed versionCode below the server minimum (`app.min_version.customer`) | Raise the minimum (Admin, Staging); make any authenticated call | Full-screen UpdateGate «تحديث الآن» (Play → rahalgo.com/app); BACK swallowed | — | `NOT_TESTED` | — | online | HTTP 426 `update_required` | — | — | — | Added: `ui/UpdateGate.kt`; `/public/*` is exempt from the 426 check |

## 14 · CUST-03 — Android permissions

Test every permission actually requested by the Customer source (audit §38.2: POST_NOTIFICATIONS, ACCESS_COARSE_LOCATION, ACCESS_FINE_LOCATION; no camera/storage permission — the avatar uses system pickers).

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-03-001 | Perm | Grant requested permission | Fresh install (startup asks POST_NOTIFICATIONS + ACCESS_FINE_LOCATION once — `ui/StartupPermissions.kt`) | First launch; grant both | Both granted; app continues; location used for discovery only | — | `NOT_TESTED` | — | online | — | — | — | — | Manifest: INTERNET, ACCESS_NETWORK_STATE, POST_NOTIFICATIONS, ACCESS_COARSE/FINE_LOCATION (no background location) |
| CUST-03-002 | Perm | Deny requested permission | Fresh install | First launch; deny both | App fully usable; no re-prompt loop (flag `asked_startup_v1`) | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-03-003 | Perm | Deny twice / don't ask again | Location denied once | Tap «موقعي» in map; deny again | Explanation + fix action (open Settings); no silent failure | — | `NOT_TESTED` | — | online | — | — | — | — | `ui/Locating.kt`, `map/PickPoint.kt:250-313` |
| CUST-03-004 | Perm | Required permission is explained, not silently failing | Location denied | Tap «موقعي» | Reason text + action shown | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-03-005 | Perm | Reduced mode when permission is optional | Location + notifications denied | Browse, add address by map search, order | Full ordering works (address chosen manually) | — | `NOT_TESTED` | — | online | order created (disposable) | — | — | — | Location is optional: delivery address is authoritative |
| CUST-03-006 | Perm | Re-enable permission from Settings while backgrounded | Location denied | Background; grant in Settings; return | «موقعي» now works without restart | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-03-007 | Perm | Return to app — state updates | After 006 | Open Account tab (re-reads location) | Discovery updates; no crash | — | `NOT_TESTED` | — | online | — | — | — | — | `MainActivity:489-494` |
| CUST-03-008 | Perm | Revoke permission while running/backgrounded | Location granted | Background; revoke; return | No crash (Android restarts process — handled as process death) | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-03-009 | Perm | No crash after revocation | After 008 | Use map/«موقعي» | Explained denial; no crash | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-03-010 | Perm | Notification denial does not break ordering | Notifications denied | Place disposable order | Order works; no push (in-app realtime still updates) | — | `NOT_TESTED` | — | online | order +1 | — | — | — | — |
| CUST-03-011 | Perm | Location services OFF distinguished from permission denial | Permission granted · OS location OFF | Tap «موقعي» | Message says location is OFF (not 'denied') | — | `NOT_TESTED` | — | online | — | — | — | — | LOC-2 profile |
| CUST-03-012 | Perm | Approximate location is safe | Grant 'approximate' only | Launch; tap «موقعي» | Discovery ignored when accuracy > 500 m (`CONFIRM_M`); no wrong serviceability | — | `NOT_TESTED` | — | online | — | — | — | — | `customer/Here.kt` |
| CUST-03-013 | Perm | Precise location correct | Precise granted | Tap «موقعي» inside Raqqa | Pin at device position; reverse-geocoded label | — | `NOT_TESTED` | — | online | `/api/v1/geo/reverse` 200 | — | — | — | — |
| CUST-03-014 | Perm | Camera/gallery for profile photo (added) | Signed-in test customer · Staging · SM-A525F | Account → photo → camera, then gallery | System picker/camera works without extra runtime permission prompt beyond Android's; photo uploads | — | `NOT_TESTED` | — | online | `/me/avatar` 200; media row | — | — | — | Added by audit: avatar uses system pickers (`ui/ImagePick.kt`) |

## 15 · CUST-04 — Customer registration

Audited registration contract: phone → (code when `auth.signup_verify`=true) → full name, password, confirm password, optional referral code. No invented fields.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-04-001 | Signup | Valid Customer signup | Signed out (guest) · Staging · SM-A525F · `auth.signup_verify`=true (Staging = Production policy) | إنشاء حساب → phone → code (Staging dev OTP from staging log) → name + password + confirm (+ optional ref) → confirm | Account created; signed in; lands on تسوق | — | `NOT_TESTED` | — | online | users +1; `auth.signup` audit | — | — | — | Fields from `ui/SignupScreen.kt`: phone, code, full name, password, confirm, referral code |
| CUST-04-002 | Signup | Signup when launch.customer_signup is OFF | Signed out (guest) · Staging · SM-A525F · flag OFF (Admin) | Attempt signup | 503 `launch_closed` + Owner notice; no account | — | `NOT_TESTED` | — | online | users unchanged | — | — | — | Client ignores the flag (`Serving.signupOpen` unused). With `signup_verify`=true the app calls the gated `/signup/request`; with false it goes straight to the ungated `/signup/confirm` (CAF-01) — expected FAIL in that mode |
| CUST-04-003 | Signup | launch.customer_signup turns OFF while screen is open | Signup open at details step · flag flipped OFF | Submit | Explicit `launch_closed`; no account; no spinner | — | `NOT_TESTED` | — | online | users unchanged | — | — | — | — |
| CUST-04-004 | Signup | Empty required fields | Signed out (guest) · Staging · SM-A525F | Submit with empty phone/name/password | Button disabled or explicit field error; no request | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-04-005 | Signup | Malformed phone number | Signed out (guest) · Staging · SM-A525F | Phone `09x`, letters | Explicit invalid-phone error | — | `NOT_TESTED` | — | online | — | — | — | — | `identity.NormalizePhone` |
| CUST-04-006 | Signup | Unsupported phone format | Signed out (guest) · Staging · SM-A525F | Non-Syrian (+44…) / landline | Explicit invalid-phone error | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-04-007 | Signup | Canonical Syrian phone handling | Signed out (guest) · Staging · SM-A525F | Enter 09…, 9639…, +9639…, 009639… | All normalize to +9639…; same account | — | `NOT_TESTED` | — | online | phone stored canonical | — | — | — | — |
| CUST-04-008 | Signup | Leading/trailing whitespace | Signed out (guest) · Staging · SM-A525F | Phone/name with spaces | Trimmed; accepted | — | `NOT_TESTED` | — | online | stored trimmed | — | — | — | — |
| CUST-04-009 | Signup | Repeated submit taps | Signed out (guest) · Staging · SM-A525F | Triple-tap final submit | One account; one session | — | `NOT_TESTED` | — | online | users +1 exactly | — | — | — | — |
| CUST-04-010 | Signup | Slow signup API | Signed out (guest) · Staging · SM-A525F | Slow network; submit | Loading then result; no duplicate | — | `NOT_TESTED` | — | slow | users +1 | — | — | — | — |
| CUST-04-011 | Signup | Network lost during signup submission | Signed out (guest) · Staging · SM-A525F | Cut network after submit | Explicit failure; retry safe; no duplicate account | — | `NOT_TESTED` | — | cut | users +0/+1 | — | — | — | — |
| CUST-04-012 | Signup | Server validation failure | Signed out (guest) · Staging · SM-A525F | Weak password (< `security.password_min_length`) | Explicit `weak_password` message | — | `NOT_TESTED` | — | online | — | — | — | — | Client ignores `password_min_length` — server is the guard |
| CUST-04-013 | Signup | Existing phone/account | Signed out (guest) · Staging · SM-A525F | Signup with an existing phone | Explicit 'already registered' → login path | — | `NOT_TESTED` | — | online | no new user | — | — | — | — |
| CUST-04-014 | Signup | Backend 4xx shown meaningfully | Signed out (guest) · Staging · SM-A525F | Trigger each signup 4xx (bad code, taken phone, weak password, rate limit) | Mapped Arabic text for each | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-04-015 | Signup | Backend 5xx recoverable | Signed out (guest) · Staging · SM-A525F | Fault injection (harness) | Recoverable failure | — | `NOT_TESTED` | — | online | — | — | — | — | Needs approved harness |
| CUST-04-016 | Signup | No duplicate accounts on repeated submission | Signed out (guest) · Staging · SM-A525F | Replay confirm | Second confirm rejected (code consumed); one account | — | `NOT_TESTED` | — | online | users +1 | — | — | — | — |
| CUST-04-017 | Signup | Back navigation during registration | Signed out (guest) · Staging · SM-A525F | BACK at each step | Returns safely; no half account | — | `NOT_TESTED` | — | online | users unchanged | — | — | — | — |
| CUST-04-018 | Signup | Kill/reopen during incomplete registration | Signed out (guest) · Staging · SM-A525F | Kill at details step; reopen | Guest shell; no half account | — | `NOT_TESTED` | — | online | users unchanged | — | — | — | — |
| CUST-04-019 | Signup | Signup with signup_verify=false (added) | Signed out (guest) · Staging · SM-A525F · flag false (Staging only, Owner-approved) | Phone → details directly (button «متابعة») | Account created without code; later ordering rules per contract | — | `NOT_TESTED` | — | online | users +1 | — | — | — | Added: client skips the code step when false (`AuthViewModel:419-519`). Production policy is true — test only if Owner approves a temporary flip |
| CUST-04-020 | Signup | Referral code at signup (added) | Signed out (guest) · Staging · SM-A525F · valid referral code | Signup with ref (typed or pre-filled) | Referral attached; invite counts update for the referrer | — | `NOT_TESTED` | — | online | `referrals` row | — | — | — | Added: ref field + invite link `https://rahalgo.com/signup?ref=` + Play install referrer |
| CUST-04-021 | Signup | Invalid referral code (added) | Signed out (guest) · Staging · SM-A525F | Signup with unknown ref | Account created; ref ignored or explicit notice; never blocks signup | — | `NOT_TESTED` | — | online | no referral row | — | — | — | — |
| CUST-04-022 | Signup | Invite deep link opens signup with code (added) | Signed out (guest) · Staging · SM-A525F | `adb shell am start -d 'https://rahalgo.com/signup?ref=CODE'` | Signup opens with the code pre-filled | — | `NOT_TESTED` | — | online | — | — | — | — | Only `/signup` is in the intent filter; `/i/CODE` parsed but not routed |
| CUST-04-023 | Signup | Signup confirm is launch-gated and rate-limited (added) | `launch.customer_signup`=OFF | API client: POST `/auth/signup/confirm` directly | 503 `launch_closed`; repeated calls rate-limited | — | `NOT_TESTED` | — | online | users unchanged | — | — | — | Added. CAF-01: only `/signup/request` is gated; `/signup/confirm` has no launch gate and no rate limit — expected FAIL |

## 16 · CUST-05 — OTP / verification

Use the actual Staging OTP mechanism (dev provider → Staging API log). Never print real Production credentials or secrets.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-05-001 | OTP | Correct OTP | Signed out (guest) · Staging · SM-A525F · signup code step | Request code; read from Staging API log (dev provider); enter | Verified; proceeds | — | `NOT_TESTED` | — | online | otp row consumed at confirm | — | — | — | Never print the code in reports |
| CUST-05-002 | OTP | Incorrect OTP | As 001 | Enter wrong code | Explicit invalid-code error | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-05-003 | OTP | Expired OTP | As 001 | Wait past OTP TTL; enter | Explicit expired/invalid; resend available | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-05-004 | OTP | Used OTP cannot be replayed | Completed signup | Replay confirm with the same code (API client) | Rejected | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-05-005 | OTP | Resend OTP | As 001 | Resend | New code issued; old invalid | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-05-006 | OTP | Repeated resend attempts | As 001 | Resend ×N | Rate limit reached → explicit message | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-05-007 | OTP | Rate-limited OTP gives explicit feedback | As 006 | Observe | `otp_rate_limited` mapped text; recovers after window | — | `NOT_TESTED` | — | online | — | — | — | — | RATE-1 |
| CUST-05-008 | OTP | OTP screen background/foreground | As 001 | HOME; return | Step and phone preserved | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-05-009 | OTP | Kill/reopen during OTP flow | As 001 | Kill; reopen | Safe restart of the flow; no half account | — | `NOT_TESTED` | — | online | users unchanged | — | — | — | — |
| CUST-05-010 | OTP | Network loss before OTP request | As 001 · offline | Request code | Blocked/explicit offline; no request | — | `NOT_TESTED` | — | offline | — | — | — | — | — |
| CUST-05-011 | OTP | Network loss after request, before verification | Code requested · then offline | Enter code | Explicit offline; code still valid after recovery | — | `NOT_TESTED` | — | offline→online | — | — | — | — | — |
| CUST-05-012 | OTP | Network restored and flow recovers | After 011 | Restore; verify | Verification succeeds | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-05-013 | OTP | OTP for one flow/account cannot verify another | Two phones | Use A's code for B / reset code for signup | Rejected (purpose + phone bound) | — | `NOT_TESTED` | — | online | — | — | — | — | OTP hash is bound to phone + purpose (`identity/service.go`) |
| CUST-05-014 | OTP | Verification produces only the intended account/session | After 001 | Inspect sessions | One user, one android-customer session | — | `NOT_TESTED` | — | online | refresh_tokens for user = 1 android-customer | — | — | — | — |
| CUST-05-015 | OTP | OTP login when otp_login=true (added) | Signed out (guest) · Staging · SM-A525F · `auth.otp_login`=true (Staging only, Owner-approved) | OTP tab → request → verify | Signed in without password | — | `NOT_TESTED` | — | online | `auth.otp_login` audit | — | — | — | Added: tab shown only when `/public/platform` says otp_login=true. Production policy is false |
| CUST-05-016 | OTP | OTP login tab hidden when otp_login=false (added) | Signed out (guest) · Staging · SM-A525F · flag false (current policy) | Open login | No OTP tab; password login only | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-05-017 | OTP | WhatsApp account verification from Account screen (added) | Signed-in test customer · Staging · SM-A525F · unverified | حسابي → «وثق حسابك» → WhatsApp ticket → code → confirm | «حسابك موثق»; `whatsapp_verified_at` set | — | `NOT_TESTED` | — | online | users.whatsapp_verified_at | — | — | — | Added: `/auth/wa/ticket` purpose=verify + `/auth/whatsapp/confirm`. Needs the Staging WhatsApp bot — BLOCKED if the bot is not paired |

## 17 · CUST-06 — Login / session / logout / password

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-06-001 | Auth | Valid login | Signed out (guest) · Staging · SM-A525F | Password login | Signed in; تسوق | — | `NOT_TESTED` | — | online | `auth.password_login` audit | — | — | — | — |
| CUST-06-002 | Auth | Incorrect password | Signed out (guest) · Staging · SM-A525F | Wrong password | Explicit invalid-credentials message | — | `NOT_TESTED` | — | online | `auth.password_failed` audit | — | — | — | — |
| CUST-06-003 | Auth | Unknown phone | Signed out (guest) · Staging · SM-A525F | Unregistered phone | Same generic invalid-credentials message (no account enumeration) | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-06-004 | Auth | Whitespace/canonical phone | Signed out (guest) · Staging · SM-A525F | Login with 09… / +963… / spaces | All succeed for the same account | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-06-005 | Auth | Repeated login tap | Signed out (guest) · Staging · SM-A525F | Triple-tap login | One session; UI consistent | — | `NOT_TESTED` | — | online | one new android-customer session | — | — | — | — |
| CUST-06-006 | Auth | Slow login response | Signed out (guest) · Staging · SM-A525F | Slow network | Loading then result | — | `NOT_TESTED` | — | slow | — | — | — | — | — |
| CUST-06-007 | Auth | Network loss during login | Signed out (guest) · Staging · SM-A525F | Cut after tap | Explicit failure; retry works | — | `NOT_TESTED` | — | cut | — | — | — | — | — |
| CUST-06-008 | Auth | Successful login lands on correct surface | Signed out (guest) · Staging · SM-A525F | Login | تسوق tab; cart/addresses of this account | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-06-009 | Auth | Access token refresh during use | Signed-in test customer · Staging · SM-A525F | Use > 15 min | Silent refresh; no interruption | — | `NOT_TESTED` | — | online | `auth.refresh` audit | — | — | — | Access TTL 15 min |
| CUST-06-010 | Auth | Expired access token with valid refresh recovers | Signed-in test customer · Staging · SM-A525F | Background > 15 min; act | Action succeeds after silent refresh | — | `NOT_TESTED` | — | online | — | — | — | — | `shared/net/ApiClient.kt:147-176` |
| CUST-06-011 | Auth | Invalid/revoked session → re-authentication | Signed-in test customer · Staging · SM-A525F | Revoke session server-side (password reset of the test account); use app | Explicit re-login path; no loop; no stale private data | — | `NOT_TESTED` | — | online | — | — | — | — | Audit risk: mid-session refresh failure shows «انتهت جلستك — ادخل من جديد» but does not sign out (no global 401 → logout) |
| CUST-06-012 | Auth | Logout removes access | Signed-in test customer · Staging · SM-A525F | Drawer → «خروج» | Signed out; server session revoked | — | `NOT_TESTED` | — | online | refresh token revoked; `auth.logout` audit | — | — | — | Logout has no confirmation |
| CUST-06-013 | Auth | BACK cannot reopen authenticated screens after logout | After 012 | Press BACK repeatedly | No authenticated screen reappears | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-06-014 | Auth | Restart after logout stays logged out | After 012 | Force-stop; relaunch | Guest shell | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-06-015 | Auth | Login as another customer exposes nothing of the previous account | A logged out | Login as B | No A cart/orders/wallet/inbox/favorites/chats | — | `NOT_TESTED` | — | online | — | — | — | — | CUST-DEF-004 (STOP, §40.6): cart not cleared, account-scoped view models not reset, socket not stopped (B reuses A's socket) — expected FAIL |
| CUST-06-016 | Auth | Password-reset flow (exposed) | Signed out (guest) · Staging · SM-A525F | نسيت كلمة المرور → phone → WhatsApp ticket → code → new password | Password changed; signed in | — | `NOT_TESTED` | — | online | `auth.password_reset` audit | — | — | — | App uses `/auth/wa/ticket` purpose=reset (WhatsApp), not SMS. Needs the Staging WhatsApp bot — BLOCKED if not paired |
| CUST-06-017 | Auth | Reset invalidates old sessions | Signed in on device + second client | Reset from one; use the other | Other session rejected | — | `NOT_TESTED` | — | online | refresh tokens revoked | — | — | — | Contract SEC8: reset revokes all sessions |
| CUST-06-018 | Auth | Old password fails after reset | After 016 | Login with old password | Rejected | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-06-019 | Auth | New password succeeds | After 016 | Login with new password | Signed in | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-06-020 | Auth | Suspended account: new login | Test account suspended via Admin | Login | Explicit suspended message; no session | — | `NOT_TESTED` | — | online | users.status | — | — | — | — |
| CUST-06-021 | Auth | Suspended account: existing session behaviour | Signed in · then suspended via Admin | Continue using app | Matches backend contract (refresh/requests rejected as the contract says) | — | `NOT_TESTED` | — | online | — | — | — | — | Confirm contract from backend before execution |
| CUST-06-022 | Auth | Multiple active devices/sessions | Second device/emulator | Login on both | Per contract: same-client login revokes the previous family (`revokeClientSessions`) — verify which | — | `NOT_TESTED` | — | online | sessions per client | — | — | — | — |
| CUST-06-023 | Auth | Change password from Account (added) | Signed-in test customer · Staging · SM-A525F | حسابي → current + new + confirm | Changed; explicit success; wrong current → explicit error | — | `NOT_TESTED` | — | online | `auth.password_change` audit | — | — | — | Added: Account screen feature |
| CUST-06-024 | Auth | Change phone number (added) | Signed-in test customer · Staging · SM-A525F | حسابي → change phone → code → confirm | Phone changed; login with new phone works | — | `NOT_TESTED` | — | online | users.phone | — | — | — | Added: `/auth/phone/request` + `/auth/phone/confirm` |
| CUST-06-025 | Auth | Edit name and profile photo (added) | Signed-in test customer · Staging · SM-A525F | Edit name; upload/remove photo | Saved; shown everywhere | — | `NOT_TESTED` | — | online | users.full_name / avatar | — | — | — | Added: `PATCH /me/name`, `POST /me/avatar` |
| CUST-06-026 | Auth | Account deletion (added) | Disposable test account | حسابي → delete → code → confirm | Account deleted; signed out; blockers (wallet_not_empty / open_orders / cash_not_settled) shown explicitly when present | — | `NOT_TESTED` | — | online | user status/deletion row | — | — | — | Added: `/auth/account/delete/request\|confirm`. Destructive — disposable account only |
| CUST-06-027 | Auth | Forced password change (password_change_required) (added) | Account flagged must-change (Admin reset) | Login; act | App routes the user to change the password | — | `NOT_TESTED` | — | online | — | — | — | — | Added: known CONTRACT_MISMATCH — no dedicated client flow; only an error text (`ApiErrors.kt:359`). Expected FAIL until built |
| CUST-06-028 | Auth | Startup session restore fails on network (added) | Signed-in test customer · Staging · SM-A525F · offline at cold start | Launch | OfflineScreen with retry (restore); session kept; recovers on retry | — | `NOT_TESTED` | — | offline | — | — | — | — | Added: `AuthGate` offline branch (`ui/AppFrame.kt:310`) |
| CUST-06-029 | Auth | Startup restore with rejected session wipes it (added) | Session revoked server-side · app killed | Launch | Session cleared; guest shell or login; no crash | — | `NOT_TESTED` | — | online | — | — | — | — | Added: `sessionRejected` on 401/invalid_refresh |
| CUST-06-030 | Auth | Guest browsing and NeedAccount gates (added) | Signed out (guest) · Staging · SM-A525F | Browse تسوق; open طلب خاص; tap + on an item; heart | Browse works; custom → «هذا القسم يحتاج حسابا»; + and heart → login | — | `NOT_TESTED` | — | online | — | — | — | — | Added: signed-out users browse by default (guest shell) |
| CUST-06-031 | Auth | Suspended customer and live orders (added) | Customer with an open order · suspended via Admin | Open app | Per contract: can still see and cancel the live order (suspension exceptions) | — | `NOT_TESTED` | — | online | — | — | — | — | Added. CAF-04 (reported by audit, to verify): exceptions list `GET /orders/{id}` which the app never calls; `/my/orders*` and chat are blocked; `/auth/me` 403 at start shows 'offline' |
| CUST-06-032 | Auth | Session-level 403 codes are not shown as network failures (added) | Temporary password / blocked / suspended account | Launch and act | Explicit account-state message, never «لا اتصال» | — | `NOT_TESTED` | — | online | — | — | — | — | Added. CAF-10: 403 on `/auth/me` at startup renders the offline state |

## 18 · CUST-07 — Location / address

**Critical contract: the selected delivery address is authoritative for serviceability. GPS/current device position is provisional assistance only.**

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-07-001 | Addr | Location permission granted | Signed-in test customer · Staging · SM-A525F · permission granted | Open map picker; «موقعي» | Pin moves to device position | — | `NOT_TESTED` | — | online | — | — | — | — | Contract: selected delivery address is authoritative; GPS is assistance only |
| CUST-07-002 | Addr | Location permission denied | Signed-in test customer · Staging · SM-A525F · denied | «موقعي» | Explained denial; manual pin/search still works | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-07-003 | Addr | GPS/location service disabled | Signed-in test customer · Staging · SM-A525F · OS location OFF | «موقعي» | Explicit 'location off' message | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-07-004 | Addr | Current position resolves normally | Signed-in test customer · Staging · SM-A525F | «موقعي» | Fix within 15 s | — | `NOT_TESTED` | — | online | — | — | — | — | `Here.kt` two-stage fix, 15 s timeout |
| CUST-07-005 | Addr | Location lookup times out | Signed-in test customer · Staging · SM-A525F · indoors/no fix | «موقعي» | Explicit timeout; manual path available | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-07-006 | Addr | Map/geocoding unavailable | Signed-in test customer · Staging · SM-A525F · maps host blocked (harness) | Open picker; search | Explicit failure; no crash; can retry | — | `NOT_TESTED` | — | maps down | — | — | — | — | Staging maps served from staging-api `/maps/` |
| CUST-07-007 | Addr | Manual recovery where contract permits | After 005/006 | Search by name / move pin | Address can still be saved | — | `NOT_TESTED` | — | online | `/geo/search` 200 | — | — | — | — |
| CUST-07-008 | Addr | Valid delivery address selected | Signed-in test customer · Staging · SM-A525F | Add address in Raqqa coverage; make default | Top chip shows kind; Shop availability = service_available | — | `NOT_TESTED` | — | online | user_addresses row; `/public/availability` | — | — | — | — |
| CUST-07-009 | Addr | Multiple saved addresses | Signed-in test customer · Staging · SM-A525F | Add up to `customers.max_addresses` (4) | All listed; one default | — | `NOT_TESTED` | — | online | rows = 4 | — | — | — | — |
| CUST-07-010 | Addr | Change selected address | Signed-in test customer · Staging · SM-A525F · ≥2 addresses | Top chip → pick another | Becomes default; availability re-evaluated | — | `NOT_TESTED` | — | online | default flag moved | — | — | — | Picking in the sheet calls make-default |
| CUST-07-011 | Addr | Delete address | Signed-in test customer · Staging · SM-A525F | حسابي → delete | Removed | — | `NOT_TESTED` | — | online | row deleted | — | — | — | No confirmation dialog today — note for UX review |
| CUST-07-012 | Addr | Edit address | Signed-in test customer · Staging · SM-A525F | حسابي → edit → change street | Saved | — | `NOT_TESTED` | — | online | row updated | — | — | — | — |
| CUST-07-013 | Addr | Continue without a required address | Signed-in test customer · Staging · SM-A525F · no address | Tap + / open cart / send | Address sheet opens; send disabled with «اختر عنوان التوصيل لتظهر الأجرة» | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-07-014 | Addr | Address outside coverage | Signed-in test customer · Staging · SM-A525F | Save address outside zones | `address_outside_coverage` explicit; add/send blocked | — | `NOT_TESTED` | — | online | availability reason | — | — | — | — |
| CUST-07-015 | Addr | Address in unsupported province | Signed-in test customer · Staging · SM-A525F | Address in another governorate | `province_not_supported` or `city_not_supported` per data | — | `NOT_TESTED` | — | online | — | — | — | — | P8-L1-013 prior evidence (API) |
| CUST-07-016 | Addr | Address in unsupported city | Signed-in test customer · Staging · SM-A525F | Address in Damascus/Aleppo | `city_not_supported` | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-07-017 | Addr | Address in unsupported area/zone | Signed-in test customer · Staging · SM-A525F | Address in an uncovered area | `area_not_supported` / `address_outside_coverage` | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-07-018 | Addr | Address near a coverage boundary | Signed-in test customer · Staging · SM-A525F | Points just inside/outside a zone edge | Inside serviceable; outside denied — consistent with server | — | `NOT_TESTED` | — | online | availability per point | — | — | — | — |
| CUST-07-019 | Addr | GPS inside coverage, selected address outside → address wins | Signed-in test customer · Staging · SM-A525F | Default address outside; device inside | Denied for the address; discovery text prefixed «موقعك الحالي:» never overrides | — | `NOT_TESTED` | — | online | — | — | — | — | `PreCart.kt:285-379` |
| CUST-07-020 | Addr | GPS outside coverage, selected address valid → ordering allowed | Signed-in test customer · Staging · SM-A525F | Default address inside; device outside | Ordering allowed to the address | — | `NOT_TESTED` | — | online | order created (disposable) | — | — | — | — |
| CUST-07-021 | Addr | Change address with items in cart | Signed-in test customer · Staging · SM-A525F · cart populated | Switch address | Quote/availability re-evaluated; out-of-zone note if needed | — | `NOT_TESTED` | — | online | `/public/quote` | — | — | — | — |
| CUST-07-022 | Addr | Coverage changes while the address is on screen | Signed-in test customer · Staging · SM-A525F · zone edited via Admin | Wait/refresh | Availability updates; send blocked if now outside | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-07-023 | Addr | Address becomes invalid before checkout | As 022 | Tap «أرسل الطلب» | Server denies explicitly; no order | — | `NOT_TESTED` | — | online | order count unchanged | — | — | — | — |
| CUST-07-024 | Addr | Restart preserves only appropriate address state | Signed-in test customer · Staging · SM-A525F | Kill; relaunch | Default address from server; discovery not persisted as address | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-07-025 | Addr | Address limit reached (added) | Signed-in test customer · Staging · SM-A525F · 4 addresses (`customers.max_addresses`=4) | Add a 5th | Explicit `too_many_addresses` message | — | `NOT_TESTED` | — | online | rows stay 4 | — | — | — | Added: no client limit; server error only |
| CUST-07-026 | Addr | Map search and reverse geocode (added) | Signed-in test customer · Staging · SM-A525F | Search ≥2 chars; move pin | Top 4 results; label filled after settle | — | `NOT_TESTED` | — | online | `/geo/search`, `/geo/reverse` | — | — | — | Added: guests cannot call geo (auth-only) — see 07-028 |
| CUST-07-027 | Addr | Address form validation (added) | Signed-in test customer · Staging · SM-A525F | Save with missing area/street; non-digit floor | Save disabled until lat/lng + area + street; floor digits only | — | `NOT_TESTED` | — | online | — | — | — | — | Added |
| CUST-07-028 | Addr | Guest cannot reach geo endpoints / address book (added) | Signed out (guest) · Staging · SM-A525F | Open cart address card; try map search | Explicit login path; no silent failure | — | `NOT_TESTED` | — | online | — | — | — | — | Added: audit found the cart address card is not guest-gated while geo calls are auth-only |
| CUST-07-029 | Addr | Browse city picker (added) | Signed-in test customer · Staging · SM-A525F | Drawer header «تتسوّق في …» → choose city / «تلقائيّاً حسب موقعي» | Catalog scoped to the chosen city; persisted | — | `NOT_TESTED` | — | online | requests carry lat/lng of the city | — | — | — | Added: `CityPicker.kt`, `CityScope.kt` |
| CUST-07-030 | Addr | Request my area / notify me (demand) (added) | Signed-in test customer · Staging · SM-A525F · address outside coverage | Tap the demand button | Explicit confirmation; one demand row | — | `NOT_TESTED` | — | online | demand row +1 | — | — | — | Added: `POST /api/v1/demand`; not offered for discovery points |

## 19 · CUST-08 — Availability / coverage

Verify actual server reason handling. Authoritative vocabulary (`orders/availability.go`): service_available · launch_closed · temporarily_unavailable · platform_closed_now · invalid_location · coverage_unavailable · province_not_supported · city_not_supported · area_not_supported · address_outside_coverage · zone_closed_now · merchant_closed_now. Do not invent substitute reason strings.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-08-001 | Avail | service_available | Signed-in test customer · Staging · SM-A525F · valid address | Open Shop/Cart | Ordering allowed | — | `NOT_TESTED` | — | online | `/public/availability` reason | — | — | — | Reason vocabulary from `orders/availability.go`; rendered by `ui/ServiceReason.kt` (12 reasons) |
| CUST-08-002 | Avail | launch_closed | `launch.customer_orders`=OFF | Send | Owner notice text; no order | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-08-003 | Avail | temporarily_unavailable | Platform closure (Admin) | Open Cart | Explicit note with Owner text/«نعود الساعة…»; send disabled | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-08-004 | Avail | platform_closed_now | Platform hours closed | Open Cart | Explicit; next_available_at shown | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-08-005 | Avail | invalid_location | API client (bad lat/lng) | `/public/availability?lat=999` | `invalid_location` handled; app never sends it in normal use | — | `NOT_TESTED` | — | online | — | — | — | — | P8-L1-016 prior evidence |
| CUST-08-006 | Avail | coverage_unavailable | Geography without coverage data | Open Shop | Explicit | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-08-007 | Avail | province_not_supported | Address in unsupported province | Open Shop/Cart | Explicit | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-08-008 | Avail | city_not_supported | Address in unsupported city | Open Shop/Cart | Explicit | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-08-009 | Avail | area_not_supported | Address in unsupported area | Open Shop/Cart | Explicit | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-08-010 | Avail | address_outside_coverage | Address outside zones | Open Cart | Out-of-zone note; send disabled | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-08-011 | Avail | zone_closed_now | Zone hours closed | Open Cart | Zone-closed note; send disabled | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-08-012 | Avail | merchant_closed_now | Source store closed by hours | View item | «المتجر مغلق حالياً» overlay; + hidden | — | `NOT_TESTED` | — | online | — | — | — | — | Structure stays (Owner decision 2026-09-16) |
| CUST-08-013 | Avail | Availability changes while browsing | Signed-in test customer · Staging · SM-A525F | Close zone via Admin; wait for realtime/refresh | UI updates to the new reason | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-08-014 | Avail | Availability changes after items entered cart | Signed-in test customer · Staging · SM-A525F · cart populated | Close zone; open Cart | Note shown; send disabled | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-08-015 | Avail | Availability changes immediately before submit | Signed-in test customer · Staging · SM-A525F · review ready | Close zone; tap send | Server denies explicitly; no order | — | `NOT_TESTED` | — | online | order count unchanged | — | — | — | — |
| CUST-08-016 | Avail | Availability/API failure never becomes a fake empty market | Signed-in test customer · Staging · SM-A525F | Offline / API blocked | Error/offline state — never the empty-market text | — | `NOT_TESTED` | — | offline / API down | — | — | — | — | L1-018/019 |
| CUST-08-017 | Avail | Pre-launch screen when browsing is closed (added) | Signed out (guest) · Staging · SM-A525F and Signed-in test customer · Staging · SM-A525F · `launch.customer_browse`=OFF | Open app | PreLaunch screen with `launch.notice`; guests see login button; Account tab still reachable | — | `NOT_TESTED` | — | online | — | — | — | — | Added: `MainActivity:1221-1252` |
| CUST-08-018 | Avail | Owner notice overrides built-in reason text (added) | Error with `details.notice` | Trigger launch_closed / temporarily_unavailable / zone_closed_now | Owner's text shown instead of the default | — | `NOT_TESTED` | — | online | — | — | — | — | Added: `ApiErrors.kt:118-134` |

## 20 · CUST-09 — Home / market / catalog

The Customer product presents a catalog (sections → items); stores are deliberately hidden. No separate public merchant-store browsing model exists.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-09-001 | Market | Home loads normally | Signed-in test customer · Staging · SM-A525F · valid default address | Open تسوق | Banners, search, section rail, items grid | — | `NOT_TESTED` | — | online | `/public/home` 200 | — | — | — | Home = Shop tab (`ShopScreen.kt`) |
| CUST-09-002 | Market | Sections load | Signed-in test customer · Staging · SM-A525F · valid default address | Observe rail | Sections with content shown | — | `NOT_TESTED` | — | online | `/public/home` sections | — | — | — | — |
| CUST-09-003 | Market | Correct active sections appear | Signed-in test customer · Staging · SM-A525F · valid default address | Compare rail vs SoT | Only active sections with count>0 for this city | — | `NOT_TESTED` | — | online | platform_sections active + counts | — | — | — | 38-section launch catalog (0156) |
| CUST-09-004 | Market | Inactive/retired sections not shown | Signed-in test customer · Staging · SM-A525F · valid default address | Compare with retired list | None of the 6 retired starters / inactive sections | — | `NOT_TESTED` | — | online | active=false rows | — | — | — | — |
| CUST-09-005 | Market | Section ordering correct | Signed-in test customer · Staging · SM-A525F · valid default address | Read rail order | Matches `sort_order` | — | `NOT_TESTED` | — | online | sort_order | — | — | — | — |
| CUST-09-006 | Market | Open section | Signed-in test customer · Staging · SM-A525F · valid default address | Tap a section chip | Its items only | — | `NOT_TESTED` | — | online | `/public/sections/{id}/items` | — | — | — | P8-C3-015 prior DEVICE_VERIFIED |
| CUST-09-007 | Market | Return without losing position/state | Signed-in test customer · Staging · SM-A525F · valid default address | Scroll; switch tab; return | Same section and position where designed | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-09-008 | Market | Products/items load | Signed-in test customer · Staging · SM-A525F · valid default address | Open section | Grid of items with price pills | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-09-009 | Market | Available product displays correctly | Signed-in test customer · Staging · SM-A525F · valid default address | Inspect card | Image, price, discount chip, + button | — | `NOT_TESTED` | — | online | price == SoT | — | — | — | — |
| CUST-09-010 | Market | Unavailable product behaviour | Item marked unavailable | Inspect card | «غير متوفر» chip; + hidden; section stays | — | `NOT_TESTED` | — | online | item available=false | — | — | — | P8-L1-017 prior evidence |
| CUST-09-011 | Market | Missing/broken image does not break the screen | Item with broken media | Open section | Placeholder; layout intact | — | `NOT_TESTED` | — | online | — | — | — | — | P8-C3-023 |
| CUST-09-012 | Market | Genuine empty catalog has explicit empty state | Geography with no content | Open تسوق | «نعمل حاليًا على إضافة المتاجر والمنتجات»; no retry (by design) | — | `NOT_TESTED` | — | online | `service_available` + zero items | — | — | — | P8-L1-018 prior evidence |
| CUST-09-013 | Market | API/network failure never shows the genuine-empty message | Signed-in test customer · Staging · SM-A525F · valid default address | Offline / API blocked | Error or OFFLINE state instead | — | `NOT_TESTED` | — | offline / API down | — | — | — | — | — |
| CUST-09-014 | Market | Refresh normally | Signed-in test customer · Staging · SM-A525F · valid default address | Pull-to-refresh | Refreshed; spinner ends | — | `NOT_TESTED` | — | online | — | — | — | — | P8-C3-017 |
| CUST-09-015 | Market | Repeated refresh | Signed-in test customer · Staging · SM-A525F · valid default address | Pull ×5 quickly | One effective refresh; no stuck spinner | — | `NOT_TESTED` | — | online | request count sane | — | — | — | `MarketplaceRaceTest` |
| CUST-09-016 | Market | Rapid navigation between sections | Signed-in test customer · Staging · SM-A525F · valid default address | Tap 5 sections in 2 s | Last tapped wins; no mixed items | — | `NOT_TESTED` | — | online | — | — | — | — | `Latest` guard; P8-C3-016/020 |
| CUST-09-017 | Market | Tap the same section repeatedly | Signed-in test customer · Staging · SM-A525F · valid default address | Tap ×5 | No spinner left | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-09-018 | Market | Scroll long content | Dense section (30–50 items) | Fling to end | Smooth; all items reachable | — | `NOT_TESTED` | — | online | — | — | — | — | Needs dense fixture (PF note) |
| CUST-09-019 | Market | Return after backgrounding | Signed-in test customer · Staging · SM-A525F · valid default address | Background 3 min; return | Refreshed only if stale; no pile-up | — | `NOT_TESTED` | — | online | — | — | — | — | AB-36 |
| CUST-09-020 | Market | Server retires a section while it is open | Signed-in test customer · Staging · SM-A525F · valid default address · Admin deactivates the open section | Refresh | Section leaves the rail; screen moves to a valid section | — | `NOT_TESTED` | — | online | active=false | — | — | — | — |
| CUST-09-021 | Market | Server disables an item while visible | Signed-in test customer · Staging · SM-A525F · valid default address · Admin marks item unavailable | Refresh | Card turns unavailable | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-09-022 | Market | Server changes product data while open | Signed-in test customer · Staging · SM-A525F · valid default address · Admin edits name/price | Refresh | New data shown | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-09-023 | Market | Refresh produces authoritative server state | Signed-in test customer · Staging · SM-A525F · valid default address | Compare UI to SoT after refresh | Equal | — | `NOT_TESTED` | — | online | SoT query | — | — | — | — |
| CUST-09-024 | Market | No hidden merchant data exposed | Signed-in test customer · Staging · SM-A525F · valid default address | Inspect payloads/screens | No store name/source in customer payloads/UI | — | `NOT_TESTED` | — | online | API JSON | — | — | — | Guards: `TestBrowse_HidesSource`, `TestRedactForCustomer_HidesSource` |
| CUST-09-025 | Market | Search | Signed-in test customer · Staging · SM-A525F · valid default address | Type ≥2 chars (300 ms debounce) | Matching items; «لا نتائج» when none; offline → OFFLINE state | — | `NOT_TESTED` | — | online | `/public/search/items` | — | — | — | Search exists; filters do not (no filter UI in source) |
| CUST-09-026 | Market | Banner slider and banner tap (added) | Signed-in test customer · Staging · SM-A525F · valid default address · banners with targets | Observe auto-rotation; tap a banner with a target | Rotation per `banner_auto/banner_every_ms`; a clickable banner must respond | — | `NOT_TESTED` | — | online | `/public/home` banners | — | — | — | Added. CAF-11 CONFIRMED (§40.10): banners with a target are clickable but ShopScreen passes no handler — silent tap, expected FAIL |
| CUST-09-027 | Market | Section rail auto-scroll setting (added) | Signed-in test customer · Staging · SM-A525F · valid default address · `shop.rail_auto` | Toggle setting (Admin) and observe | Rail follows the Owner setting | — | `NOT_TESTED` | — | online | setting value | — | — | — | Added. CAF-15 NEEDS_OWNER_DECISION (§40.10): `rail_auto/rail_every_ms` parsed but never used; the setting sits in the Site group |
| CUST-09-028 | Market | Browse scoped to city/address (added) | Signed-in test customer · Staging · SM-A525F · valid default address | Change city (drawer) / default address | Catalog, search, offers and suggestions follow the chosen scope | — | `NOT_TESTED` | — | online | requests carry lat/lng | — | — | — | Added. PC-1 gap noted: `CityScope.kt:90` returns the chosen city first |
| CUST-09-029 | Market | Guest browsing of the market (added) | Signed out | Browse, search, open sections | Works without account; add/heart lead to login | — | `NOT_TESTED` | — | online | — | — | — | — | Added |

## 21 · CUST-10 — Product interaction

Audited product model: no product-detail screen; items with options open the options sheet (required/optional groups, min/max, live price); items without options add directly.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-10-001 | Item | Open product (options sheet) | Signed-in test customer · Staging · SM-A525F · valid default address · item with options | Tap + on an item with options | ItemOptionsSheet opens with groups (مطلوب/اختياري) | — | `NOT_TESTED` | — | online | `/public/items/{id}` | — | — | — | No standalone product-detail screen exists — the options sheet is the product surface; items without options add directly |
| CUST-10-002 | Item | Return/back from the sheet | Sheet open | BACK / swipe down | Sheet closes; nothing added | — | `NOT_TESTED` | — | online | cart unchanged | — | — | — | — |
| CUST-10-003 | Item | Rapid repeated product taps | Signed-in test customer · Staging · SM-A525F · valid default address | Tap + ×5 fast on an item with options | One sheet; no duplicates | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-10-004 | Item | Available product can progress to cart | Sheet open | Meet minimums; «أضف — X» | Line added; badge +1 | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-10-005 | Item | Unavailable product cannot be ordered | Unavailable item | Tap card | No + button; not addable | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-10-006 | Item | Product becomes unavailable while sheet open | Sheet open · Admin disables item | Add | Server or refresh blocks; explicit | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-10-007 | Item | Price changes while sheet open | Sheet open · Admin changes price | Add; open cart | Cart review shows the change («متابعة بالقيم الحالية») | — | `NOT_TESTED` | — | online | quote price | — | — | — | `CartChanges` review gate |
| CUST-10-008 | Item | Product retired while sheet open | Sheet open · Admin retires item | Add; submit | Blocked explicitly at quote/submit | — | `NOT_TESTED` | — | online | no order | — | — | — | — |
| CUST-10-009 | Item | Quantity boundaries | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Cart + to large qty; − to 0 | 0 removes line; server enforces max (`bad_qty`/`quantity_invalid`) | — | `NOT_TESTED` | — | online | — | — | — | — | No client max; `TestQI*` server guards |
| CUST-10-010 | Item | Invalid quantity cannot be created through UI | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Try negative/zero via UI | Impossible via UI | — | `NOT_TESTED` | — | online | — | — | — | — | `CartTest` covers negative/zero |
| CUST-10-011 | Item | Options/variants/add-ons | Item with option groups | Pick options (max=1 replaces) | Live price updates; options sent as ids | — | `NOT_TESTED` | — | online | order item options == picked | — | — | — | — |
| CUST-10-012 | Item | Required option missing | Item with required group | Try to add without choosing | Add disabled until minimums met | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-10-013 | Item | Long product names do not break layout | Long-name fixture | Open grid/sheet/cart | Readable; actions reachable | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-10-014 | Item | Unavailable option disabled (added) | Item with an unavailable option | Open sheet | Option disabled; cannot be picked | — | `NOT_TESTED` | — | online | — | — | — | — | Added |

## 22 · CUST-11 — Cart

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-11-001 | Cart | Add one item | Signed-in test customer · Staging · SM-A525F · valid default address | Tap + (no options) | Badge 1; line in سلتي | — | `NOT_TESTED` | — | online | — | — | — | — | Cart is local (SharedPreferences `rahalgo_cart`) |
| CUST-11-002 | Cart | Add multiple quantities | Signed-in test customer · Staging · SM-A525F · valid default address | + ×3 | Qty 3 on one line | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-11-003 | Cart | Add multiple products | Signed-in test customer · Staging · SM-A525F · valid default address | Add 3 items | 3 lines | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-11-004 | Cart | Rapid Add taps | Signed-in test customer · Staging · SM-A525F · valid default address | `input tap` ×5 in 1 s | Qty equals taps counted by design (each tap adds 1) — no lost/duplicated adds | — | `NOT_TESTED` | — | online | — | — | — | — | `ButtonsUiTest.rapidQuantityTapsAllCount` |
| CUST-11-005 | Cart | Quantity correct after rapid tapping | After 004 | Read cart | Equals number of registered taps | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-11-006 | Cart | Increase quantity | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | + in cart | Qty +1; totals update | — | `NOT_TESTED` | — | online | quote refetched | — | — | — | — |
| CUST-11-007 | Cart | Decrease quantity | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | − in cart | Qty −1 | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-11-008 | Cart | Remove item | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Trash icon | Line removed | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-11-009 | Cart | Remove last item | One line | Remove | Empty state | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-11-010 | Cart | Empty cart state | Empty | Open سلتي | Explicit empty state; send absent | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-11-011 | Cart | Repeated remove taps safe | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Trash ×3 fast | One removal; no crash | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-11-012 | Cart | Repeated quantity taps safe | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | − ×5 fast from qty 2 | Line removed once; no negative | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-11-013 | Cart | Cart survives screen navigation | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Switch tabs | Intact | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-11-014 | Cart | Cart survives background/foreground | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | HOME; return | Intact | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-11-015 | Cart | Cart survives process recreation | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | `am kill`; relaunch | Intact (persisted) | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-11-016 | Cart | Cart after application restart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Force-stop; relaunch | Intact | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-11-017 | Cart | Logout behaviour with existing cart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Logout | Logout detaches the account's private cart (Owner decision §40.1-1); a guest cart only if explicitly scoped | — | `NOT_TESTED` | — | online | — | — | — | — | CUST-DEF-004 (STOP, §40.6): cart is device-global and NOT cleared on logout — expected FAIL |
| CUST-11-018 | Cart | Different customer does not inherit previous cart | A's cart; A logs out | B logs in; open cart | B never sees A's cart (Owner decision §40.1-1) | — | `NOT_TESTED` | — | online | — | — | — | — | CUST-DEF-004 (STOP, §40.6) — expected FAIL |
| CUST-11-019 | Cart | Change delivery address with populated cart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Switch address | Quote re-fetched; notes for out-of-zone | — | `NOT_TESTED` | — | online | `/public/quote` | — | — | — | AB-03 guard |
| CUST-11-020 | Cart | Item becomes unavailable while in cart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · Admin disables item | Open cart | Change listed; submit blocked until reviewed/removed | — | `NOT_TESTED` | — | online | — | — | — | — | P8-C3-027/036 |
| CUST-11-021 | Cart | Price changes while in cart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · Admin changes price | Open cart | «cart changes» list + «متابعة بالقيم الحالية» | — | `NOT_TESTED` | — | online | quote | — | — | — | P8-C3-028 |
| CUST-11-022 | Cart | Item retired while in cart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · Admin retires item | Open cart; submit | Explicit; no order with retired item | — | `NOT_TESTED` | — | online | no order | — | — | — | — |
| CUST-11-023 | Cart | Section becomes inactive while item in cart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · Admin deactivates section | Open cart; submit | Explicit per contract | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-11-024 | Cart | Zone closes with populated cart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · zone hours closed | Open cart | Zone-closed note; send disabled | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-11-025 | Cart | Platform ordering closes with populated cart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · platform closure | Open cart | Owner text; send disabled | — | `NOT_TESTED` | — | online | — | — | — | — | `Serving` refreshed on cart open |
| CUST-11-026 | Cart | Source becomes closed/unavailable | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · source store closed | Open cart | Explicit; per contract | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-11-027 | Cart | Server remains source of truth for orderability | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Force-submit via stale UI after server change | Server decision shown | — | `NOT_TESTED` | — | online | no invalid order | — | — | — | — |
| CUST-11-028 | Cart | Offline blocks Add | Signed-in test customer · Staging · SM-A525F · valid default address · offline | Tap + | Blocked with explanation | — | `NOT_TESTED` | — | offline | — | — | — | — | §7 — not built (expected FAIL) |
| CUST-11-029 | Cart | Offline blocks Remove | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · offline | Trash | Blocked | — | `NOT_TESTED` | — | offline | — | — | — | — | §7 |
| CUST-11-030 | Cart | Offline blocks quantity changes | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · offline | + / − | Blocked | — | `NOT_TESTED` | — | offline | — | — | — | — | §7 |
| CUST-11-031 | Cart | Offline visibly explains why | Signed-in test customer · Staging · SM-A525F · valid default address · offline | Attempt 028–030 | Explanation shown each time | — | `NOT_TESTED` | — | offline | — | — | — | — | §7 |
| CUST-11-032 | Cart | Recovery restores safe cart interaction | After 028–031 | Restore network | Cart usable after authoritative refresh | — | `NOT_TESTED` | — | recovering | quote refetched | — | — | — | §7.11 |
| CUST-11-033 | Cart | Suggestions row «يُطلب معه» (added) | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Tap a suggestion | Added with one tap; respects gating | — | `NOT_TESTED` | — | online | `/public/suggest` | — | — | — | Added: `SuggestRow.kt` |
| CUST-11-034 | Cart | Empty cart via «إفراغ السلة» (added) | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Tap «إفراغ السلة» | Cart empty (no confirmation by design — note) | — | `NOT_TESTED` | — | online | — | — | — | — | Added |
| CUST-11-035 | Cart | Add from Offers respects address/coverage gate (added) | Signed-in test customer · Staging · SM-A525F · valid default address · address outside coverage | Offers → «أضف إلى السلة» | Same gating as Shop add, or submit blocked explicitly | — | `NOT_TESTED` | — | online | no order | — | — | — | Added. CAF-12: offers add path skips the PreCart gate (server still validates at submit) |
| CUST-11-036 | Cart | Multi-source limit (added) | Signed-in test customer · Staging · SM-A525F · valid default address · `orders.max_sources`=1 | Add items from two sources; submit | Explicit `too_many_sources`/`multi_source_order` | — | `NOT_TESTED` | — | online | no order | — | — | — | Added: no client check; server enforces |
| CUST-11-037 | Cart | Corrupt persisted cart is discarded safely (added) | Emulator: corrupt `rahalgo_cart` prefs | Launch | Empty cart; no crash | — | `NOT_TESTED` | — | any | — | — | — | — | Added: unreadable cart is deleted by design |

## 23 · CUST-12 — Quote / checkout

Audited checkout: the cart screen is the checkout. Payment methods that exist: cash on delivery and wallet. No card, no mixed payment.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-12-001 | Checkout | Enter checkout with valid cart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Open سلتي (checkout is the cart screen) | Address card, promo, payment, totals, «أرسل الطلب» | — | `NOT_TESTED` | — | online | `/public/quote` 200 | — | — | — | No separate checkout screen — the cart screen is the review |
| CUST-12-002 | Checkout | Checkout with empty cart impossible | Empty cart | Open سلتي | No send button | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-12-003 | Checkout | Selected address shown correctly | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Read address card | Default address kind/text | — | `NOT_TESTED` | — | online | default address row | — | — | — | — |
| CUST-12-004 | Checkout | Authoritative quote obtained | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Open cart | Fee from server quote | — | `NOT_TESTED` | — | online | quote JSON | — | — | — | — |
| CUST-12-005 | Checkout | Item totals match server | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Compare subtotal with quote subtotal | Equal | — | `NOT_TESTED` | — | online | quote.subtotal | — | — | — | Subtotal is local (Cart.subtotal) — must equal server |
| CUST-12-006 | Checkout | Delivery fee matches server contract | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Compare fee | Equal to quote (`delivery.fee` = 100 policy) | — | `NOT_TESTED` | — | online | quote.delivery_fee | — | — | — | — |
| CUST-12-007 | Checkout | Displayed final total matches server | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Compare displayed total with quote/order total | Equal | — | `NOT_TESTED` | — | online | quote.total / order total | — | — | — | CUST-DEF-005 (CAF-08, STOP §40.7): total displayed = local subtotal + fee − discount (`CartScreen.kt:326`); server total used only in the change fingerprint |
| CUST-12-008 | Checkout | Client does not invent authoritative monetary totals | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Price change + promo edge cases | Displayed total never differs from the charged total | — | `NOT_TESTED` | — | online | order.total | — | — | — | Charged amounts are server-side (AB-35, `TestFIN_*`); display risk CUST-DEF-005 (CAF-08, STOP §40.7) |
| CUST-12-009 | Checkout | Change address before final submit | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Switch address | Quote refreshes | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-12-010 | Checkout | Quote refreshes when required | After 009 | Observe | New fee/availability | — | `NOT_TESTED` | — | online | new quote | — | — | — | — |
| CUST-12-011 | Checkout | Price change between cart and checkout | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · Admin price change | Open cart | Change review gate | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-12-012 | Checkout | Availability change between cart and checkout | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · item disabled | Open cart | Explicit | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-12-013 | Checkout | Coverage change between cart and checkout | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · zone shrunk | Open cart | Out-of-zone note | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-12-014 | Checkout | Zone closes during checkout | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Close zone; submit | Explicit denial | — | `NOT_TESTED` | — | online | no order | — | — | — | — |
| CUST-12-015 | Checkout | Platform closes ordering during checkout | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Close platform; submit | Explicit | — | `NOT_TESTED` | — | online | no order | — | — | — | — |
| CUST-12-016 | Checkout | launch.customer_orders changes during checkout | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Flip OFF; submit | `launch_closed` notice | — | `NOT_TESTED` | — | online | no order | — | — | — | P8-L1-020 |
| CUST-12-017 | Checkout | Slow quote | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Slow network | Loading; send disabled until quote | — | `NOT_TESTED` | — | slow | — | — | — | — | — |
| CUST-12-018 | Checkout | Quote timeout | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Timeout harness | Explicit failure + retry | — | `NOT_TESTED` | — | timeout | — | — | — | — | — |
| CUST-12-019 | Checkout | Quote 4xx | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Invalid lines (API client) / bad address | Explicit message | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-12-020 | Checkout | Quote 5xx | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Fault injection | Recoverable failure | — | `NOT_TESTED` | — | online | — | — | — | — | Harness to be approved |
| CUST-12-021 | Checkout | Offline checkout follows blocking contract | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · offline | Open cart; tap send | OFFLINE state; send blocked | — | `NOT_TESTED` | — | offline | no order | — | — | — | §7 |
| CUST-12-022 | Checkout | Back and return to checkout | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Leave cart and return | Quote re-evaluated; promo/payment kept in session | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-12-023 | Checkout | Payment methods that exist: cash and wallet | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Pay cash; pay from wallet (sufficient / insufficient balance) | Cash works; wallet works when covered; insufficient → explicit error, no order | — | `NOT_TESTED` | — | online | wallet tx; order payment | — | — | — | Methods from source: «نقدا عند التسليم», «من محفظتي» — no mixed, no card |
| CUST-12-024 | Checkout | Promo code preview (added) | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Enter valid / invalid / expired code → «تطبيق» | Valid → «تم تطبيق الكود — خصم X»; invalid → «الكود غير صالح أو منتهي» | — | `NOT_TESTED` | — | online | `/promo/preview` | — | — | — | Added |
| CUST-12-025 | Checkout | Promo becomes invalid before submit (added) | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · promo applied · Admin disables promo | Submit | Explicit; no stale discount charged | — | `NOT_TESTED` | — | online | order.discount | — | — | — | Added: `TestPR03_StaleDiscountCannotSubmit` |
| CUST-12-026 | Checkout | Promo & payment choice across process death (added) | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · promo + wallet chosen | `am kill`; relaunch; open cart | State lost is re-entered explicitly — never silently submitted with other values | — | `NOT_TESTED` | — | online | — | — | — | — | Added. CAF-17: promo and payment choice are memory-only |
| CUST-12-027 | Checkout | Below minimum order (added) | Cart below merchant minimum | Submit | Explicit `below_min_order` | — | `NOT_TESTED` | — | online | no order | — | — | — | Added: P8-C3-029..031; minimum shown only on rejection (TRUTH §6) |
| CUST-12-028 | Checkout | Cart-changes review gate (added) | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · server change | Open cart | Changes listed; send blocked until «متابعة بالقيم الحالية» | — | `NOT_TESTED` | — | online | — | — | — | — | Added: `CartChangesTest` (10) |

## 24 · CUST-13 — Order submission (critical)

**Critical section. Server order count and authoritative state must be checked for every row.**

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-13-001 | Submit | Single valid submit creates exactly one order | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | «أرسل الطلب» | Success; cart emptied; طلباتي opens | — | `NOT_TESTED` | — | online | orders +1 | — | — | — | Disposable test order only |
| CUST-13-002 | Submit | Order data matches server | After 001 | Compare card with SoT | Items, qty, fee, total, payment equal | — | `NOT_TESTED` | — | online | order row + items | — | — | — | — |
| CUST-13-003 | Submit | Repeated fast taps create one order | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Triple-tap send | One order | — | `NOT_TESTED` | — | online | orders +1 | — | — | — | AB-01, P8-C3-039 |
| CUST-13-004 | Submit | Button guarded while in flight | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Tap; inspect button | Busy/disabled until result | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-13-005 | Submit | Slow submit response | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Slow network | Busy then result; no duplicate | — | `NOT_TESTED` | — | slow | orders +1 | — | — | — | — |
| CUST-13-006 | Submit | Network loss before request reaches server | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Cut before tap (after §7 block) / during connect | Explicit failure; no order; key kept | — | `NOT_TESTED` | — | cut | orders +0 | — | — | — | — |
| CUST-13-007 | Submit | Loss after server commit, before client response | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Cut right after send (harness delay) | No duplicate on retry; order discoverable | — | `NOT_TESTED` | — | cut | orders +1 total | — | — | — | Idempotency-Key persisted (`Attempt`, `TestIDEM_002_LostResponse`) |
| CUST-13-008 | Submit | Retry after ambiguous failure is idempotent | After 007 | Retry send | Same order returned; no second | — | `NOT_TESTED` | — | recovering | orders +1 total | — | — | — | P8-C3-040/041 |
| CUST-13-009 | Submit | HTTP conflict gives explicit safe result | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Concurrent in-flight key (`409 in_progress`) | Explicit message; no duplicate | — | `NOT_TESTED` | — | online | — | — | — | — | PC-8 wording not verified |
| CUST-13-010 | Submit | Validation failure explicit | API/UI invalid payload | Submit | Explicit | — | `NOT_TESTED` | — | online | no order | — | — | — | — |
| CUST-13-011 | Submit | Backend 500 does not fake success | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Fault injection | Explicit failure | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-13-012 | Submit | Timeout does not fake success | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Timeout harness | Explicit ambiguous result; order checked before retry | — | `NOT_TESTED` | — | timeout | orders +0/+1 | — | — | — | PC-8: `CartViewModel.uncertain` is set but never displayed |
| CUST-13-013 | Submit | App restart immediately after submit | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Send; force-stop at once; relaunch | Order discoverable once | — | `NOT_TESTED` | — | online | orders +1 | — | — | — | — |
| CUST-13-014 | Submit | Process killed immediately after submit | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Send; kill; relaunch | As 013 | — | `NOT_TESTED` | — | online | orders +1 | — | — | — | P8-C3-045 |
| CUST-13-015 | Submit | Committed order discoverable after reconnect/reopen | After 007/013 | Open طلباتي | Order visible | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-13-016 | Submit | No locally created phantom order | After failures | Open طلباتي | Only server orders | — | `NOT_TESTED` | — | online | UI == SoT | — | — | — | — |
| CUST-13-017 | Submit | No duplicate after reconnect | After 007 | Reconnect; refresh | One order | — | `NOT_TESTED` | — | online | orders +1 total | — | — | — | — |
| CUST-13-018 | Submit | Ordering disabled at final moment | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Flip launch/platform just before send | Explicit denial | — | `NOT_TESTED` | — | online | no order | — | — | — | `TestPH29_StaleClientCannotSubmitAfterClose` |
| CUST-13-019 | Submit | Address invalidated at final moment | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Shrink zone just before send | Explicit denial | — | `NOT_TESTED` | — | online | no order | — | — | — | — |
| CUST-13-020 | Submit | Item invalidated at final moment | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Disable item just before send | Explicit | — | `NOT_TESTED` | — | online | no order | — | — | — | — |
| CUST-13-021 | Submit | Price changed at final moment | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Change price just before send | Change review / explicit; charged = server price | — | `NOT_TESTED` | — | online | order price | — | — | — | — |
| CUST-13-022 | Submit | Session invalid before final submit | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Revoke session; send | Explicit re-login; no order | — | `NOT_TESTED` | — | online | no order | — | — | — | — |
| CUST-13-023 | Submit | Offline submit blocked before misleading success | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · offline | Tap send | Blocked; OFFLINE explanation | — | `NOT_TESTED` | — | offline | no order | — | — | — | §7 |
| CUST-13-024 | Submit | Order count before/after proves exact mutation | Every submit case | Read counts | Exactly the intended delta | — | `NOT_TESTED` | — | online | read-only SQL | — | — | — | Applies to all CUST-13 rows |
| CUST-13-025 | Submit | Open-order cap (added) | Signed-in test customer · Staging · SM-A525F · valid default address · open orders at cap | Submit another | Explicit cap message; no order | — | `NOT_TESTED` | — | online | orders unchanged | — | — | — | Added: D4 fixed; `TestD4_*` |
| CUST-13-026 | Submit | WhatsApp verification requirement on normal orders (added) | Signed-in test customer · Staging · SM-A525F · valid default address · unverified · `auth.require_whatsapp` policy | Submit | Behaviour per policy (false today → allowed) | — | `NOT_TESTED` | — | online | — | — | — | — | Added: normal path checks WhatsApp; custom path does not (D8) |
| CUST-13-027 | Submit | Cash-blocked customer (added) | Test customer cash-blocked | Submit cash order | Explicit denial | — | `NOT_TESTED` | — | online | no order | — | — | — | Added: normal path checks cash ban; custom path does not (D6) |
| CUST-13-028 | Submit | 409 `in_progress` never leads to a duplicate order (added) | Harness: slow first submit (> client 20 s timeout, < server 30 s) | Submit; after client timeout tap send again while the first is still running; then tap again | Retry key kept; the user is told the order is still processing; exactly one order | — | `NOT_TESTED` | — | slow | orders +1 exactly | — | — | — | Added. CAF-02 (source-confirmed): `Attempt.isDecided` treats any ApiException (incl. 409 `in_progress`) as final and clears the key; `in_progress` is unmapped. Expected FAIL |
| CUST-13-029 | Submit | Retry after cart edit does not replay the old order (added) | Submit failed by network (key kept) | Edit cart; submit | Server returns the old committed order OR the new cart is submitted — never a silent mismatch between cart and created order | — | `NOT_TESTED` | — | cut | order items vs cart | — | — | — | Added. CAF-02: idempotency does not fingerprint the body |

## 25 · CUST-CUSTOM — Custom order «طلب خاص»

**The custom-order surface exists** (tab «طلب خاص», `custom/CustomScreen.kt`, `POST /api/v1/orders/custom`). Production has `launch.customer_custom_orders`=false; Staging has it open. The known defects D6/D8/D9 stay in scope — the launch flag does not make the code safe.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-CUSTOM-001 | Custom | Create a custom order | Signed-in test customer · Staging · SM-A525F · valid default address · `launch.customer_custom_orders`=ON (Staging) | طلب خاص → request text → address → optional driver note → send | «تم إرسال طلبك رقم N»; switches to طلباتي | — | `NOT_TESTED` | — | online | orders +1 (custom) | — | — | — | Feature exists (`custom/CustomScreen.kt`, `POST /api/v1/orders/custom`). Production flag is OFF — the code is still judged, not hidden behind the flag |
| CUST-CUSTOM-002 | Custom | Empty request rejected | Signed-in test customer · Staging · SM-A525F · valid default address | Send blank | Send disabled | — | `NOT_TESTED` | — | online | — | — | — | — | `TestCUST_002_EmptyRequestRejected` |
| CUST-CUSTOM-003 | Custom | No address | Signed-in test customer · Staging · SM-A525F · valid default address · no address | Try to send | Hint «اختر عنوان التوصيل…»; disabled | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-CUSTOM-004 | Custom | Repeated send taps → one order | Signed-in test customer · Staging · SM-A525F · valid default address | Triple-tap send | One custom order | — | `NOT_TESTED` | — | online | orders +1 | — | — | — | Own idempotency slot (`Attempt.CUSTOM`); `TestCUST_003_Idempotent` |
| CUST-CUSTOM-005 | Custom | Lost response / retry | Signed-in test customer · Staging · SM-A525F · valid default address | Cut after send; retry | No duplicate | — | `NOT_TESTED` | — | cut | orders +1 total | — | — | — | — |
| CUST-CUSTOM-006 | Custom | Launch flag OFF | Signed-in test customer · Staging · SM-A525F · valid default address · flag OFF | Send | Explicit `launch_closed`; no order | — | `NOT_TESTED` | — | online | no order | — | — | — | Client ignores the flag (parsed, unused) — server is the guard |
| CUST-CUSTOM-007 | Custom | Outside coverage / zone closed / platform closed | Signed-in test customer · Staging · SM-A525F · valid default address | Send under each condition | Explicit denial each | — | `NOT_TESTED` | — | online | no order | — | — | — | `TestSRV4_CustomOrderFollowsCoverage`, `TestZH19`, `TestPH16/18` |
| CUST-CUSTOM-008 | Custom | Cash-blocked customer | Cash-blocked test customer | Send | Must be denied like the normal path | — | `NOT_TESTED` | — | online | no order | — | — | — | KNOWN DEFECT D6 (EXPECTED_FAIL `TestCENSUS_D6_CustomOrderSkipsCashBan`) — expected FAIL |
| CUST-CUSTOM-009 | Custom | WhatsApp verification requirement | Unverified · require_whatsapp=true (Staging test) | Send | Must follow the same rule as the normal path | — | `NOT_TESTED` | — | online | no order | — | — | — | KNOWN DEFECT D8 (EXPECTED_FAIL) — expected FAIL |
| CUST-CUSTOM-010 | Custom | Creation event recorded | After 001 | Read order_events | `''→pending` event exists like normal orders | — | `NOT_TESTED` | — | online | order_events | — | — | — | KNOWN DEFECT D9 (EXPECTED_FAIL) — expected FAIL |
| CUST-CUSTOM-011 | Custom | Open-order cap shared with normal orders | Signed-in test customer · Staging · SM-A525F · valid default address · at cap | Send | Explicit cap | — | `NOT_TESTED` | — | online | no order | — | — | — | `TestD4_CustomOrdersShareTheSameCap` |
| CUST-CUSTOM-012 | Custom | No price before agreement | After 001 | Read card | Fee shown as «يحددها السائق عند الاتفاق» until agreed | — | `NOT_TESTED` | — | online | order fee 0 | — | — | — | `TestCUST_010_NoPriceBeforeAgreement` |
| CUST-CUSTOM-013 | Custom | Cancel custom order until bought | Open custom order | Cancel | Allowed until bought; then denied explicitly | — | `NOT_TESTED` | — | online | status | — | — | — | `TestCustomCancel_OwnerUntilBought` |
| CUST-CUSTOM-014 | Custom | Foreign customer cannot read it | Two customers | B reads A's custom order id | 404/403 | — | `NOT_TESTED` | — | online | — | — | — | — | `TestCUST_020_ForeignCannotRead` |
| CUST-CUSTOM-015 | Custom | Offline send blocked | Signed-in test customer · Staging · SM-A525F · valid default address · offline | Send | OFFLINE state; blocked | — | `NOT_TESTED` | — | offline | no order | — | — | — | §7 |
| CUST-CUSTOM-016 | Custom | Text preserved across rotation/background | Signed-in test customer · Staging · SM-A525F · valid default address | Type; rotate/background | Text kept (rememberSaveable) | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-CUSTOM-017 | Custom | Guest sees NeedAccount | Signed out | Open طلب خاص | «هذا القسم يحتاج حسابا» + login | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-CUSTOM-018 | Custom | Custom-order realtime to owner | After 001 | Driver/ops change it | Customer sees update | — | `NOT_TESTED` | — | online | — | — | — | — | `TestD22_*` |
| CUST-CUSTOM-019 | Custom | Driver note is saved and shown (added) | Signed-in test customer · Staging · SM-A525F · valid default address | Send with «ملاحظات للسائق» | Note stored and visible to the driver | — | `NOT_TESTED` | — | online | order notes | — | — | — | Added. CAF-07 (reported by audit): custom `notes` are sent but not decoded/stored — expected FAIL |
| CUST-CUSTOM-020 | Custom | Custom order payment method (added) | Signed-in test customer · Staging · SM-A525F · valid default address | Inspect payment options | Cash or wallet selectable (Owner decision 2026-08-09, `orders/custom.go:74-78`) | — | `NOT_TESTED` | — | online | order payment_method | — | — | — | Added. Answered by contract (§40.11): the app sends no method → wallet option missing — expected FAIL |

## 26 · CUST-14 — Order list / lifecycle

Full multi-role order progression belongs to later E2E acceptance. #1050 may be observed read-only but is never progressed.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-14-001 | Orders | Current order list | Signed-in test customer · Staging · SM-A525F · valid default address · open orders exist | Open طلباتي | Open orders only | — | `NOT_TESTED` | — | online | `/my/orders` open | — | — | — | — |
| CUST-14-002 | Orders | Past/history list | Signed-in test customer · Staging · SM-A525F · valid default address | Drawer → سجل الطلبات | Ended orders (delivered/cancelled/failed/rejected/refunded) | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-14-003 | Orders | Open correct order detail | — | — | — | — | `NOT_APPLICABLE` | — | — | — | — | — | — | **N/A:** No order-detail screen exists: the order card is the only view; `CustomerApi.order(id)` is unused (audit §38). Card correctness is covered by CUST-14-021. |
| CUST-14-004 | Orders | Wrong customer cannot see another's order | Two customers | B calls A's order via API / UI list | Not visible; 404/403 | — | `NOT_TESTED` | — | online | — | — | — | — | `TestSECIDOR_Orders`, `TestOrder_IntruderCannotRead` |
| CUST-14-005 | Orders | Order status matches backend | Signed-in test customer · Staging · SM-A525F · valid default address | Compare chip with SoT | Equal | — | `NOT_TESTED` | — | online | orders.status | — | — | — | — |
| CUST-14-006 | Orders | Refresh order state | Signed-in test customer · Staging · SM-A525F · valid default address | Pull/return to tab | Fresh status | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-14-007 | Orders | Realtime state update | Signed-in test customer · Staging · SM-A525F · valid default address · order progressing (ops/driver on a disposable order) | Keep طلباتي open | Status changes without manual refresh (WebSocket → Refresh.bump) | — | `NOT_TESTED` | — | online | — | — | — | — | Never progress #1050 |
| CUST-14-008 | Orders | No duplicate rows after refresh/reconnect | Signed-in test customer · Staging · SM-A525F · valid default address | Reconnect ×3 | No duplicates | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-14-009 | Orders | Ordering/sorting correct | Signed-in test customer · Staging · SM-A525F · valid default address | Compare with SoT | Newest first as designed | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-14-010 | Orders | Pending state | Disposable order pending | Read card | Stage bar at pending; cancel shown while window open | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-14-011 | Orders | Accepted state | Order accepted | Read card | Stage accepted | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-14-012 | Orders | Dispatch/driver assignment | Driver assigned | Read card | Driver name shown; no driver phone | — | `NOT_TESTED` | — | online | — | — | — | — | D21 fixed: no driver phone in payload. PC-12: no push on `assigned` |
| CUST-14-013 | Orders | On-the-way state | #1050 read-only | Read card | «في الطريق» | — | `NOT_TESTED` | — | online | #1050 unchanged | — | — | — | Observed 2026-09-19 (read-only) |
| CUST-14-014 | Orders | Delivered/completed | Disposable delivered order | Read card | Delivered; rate available | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-14-015 | Orders | Cancelled/rejected/failed/refunded | Orders in those states | Read cards | Correct Arabic status for each | — | `NOT_TESTED` | — | online | — | — | — | — | PC-3: `refunded` shows raw English; `rejected` shares cancelled text (`orders/Status.kt:31`) — expected FAIL |
| CUST-14-016 | Orders | Actions only in appropriate states | Signed-in test customer · Staging · SM-A525F · valid default address | Inspect cancel/complaint/rate per state | Cancel only in window; rate only delivered+unrated | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-14-017 | Orders | Forbidden action via client manipulation denied | API client | Cancel after delivered; rate twice; complain on running order | Server denies each | — | `NOT_TESTED` | — | online | — | — | — | — | `TestCANC_010`, `TestComplaint_NotOnRunningOrder` |
| CUST-14-018 | Orders | Order detail survives background | — | — | — | — | `NOT_APPLICABLE` | — | — | — | — | — | — | **N/A:** No order-detail screen (see 14-003). Orders tab lifecycle is covered by CUST-17-022. |
| CUST-14-019 | Orders | Order detail after process restart | — | — | — | — | `NOT_APPLICABLE` | — | — | — | — | — | — | **N/A:** No order-detail screen (see 14-003). Covered for the Orders tab by CUST-17-022. |
| CUST-14-020 | Orders | #1050 observed read-only, never progressed | Signed-in test customer · Staging · SM-A525F · valid default address | Observe only | #1050 status/events unchanged before/after every session | — | `NOT_TESTED` | — | online | #1050 row | — | — | — | — |
| CUST-14-021 | Orders | Order card shows authoritative data (added) | Signed-in test customer · Staging · SM-A525F · valid default address | Compare each card field with SoT | Number, status, stages, items, subtotal, delivery, discount, total, cancel reason correct | — | `NOT_TESTED` | — | online | order JSON | — | — | — | Added (replaces the absent detail screen). Card omits payment method, address, times, wallet_paid, cash_due — Owner to decide if required |
| CUST-14-022 | Orders | Cancel order within window (added) | Disposable pending order | Cancel → confirm | Cancelled; wallet refund if paid by wallet | — | `NOT_TESTED` | — | online | status; wallet tx | — | — | — | Added: `TestCANC_001`, `TestCancelBeforeDelivery_RefundsWalletOnly` |
| CUST-14-023 | Orders | Double cancel / cancel after window (added) | After 022 | Cancel again / after window | Explicit denial | — | `NOT_TESTED` | — | online | — | — | — | — | Added: `TestCANC_002` |
| CUST-14-024 | Orders | Rate a delivered order (added) | Delivered unrated order | Rate service (+driver) | Saved; not re-prompted | — | `NOT_TESTED` | — | online | rating row | — | — | — | Added. No backend test for the customer rating happy path/authz (audit) |
| CUST-14-025 | Orders | Automatic rating prompt (added) | Newest delivered unrated | Open app | Prompt once per session; not for guests; not on Cart tab | — | `NOT_TESTED` | — | online | — | — | — | — | Added |
| CUST-14-026 | Orders | History beyond 30 orders (added) | Account with >30 orders (fixture) | Open history; scroll | All orders reachable | — | `NOT_TESTED` | — | online | count > 30 | — | — | — | Added. CAF-14: the app requests page 1 only (`OrdersViewModel.kt:99,145`) — expected FAIL |

## 26A · CUST-SUP — Chat, complaints, tickets and warnings (added by audit)

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-SUP-001 | Chat | Chat button appears for an open order with a driver | Signed-in test customer · Staging · SM-A525F · valid default address · disposable order with driver | Observe ChatFab | FAB with unread badge; one order → opens its chat; several → list | — | `NOT_TESTED` | — | online | `/my/chats` | — | — | — | `ChatMultiOrderTest` (21) |
| CUST-SUP-002 | Chat | Send and receive messages | As 001 | Send text; driver replies | Both appear in order; ringtone when chat closed | — | `NOT_TESTED` | — | online | `/orders/{id}/messages` | — | — | — | — |
| CUST-SUP-003 | Chat | Chat read-only after order ends | Ended order | Open chat | Read-only; send absent; `comms_closed` explicit if forced | — | `NOT_TESTED` | — | online | — | — | — | — | `comms_closed`/`comms_no_driver` are unmapped codes (CAF-18) |
| CUST-SUP-004 | Chat | Two orders' chats never mix | Two open orders with drivers | Switch chats rapidly | Messages/drafts stay with their order | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-SUP-005 | Chat | Stranger cannot read/post in another's chat | Two test customers (A, B) · API client with each token | B GET/POST A's messages | 404 without leakage | — | `NOT_TESTED` | — | online | — | — | — | — | `TestCHAT05_StrangerGetsNotFound` |
| CUST-SUP-006 | Chat | Chats list (دردشاتي السابقة) | Signed-in test customer · Staging · SM-A525F · valid default address | Drawer → chats | Open first; closed expand read-only | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-SUP-007 | Chat | Offline chat send blocked | Signed-in test customer · Staging · SM-A525F · valid default address · offline | Send | OFFLINE state; blocked | — | `NOT_TESTED` | — | offline | no message row | — | — | — | §7 |
| CUST-SUP-008 | Support | Complaint on an order | Delivered disposable order | Complaint → reason (note required for 'other') → send | Ticket created; shown in الشكاوى والبلاغات | — | `NOT_TESTED` | — | online | ticket row | — | — | — | `TestComplaint_*` |
| CUST-SUP-009 | Support | Complaint once / window / not on running order | As 008 | Complain twice; after window; on running order | Explicit denial each | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-SUP-010 | Support | Tickets list shows status and resolution | Signed-in test customer · Staging · SM-A525F · valid default address | Drawer → الشكاوى والبلاغات | Number, subject, status, resolution | — | `NOT_TESTED` | — | online | `/my/tickets` | — | — | — | Ticket reply: PRQ-2 approved (customer should reply) but not built — release scope is the open Owner question (§40.11) |
| CUST-SUP-011 | Support | Foreign order complaint denied | Two test customers (A, B) · API client with each token | B complains on A's order | Denied without leakage | — | `NOT_TESTED` | — | online | — | — | — | — | `TestVAL_040_ForeignOrderComplaintCode` |
| CUST-SUP-012 | Support | Admin warnings visible to the customer | Admin warns the test customer | Open app | Warning reaches the customer (Admin contract: «يصل الإنذار صاحب الحساب بنصه، ويبقى في سجله») in safe customer-facing wording (Owner decision §40.1-2) | — | `NOT_TESTED` | — | online | `/my/warnings` | — | — | — | CAF-19 CONFIRMED (§40.10): no notification is sent and the app never shows warnings — expected FAIL |

## 26B · CUST-WAL — Wallet (added by audit)

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-WAL-001 | Wallet | Balance chip and wallet screen | Signed-in test customer · Staging · SM-A525F · valid default address | Tap wallet chip | Balance equals SoT; transactions listed | — | `NOT_TESTED` | — | online | `/my/wallet`; wallets.balance | — | — | — | — |
| CUST-WAL-002 | Wallet | Transaction list correctness | Signed-in test customer · Staging · SM-A525F · valid default address · known transactions | Compare rows | Signed amounts, kinds, notes, dates, order numbers match SoT | — | `NOT_TESTED` | — | online | wallet_transactions | — | — | — | — |
| CUST-WAL-003 | Wallet | Statement: this month / previous / all | Signed-in test customer · Staging · SM-A525F · valid default address | كشف حساب → switch ranges | Opening/closing balances consistent | — | `NOT_TESTED` | — | online | `TestStatement_BalancesEvenWhenTruncated` | — | — | — | — |
| CUST-WAL-004 | Wallet | Statement print/PDF | Signed-in test customer · Staging · SM-A525F · valid default address | Print → Save as PDF | PDF with logo, name, support phone | — | `NOT_TESTED` | — | any | — | — | — | — | — |
| CUST-WAL-005 | Wallet | Wallet shows only own balance | Two test customers (A, B) · API client with each token | B reads A's wallet via API | Own data only | — | `NOT_TESTED` | — | online | — | — | — | — | `TestSECIDOR_WalletIsOwn`, `TestWallet_ShowsOnlyOwnBalance` |
| CUST-WAL-006 | Wallet | Pay order from wallet (sufficient) | Signed-in test customer · Staging · SM-A525F · valid default address · balance ≥ total | Pay «من محفظتي» | Order created; balance debited once | — | `NOT_TESTED` | — | online | wallet tx −total | — | — | — | Staging wallet credit via supported Admin/incentive path only |
| CUST-WAL-007 | Wallet | Pay from wallet (insufficient) | Signed-in test customer · Staging · SM-A525F · valid default address · balance < total | Pay «من محفظتي» | Explicit `insufficient_balance`; no order | — | `NOT_TESTED` | — | online | no order; no tx | — | — | — | `TestWALL_010_CannotPayBeyondBalance` |
| CUST-WAL-008 | Wallet | Cancel wallet-paid order refunds wallet | After 006 (within cancel window) | Cancel | Wallet refunded exactly once | — | `NOT_TESTED` | — | online | wallet tx +total | — | — | — | `TestCancelBeforeDelivery_RefundsWalletOnly` |
| CUST-WAL-009 | Wallet | No payouts/top-up offered to customers | Signed-in test customer · Staging · SM-A525F · valid default address | Inspect wallet UI | No payout/top-up controls | — | `NOT_TESTED` | — | online | — | — | — | — | POST `/me/payouts` returns 403 `payout_not_allowed` for customers |
| CUST-WAL-010 | Wallet | Wallet realtime refresh | Signed-in test customer · Staging · SM-A525F · valid default address | Credit via Admin while screen open | Balance updates via realtime | — | `NOT_TESTED` | — | online | — | — | — | — | — |

## 26C · CUST-ENG — Engagement: favorites, offers, referrals, pages, theme (added by audit)

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-ENG-001 | Fav | Toggle favorite from a card | Signed-in test customer · Staging · SM-A525F · valid default address | Tap heart | Toggled; persists | — | `NOT_TESTED` | — | online | `/my/favorites` | — | — | — | — |
| CUST-ENG-002 | Fav | Favorites screen | Signed-in test customer · Staging · SM-A525F · valid default address | Drawer → المفضلة | Grid; ♥ removes; empty state | — | `NOT_TESTED` | — | online | — | — | — | — | No add-to-cart from Favorites (by design?) — Owner note |
| CUST-ENG-003 | Fav | Guest heart → login | Signed out | Tap heart | Login path | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-ENG-004 | Offers | Offers list | Signed-in test customer · Staging · SM-A525F · valid default address | Drawer → العروض | Image, title, prices, discount chip | — | `NOT_TESTED` | — | online | `/public/offers` | — | — | — | — |
| CUST-ENG-005 | Offers | Add offer item to cart | Signed-in test customer · Staging · SM-A525F · valid default address | «أضف إلى السلة» on an offer with an item | «أُضيف إلى السلة»; options sheet if needed; gating per CUST-11-035 | — | `NOT_TESTED` | — | online | — | — | — | — | CAF-12 |
| CUST-ENG-006 | Offers | Offer opened from push; expired offer | Offer push | Tap push; tap expired | Focused at top; expired → «العرض الذي وصلك لم يعد سارياً» | — | `NOT_TESTED` | — | online | — | — | — | — | `DeepLinkTest` |
| CUST-ENG-007 | Refer | Invite screen | Signed-in test customer · Staging · SM-A525F · valid default address | Drawer → ادع صديقا | Code, next reward, counts; share opens chooser | — | `NOT_TESTED` | — | online | `/auth/referral` | — | — | — | — |
| CUST-ENG-008 | Refer | Referral reward paid per policy | New signup with the code | Complete signup (and first order if policy) | Reward credited once per policy (`referral.*`) | — | `NOT_TESTED` | — | online | wallet tx | — | — | — | `TestRewardOn*`, `TestBonusAndReferral_OncePerPhone` |
| CUST-ENG-009 | Refer | Signup bonus | New account | Signup | `customers.signup_bonus` (15) credited once | — | `NOT_TESTED` | — | online | wallet tx | — | — | — | `TestGrantSignupBonus_Credits` |
| CUST-ENG-010 | Pages | Static pages | Any | Drawer → التعليمات · من نحن · شروط الاستخدام · سياسة الخصوصية | Server texts shown; offline → explicit | — | `NOT_TESTED` | — | online | `/public/contact` | — | — | — | — |
| CUST-ENG-011 | Pages | Contact page links | Any | تواصل معنا → phone / WhatsApp / map / social | Each opens the right app/intent | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-ENG-012 | Theme | Theme toggle | Any | Drawer theme toggle; restart | System → explicit mode; persisted | — | `NOT_TESTED` | — | any | — | — | — | — | — |
| CUST-ENG-013 | Brand | Brand intro once per process; reduce-motion respected | Any | Cold start; with animations off | Intro once; skipped/minimal with reduce motion | — | `NOT_TESTED` | — | any | — | — | — | — | — |
| CUST-ENG-014 | Menu | Drawer items per auth state | Guest and signed-in | Open drawer | Guest: public items + «دخول أو إنشاء حساب»; signed-in: history, favorites, offers, chats, invite, complaints + «خروج» | — | `NOT_TESTED` | — | any | — | — | — | — | — |

## 27 · CUST-15 — Realtime / notifications

Audited: FCM push (channels `rahalgo_urgent` / `rahalgo_default`) and a WebSocket (`/api/v1/ws`) whose frames trigger refresh. No polling.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-15-001 | Push | Order update reaches foreground app | Signed-in test customer · Staging · SM-A525F · valid default address · disposable order progressing | Keep app open | UI updates via WebSocket; notification per policy | — | `NOT_TESTED` | — | online | — | — | — | — | Realtime = WebSocket `/api/v1/ws` (no polling) |
| CUST-15-002 | Push | Update while backgrounded | As 001 · app backgrounded | Progress order | Push notification (urgent channel) | — | `NOT_TESTED` | — | online | notification_deliveries | — | — | — | — |
| CUST-15-003 | Push | Notification while process killed | App force-stopped (not 'Force stop' in settings) | Progress order | Push shown; tap opens app | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-15-004 | Push | Notification permission denied | Denied | Progress order | No push; in-app state still correct on open | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-15-005 | Push | Permission granted later | Denied then granted | Progress order | Push arrives | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-15-006 | Push | Tapping a notification opens the intended safe destination | Push received | Tap order_chat / offer / order-status pushes | order_chat → chat sheet; offer → offer; order status → the order | — | `NOT_TESTED` | — | online | — | — | — | — | XG-9 (CAF-13) / XG-8 / XG-9: order-status pushes open the default screen — expected FAIL for status pushes |
| CUST-15-007 | Push | Old/stale notification | Old push in tray | Tap after state changed | Opens current truth; no stale action | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-15-008 | Push | Duplicate notification | Two pushes same class | Observe tray | No confusing duplicates | — | `NOT_TESTED` | — | online | — | — | — | — | XG-38 / PC-5: two fixed IDs (3001/3002) — newer replaces older |
| CUST-15-009 | Push | Notification for an inaccessible order | Push for order of another account | Tap | Safe fallback; no data | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-15-010 | Push | Account switched after notification generated | A's push; B logged in | Tap | No A data shown to B | — | `NOT_TESTED` | — | online | — | — | — | — | `TestPush_TokenMovesToNewOwner` |
| CUST-15-011 | Push | No cross-account content leakage | Shared device, two accounts | Receive pushes after switch | Only current account's pushes | — | `NOT_TESTED` | — | online | device token owner | — | — | — | D12 fixed |
| CUST-15-012 | Push | Realtime disconnect | Signed-in test customer · Staging · SM-A525F · valid default address | Cut network while on طلباتي | Reconnect backoff 2→30 s; no crash | — | `NOT_TESTED` | — | flapping | — | — | — | — | — |
| CUST-15-013 | Push | Realtime reconnect | After 012 | Restore | Reconnects; refresh fires | — | `NOT_TESTED` | — | recovering | — | — | — | — | — |
| CUST-15-014 | Push | Expired access token during reconnect | Signed-in test customer · Staging · SM-A525F · valid default address | Background > 15 min; return | Refresh once; one socket | — | `NOT_TESTED` | — | online | — | — | — | — | `D19ReconnectTest`, `D19StressTest` |
| CUST-15-015 | Push | Device-token rotation | Signed-in test customer · Staging · SM-A525F · valid default address | Reinstall FCM token / clear Play services data (emulator) | `/me/devices` re-registers | — | `NOT_TESTED` | — | online | device_tokens | — | — | — | — |
| CUST-15-016 | Push | Logout removes notification association | Signed-in test customer · Staging · SM-A525F · valid default address | Logout | Device token unregistered | — | `NOT_TESTED` | — | online | device_tokens row removed | — | — | — | `TestD12_*` |
| CUST-15-017 | Push | Notification inbox (added) | Signed-in test customer · Staging · SM-A525F · valid default address | Bell → list; «تعليم الكل كمقروء» | Grouped by day; unread dot; all marked read; tapping an item does nothing (by design) | — | `NOT_TESTED` | — | online | `/me/notifications` | — | — | — | Added |
| CUST-15-018 | Push | Chat message push opens the chat (added) | Signed-in test customer · Staging · SM-A525F · valid default address · driver sends message | Tap push | Opens that order's chat | — | `NOT_TESTED` | — | online | — | — | — | — | Added: `DeepLinkTest`, `ChatMultiOrderTest` |
| CUST-15-019 | Push | Logout stops realtime (added) | Signed-in test customer · Staging · SM-A525F · valid default address | Logout; watch socket/logcat | Socket closed; no updates for the old account | — | `NOT_TESTED` | — | online | — | — | — | — | Added. CUST-DEF-004 (STOP §40.6): `LiveSocket.stop` is not called by the customer app on logout — expected FAIL |

## 28 · CUST-16 — Network / offline / degraded connectivity (mandatory)

**This entire group is mandatory. Final closure of L1-019 requires the applicable mandatory cases in this group — especially blocking/retry — to PASS on the physical device.**

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-16-001 | Offline | Fresh app launch fully offline | Force-stopped · net=none | Launch; UIA +3/+8/+15 s | OFFLINE state: «لا يوجد اتصال بالإنترنت» + «أعد المحاولة»; interaction blocked (§7); no empty market; stable | — | `NOT_TESTED` | — | offline | — | — | — | — | L1-019 cold path |
| CUST-16-002 | Offline | Marketplace loaded, Wi-Fi disappears, no other network | Signed-in test customer · Staging · SM-A525F · market loaded · data OFF | Disable Wi-Fi only; UIA +3/+8/+15 s | OFFLINE state: «لا يوجد اتصال بالإنترنت» + «أعد المحاولة»; interaction blocked (§7) | — | `NOT_TESTED` | — | offline | — | — | — | — | Exact P8-DEF-001 scenario |
| CUST-16-003 | Offline | Marketplace loaded, mobile data disappears | Signed-in test customer · Staging · SM-A525F · Wi-Fi OFF · data ON | Disable data; UIA | OFFLINE state: «لا يوجد اتصال بالإنترنت» + «أعد المحاولة»; interaction blocked (§7) | — | `NOT_TESTED` | — | offline | — | — | — | — | — |
| CUST-16-004 | Offline | Wi-Fi switches to mobile data successfully | Signed-in test customer · Staging · SM-A525F · both ON · on Wi-Fi | Disable Wi-Fi; wait for cellular VALIDATED | No OFFLINE state once cellular validates (a brief state during validation is acceptable and must clear); data keeps working | — | `NOT_TESTED` | — | switching | — | — | — | — | 2026-09-19 evidence: a new cellular network shows banner until VALIDATED |
| CUST-16-005 | Offline | Mobile data switches to Wi-Fi successfully | Signed-in test customer · Staging · SM-A525F · on cellular | Enable Wi-Fi | Stays online; no false OFFLINE | — | `NOT_TESTED` | — | switching | — | — | — | — | — |
| CUST-16-006 | Offline | All connectivity disappears | Signed-in test customer · Staging · SM-A525F | Cut Wi-Fi; cut data only if a default network remains; wait for net=none (on-device script) | OFFLINE state: «لا يوجد اتصال بالإنترنت» + «أعد المحاولة»; interaction blocked (§7) | — | `NOT_TESTED` | — | offline | — | — | — | — | — |
| CUST-16-007 | Offline | Network transition while the old network is disappearing (P8-DEF-001 guard) | Signed-in test customer · Staging · SM-A525F | Drop the only validated network while another is still connecting/unvalidated | OFFLINE until a validated network exists; never stuck 'online' | — | `NOT_TESTED` | — | switching | — | — | — | — | Automated guard: `NetTrackerTest` (11 tests incl. `callbacksNeverQueryTheSystem`) |
| CUST-16-008 | Offline | Offline state blocks catalog network-dependent interaction | Signed-in test customer · Staging · SM-A525F · offline | Tap section chips, item cards, search, banners | Blocked with explanation; no navigation into 'live' data | — | `NOT_TESTED` | — | offline | — | — | — | — | Not implemented today (§6) — expected FAIL until built |
| CUST-16-009 | Offline | Offline blocks cart Add | Signed-in test customer · Staging · SM-A525F · offline | Tap «أضف إلى السلة» | Blocked with explanation; cart unchanged | — | `NOT_TESTED` | — | offline | local cart unchanged | — | — | — | Gap proven 2026-09-19 (contaminated run) |
| CUST-16-010 | Offline | Offline blocks cart Remove | Signed-in test customer · Staging · SM-A525F · offline · cart populated | Tap remove / «إفراغ السلة» | Blocked; cart unchanged | — | `NOT_TESTED` | — | offline | — | — | — | — | — |
| CUST-16-011 | Offline | Offline blocks cart quantity mutation | Signed-in test customer · Staging · SM-A525F · offline | Tap + / − | Blocked; quantity unchanged | — | `NOT_TESTED` | — | offline | — | — | — | — | — |
| CUST-16-012 | Offline | Offline blocks order submission | Signed-in test customer · Staging · SM-A525F · offline · review open | Tap «أرسل الطلب» | Blocked before any request; no success UI | — | `NOT_TESTED` | — | offline | order count unchanged | — | — | — | — |
| CUST-16-013 | Offline | Offline blocks other network mutations | Signed-in test customer · Staging · SM-A525F · offline | Address add/edit/delete, custom order send, rating, profile edit, favorites, promo apply | Each blocked with explanation | — | `NOT_TESTED` | — | offline | no rows written | — | — | — | List of mutations comes from the §38 audit |
| CUST-16-014 | Offline | Every attempted blocked action has an understandable response | Signed-in test customer · Staging · SM-A525F · offline | Attempt 009–013 | Each attempt shows the offline explanation | — | `NOT_TESTED` | — | offline | — | — | — | — | — |
| CUST-16-015 | Offline | No silent taps | Signed-in test customer · Staging · SM-A525F · offline | Tap every clickable node | Every tap → visible response | — | `NOT_TESTED` | — | offline | — | — | — | — | — |
| CUST-16-016 | Offline | No infinite loading | Signed-in test customer · Staging · SM-A525F · offline | Attempt actions; UIA +30 s | No spinner after 30 s | — | `NOT_TESTED` | — | offline | — | — | — | — | — |
| CUST-16-017 | Offline | No false empty-market state | Signed-in test customer · Staging · SM-A525F · offline | All offline probes | «نعمل حاليًا على إضافة المتاجر والمنتجات» never shown | — | `NOT_TESTED` | — | offline | — | — | — | — | Held in every 2026-09-19 probe |
| CUST-16-018 | Offline | No fake success | Signed-in test customer · Staging · SM-A525F · offline | All offline attempts | No success toasts/states | — | `NOT_TESTED` | — | offline | SoT unchanged | — | — | — | — |
| CUST-16-019 | Offline | «أعد المحاولة» while still offline stays safely offline | Signed-in test customer · Staging · SM-A525F · offline | Tap «أعد المحاولة» ×3 | Stays OFFLINE; no crash; no spinner loop | — | `NOT_TESTED` | — | offline | — | — | — | — | — |
| CUST-16-020 | Offline | «أعد المحاولة» after connectivity is restored | Signed-in test customer · Staging · SM-A525F · offline | Restore; wait VALIDATED; tap «أعد المحاولة» | Leaves OFFLINE; fresh data | — | `NOT_TESTED` | — | recovering | — | — | — | — | — |
| CUST-16-021 | Offline | Automatic recovery after connectivity returns | Signed-in test customer · Staging · SM-A525F · offline | Restore; no taps; UIA +3/+8/+15 s | OFFLINE state removed automatically after real connectivity | — | `NOT_TESTED` | — | recovering | — | — | — | — | 2026-09-19: banner removed +6 s after restore |
| CUST-16-022 | Offline | Authoritative refresh after recovery | As 021 with a server change made while offline (Admin) | Recover; read screen | Screen shows the server change | — | `NOT_TESTED` | — | recovering | UI == SoT | — | — | — | — |
| CUST-16-023 | Offline | Safe cart/user state preserved after recovery | Cart populated before offline | Recover | Cart kept (re-validated against server) | — | `NOT_TESTED` | — | recovering | — | — | — | — | — |
| CUST-16-024 | Offline | No reinstall required | After recovery | Observe | Works without reinstall | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-16-025 | Offline | No data clear required | After recovery | Observe | Works without clearing data | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-16-026 | Offline | No unnecessary logout | After recovery | Observe | Still signed in | — | `NOT_TESTED` | — | online | no new password_login | — | — | — | — |
| CUST-16-027 | Degraded | Internet interface exists but API host unreachable | Signed-in test customer · Staging · SM-A525F | Block only staging-api (harness §38) | Explicit recoverable failure; OFFLINE-equivalent handling per §7.12; no empty market | — | `NOT_TESTED` | — | API unreachable | — | — | — | — | §7.12: interface ≠ reachability |
| CUST-16-028 | Degraded | DNS resolution failure | Signed-in test customer · Staging · SM-A525F | Private DNS pointed to an unresolvable host (restore after) | As 027 | — | `NOT_TESTED` | — | DNS fail | — | — | — | — | Changes a phone setting — Owner approval, restore |
| CUST-16-029 | Degraded | Connection timeout | Signed-in test customer · Staging · SM-A525F | Black-hole route to API (harness) | Timeout → explicit failure within client timeout; no hang | — | `NOT_TESTED` | — | timeout | — | — | — | — | — |
| CUST-16-030 | Degraded | Very slow network | Signed-in test customer · Staging · SM-A525F | Throttled link (emulator netspeed or router shaping) | Loading then result; no premature error; no double submit | — | `NOT_TESTED` | — | slow | — | — | — | — | Emulator `-netspeed` |
| CUST-16-031 | Degraded | High latency | Signed-in test customer · Staging · SM-A525F | Emulator `-netdelay` | Usable; explicit loading | — | `NOT_TESTED` | — | latency | — | — | — | — | — |
| CUST-16-032 | Degraded | Repeated network flapping | Signed-in test customer · Staging · SM-A525F | Toggle Wi-Fi ×10 at 5 s intervals | Final state correct; no crash; no stuck state | — | `NOT_TESTED` | — | flapping | — | — | — | — | — |
| CUST-16-033 | Degraded | Network disappears while loading catalog | Signed-in test customer · Staging · SM-A525F | Cut during initial load | OFFLINE state; no partial 'empty' market | — | `NOT_TESTED` | — | cut mid-load | — | — | — | — | — |
| CUST-16-034 | Degraded | Network disappears while refreshing | Signed-in test customer · Staging · SM-A525F | Cut during pull-to-refresh | OFFLINE state; content kept but not live-interactive | — | `NOT_TESTED` | — | cut mid-refresh | — | — | — | — | — |
| CUST-16-035 | Degraded | Network disappears while opening product | Signed-in test customer · Staging · SM-A525F | Cut during item detail/options load | OFFLINE state | — | `NOT_TESTED` | — | cut | — | — | — | — | — |
| CUST-16-036 | Degraded | Network disappears while obtaining quote | Signed-in test customer · Staging · SM-A525F | Cut during review/quote | OFFLINE state; no stale total shown as final | — | `NOT_TESTED` | — | cut | — | — | — | — | — |
| CUST-16-037 | Degraded | Network disappears during final order submission | Signed-in test customer · Staging · SM-A525F | Cut after tapping «أرسل الطلب» | Ambiguous result handled: on reconnect the app shows the committed order once or allows a safe retry; never a duplicate | — | `NOT_TESTED` | — | cut mid-submit | order count +0 or +1, never +2 | — | — | — | Ties to CUST-13-007/008 |
| CUST-16-038 | API | API 401 while network is available | Signed-in test customer · Staging · SM-A525F | Revoke session server-side (password reset on test account) then act | Refresh fails → explicit re-login; no loop | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-16-039 | API | API 403 | Signed-in test customer · Staging · SM-A525F | Hit a forbidden action (e.g. blocked account)  | Explicit denial message | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-16-040 | API | API 409 | Signed-in test customer · Staging · SM-A525F | Trigger a conflict (e.g. address limit / state conflict) | Explicit conflict message | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-16-041 | API | API 422 | Signed-in test customer · Staging · SM-A525F | Trigger 422 if any customer path returns it | Explicit message | — | `NOT_TESTED` | — | online | — | — | — | — | Audit (§38) states which customer paths return 422 |
| CUST-16-042 | API | API 429 if applicable | Signed-in test customer · Staging · SM-A525F | Exceed OTP/login rate limit | Explicit 'try later'; recovers after window | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-16-043 | API | API 500 | Signed-in test customer · Staging · SM-A525F | Simulated 5xx (harness) | Explicit recoverable failure; no fake success | — | `NOT_TESTED` | — | online | — | — | — | — | Needs a fault-injection harness — to be approved |
| CUST-16-044 | API | Temporary API outage then recovery | Signed-in test customer · Staging · SM-A525F | Stop reaching API 60 s then restore | Failure then automatic/Retry recovery | — | `NOT_TESTED` | — | outage | — | — | — | — | Staging API must not be stopped without Owner approval — prefer client-side block |
| CUST-16-045 | API | True empty state distinguishable from network/API failure | Genuinely empty geography vs offline | Compare both screens | Different texts: empty = «نعمل حاليًا على إضافة المتاجر والمنتجات»; failure = offline/error | — | `NOT_TESTED` | — | online / offline | — | — | — | — | L1-018 + L1-019 evidence |

## 29 · CUST-17 — Android app lifecycle / interruption

Test key screens under foreground, background, process death, reopen, screen lock and memory pressure where practical.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-17-001 | Lifecycle | Home → background → foreground | Signed-in test customer · Staging · SM-A525F | Open تسوق; HOME; wait 30 s; return | Same screen; data refreshed or kept; no crash | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-17-002 | Lifecycle | Catalog → background → foreground | Signed-in test customer · Staging · SM-A525F | Open a section; HOME; return | Same section and scroll position where designed | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-17-003 | Lifecycle | Cart → background → foreground | Signed-in test customer · Staging · SM-A525F · cart populated | Open سلتي; HOME; return | Cart intact | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-17-004 | Lifecycle | Checkout → background → foreground | Signed-in test customer · Staging · SM-A525F · checkout review open | HOME; return | Review intact; totals re-read; no auto-submit | — | `NOT_TESTED` | — | online | order count unchanged | — | — | — | — |
| CUST-17-005 | Lifecycle | Order detail → background → foreground | Signed-in test customer · Staging · SM-A525F · an order exists | Open order detail; HOME; return | Detail intact; status refreshed | — | `NOT_APPLICABLE` | — | online | status equals SoT | — | — | — | **N/A:** No order-detail screen exists (order cards only; `CustomerApi.order(id)` unused). Orders-tab lifecycle is covered by CUST-17-022. · Use a disposable test order, not #1050 progression |
| CUST-17-006 | Lifecycle | Android kills process from Home state | Signed-in test customer · Staging · SM-A525F | HOME; `am kill com.rahalgo.customer.debug`; relaunch | Restored cleanly; signed in | — | `NOT_TESTED` | — | online | — | — | — | — | `am kill` only kills background processes — background first |
| CUST-17-007 | Lifecycle | Process killed with cart populated | Signed-in test customer · Staging · SM-A525F · cart populated | HOME; `am kill`; relaunch | Cart restored per persistence contract | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-17-008 | Lifecycle | Process killed during safe non-committed checkout | Signed-in test customer · Staging · SM-A525F · review open, not submitted | HOME; `am kill`; relaunch | No order created; cart intact | — | `NOT_TESTED` | — | online | order count unchanged | — | — | — | — |
| CUST-17-009 | Lifecycle | Process killed after order may have been committed | Signed-in test customer · Staging · SM-A525F | Submit; kill immediately; relaunch | Committed order visible once in طلباتي; no duplicate | — | `NOT_TESTED` | — | online | order count +1 exactly | — | — | — | Same idempotency concern as CUST-13-014 |
| CUST-17-010 | Lifecycle | Reopen after process death | After 006–009 | Relaunch | Consistent state; no stale error overlay | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-17-011 | Lifecycle | Screen lock/unlock | Signed-in test customer · Staging · SM-A525F | Power off screen; unlock | Same screen; no crash | — | `NOT_TESTED` | — | online | — | — | — | — | Owner unlocks; no PIN handling by tooling |
| CUST-17-012 | Lifecycle | App left backgrounded for an extended period | Signed-in test customer · Staging · SM-A525F | Background ≥ 30 min; return | Refreshes authoritative data; no stale live data | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-17-013 | Lifecycle | Access token expires while backgrounded | Signed-in test customer · Staging · SM-A525F | Background > access-token TTL (15 min); return; act | Silent refresh; action succeeds; no forced login | — | `NOT_TESTED` | — | online | `auth.refresh` audit | — | — | — | Access TTL 15 min (`cmd/api/main.go`) |
| CUST-17-014 | Lifecycle | Network changes while backgrounded | Signed-in test customer · Staging · SM-A525F | Background; toggle Wi-Fi↔data; return | Correct online/offline state on return | — | `NOT_TESTED` | — | switching | — | — | — | — | — |
| CUST-17-015 | Lifecycle | Location permission changes while backgrounded | Signed-in test customer · Staging · SM-A525F | Background; revoke/grant in Settings; return | No crash; state reflects permission | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-17-016 | Lifecycle | Notification permission changes while backgrounded | Signed-in test customer · Staging · SM-A525F | Background; toggle notifications; return | No crash; ordering unaffected | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-17-017 | Lifecycle | Android reboot with an existing valid session | Signed-in test customer · Staging · SM-A525F | Reboot device (Owner consent) | No crash at boot; session and cart persisted for the next launch | — | `NOT_TESTED` | — | online | — | — | — | — | Owner consent required for reboot |
| CUST-17-018 | Lifecycle | Reopen after reboot | After 017 | Launch | Signed in; cart per contract; FCM re-registers if needed | — | `NOT_TESTED` | — | online | device token row present | — | — | — | — |
| CUST-17-019 | Lifecycle | Repeated Back presses | Signed-in test customer · Staging · SM-A525F | From deep screen press BACK ×10 | Leaves app cleanly; no crash; no loop | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-17-020 | Lifecycle | Repeated Home/app-switch transitions | Signed-in test customer · Staging · SM-A525F | HOME/recents ×10 in 30 s | No crash; no duplicate requests beyond refresh | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-17-021 | Lifecycle | No impossible navigation stack after restoration | After 006–018 | Navigate tabs and BACK | No duplicated screens; BACK behaves normally | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-17-022 | Lifecycle | Orders tab across background and process death (added) | Signed-in test customer · Staging · SM-A525F · valid default address · open disposable order | Open طلباتي; HOME; `am kill`; relaunch | Orders tab data re-read from server; no stale status | — | `NOT_TESTED` | — | online | status == SoT | — | — | — | Added: replaces the order-detail lifecycle rows (no detail screen) |

## 30 · CUST-18 — Remote / Admin / server-side operational changes

These tests verify Customer reaction to backend/Admin truth. They are NOT a repeat of Admin parity testing. For every remote change verify both UI reaction and backend truth.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-18-001 | Remote | launch.customer_signup ON → OFF | Signup screen open · Change made through the Staging Admin panel (recorded before/after, restored) | Flip OFF; submit signup | 503 `launch_closed` → Owner notice; no account created | — | `NOT_TESTED` | — | online | users count unchanged | — | — | — | — |
| CUST-18-002 | Remote | launch.customer_signup OFF → ON | Signup closed · Change made through the Staging Admin panel (recorded before/after, restored) | Flip ON; retry | Signup proceeds | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-18-003 | Remote | launch.customer_browse ON → OFF | Signed-in test customer · Staging · SM-A525F · market open · Change made through the Staging Admin panel (recorded before/after, restored) | Flip OFF; refresh/navigate | Owner notice (`launch.notice`); no empty market; no crash | — | `NOT_TESTED` | — | online | flag before/after | — | — | — | — |
| CUST-18-004 | Remote | launch.customer_browse OFF → ON | After 003 | Flip ON; refresh | Market returns without reinstall | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-18-005 | Remote | launch.customer_orders ON → OFF | Signed-in test customer · Staging · SM-A525F · cart built · Change made through the Staging Admin panel (recorded before/after, restored) | Flip OFF; tap «أرسل الطلب» | 503 `launch_closed` + notice; no order; no spinner | — | `NOT_TESTED` | — | online | order count unchanged | — | — | — | Prior evidence: P8-L1-020 (DEVICE_VERIFIED 2026-09-18) |
| CUST-18-006 | Remote | launch.customer_orders OFF → ON | After 005 | Flip ON; navigate away/back; submit test order (disposable) | Submit available again; cart preserved | — | `NOT_TESTED` | — | online | order count +1 exactly | — | — | — | — |
| CUST-18-007 | Remote | launch.customer_custom_orders transitions | Custom-order screen open · Change made through the Staging Admin panel (recorded before/after, restored) | Flip OFF/ON; send | OFF → explicit denial; ON → works | — | `NOT_TESTED` | — | online | custom order count | — | — | — | Feature exists (tab «طلب خاص») — see CUST-CUSTOM |
| CUST-18-008 | Remote | Platform temporarily closes while app is open | Signed-in test customer · Staging · SM-A525F · Change made through the Staging Admin panel (recorded before/after, restored) (service closure) | Close platform; act | `temporarily_unavailable` explicit; structure stays; ordering blocked | — | `NOT_TESTED` | — | online | service_closure row | — | — | — | Admin action is audited (`admin.platform_closure`) |
| CUST-18-009 | Remote | Platform reopens | After 008 | Reopen; refresh | Ordering available again | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-18-010 | Remote | Zone closes while browsing | Signed-in test customer · Staging · SM-A525F · Change made through the Staging Admin panel (recorded before/after, restored) (zone hours) | Close the test zone | `zone_closed_now` explicit | — | `NOT_TESTED` | — | online | zone hours row | — | — | — | Never the zone referenced by #1050 |
| CUST-18-011 | Remote | Zone reopens | After 010 | Reopen | Ordering available | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-18-012 | Remote | Product becomes unavailable | Signed-in test customer · Staging · SM-A525F · item visible · Change made through the Staging Admin panel (recorded before/after, restored) | Mark item unavailable; refresh | Shown unavailable; cannot be ordered | — | `NOT_TESTED` | — | online | item row | — | — | — | — |
| CUST-18-013 | Remote | Product becomes available again | After 012 | Mark available; refresh | Orderable again | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-18-014 | Remote | Section is retired | Signed-in test customer · Staging · SM-A525F · section open · Change made through the Staging Admin panel (recorded before/after, restored) | Deactivate a test section (PATCH active=false) | Section disappears after refresh; open screen handles it explicitly | — | `NOT_TESTED` | — | online | section active=false | — | — | — | Delete of a used section is 409 by contract — use deactivate |
| CUST-18-015 | Remote | Section activates | After 014 | Activate | Section returns | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-18-016 | Remote | Price changes | Signed-in test customer · Staging · SM-A525F · item in cart · Change made through the Staging Admin panel (recorded before/after, restored) | Change price; open review | Review shows the new server price; no silent old total | — | `NOT_TESTED` | — | online | quote == server | — | — | — | — |
| CUST-18-017 | Remote | Coverage configuration changes | Signed-in test customer · Staging · SM-A525F · Change made through the Staging Admin panel (recorded before/after, restored) | Shrink the test zone so the address falls outside | `address_outside_coverage` explicit | — | `NOT_TESTED` | — | online | zone geometry | — | — | — | — |
| CUST-18-018 | Remote | Selected address becomes unsupported | As 017 | Refresh / proceed to review | Explicit denial; must pick another address | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-18-019 | Remote | Minimum-version policy if exposed | `app.min_version.customer` setting | Raise min version above installed versionCode (Admin); relaunch | Explicit update-required behaviour if implemented | — | `NOT_TESTED` | — | online | setting value | — | — | — | Audit decides applicability (§38) |
| CUST-18-020 | Remote | Required-update behaviour if implemented | As 019 | As 019 | As 019 | — | `NOT_TESTED` | — | online | — | — | — | — | Audit decides applicability (§38) |
| CUST-18-021 | Remote | Browse closure closes every browse surface (added) | `launch.customer_browse`=OFF | API client: GET `/public/sections`, `/public/sections/{id}/items`, `/public/search/items`, `/public/suggest` | Consistent with the pre-launch contract (closed or explicitly allowed by Owner) | — | `NOT_TESTED` | — | online | — | — | — | — | Added. CAF-05 (reported by audit): only `/public/home`, `/public/offers`, `/public/items/{id}` are gated |

## 31 · CUST-19 — Adversarial / security acceptance

Defensive acceptance testing of RahalGo's own application.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-19-001 | Sec | Rapid double/triple taps on important mutation buttons | Signed-in test customer · Staging · SM-A525F | Triple-tap add, submit, address save, rating, custom-order send (`input tap` ×3 in <300 ms) | Exactly one effect each | — | `NOT_TESTED` | — | online | row counts +1 | — | — | — | — |
| CUST-19-002 | Sec | Simultaneous navigation and mutation | Signed-in test customer · Staging · SM-A525F | Tap add then immediately switch tab | Effect applied once; UI consistent | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-19-003 | Sec | Back press during API request | Signed-in test customer · Staging · SM-A525F | Submit then BACK within 200 ms | No duplicate; result discoverable | — | `NOT_TESTED` | — | online | order count ≤ +1 | — | — | — | — |
| CUST-19-004 | Sec | Screen change during API request | Signed-in test customer · Staging · SM-A525F | Trigger load then change screen | No crash; no leaked result into wrong screen | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-19-005 | Sec | Repeated order submit | Signed-in test customer · Staging · SM-A525F | Submit ×3 rapidly | One order | — | `NOT_TESTED` | — | online | order count +1 | — | — | — | — |
| CUST-19-006 | Sec | Replay-safe order creation | API client with the test customer's token | Replay the exact submit request (same idempotency key if any) ×2 | Same order returned; no second order | — | `NOT_TESTED` | — | online | `idempotency_keys` / order count | — | — | — | Audit (§38) identifies the key mechanism |
| CUST-19-007 | Sec | Invalid quantity through client/API boundary | API client | Submit qty 0 / 1000000 | 400 validation; no order | — | `NOT_TESTED` | — | online | no order row | — | — | — | — |
| CUST-19-008 | Sec | Negative/zero/absurd quantity via tooling | API client | qty −1, 0, 2^31 | 400; never 500 | — | `NOT_TESTED` | — | online | — | — | — | — | VAL-1 profile |
| CUST-19-009 | Sec | Client-supplied monetary values are not trusted | API client | Add price/total fields to the order body | Ignored; server prices used | — | `NOT_TESTED` | — | online | order totals == server quote | — | — | — | — |
| CUST-19-010 | Sec | Access another Customer's order ID | Two test customers | Customer B GET/act on A's order id | 403/404 without existence leak | — | `NOT_TESTED` | — | online | — | — | — | — | AUTHZ-1 |
| CUST-19-011 | Sec | Customer token against privileged endpoints | API client | Call /admin/*, /driver/*, /merchant/*, /rep/* with a customer token | 401/403 everywhere | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-19-012 | Sec | Expired token | API client | Use an expired access token | 401 → refresh path; no data | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-19-013 | Sec | Revoked token/session | API client | Use tokens after password reset | 401; refresh fails | — | `NOT_TESTED` | — | online | — | — | — | — | Reset revokes all sessions (`revokeAllSessions`, SEC8) |
| CUST-19-014 | Sec | Malformed identifiers | API client | Non-UUID ids on every customer path | 400/404, never 500 | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-19-015 | Sec | Deep-link route manipulation if deep links exist | — | Send crafted intents/URIs | Safe handling or N/A | — | `NOT_TESTED` | — | any | — | — | — | — | Audit decides applicability (§38) |
| CUST-19-016 | Sec | Old screen/state cannot bypass a newly closed server rule | Signed-in test customer · Staging · SM-A525F | Close ordering server-side; submit from the stale screen | Server denies; UI explicit | — | `NOT_TESTED` | — | online | order count unchanged | — | — | — | — |
| CUST-19-017 | Sec | Account A logout → Account B login: no A data | Two test customers | A: cart/addresses/orders; logout; B login | No A cart/addresses/orders/notifications visible | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-19-018 | Sec | Two devices on the same Customer account | Second device/emulator | Login on both; act on both | Behaviour matches session contract | — | `NOT_TESTED` | — | online | sessions per client | — | — | — | — |
| CUST-19-019 | Sec | Concurrent actions from two sessions do not corrupt order state | As 018 | Submit/cancel concurrently | Consistent single outcome | — | `NOT_TESTED` | — | online | order state | — | — | — | — |
| CUST-19-020 | Sec | Cannot order outside serviceability by manipulating local state | API client | Submit with coordinates outside coverage / foreign address id | Server denies (`address_outside_coverage` / 404) | — | `NOT_TESTED` | — | online | no order | — | — | — | — |
| CUST-19-021 | Sec | Cannot order a retired/unavailable item via stale screen | Signed-in test customer · Staging · SM-A525F | Retire item server-side; submit stale cart | Server denies; explicit message | — | `NOT_TESTED` | — | online | no order | — | — | — | — |
| CUST-19-022 | Sec | Cannot bypass launch closure with an open screen | Signed-in test customer · Staging · SM-A525F | Close launch.customer_orders; submit | 503 `launch_closed` | — | `NOT_TESTED` | — | online | no order | — | — | — | P8-L1-020 prior evidence |
| CUST-19-023 | Sec | Tokens/secrets not printed in normal application logs | Signed-in test customer · Staging · SM-A525F | logcat during login/refresh/order | No tokens, OTP, passwords in logcat | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-19-024 | Sec | Production secrets not embedded in the Staging debug app | APK | Search dex/resources for production keys/hosts | None (Staging Firebase project only) | — | `NOT_TESTED` | — | any | — | — | — | — | CUST-00-006 complement |
| CUST-19-025 | Sec | No Customer-visible error dumps internal/server detail | Signed-in test customer · Staging · SM-A525F | Trigger 4xx/5xx | Mapped Arabic messages only; no stack/SQL text | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-19-026 | Sec | Signup confirm cannot take over an existing account (added) | Existing test customer X · attacker knows X's phone | API client: POST `/auth/signup/confirm` with X's phone and a new password — (a) `signup_verify`=true without a code; (b) `signup_verify`=false (Staging flip only with Owner approval) | Both denied; X's password unchanged; no session issued | — | `NOT_TESTED` | — | online | X password hash fingerprint unchanged; no new session | — | — | — | Added. **CAF-01 P0 (source-confirmed)**: with `signup_verify`=false `ConfirmSignup` skips the code and overwrites an existing account's password, then issues a session (`identity/service.go:486-559`). Current Prod/Staging value = true (Staging since 2026-09-19). Expected FAIL for (b) |
| CUST-19-027 | Sec | Client-supplied merchant_id is ignored (added) | API client | POST `/orders` with a valid cart plus a foreign/closed `merchant_id` | Server derives the merchant from the items; open-hours check uses the real source | — | `NOT_TESTED` | — | online | order.merchant_id == item source | — | — | — | Added. **CAF-03 HIGH (source-confirmed)**: `CreateTx` keeps a non-empty client `merchant_id` (`orders/service.go:303-305`) and runs the open-hours check on it |
| CUST-19-028 | Sec | Order in an unlaunched city/province is denied at create (added) | Active zone inside an inactive city (fixture) | API client submit; custom submit | Denied with the same reason availability gives | — | `NOT_TESTED` | — | online | no order | — | — | — | Added. CAF-06 (reported by audit): place classification is advisory; create paths enforce zones only |
| CUST-19-029 | Sec | Any-role token cannot misuse customer order routes (added) | Driver/merchant/rep test tokens | POST `/orders`, rating, complaint with non-customer roles | Per contract (every role also carries customer — `TestOneRole_EveryRoleBringsCustomer`) — decide and verify | — | `NOT_TESTED` | — | online | — | — | — | — | Added: the customer route group has no role check |

## 32 · CUST-20 — UI / UX / Arabic / RTL

Functional correctness includes understandable UI behaviour.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-20-001 | UI | Arabic RTL layout | Signed-in test customer · Staging · SM-A525F | UIA of every main screen; check bounds order | RTL mirroring correct | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-20-002 | UI | Main navigation direction/layout | Signed-in test customer · Staging · SM-A525F | UIA bottom bar | تسوق at right … حسابي at left (RTL) | — | `NOT_TESTED` | — | online | — | — | — | — | Tabs today: تسوق · سلتي · طلباتي · طلب خاص · حسابي |
| CUST-20-003 | UI | No clipped critical Arabic text | Signed-in test customer · Staging · SM-A525F | UIA text vs bounds on all screens | No truncation of prices/actions/errors | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-20-004 | UI | No overlapping buttons/text | Signed-in test customer · Staging · SM-A525F | UIA bounds intersection check | No overlapping clickable nodes | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-20-005 | UI | Long product name | Fixture item with long name (Staging, via Admin) | Open section/cart/review | Wraps/ellipsizes without breaking actions | — | `NOT_TESTED` | — | online | — | — | — | — | Fixture created and removed through Admin — recorded |
| CUST-20-006 | UI | Long address | Address with long label/details | Open review/address list | Readable; actions reachable | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-20-007 | UI | Large monetary values | Item priced high | Cart/review | Grouping separators; no overflow | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-20-008 | UI | Small/zero monetary values where valid | Zero delivery fee case | Review | Shows 0 ل.س correctly | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-20-009 | UI | Keyboard does not cover critical fields/actions | Signed-in test customer · Staging · SM-A525F | Focus each text field (login, address, promo, custom order) | Field and its action visible | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-20-010 | UI | Keyboard dismissal | Signed-in test customer · Staging · SM-A525F | BACK / tap outside | Keyboard closes; no lost input | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-20-011 | UI | Loading indicator disappears when operation finishes | Signed-in test customer · Staging · SM-A525F | Trigger each async action | Spinner gone on completion/failure | — | `NOT_TESTED` | — | online / slow | — | — | — | — | — |
| CUST-20-012 | UI | Errors are understandable | Signed-in test customer · Staging · SM-A525F | Trigger known error codes | Arabic message mapped from `errors.*`, not raw codes | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-20-013 | UI | Retry action visible where recovery is possible | Signed-in test customer · Staging · SM-A525F | Trigger recoverable failures | «أعد المحاولة» present | — | `NOT_TESTED` | — | offline / API down | — | — | — | — | — |
| CUST-20-014 | UI | Disabled action visually understandable | Signed-in test customer · Staging · SM-A525F | Disabled submit/add states | Disabled state visible with reason where relevant | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-20-015 | UI | No tap target silently accepts touch without response | Signed-in test customer · Staging · SM-A525F | Tap every clickable node per screen | Each tap yields a visible response | — | `NOT_TESTED` | — | online | — | — | — | — | Automatable with UIA clickable enumeration |
| CUST-20-016 | UI | Usable on the actual SM-A525F screen | Signed-in test customer · Staging · SM-A525F | Full walkthrough | Everything reachable | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-20-017 | UI | System font scaling keeps critical actions usable | Signed-in test customer · Staging · SM-A525F | `settings put system font_scale 1.3` (restore after) | Actions reachable; no overlap | — | `NOT_TESTED` | — | online | — | — | — | — | Restore font_scale after |
| CUST-20-018 | UI | English/localization only if supported | — | Check app resources for non-Arabic locales | Arabic only → N/A unless a locale switch exists | — | `NOT_APPLICABLE` | — | any | — | — | — | — | **N/A:** Arabic only: no `values-xx` locale folders and no language switch in the Customer app (audit §38). · Decided by the audit (§38) |
| CUST-20-019 | UI | Every customer-path error code has a meaningful message (added) | API errors | Trigger `not_found`, `in_progress`, `comms_closed`, `comms_no_driver` | Specific Arabic messages — not «تعذر الاتصال — حاول بعد قليل» | — | `NOT_TESTED` | — | online | — | — | — | — | Added. CAF-18: these codes are unmapped and fall back to a misleading 'connection' message |

## 33 · CUST-21 — Performance / resilience

Capture real measurements, not subjective statements. Do not set arbitrary pass thresholds without documenting where the threshold came from; use established PF thresholds where they exist (P-8 `PF` is NOT YET QUALIFIED).

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-21-001 | Perf | Cold start | Signed-in test customer · Staging · SM-A525F | `am start -W` ×5 after force-stop | TotalTime recorded; threshold per PF source | — | `NOT_TESTED` | — | online | — | — | — | — | No arbitrary thresholds — P-8 `PF` wave is NOT YET QUALIFIED (needs dense fixture) |
| CUST-21-002 | Perf | Warm start | Signed-in test customer · Staging · SM-A525F | `am start -W` ×5 from background | Recorded | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-21-003 | Perf | Home/catalog initial load | Signed-in test customer · Staging · SM-A525F | Time to first item text in UIA | Recorded | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-21-004 | Perf | Section navigation responsiveness | Signed-in test customer · Staging · SM-A525F | Switch sections ×10; gfxinfo | Recorded jank % | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-21-005 | Perf | Long-scroll responsiveness | Dense fixture section (30–50 items) | Fling ×10; gfxinfo | Recorded p90/p99 frame times | — | `NOT_TESTED` | — | online | — | — | — | — | Fixture per P8 PF note |
| CUST-21-006 | Perf | Cart mutation responsiveness | Signed-in test customer · Staging · SM-A525F | +/− ×20 | Recorded; no lag spikes | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-21-007 | Perf | Checkout load | Signed-in test customer · Staging · SM-A525F | Open review; time to totals | Recorded | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-21-008 | Perf | Order-list load | Signed-in test customer · Staging · SM-A525F | Open طلباتي | Recorded | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-21-009 | Perf | Order-detail load | Signed-in test customer · Staging · SM-A525F | Open detail | Recorded | — | `NOT_APPLICABLE` | — | online | — | — | — | — | **N/A:** No order-detail screen exists. Order-list load is CUST-21-008. |
| CUST-21-010 | Perf | Network recovery time | Signed-in test customer · Staging · SM-A525F | Offline → online; time to banner removal and fresh data | Recorded (2026-09-19 baseline: validated +6 s) | — | `NOT_TESTED` | — | flapping | — | — | — | — | — |
| CUST-21-011 | Perf | Repeated foreground/background cycle | Signed-in test customer · Staging · SM-A525F | ×50 via `am start`/HOME | No crash; memory recorded | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-21-012 | Perf | Repeated refresh cycle | Signed-in test customer · Staging · SM-A525F | Pull-to-refresh ×30 | No crash; request count sane | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-21-013 | Perf | No obvious unbounded memory growth | Signed-in test customer · Staging · SM-A525F | `dumpsys meminfo` at start/after 20 min use | No monotonic growth beyond noise | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-21-014 | Perf | No crash under repeated normal navigation | Signed-in test customer · Staging · SM-A525F | Scripted navigation loop 20 min | 0 crashes/ANRs | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-21-015 | Perf | Degraded network remains understandable and recoverable | Signed-in test customer · Staging · SM-A525F | Throttled/slow network walkthrough | Explicit loading → result or recoverable failure | — | `NOT_TESTED` | — | slow | — | — | — | — | — |

## 34 · CUST-22 — Final Customer regression gate

Customer must NOT be declared ACCEPTED until every row below is PASS.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-22-001 | Gate | Every actual Customer surface mapped to this document | — | Re-run the §38 audit against the release candidate | No unmapped surface | — | `NOT_TESTED` | — | — | — | — | — | — | Gate item 1 |
| CUST-22-002 | Gate | Every mandatory case PASS or justified NOT_APPLICABLE | — | Recount §39 from the tables | 0 NOT_TESTED / FAIL / BLOCKED | — | `NOT_TESTED` | — | — | — | — | — | — | Gate item 2 |
| CUST-22-003 | Gate | L1-019 truly PASS under the offline blocking/retry contract | — | CUST-16 mandatory rows on the physical device | PASS | — | `NOT_TESTED` | — | offline | — | — | — | — | Gate item 3 |
| CUST-22-004 | Gate | All Customer P0/P1 defects CLOSED | — | Defect ledger | None open | — | `NOT_TESTED` | — | — | — | — | — | — | Gate item 4 — includes CAF-01 |
| CUST-22-005 | Gate | No unresolved launch-affecting security defect | — | Defect ledger | None | — | `NOT_TESTED` | — | — | — | — | — | — | Gate item 5 |
| CUST-22-006 | Gate | No unresolved duplicate-order defect | — | Defect ledger | None | — | `NOT_TESTED` | — | — | — | — | — | — | Gate item 6 — includes CAF-02 |
| CUST-22-007 | Gate | No unresolved cross-account leakage | — | Defect ledger | None | — | `NOT_TESTED` | — | — | — | — | — | — | Gate item 7 — includes CAF-09 |
| CUST-22-008 | Gate | No unresolved financial/source-of-truth defect | — | Defect ledger | None | — | `NOT_TESTED` | — | — | — | — | — | — | Gate item 8 — includes CAF-03, CAF-08 |
| CUST-22-009 | Gate | Every fixed defect has regression evidence | — | Regression column | Filled for every fixed defect | — | `NOT_TESTED` | — | — | — | — | — | — | Gate item 9 |
| CUST-22-010 | Gate | Automated impacted suites pass | — | Go full suite; Kotlin unit suites; guards | Green | — | `NOT_TESTED` | — | — | — | — | — | — | Gate item 10 |
| CUST-22-011 | Gate | Physical-device mandatory cases pass | — | Device rows | PASS | — | `NOT_TESTED` | — | — | — | — | — | — | Gate item 11 |
| CUST-22-012 | Gate | Zero accidental Production dependency | — | CUST-00-005/006, CUST-19-024 | PASS | — | `NOT_TESTED` | — | — | — | — | — | — | Gate item 12 |
| CUST-22-013 | Gate | Staging data reconciled/known after tests | — | Before/after SoT reads; moneycheck | Known; 51/51 | — | `NOT_TESTED` | — | — | read-only SQL | — | — | — | Gate item 13 |
| CUST-22-014 | Gate | Production mutations zero unless authorized | — | Production identity + audit read | 0 | — | `NOT_TESTED` | — | — | identity endpoint | — | — | — | Gate item 14 |
| CUST-22-015 | Gate | No unexplained NOT_TESTED/BLOCKED rows | — | §39 | 0 | — | `NOT_TESTED` | — | — | — | — | — | — | Gate item 15 |
| CUST-22-016 | Gate | Clean end-to-end Customer smoke run after all fixes | Release-candidate debug build | Install → signup → address → browse → cart → submit → track → rate → logout | All PASS on the device | — | `NOT_TESTED` | — | online | SoT per step | — | — | — | Gate item 16 |

---

## 35 · Release build is a separate gate

Passing Customer Debug/Staging acceptance does NOT automatically mean the Google Play
build is accepted. After the whole project reaches final freeze, Customer release
acceptance must later include the actual signed release/AAB/Play distribution path.
**Do not build that now.**

## 36 · Later project sequence

This Customer Master does not replace the other application acceptance documents.
After Customer is fully accepted, create equivalent independent acceptance masters for
the remaining applications; each app is tested independently with the same philosophy.

After all applications are independently accepted, perform full operational E2E
testing: Customer ↔ Backend ↔ Admin/Ops ↔ Driver ↔ Merchant ↔ Rep ↔
Realtime/Notifications ↔ Financial state ↔ Audit.

Then test Admin operational behaviour itself: every relevant action; every state
transition; every permission boundary; application → Admin propagation; Admin →
application propagation; realtime effects; financial effects; audit effects;
recovery/failure effects. **Admin functional parity being CLOSED does not replace this
later operational E2E acceptance.**

## 37 · Final project principle

We do not want to say "The app opened and looked fine." We want to be able to prove:
we tested normal behaviour; wrong behaviour; failures; recovery; interruptions;
repeated actions; network loss; stale state; server-side changes; authorization
boundaries; duplicate prevention; source-of-truth consistency; real physical-device
behaviour; every defect was recorded; every fix received regression coverage; every
application was accepted independently; and finally the whole RahalGo operational
cycle was accepted as one system. **Only after that may RahalGo be considered ready for
final release preparation.**

---

## 38 · Coverage audit (read-only, 2026-09-19)

### 38.1 · Method

Read-only inspection at `HEAD 1fc4da52` — **no code changed, no Staging or Production
mutation, no test executed.** Three independent source audits plus direct verification
of every high-severity claim:

| Audit | Scope |
|---|---|
| A · Customer app surfaces | every Kotlin file of `mobile/app-customer`, the shared `ui`, `shared`, `map` modules it uses, manifests, strings, `build.gradle.kts` |
| B · API contracts | every endpoint the app calls vs every customer-reachable route in `backend/internal/server/server.go`, middleware gates, error codes, idempotency, quote/pricing, custom vs normal order rules |
| C · Tests and defects | Kotlin unit/instrumented tests, Go tests on customer flows, `TEST_TRUTH`, P-8 rows (C2 · C3 · C4 · C5 · L1 · N1 · PF · E2E), defect/risk/gap ledgers, older customer closure documents |
| Verification | the P0/HIGH claims were re-read directly: `identity/service.go:486-559`, `ui/Attempt.kt` (`isDecided`), `server/idempotency.go:221-226`, `orders/service.go:303-305,378-396`, `server/customer_handlers.go:422-428`, `orders/OrdersViewModel.kt:99,145`, `cart/CartScreen.kt:326` |

### 38.2 · Actual Customer screens and surfaces discovered

Single activity (`MainActivity`, no navigation library). Screens are chosen by a tab
enum plus an overlay state.

| Surface | Source | Master groups |
|---|---|---|
| Gate: update required (HTTP 426) | `ui/UpdateGate.kt` | CUST-02-011, CUST-18-019/020 |
| Gate: session restore · restore-offline screen | `ui/AppFrame.kt` AuthGate | CUST-06-028/029 |
| Guest shell (default signed-out browsing) · NeedAccount | `MainActivity:1166-1278` | CUST-06-030, CUST-09-029, CUST-CUSTOM-017 |
| PreLaunch screen (`launch.customer_browse`=false) | `MainActivity:1221-1252` | CUST-02-007, CUST-08-017 |
| Brand intro splash | `ui/BrandIntroHost.kt` | CUST-ENG-013 |
| Auth: password login · OTP login (flag) · signup (phone→code→details, referral) · password reset (WhatsApp ticket) | `ui/AuthScreen.kt`, `SignupScreen.kt`, `ResetScreen.kt`, `AuthViewModel.kt` | CUST-04, CUST-05, CUST-06 |
| Tab تسوق: banners · search · coverage notice · section rail · 3-column grid · pull-to-refresh | `shop/ShopScreen.kt` | CUST-08, CUST-09 |
| Item options sheet (the only product surface) | `shop/ItemOptionsSheet.kt` | CUST-10 |
| Tab سلتي (cart = checkout): lines · suggestions · address card · promo · payment (cash / wallet) · totals · availability notes · change review · send | `cart/CartScreen.kt` | CUST-11, CUST-12, CUST-13 |
| Tab طلباتي and drawer سجل الطلبات: order cards · cancel · complaint · rate · auto rating prompt | `orders/OrdersScreen.kt`, `OrderCard.kt` | CUST-14 |
| Tab طلب خاص (custom order) | `custom/CustomScreen.kt` | CUST-CUSTOM |
| Tab حسابي: photo · name · phone change · WhatsApp verification · password change · address book · account deletion | `ui/AccountScreen.kt` | CUST-06, CUST-07 |
| Address sheet · address editor · map picker (search, reverse geocode, «موقعي») | `ui/Address.kt`, `map/PickPoint.kt` | CUST-03, CUST-07 |
| City picker (drawer header) · demand «request my area / notify me» | `CityPicker.kt`, `PreCart.kt` | CUST-07-029/030 |
| Wallet overlay · statement · print/PDF | `ui/WalletScreen.kt`, `StatementPrint.kt` | CUST-WAL |
| Inbox overlay (bell) | `ui/InboxSheet.kt` | CUST-15-017 |
| Favorites · Offers · Invite · Tickets | `mine/MineScreens.kt` | CUST-ENG, CUST-SUP |
| Chat FAB · order chat sheet · chats list | `ui/OrderChat.kt`, `ui/Chats.kt`, `chat/LiveChat.kt` | CUST-SUP |
| Static pages · contact links | `ui/PlatformPages.kt`, `ContactPage.kt` | CUST-ENG-010/011 |
| Drawer (items per auth state) · theme toggle · logout | `ui/Drawer.kt`, `Menu.kt` | CUST-ENG-012/014, CUST-06-012 |
| NetBanner · OfflineScreen · per-screen error states | `ui/Connectivity.kt`, `AppFrame.kt` | CUST-16 |
| FCM push (urgent/default channels) · destinations (order_chat, offer) | `push/PushService.kt`, `ui/Engagement.kt` | CUST-15 |
| Realtime WebSocket `/api/v1/ws` → refresh signal | `shared/net/LiveSocket.kt` | CUST-15, CUST-14-007 |
| Invite deep link `https://rahalgo.com/signup?ref=` · Play install referrer | manifest, `Invited.kt` | CUST-04-020..022, CUST-19-015 |

**Not present (drives `NOT_APPLICABLE` rows):** product-detail screen; order-detail
screen (`CustomerApi.order` unused); tracking map / driver location / driver phone
(removed deliberately); reorder; scheduled orders; order notes for normal orders;
language switch (Arabic only); in-app update API; Play in-app review; analytics SDK;
customer emergency button; ticket reply; top-up/payout for customers.

**Permissions (manifest):** INTERNET · ACCESS_NETWORK_STATE · POST_NOTIFICATIONS ·
ACCESS_COARSE_LOCATION · ACCESS_FINE_LOCATION. No background location, no CAMERA/storage
(the avatar uses system pickers). `allowBackup=false`.

### 38.3 · Actual Customer API / contract groups discovered

Every path the app calls exists in the backend (no path mismatch).

| Group | Endpoints (base `/api/v1`) | Gating |
|---|---|---|
| Public catalog | `public/home`, `public/cities`, `public/sections/{id}/items`, `public/items/{id}`, `public/suggest`, `public/search/items`, `public/offers` | browse gate only on home, offers, items/{id} (CAF-05) |
| Serviceability | `public/availability`, `public/platform`, `public/contact`, POST `public/quote` | ungated |
| Auth | `auth/login`, `otp/request|verify`, `wa/ticket`, `password/reset/verify|confirm`, `signup/request|verify|confirm`, `me`, `logout`, `refresh`, `referral`, `password`, `phone/request|confirm`, `whatsapp/confirm`, `account/delete/request|confirm` | only `signup/request` is launch-gated (CAF-01) |
| Orders | POST `orders` (+Idempotency-Key), POST `orders/custom` (+key), POST `promo/preview`, POST `demand`, POST `orders/{id}/cancel`, POST `orders/{id}/rating`, GET `my/ratings`, GET `my/orders` (page 1 only), complaint-reasons, POST complaint, GET `my/tickets` | normal: customer_orders + merchant_orders + ordering; custom: customer_custom_orders + ordering |
| Profile & addresses | `my/favorites`, `me/summary`, `me/name`, `me/avatar`, `my/addresses` (+`{id}`, `{id}/default`) | auth |
| Wallet & inbox | `my/wallet`, `me/payouts` (read only), `me/notifications`, `me/notifications/read` | auth |
| Devices, chat, geo, realtime | `me/devices`, `my/chats`, `orders/{id}/messages`, `geo/reverse`, `geo/search`, `ws` | auth |

**Reachable but never called by the app:** `my/orders/{id}`, `my/orders/{id}/complaint`
(GET), `my/warnings`, `me/reputation`, `me/devices/test`, POST `me/payouts` (403 for
customers), `me/demand/cancel`, `auth/password/reset/request`, `auth/whatsapp/request`,
`public/sections`, `public/search`, `public/banners`, `public/zone`, `public/invite`,
`public/join`, `public/coverage-request`. They remain security-relevant surface
(CUST-19).

**Availability vocabulary** (`orders/availability.go`) — all 12 reasons have app strings
(`ui/ServiceReason.kt`); province/city/area reasons are advisory only at create time
(CAF-06).

### 38.4 · Audit findings carried into this document (not yet runtime-proven)

> **Reconciled 2026-09-19 in §40.10** — every finding re-read in the source; stop-class ones received `CUST-DEF-nnn` IDs. The status column below is the original audit wording.

Found by reading the source. **None is fixed. None is declared a confirmed defect until
its acceptance row is executed** — but the P0/HIGH ones require an Owner decision before
Customer testing starts (§3: P0/security/duplicate-order → stop progression).

| ID | Sev | Finding | Evidence | Status | Rows |
|---|---|---|---|---|---|
| **CAF-01** | **P0** | With `auth.signup_verify`=false, `POST /auth/signup/confirm` skips the code, **overwrites the password of an existing account** and issues a session — account takeover by phone number. `signup/confirm` is also not launch-gated and not rate-limited; the app always shows the signup door | `identity/service.go:486-559` | **source-confirmed** · latent today: Prod and Staging store `signup_verify`=true (catalog default is false; Owner decision 2026-08-25 intended false) | CUST-19-026, CUST-04-023, CUST-04-002 |
| **CAF-02** | **HIGH** | A 409 `in_progress` is treated as a final answer: the persisted Idempotency-Key is cleared, so a later tap can create a **second order** while the first is still committing; `in_progress` is unmapped (shows a misleading connection message); the key does not fingerprint the body | `ui/Attempt.kt` `isDecided`; `server/idempotency.go:221-226`; client timeout 20 s vs server 30 s, claim 60 s | **source-confirmed** (runtime path to prove) | CUST-13-028/029, CUST-20-019 |
| **CAF-03** | **HIGH** | `POST /orders` honours a client-supplied `merchant_id` (open-hours check and attribution) | `orders/service.go:303-305,378-396`; handler does not clear it | **source-confirmed** (app never sends it) | CUST-19-027 |
| CAF-04 | HIGH | Suspended customer cannot see/cancel a live order in the app: suspension exceptions name `GET /orders/{id}`, which the app never uses; `/my/orders*` and chat blocked; `/auth/me` 403 at start shows «offline» | `server/suspension.go:65-66` (audit B) | reported · to verify | CUST-06-031/032 |
| CAF-05 | MED | Browse gate is partial: sections, section items, search, suggest stay open before launch | `server.go:438-442,492-493` (audit B) | reported · to verify | CUST-18-021 |
| CAF-06 | MED | Unlaunched province/city is explained by availability but not enforced when creating orders (zones only) | `availability.go:264-313`, `service.go:474-490`, `custom.go:152-158` (audit B) | reported · to verify | CUST-19-028 |
| CAF-07 | MED | Custom order: driver note sent but dropped; payment always cash | `custom_order_handlers.go:36-43`, `custom.go:162-164` (audit B) | reported · to verify | CUST-CUSTOM-019/020 |
| CAF-08 | MED | Displayed cart total = device subtotal + server fee − discount; server total never shown; first quote sends no `expected` so price drift since add is not flagged | `cart/CartScreen.kt:326` (verified), `:741-749` (audit B) | line 326 source-confirmed | CUST-12-005/007/008 |
| CAF-09 | HIGH | Logout clears only the session: the device-global cart, activity-scoped state (balance, inbox, orders, favorites) and the WebSocket survive into the next account | audit A; PC-2 | reported · to verify | CUST-06-015, CUST-11-017/018, CUST-15-019, CUST-19-017 |
| CAF-10 | MED | No global 401→logout mid-session; `password_change_required`/`forbidden` never clear the session | audit A/B | reported · to verify | CUST-06-011/032 |
| CAF-11 | MED | Banners with a target are clickable but have no handler — silent tap | `ShopScreen` (audit A) | reported · to verify | CUST-09-026 |
| CAF-12 | MED | Adding from Offers skips the address/coverage gate (server still validates at submit) | `MineScreens.kt:181-361` (audit A) | reported · to verify | CUST-11-035, CUST-ENG-005 |
| CAF-13 | MED | Order-status pushes do not open the order (only order_chat and offer destinations are routed) | `MainActivity:1070-1097` (audit A); XG-8/XG-9 | reported · to verify | CUST-15-006 |
| CAF-14 | MED | Order list/history loads page 1 (30) only | `OrdersViewModel.kt:99,145` | **source-confirmed** | CUST-14-026 |
| CAF-15 | LOW | `shop.rail_auto` / `rail_every_ms` parsed, never used | `model/Shop.kt:27-28` (audit A) | reported | CUST-09-027 |
| CAF-16 | LOW | Launch flags signup/orders/custom_orders parsed, unused by the client (server is the only guard) | `Serving.kt:92` (audit A/B) | reported | CUST-04-002, CUST-CUSTOM-006 |
| CAF-17 | LOW | Promo code and payment choice are lost on process death | audit A | reported | CUST-12-026 |
| CAF-18 | LOW | Unmapped customer-path codes (`not_found`, `in_progress`, `comms_closed`, `comms_no_driver`) show «تعذر الاتصال — حاول بعد قليل» | `ui/ApiErrors.kt` (audit B) | reported | CUST-20-019, CUST-SUP-003 |
| CAF-19 | LOW | `/my/warnings` exists (Admin can warn customers) but the app never shows warnings | audit B | reported · Owner decision | CUST-SUP-012 |
| CAF-20 | INFO | `app_opens_daily` is inflated because every realtime frame re-fetches `/public/home` | `app_opens.go` (audit B) | reported · analytics only | — |
| CAF-21 | LOW | Guest: cart address card not guest-gated while geo endpoints require auth | audit A | reported | CUST-07-028 |
| CAF-22 | TEST DEP | Password reset and WhatsApp verification go through `/auth/wa/ticket` (WhatsApp bot), not the dev OTP log | `AuthViewModel:311-407` | source-confirmed | CUST-05-017, CUST-06-016 |

### 38.5 · Known defects and gaps carried into this document

| ID | Source | Status | Rows |
|---|---|---|---|
| P8-DEF-001 / L1-019 | P-8 matrix | root cause FIXED (`cc6fe128`); detection/recovery PASS; **§7 blocking contract not built** | CUST-16 (all), CUST-11-028..032, CUST-12-021, CUST-13-023 |
| D11 | TEST_TRUTH, P-8 C2-019, AND-50 | CONTRACT_MISMATCH — no `must_change_password` flow | CUST-06-027 |
| D6 · D8 · D9 | TEST_TRUTH (EXPECTED_FAIL) | custom order skips cash-ban, WhatsApp check, creation event | CUST-CUSTOM-008/009/010, CUST-13-026/027 |
| D3 | TEST_TRUTH, P-8 C2-016 | "remember me" becomes permanent after first refresh — no guard | CUST-06-009/010 (note) |
| XG-8 · XG-9 | TEST_TRUTH | notifications open the first screen / 24 notifications unrouted | CUST-15-006 |
| XG-19 · PRQ-2 | TEST_TRUTH, CUSTOMER_FINAL_PRODUCT_DECISIONS | no support entry; tickets need an order; no ticket reply | CUST-SUP-010 |
| XG-38 · PC-5 | TEST_TRUTH | two fixed notification IDs (3001/3002) | CUST-15-008 |
| PC-2 | CUSTOMER_FINAL_PRODUCT_DECISIONS | cart has no owner account | CUST-11-017/018 |
| PC-3 | same | `refunded` shows raw English; `rejected` shares the cancelled text | CUST-14-015 |
| PC-12 | same | no customer push on `assigned` | CUST-14-012 |
| R9 | TEST_TRUTH | no global rate limit (affects `POST /orders`) — no test | CUST-16-042, CUST-19-005 |
| R18 | TEST_TRUTH | upload does not refresh the token (avatar) — no test | CUST-06-025 |
| D27 | doc inconsistency | TEST_TRUTH says fixed; P-8 N1-021 says open — to reconcile | CUST-15 (note) |

### 38.6 · Existing automated coverage mapped to this master

| Master area | Kotlin (JVM unless noted) | Go |
|---|---|---|
| CUST-00 identity | `ProductionEndpointGuardTest` (6), `CleartextPolicyTest` (3), `StagingFirebaseGuardTest` (4) | `TestDevOTPProviderIsRefusedInProduction` |
| CUST-04/05/06 auth | `DecodeContractTest` SC-OTP, `ApiErrorsTest` (8) | `TestPID*`, `TestNormalizePhone`, `TestCheckOTP*`, `TestOTPIsCappedPerIP`, `TestAUTH_*`, `TestR13_*`, `TestXG40_*`, `TestSEC1..8`, `TestR16_*`, `TestXG39_*`, `TestXG22_*`, `TestTMP1..4`, `TestLM4`, `TestPL02` |
| CUST-07/08 address & availability | `DiscoveryPolicyTest` (8), `PreCartPolicyTest` (9), `ServingPolicyTest` (6), `ServiceCtaPolicyTest` (2), `LocatingTest` (10), `MyLocationWiringTest` (13) | `TestAV*`, `TestCFC*`, `TestSRV*`, `TestZONE_*`, `TestZH*`, `TestPH*`, `TestPC*`, `TestLM*`, `TestPL*`, `TestCR*`, `TestCITY_*`, `TestSECIDOR_Addresses` |
| CUST-09/10 market | `MarketplaceLifecycleTest` (6), `MarketplaceRaceTest` (13), `PreLaunchScreenTest` (6) | `TestML1..14`, `TestBrowse_HidesSource`, `TestRedact*`, `TestSuggestAndHasOptions`, `TestSD1..4` |
| CUST-11/12 cart & checkout | `CartTest` (17), `CartChangesTest` (10), `AbuseMatrixTest` AB-03/AB-35 | `TestFIN_*`, `TestVAL_*`, `TestCC*`, `TestQI*`, `TestPR*`, `TestPromo_*`, `TestDiscount_ShownPriceMatchesCharged` |
| CUST-13 submit | `AttemptTest` (5), `PerfFlowTest` (9), `AbuseMatrixTest` AB-01, `ButtonsUiTest` (instrumented) | `TestIDEM_*`, `TestD4_*`, `TestPH29_*`, `TestXG46_*`, `TestCreate_RequiresWhatsAppVerified` |
| CUST-CUSTOM | — | `TestCUST_*`, `TestCustomCancel_*`, `TestD22_*`, `TestCENSUS_D6*` (EXPECTED_FAIL) |
| CUST-14 orders | — | `TestCANC_*`, `TestLIFE_*`, `TestOrder_IntruderCannot*`, `TestSECIDOR_Orders`; **no test for the customer rating happy path/authz** |
| CUST-15 push & realtime | `DeepLinkTest` (15), `EngagementTest` (7), `ChatMultiOrderTest` (21), `D19ReconnectTest` (8), `D19StressTest` (4), `PerfPolicyTest` (11) | `TestD12_*`, `TestPush_*`, `TestPF09_*`, `TestEV_*`, `TestD14_*`, `TestD19_*` |
| CUST-16 offline | `NetTrackerTest` (11), `StatesUiTest` (instrumented) | — (**nothing tests blocking while offline**) |
| CUST-SUP / WAL / ENG | `ChatMultiOrderTest`, `MoneyTest`, `OffersPolicyTest` | `TestCHAT*`, `TestComplaint_*`, `TestWALL_*`, `TestStatement_*`, `TestRewardOn*`, `TestGrantSignupBonus_Credits` |
| CUST-19 security | `AbuseMatrixTest` | `TestSECIDOR_*`, `TestAUTH_*`, `TestVAL_*`, `TestIDEM_004_KeyIsPerUser`, privacy suites (`TestPushPayloadScope`, D20/D21/D23) |

**Not covered by any automated test:** offline blocking (§7), login/OTP/reset screens,
address CRUD happy path and the `customers.max_addresses` cap, customer rating
happy path/authz, promo/wallet screens, HTTP 429 beyond OTP, CAF-01, CAF-02 (409 path),
CAF-03. `TEST_TRUTH` maps Go tests only (961 of 1522 unmapped); Kotlin is not in it.

### 38.7 · Areas that require physical-device testing

CUST-00 (identity), CUST-01 (install/update on the Owner's phone; destructive variants on
an emulator), CUST-02, CUST-03 (real permission dialogs), CUST-07 map/«موقعي»,
CUST-11/12/13 UI paths, CUST-15 (real FCM delivery, background/killed, notification
tap), **CUST-16 (real network loss — mandatory for L1-019)**, CUST-17 (process death,
lock, reboot), CUST-20 (RTL, font scale, keyboard), CUST-21 (performance on SM-A525F).

### 38.8 · Owner decisions required before execution

1. **Stop-class blockers (§40.2)** — CUST-DEF-001…005, D6, D8 — must be resolved
   through §3 before Customer acceptance progresses. Keep `auth.signup_verify`=true
   everywhere until CUST-DEF-001 is fixed.
2. **Closed by the Owner (§40.1):** cross-account cart/state; Admin/internal warnings.
   **Answered by existing contracts (§40.11):** custom-order payment (cash or wallet,
   2026-08-09); ticket reply behaviour (PRQ-2 approved).
3. **Still open:** PRQ-2 release scope; CAF-15 (is `shop.rail_auto` meant for the
   app?); required order-card fields (CUST-14-021).
4. **Staging policy flips for conditional rows** (each deviates from Production policy
   and must be approved and restored): `auth.otp_login`=true (CUST-05-015),
   `auth.signup_verify`=false (CUST-04-019, CUST-19-026b), `launch.*` flips.
5. **Test harnesses:** API-unreachable/DNS/timeout/5xx fault injection (CUST-02-010,
   CUST-16-027..044, CUST-04-015, CUST-12-018/020, CUST-13-011/012/028); emulator for
   destructive install cases; the Staging WhatsApp bot for reset/verification
   (CAF-22); dense catalog fixture (CUST-09-018, CUST-21-005).

### 38.9 · Open housekeeping from 2026-09-19 (not part of acceptance)

**Corrected (§40.13):** the Owner manually removed the four RahalGo apps from Dual
Messenger (user 95) — DONE. **ADB verification is pending** until the phone is
reachable; CUST-00-009 stays `NOT_TESTED` until then. Test residue (`/sdcard/def001`,
`/data/local/tmp/def001.sh`, `/sdcard/rg_pre.xml`, `/sdcard/rg_ui.xml`) may remain
until ADB is available — not an acceptance blocker.

---

## 39 · Ledger — the count that must reach zero open rows

**Total acceptance cases: 576** — from the Owner contract: 474 (440 explicit cases in §11–§33 + 16 final-gate criteria in §34 + 18 CUST-CUSTOM cases written because §25 of the contract requires a dedicated group) · added by the coverage audit: 102.

**Status now:** `NOT_TESTED` 570 · `NOT_APPLICABLE` 6 · `PASS` 0 · `FAIL` 0 · `BLOCKED` 0. **Open mandatory rows: 570.** Nothing has been executed under this document.

| § | Group | Rows | Supplied | Added | NOT_TESTED | NOT_APPLICABLE | PASS | FAIL | BLOCKED |
|---|---|---|---|---|---|---|---|---|---|
| 11 | CUST-00 | 15 | 15 | 0 | 15 | 0 | 0 | 0 | 0 |
| 12 | CUST-01 | 11 | 11 | 0 | 11 | 0 | 0 | 0 | 0 |
| 13 | CUST-02 | 11 | 10 | 1 | 11 | 0 | 0 | 0 | 0 |
| 14 | CUST-03 | 14 | 13 | 1 | 14 | 0 | 0 | 0 | 0 |
| 15 | CUST-04 | 23 | 18 | 5 | 23 | 0 | 0 | 0 | 0 |
| 16 | CUST-05 | 17 | 14 | 3 | 17 | 0 | 0 | 0 | 0 |
| 17 | CUST-06 | 32 | 22 | 10 | 32 | 0 | 0 | 0 | 0 |
| 18 | CUST-07 | 30 | 24 | 6 | 30 | 0 | 0 | 0 | 0 |
| 19 | CUST-08 | 18 | 16 | 2 | 18 | 0 | 0 | 0 | 0 |
| 20 | CUST-09 | 29 | 25 | 4 | 29 | 0 | 0 | 0 | 0 |
| 21 | CUST-10 | 14 | 13 | 1 | 14 | 0 | 0 | 0 | 0 |
| 22 | CUST-11 | 37 | 32 | 5 | 37 | 0 | 0 | 0 | 0 |
| 23 | CUST-12 | 28 | 23 | 5 | 28 | 0 | 0 | 0 | 0 |
| 24 | CUST-13 | 29 | 24 | 5 | 29 | 0 | 0 | 0 | 0 |
| 25 | CUST-CUSTOM | 20 | 18 | 2 | 20 | 0 | 0 | 0 | 0 |
| 26 | CUST-14 | 26 | 20 | 6 | 23 | 3 | 0 | 0 | 0 |
| 26A | CUST-SUP | 12 | 0 | 12 | 12 | 0 | 0 | 0 | 0 |
| 26B | CUST-WAL | 10 | 0 | 10 | 10 | 0 | 0 | 0 | 0 |
| 26C | CUST-ENG | 14 | 0 | 14 | 14 | 0 | 0 | 0 | 0 |
| 27 | CUST-15 | 19 | 16 | 3 | 19 | 0 | 0 | 0 | 0 |
| 28 | CUST-16 | 45 | 45 | 0 | 45 | 0 | 0 | 0 | 0 |
| 29 | CUST-17 | 22 | 21 | 1 | 21 | 1 | 0 | 0 | 0 |
| 30 | CUST-18 | 21 | 20 | 1 | 21 | 0 | 0 | 0 | 0 |
| 31 | CUST-19 | 29 | 25 | 4 | 29 | 0 | 0 | 0 | 0 |
| 32 | CUST-20 | 19 | 18 | 1 | 18 | 1 | 0 | 0 | 0 |
| 33 | CUST-21 | 15 | 15 | 0 | 14 | 1 | 0 | 0 | 0 |
| 34 | CUST-22 | 16 | 16 | 0 | 16 | 0 | 0 | 0 | 0 |
| | **Total** | **576** | **474** | **102** | **570** | **6** | **0** | **0** | **0** |

Rows marked *conditional* in Notes need an Owner-approved Staging policy flip (§38.8); until approved they stay `NOT_TESTED`. Rows noting *expected FAIL* point at a source-confirmed or known gap — they are still executed and recorded honestly.

### 39.1 · Cases added beyond the supplied master (102)

| ID | Case | Why (evidence) |
|---|---|---|
| CUST-02-011 | Update-required gate | Added: `ui/UpdateGate.kt`; `/public/*` is exempt from the 426 check |
| CUST-03-014 | Camera/gallery for profile photo | Added by audit: avatar uses system pickers (`ui/ImagePick.kt`) |
| CUST-04-019 | Signup with signup_verify=false | Added: client skips the code step when false (`AuthViewModel:419-519`). Production policy is true — test only if Owner approves a temporary flip |
| CUST-04-020 | Referral code at signup | Added: ref field + invite link `https://rahalgo.com/signup?ref=` + Play install referrer |
| CUST-04-021 | Invalid referral code | Audit: real surface not covered by the supplied list |
| CUST-04-022 | Invite deep link opens signup with code | Only `/signup` is in the intent filter; `/i/CODE` parsed but not routed |
| CUST-04-023 | Signup confirm is launch-gated and rate-limited | Added. CAF-01: only `/signup/request` is gated; `/signup/confirm` has no launch gate and no rate limit — expected FAIL |
| CUST-05-015 | OTP login when otp_login=true | Added: tab shown only when `/public/platform` says otp_login=true. Production policy is false |
| CUST-05-016 | OTP login tab hidden when otp_login=false | Audit: real surface not covered by the supplied list |
| CUST-05-017 | WhatsApp account verification from Account screen | Added: `/auth/wa/ticket` purpose=verify + `/auth/whatsapp/confirm`. Needs the Staging WhatsApp bot — BLOCKED if the bot is not paired |
| CUST-06-023 | Change password from Account | Added: Account screen feature |
| CUST-06-024 | Change phone number | Added: `/auth/phone/request` + `/auth/phone/confirm` |
| CUST-06-025 | Edit name and profile photo | Added: `PATCH /me/name`, `POST /me/avatar` |
| CUST-06-026 | Account deletion | Added: `/auth/account/delete/request\|confirm`. Destructive — disposable account only |
| CUST-06-027 | Forced password change (password_change_required) | Added: known CONTRACT_MISMATCH — no dedicated client flow; only an error text (`ApiErrors.kt:359`). Expected FAIL until built |
| CUST-06-028 | Startup session restore fails on network | Added: `AuthGate` offline branch (`ui/AppFrame.kt:310`) |
| CUST-06-029 | Startup restore with rejected session wipes it | Added: `sessionRejected` on 401/invalid_refresh |
| CUST-06-030 | Guest browsing and NeedAccount gates | Added: signed-out users browse by default (guest shell) |
| CUST-06-031 | Suspended customer and live orders | Added. CAF-04 (reported by audit, to verify): exceptions list `GET /orders/{id}` which the app never calls; `/my/orders*` and chat are blocked; `/auth/me` 403 at start shows 'offline' |
| CUST-06-032 | Session-level 403 codes are not shown as network failures | Added. CAF-10: 403 on `/auth/me` at startup renders the offline state |
| CUST-07-025 | Address limit reached | Added: no client limit; server error only |
| CUST-07-026 | Map search and reverse geocode | Added: guests cannot call geo (auth-only) — see 07-028 |
| CUST-07-027 | Address form validation | Added |
| CUST-07-028 | Guest cannot reach geo endpoints / address book | Added: audit found the cart address card is not guest-gated while geo calls are auth-only |
| CUST-07-029 | Browse city picker | Added: `CityPicker.kt`, `CityScope.kt` |
| CUST-07-030 | Request my area / notify me (demand) | Added: `POST /api/v1/demand`; not offered for discovery points |
| CUST-08-017 | Pre-launch screen when browsing is closed | Added: `MainActivity:1221-1252` |
| CUST-08-018 | Owner notice overrides built-in reason text | Added: `ApiErrors.kt:118-134` |
| CUST-09-026 | Banner slider and banner tap | Added. CAF-11 CONFIRMED (§40.10): banners with a target are clickable but ShopScreen passes no handler — silent tap, expected FAIL |
| CUST-09-027 | Section rail auto-scroll setting | Added. CAF-15 NEEDS_OWNER_DECISION (§40.10): `rail_auto/rail_every_ms` parsed but never used; the setting sits in the Site group |
| CUST-09-028 | Browse scoped to city/address | Added. PC-1 gap noted: `CityScope.kt:90` returns the chosen city first |
| CUST-09-029 | Guest browsing of the market | Added |
| CUST-10-014 | Unavailable option disabled | Added |
| CUST-11-033 | Suggestions row «يُطلب معه» | Added: `SuggestRow.kt` |
| CUST-11-034 | Empty cart via «إفراغ السلة» | Added |
| CUST-11-035 | Add from Offers respects address/coverage gate | Added. CAF-12: offers add path skips the PreCart gate (server still validates at submit) |
| CUST-11-036 | Multi-source limit | Added: no client check; server enforces |
| CUST-11-037 | Corrupt persisted cart is discarded safely | Added: unreadable cart is deleted by design |
| CUST-12-024 | Promo code preview | Added |
| CUST-12-025 | Promo becomes invalid before submit | Added: `TestPR03_StaleDiscountCannotSubmit` |
| CUST-12-026 | Promo & payment choice across process death | Added. CAF-17: promo and payment choice are memory-only |
| CUST-12-027 | Below minimum order | Added: P8-C3-029..031; minimum shown only on rejection (TRUTH §6) |
| CUST-12-028 | Cart-changes review gate | Added: `CartChangesTest` (10) |
| CUST-13-025 | Open-order cap | Added: D4 fixed; `TestD4_*` |
| CUST-13-026 | WhatsApp verification requirement on normal orders | Added: normal path checks WhatsApp; custom path does not (D8) |
| CUST-13-027 | Cash-blocked customer | Added: normal path checks cash ban; custom path does not (D6) |
| CUST-13-028 | 409 `in_progress` never leads to a duplicate order | Added. CAF-02 (source-confirmed): `Attempt.isDecided` treats any ApiException (incl. 409 `in_progress`) as final and clears the key; `in_progress` is unmapped. Expected FAIL |
| CUST-13-029 | Retry after cart edit does not replay the old order | Added. CAF-02: idempotency does not fingerprint the body |
| CUST-CUSTOM-019 | Driver note is saved and shown | Added. CAF-07 (reported by audit): custom `notes` are sent but not decoded/stored — expected FAIL |
| CUST-CUSTOM-020 | Custom order payment method | Added. Answered by contract (§40.11): the app sends no method → wallet option missing — expected FAIL |
| CUST-14-021 | Order card shows authoritative data | Added (replaces the absent detail screen). Card omits payment method, address, times, wallet_paid, cash_due — Owner to decide if required |
| CUST-14-022 | Cancel order within window | Added: `TestCANC_001`, `TestCancelBeforeDelivery_RefundsWalletOnly` |
| CUST-14-023 | Double cancel / cancel after window | Added: `TestCANC_002` |
| CUST-14-024 | Rate a delivered order | Added. No backend test for the customer rating happy path/authz (audit) |
| CUST-14-025 | Automatic rating prompt | Added |
| CUST-14-026 | History beyond 30 orders | Added. CAF-14: the app requests page 1 only (`OrdersViewModel.kt:99,145`) — expected FAIL |
| CUST-SUP-001 | Chat button appears for an open order with a driver | `ChatMultiOrderTest` (21) |
| CUST-SUP-002 | Send and receive messages | Audit: real surface not covered by the supplied list |
| CUST-SUP-003 | Chat read-only after order ends | `comms_closed`/`comms_no_driver` are unmapped codes (CAF-18) |
| CUST-SUP-004 | Two orders' chats never mix | Audit: real surface not covered by the supplied list |
| CUST-SUP-005 | Stranger cannot read/post in another's chat | `TestCHAT05_StrangerGetsNotFound` |
| CUST-SUP-006 | Chats list (دردشاتي السابقة) | Audit: real surface not covered by the supplied list |
| CUST-SUP-007 | Offline chat send blocked | §7 |
| CUST-SUP-008 | Complaint on an order | `TestComplaint_*` |
| CUST-SUP-009 | Complaint once / window / not on running order | Audit: real surface not covered by the supplied list |
| CUST-SUP-010 | Tickets list shows status and resolution | Ticket reply: PRQ-2 approved (customer should reply) but not built — release scope is the open Owner question (§40.11) |
| CUST-SUP-011 | Foreign order complaint denied | `TestVAL_040_ForeignOrderComplaintCode` |
| CUST-SUP-012 | Admin warnings visible to the customer | CAF-19 CONFIRMED (§40.10): no notification is sent and the app never shows warnings — expected FAIL |
| CUST-WAL-001 | Balance chip and wallet screen | Audit: real surface not covered by the supplied list |
| CUST-WAL-002 | Transaction list correctness | Audit: real surface not covered by the supplied list |
| CUST-WAL-003 | Statement: this month / previous / all | Audit: real surface not covered by the supplied list |
| CUST-WAL-004 | Statement print/PDF | Audit: real surface not covered by the supplied list |
| CUST-WAL-005 | Wallet shows only own balance | `TestSECIDOR_WalletIsOwn`, `TestWallet_ShowsOnlyOwnBalance` |
| CUST-WAL-006 | Pay order from wallet (sufficient) | Staging wallet credit via supported Admin/incentive path only |
| CUST-WAL-007 | Pay from wallet (insufficient) | `TestWALL_010_CannotPayBeyondBalance` |
| CUST-WAL-008 | Cancel wallet-paid order refunds wallet | `TestCancelBeforeDelivery_RefundsWalletOnly` |
| CUST-WAL-009 | No payouts/top-up offered to customers | POST `/me/payouts` returns 403 `payout_not_allowed` for customers |
| CUST-WAL-010 | Wallet realtime refresh | Audit: real surface not covered by the supplied list |
| CUST-ENG-001 | Toggle favorite from a card | Audit: real surface not covered by the supplied list |
| CUST-ENG-002 | Favorites screen | No add-to-cart from Favorites (by design?) — Owner note |
| CUST-ENG-003 | Guest heart → login | Audit: real surface not covered by the supplied list |
| CUST-ENG-004 | Offers list | Audit: real surface not covered by the supplied list |
| CUST-ENG-005 | Add offer item to cart | CAF-12 |
| CUST-ENG-006 | Offer opened from push; expired offer | `DeepLinkTest` |
| CUST-ENG-007 | Invite screen | Audit: real surface not covered by the supplied list |
| CUST-ENG-008 | Referral reward paid per policy | `TestRewardOn*`, `TestBonusAndReferral_OncePerPhone` |
| CUST-ENG-009 | Signup bonus | `TestGrantSignupBonus_Credits` |
| CUST-ENG-010 | Static pages | Audit: real surface not covered by the supplied list |
| CUST-ENG-011 | Contact page links | Audit: real surface not covered by the supplied list |
| CUST-ENG-012 | Theme toggle | Audit: real surface not covered by the supplied list |
| CUST-ENG-013 | Brand intro once per process; reduce-motion respected | Audit: real surface not covered by the supplied list |
| CUST-ENG-014 | Drawer items per auth state | Audit: real surface not covered by the supplied list |
| CUST-15-017 | Notification inbox | Added |
| CUST-15-018 | Chat message push opens the chat | Added: `DeepLinkTest`, `ChatMultiOrderTest` |
| CUST-15-019 | Logout stops realtime | Added. CUST-DEF-004 (STOP §40.6): `LiveSocket.stop` is not called by the customer app on logout — expected FAIL |
| CUST-17-022 | Orders tab across background and process death | Added: replaces the order-detail lifecycle rows (no detail screen) |
| CUST-18-021 | Browse closure closes every browse surface | Added. CAF-05 (reported by audit): only `/public/home`, `/public/offers`, `/public/items/{id}` are gated |
| CUST-19-026 | Signup confirm cannot take over an existing account | Added. **CAF-01 P0 (source-confirmed)**: with `signup_verify`=false `ConfirmSignup` skips the code and overwrites an existing account's password, then issues a session (`identity/service.go:486-559`). Current Prod/Staging value = true (Staging since 2026-09-19). Expected FAIL for (b) |
| CUST-19-027 | Client-supplied merchant_id is ignored | Added. **CAF-03 HIGH (source-confirmed)**: `CreateTx` keeps a non-empty client `merchant_id` (`orders/service.go:303-305`) and runs the open-hours check on it |
| CUST-19-028 | Order in an unlaunched city/province is denied at create | Added. CAF-06 (reported by audit): place classification is advisory; create paths enforce zones only |
| CUST-19-029 | Any-role token cannot misuse customer order routes | Added: the customer route group has no role check |
| CUST-20-019 | Every customer-path error code has a meaningful message | Added. CAF-18: these codes are unmapped and fall back to a misleading 'connection' message |

### 39.2 · Supplied cases marked NOT_APPLICABLE (6)

| ID | Case | Why (evidence) |
|---|---|---|
| CUST-14-003 | Open correct order detail | No order-detail screen exists: the order card is the only view; `CustomerApi.order(id)` is unused (audit §38). Card correctness is covered by CUST-14-021. |
| CUST-14-018 | Order detail survives background | No order-detail screen (see 14-003). Orders tab lifecycle is covered by CUST-17-022. |
| CUST-14-019 | Order detail after process restart | No order-detail screen (see 14-003). Covered for the Orders tab by CUST-17-022. |
| CUST-17-005 | Order detail → background → foreground | No order-detail screen exists (order cards only; `CustomerApi.order(id)` unused). Orders-tab lifecycle is covered by CUST-17-022. |
| CUST-20-018 | English/localization only if supported | Arabic only: no `values-xx` locale folders and no language switch in the Customer app (audit §38). |
| CUST-21-009 | Order-detail load | No order-detail screen exists. Order-list load is CUST-21-008. |

---

*Maintained with the repository. Status changes are made in this file, with evidence, row by row. Never renumber an ID.*

---

## 40 · Pre-acceptance blocker gate (2026-09-19) — read-only triage

> **Owner instruction:** VERIFY → CLASSIFY → ROOT CAUSE → P-9 IMPACT → REGRESSION DESIGN →
> FIX PLAN. **Read-only.** No fix implemented, no Staging or Production mutation, no
> acceptance case executed, R1 not started.
>
> **Every claim below was re-read in the source by the triage itself** — sub-agent
> reports were used only as leads.

### 40.1 · Owner decisions recorded (closed 2026-09-19)

1. **Cross-account cart / state.** Account B must not see or inherit Account A's
   private cart or private authenticated state. Logout must detach the previous
   account's private state and its authenticated realtime/session context. If an
   anonymous/guest cart exists by design, it must be explicitly scoped and must never
   leak private state between authenticated accounts.
2. **Admin/internal warnings.** Customers must not receive raw internal/Admin-only
   warning text, stack detail, debug detail, secrets or internal operational
   information. Customer-visible errors must be safe, understandable Customer-facing
   messages.

### 40.2 · Stop-class blockers — Customer acceptance does not progress until these are resolved

| Defect ID | Was | Sev | Class (§3 / Owner rule B) | Status |
|---|---|---|---|---|
| **CUST-DEF-001** | CAF-01 (+ CAF-16 signup part) | **P0** | account takeover · authentication boundary | **CONFIRMED (source)** — latent while `auth.signup_verify`=true |
| **CUST-DEF-002** | CAF-02 (+ CAF-18 `in_progress` part) | **P1** | duplicate-order risk | **CONFIRMED (source)** — timing-dependent |
| **CUST-DEF-003** | CAF-03 | **P1** | unsafe client/server trust boundary · financial source of truth | **CONFIRMED (source)** — exploitable by any signed-in customer with a crafted request |
| **CUST-DEF-004** | CAF-09 + PC-2 (one root cause) | **P1** | cross-account data leakage | **CONFIRMED (source)** |
| **CUST-DEF-005** | CAF-08 | **P1** | financial source of truth (customer shown one total, charged another) | **CONFIRMED (source)** |
| **D6** | known (EXPECTED_FAIL) | **P1** | financial risk-control bypass | **CONFIRMED (source)** — live on Staging; Production flag OFF is not a boundary |
| **D8** | known (EXPECTED_FAIL) | **P1 · latent** | verification boundary bypass | **CONFIRMED (source)** — dormant while `auth.require_whatsapp`=false |

**New IDs use the `CUST-DEF-nnn` namespace of this document.** Existing IDs (D6, D8,
D9, D11, R14, XG-9, PC-2…) are kept, never re-numbered; where a finding and a known ID
share one root cause, the known ID is referenced instead of opening a new one.

### 40.3 · CUST-DEF-001 — signup confirm can take over an existing account

**Path.** `POST /api/v1/auth/signup/confirm` → `handleSignupConfirm`
(`server/auth_handlers.go:354-418`) → `identity.ConfirmSignup`
(`identity/service.go:486-559`).

**Root cause.**

1. The code check is **conditional**: `if s.boolSetting(ctx, "auth.signup_verify", false)`
   (`service.go:520`). Catalog default is **false**; the setting is read straight from
   the database (`settings.Store.Get`, no cache) and **any read error returns the
   fallback false → the check is skipped (fail-open)**.
2. For a phone that already has an account, the `else` branch calls
   `repo.SetPassword` — `UPDATE users SET password_hash=$2, must_change_password=false`
   (`identity/repo.go:509-513`) — and overwrites the full name. **There is no check
   that the existing account is password-less**; that guard exists only in
   `RequestSignup` (`service.go:436-439`), which the app skips when verification is off
   and which an API caller need not call at all.
3. It then calls `issueFor` → `issueSessionFor`, which revokes the victim's sessions of
   the same client kind and returns a new session.
4. `handleSignupConfirm` has **no `requireLaunch(launch.customer_signup)` and no rate
   limit** — only `/signup/request` is gated (`auth_handlers.go:297`).

**What an attacker gets with verification OFF (phone number only):**

| Target account | Result |
|---|---|
| customer · driver · merchant · rep (any non-admin role), active | **password + name overwritten, full session issued, victim's same-client session revoked** |
| admin **without** a PIN yet | password overwritten; response carries a PIN **setup** challenge → `POST /auth/pin/setup` (`SetPinFirstTime`) sets the attacker's PIN → **admin session** |
| admin **with** a PIN | password overwritten (**lock-out**); blocked at the PIN step |
| suspended / blocked | password overwritten **before** the status check refuses the session (state mutation) |
| any account flagged `must_change_password` | flag cleared → forced-change bypass (links D11) |

**With verification ON (today, Production and Staging).** `ConsumeOTP(phone, code,
"signup")` must succeed first; signup codes are only sent by `RequestSignup`, which
refuses accounts that already have a password. **A password-protected account cannot be
reached** — the remaining path (a password-less account created by OTP login) needs a
code delivered to that phone, i.e. its owner. **Not exploitable at runtime today**,
but one Admin toggle, a missing setting row, or a transient settings-read error
re-opens it.

**Shared logic?** No other auth path has the pattern. `ConfirmPasswordReset`,
`VerifyOTP` (OTP login), `ConfirmPhoneChange` (also refuses a phone owned by another
user) and WhatsApp confirm **always** consume a code sent to the phone before touching
an account; `/auth/sso` exchanges a one-time code minted by an authenticated
`/auth/handoff`. **The flaw is unique to `ConfirmSignup`** — but it reaches **every
role**, not only Customer signup.

**Required safe contract (Owner).** An existing account must never have its password
replaced or receive a new session through the signup path merely because someone knows
its phone number. Verification OFF must never turn signup into password reset or
takeover. Existing-account signup is rejected or redirected to login/recovery.
Backend-enforced.

**Regression (must fail before the fix, pass after):** Go, `internal/identity` +
`internal/qa`:

- verify=false · existing account with password → rejected; hash, name, sessions
  unchanged; no session issued;
- verify=true · no/invalid code → rejected;
- admin without PIN → no challenge reachable through signup;
- suspended/blocked account → nothing written;
- `must_change_password` not cleared by signup;
- settings read failure → fail-closed;
- `/signup/confirm` launch-gated and rate-limited;
- the legitimate password-less (OTP-login) completion still works only with a valid
  signup code.

**There is no test today for signup confirm or `auth.signup_verify` at all.**

**P-9:** RISK CLASS **CRITICAL** · apps admin/customer/driver/merchant/rep · flows
F-28 F-29 F-30 F-34 · defects D10 D11 D12 D14 · risks R13 R15 R16 · gaps XG-22 XG-23
XG-24 XG-39 XG-9 · 158 mandatory tests (`./internal/identity ./internal/qa`) · Staging
required (R16).

### 40.4 · CUST-DEF-002 — a 409 `in_progress` can lead to a duplicate order

**Client.** `ui/Attempt.kt` `isDecided(e) = e is ApiException` → every server answer,
**including 409 `in_progress`**, is "decided". `CartViewModel.send`
(`cart/CartScreen.kt:~790-836`) and `CustomViewModel.send` then `Attempt.clear(slot)`;
the next tap mints a fresh UUID. `in_progress` has no entry in `ui/ApiErrors.kt`, so
the user reads «تعذر الاتصال — حاول بعد قليل».

**Server.** `server/idempotency.go`: claim keyed `(user, endpoint, key)` with a 60 s
lease (`idempotencyLease`); a second request on a live, uncommitted claim gets **409
`in_progress`** (`:221-226`); a committed claim replays the stored response; errors
≥ 400 release. Handler timeout 30 s (`server.go:302`) vs client request timeout 20 s
(`shared/net/ApiClient.kt:128-132`). The body is not fingerprinted.

**Reachable sequence.**

1. Submit #1 is slow and the client times out at 20 s; the key is kept (correct).
2. The user retries within #1's lease; the server answers 409 `in_progress`.
3. The client treats that as final and **clears the key**.
4. Submit #1 commits.
5. The next tap sends a new key, waits on the per-customer advisory lock, and passes
   the open-order cap — `orders.max_open_per_customer` is not stored in either
   environment, so the default is **3**.
6. The result is **a second order**.

**Regression:**

- Kotlin (`ui`): `isDecided(ApiException(409,"in_progress"))` is false; Cart and
  Custom view models keep the key on `in_progress`; `in_progress` maps to a
  "still processing" message.
- Integration: after a lost response and an `in_progress` answer, a retry with the
  kept key replays the committed order (orders +1 exactly).
- Server-side body fingerprinting is a separate, optional hardening decision.

**P-9:** apps customer/driver/merchant/rep (shared `ui`) · RISK MEDIUM (Go-mapped) ·
device required · plus Kotlin suites `AttemptTest`, `PerfFlowTest`, `AbuseMatrixTest`
(P-9 maps Go tests only).

### 40.5 · CUST-DEF-003 — `POST /orders` trusts a client-supplied `merchant_id`

**Root cause.** `handleCustomerCreateOrder` (`server/customer_handlers.go:422-428`)
decodes `orders.CreateInput` and clears only `CustomerID`/`CustomerPhone`; `CreateTx`
fills `MerchantID` from the items **only when empty** (`orders/service.go:303-305`),
checks only that it is a UUID (`:314-320`), runs the **open-hours check on it**
(`:378-396`) and writes it to `orders.merchant_id` (`:605`). Items keep their true
source (`order_items.merchant_id`).

**What the header `merchant_id` drives** (verified consumers):

- merchant notification (`service.go:158`);
- the merchant's order list (`queries.go:314`);
- pickup routing and ETA (`sameroute.go`, `routeeta.go`);
- ratings attribution (`ratings.go:55`);
- **sales-rep commission** (`settleRep`, `transitions.go:~1040`) and its reversal
  (`reverseCommissions`, `:~1139`);
- **merchant activation counting** (`merchantActivated`, `:1581`), which feeds rep
  rewards.

Goods settlement uses the item's merchant (`COALESCE(oi.merchant_id, o.merchant_id)`).

**Impact.** Any signed-in customer can attribute an order to an unrelated open merchant.
That misroutes the order and steers rep commission and activation — **financial
source-of-truth corruption through an unsafe trust boundary.** The normal app never
sends the field.

**Regression (Go):**

- a foreign or closed `merchant_id` is ignored or rejected;
- the stored header equals the item source;
- rep commission and activation are computed on the true merchant;
- the open-hours check uses the true merchant.

**P-9:** RISK **CRITICAL** · 4 apps + admin · order lifecycle, privacy · 212 mandatory
tests (`./internal/orders ./internal/qa`).

### 40.6 · CUST-DEF-004 — logout does not detach account-scoped state (cross-account)

One root cause: **the client has no session-boundary reset.** It absorbs PC-2 (cart
ownership) and CAF-09.

| Part | Evidence | Behaviour |
|---|---|---|
| Cart | `customer/cart/Cart.kt` — one global object in SharedPreferences `rahalgo_cart`, no owner; `Cart.clear()` only on «إفراغ السلة» and after a successful order (`CartScreen.kt:278,816`) | **A's cart is shown to guest and to B** |
| Logout | `ui/AuthViewModel.kt:558-570` — clears session + `user`, calls `/auth/logout`; nothing else | activity-scoped view models keep A's balance, inbox, orders and favorites until they reload |
| Realtime | `shared/net/LiveSocket.start` returns if a job is already active; the customer app never calls `stop()` (only the driver app does) | **A's authenticated socket stays open after logout; when B signs in, B's shell reuses A's socket** — B gets refresh signals driven by A's private events and misses its own until the socket drops |
| Server side | `server/ws.go:86-99` checks the session only at handshake | known risk **R14** ("the realtime connection is not re-checked after the handshake") — referenced, not duplicated |

**Violates Owner decision 40.1-1.**

**Regression:**

- Kotlin: logout clears or re-scopes the cart (a guest cart only if explicitly scoped);
  logout stops the socket; sign-in starts a socket with the new token; account-scoped
  view-model state resets.
- Device: A → logout → B shows none of A's cart, orders, wallet, inbox or favorites,
  and receives only B's realtime.

**P-9:** RISK **CRITICAL** · apps customer/driver/merchant/rep · F-04 F-30 F-34 ·
D10 D11 D12 D19 D20 D21 · R13 R15 R16 R21 · 182 mandatory Go tests · device required
(AND-31).

### 40.7 · CUST-DEF-005 — the cart can show a stale total and the server charges another

**Root cause.**

- `CartViewModel.quote` sends **no `expected`** when `priced == null`, i.e. on the
  first quote after opening the cart (`cart/CartScreen.kt:741-749`), so the server
  returns no `changes` and the review gate never opens.
- The screen shows `Cart.subtotal` from the unit prices stored when the item was added
  (the cart persists across restarts), and the total is computed on the device
  (`:326`).
- The order is priced by the server at current prices (`orders/service.go:690-833`).

**Impact.** A price change since the item was added is invisible at send time, so the
customer agrees to one total and is charged another. **Stale data presented as
authoritative.**

**Regression (Kotlin):**

- the first quote sends expected unit prices from the stored lines;
- a stale price opens the change-review gate;
- the displayed subtotal and total come from the server quote.

**P-9:** customer only · device required.

### 40.8 · D6 · D8 · D9 — custom order skips rules the normal order enforces

| Rule | Normal `CreateTx` | Custom `CreateCustomTx` (`orders/custom.go:135-230`) |
|---|---|---|
| Cash ban (`customers.cash_ban_*`) | `service.go:~363` `cashBlocked` | **absent** → **D6 confirmed** |
| WhatsApp verification (`customers.require_whatsapp` under `auth.require_whatsapp`) | `service.go:~390` | **absent** → **D8 confirmed** (latent while the master switch is off) |
| Creation event `''→pending` + `notifyCreated` | `service.go:639,678` | **absent** (publish only) → **D9 confirmed** |

**Classification:**

- **D6 — STOP, P1.** A cash-banned customer can place a cash custom order, where the
  driver buys the goods, so this is a financial risk-control bypass. Custom orders are
  open on Staging.
- **D8 — STOP, P1, latent.** Verification boundary bypass.
- **D9 — not stop-class, P2.** Audit-trail and notification completeness.

Launch flags are not a boundary (Owner rule).

**Regression:** flip `TestCENSUS_D6_CustomOrderSkipsCashBan` and
`TestCENSUS_D6_D9_CustomOrderCreationGuards` from EXPECTED_FAIL to PASS, plus explicit
tests per rule.

**P-9:** RISK **CRITICAL** · 212 mandatory tests (`./internal/orders ./internal/qa`).

### 40.9 · D11 — forced password change

**Backend holds.**

- `RequireAuth` returns 403 `password_change_required` on everything except
  `/auth/me`, `/auth/password`, `/auth/logout` and `/auth/capabilities`
  (`server/middleware.go:~57-67`).
- Guarded by `TestTMP1..4`.
- **A flagged customer cannot continue normal use.**

**App:**

- no dedicated gate;
- every screen shows «كلمة المرور الحالية وضعها شخص آخر — بدّلها لتتابع»;
- the Account screen still renders its password section when the summary call fails,
  so a way out exists.

**CONFIRMED · P2 · not stop-class** (no bypass). The only bypass route is signup
clearing the flag, which is part of CUST-DEF-001.

### 40.10 · Reconciliation of all 22 §38.4 findings

| Finding | Verdict | Sev | Stop | Surface · component | Note |
|---|---|---|---|---|---|
| CAF-01 | CONFIRMED → **CUST-DEF-001** | P0 | **YES** | signup · `identity/service.go` | §40.3 |
| CAF-02 | CONFIRMED → **CUST-DEF-002** | P1 | **YES** | cart/custom send · `ui/Attempt.kt` | §40.4 |
| CAF-03 | CONFIRMED → **CUST-DEF-003** | P1 | **YES** | `POST /orders` · `orders/service.go` | §40.5 |
| CAF-04 | CONFIRMED | P2 | no | suspended customer · `server/suspension.go:55-66` | exception names `GET /api/v1/orders/{uuid}`, which has no customer route; `/my/orders*` blocked → live order unreachable in the app; cancel by id still allowed. Restrictive, not a bypass |
| CAF-05 | CONFIRMED | P3 | no | pre-launch browse · `server.go:~439-499` | sections, section items, suggest, search stay open while `customer_browse` is off; public catalog data, app shows PreLaunch |
| CAF-06 | CONFIRMED | P2 | no | order create · `orders/availability.go:169` | `classifyPlace` used only by availability; create enforces zones only |
| CAF-07 | CONFIRMED | P2 | no | custom order · `custom_order_handlers.go` | `notes` not decoded → driver note silently dropped. Wallet option: see §40.11 |
| CAF-08 | CONFIRMED → **CUST-DEF-005** | P1 | **YES** | cart · `CartScreen.kt` | §40.7 |
| CAF-09 | CONFIRMED → **CUST-DEF-004** | P1 | **YES** | logout / account switch | §40.6 |
| CAF-10 | CONFIRMED | P2 | no | session expiry mid-use · `AuthViewModel.kt` | session cleared only at startup and logout; a revoked session keeps loaded data on screen with an error message; server refuses new data. Socket side = R14 |
| CAF-11 | CONFIRMED | P3 | no | Shop banners · `ShopScreen.kt:194`, `BannerSlider.kt:188` | target banners clickable, no `onOpen` → silent tap |
| CAF-12 | CONFIRMED | P3 | no | Offers · `MineScreens.kt:204,345` | `Cart.add` without the PreCart gate; send still requires address, availability and quote |
| CAF-13 | DUPLICATE_OF_XG-9 | — | no | push tap | order-status pushes fall to `else -> Unit` (`MainActivity.kt:1095`) |
| CAF-14 | CONFIRMED | P2 | no | order history · `OrdersViewModel.kt:99,145` | page 1 (30) only |
| CAF-15 | NEEDS_OWNER_DECISION | P3 | no | rail auto-scroll · `model/Shop.kt` | `shop.rail_auto` sits in the **Site** settings group (page.shop); the Android app never had auto-scroll — is the setting meant for the app? |
| CAF-16 | NOT_A_DEFECT (signup part → CUST-DEF-001) | — | no | launch flags on client | server gates orders/custom/signup-request and answers with explicit `launch_closed` + Owner notice |
| CAF-17 | CONFIRMED | P3 | no | cart · `CartScreen.kt:518,587` | promo and payment choice in memory only; reset visibly after process death |
| CAF-18 | CONFIRMED (`in_progress` part → CUST-DEF-002) | P3 | no | error texts · `ui/ApiErrors.kt` | `not_found`, `comms_closed`, `comms_no_driver` unmapped → misleading «تعذر الاتصال» |
| CAF-19 | CONFIRMED | P2 | no | warnings · `/my/warnings` | Admin panel promises «يصل الإنذار صاحب الحساب بنصه، ويبقى في سجله», but `issueWarning` sends no notification and the app never shows warnings → the customer is never told. Warnings are user-addressed by design (reason + Admin note), so decision 40.1-2 governs **how** they are worded, not **whether** they reach the customer |
| CAF-20 | CONFIRMED | INFO | no | analytics · `app_opens.go` | every `/public/home` call counts an open; home reloads on every realtime frame → inflated |
| CAF-21 | CONFIRMED | P3 | no | guest cart · `CartScreen.kt:202` | address card opens the sheet for guests; geo endpoints need auth |
| CAF-22 | NOT_A_DEFECT | — | no | reset / verify | test dependency: `/auth/wa/ticket` needs the Staging WhatsApp bot |

**Consolidated (no duplicate IDs):**

- CAF-09 + PC-2 → CUST-DEF-004;
- CAF-16 signup part → CUST-DEF-001;
- CAF-18 `in_progress` → CUST-DEF-002;
- CAF-13 → XG-9;
- server socket side → R14.

### 40.11 · Product questions — answered by existing contracts or genuinely open

| Question | App today | Backend | Contract / documents | Verdict |
|---|---|---|---|---|
| Custom-order payment options | cash only (no `payment_method` sent) | accepts `wallet` (`custom.go:160-164`); wallet settled at delivery; short balance logged as debt (`:400-417`) | **Owner decision 2026-08-09** in `orders/custom.go:74-78`: «والدفعُ نقدٌ أو محفظة… والمحفظةُ خيارٌ» | **Answered** → the app is missing the wallet option (CONFIRMED gap, P2, not stop) |
| Ticket reply | tickets list, read-only; tickets created only through the complaint dialog | replies are Admin-only | **PRQ-2 «CENTRAL SUPPORT CENTER» — «معتمدٌ»** (`CUSTOMER_FINAL_PRODUCT_DECISIONS.md:424`); the missing reply is named its heaviest gap; «ولا يُبنيان الآن» | **Behaviour answered** (customer should reply). **Genuinely open: is PRQ-2 in scope for this release, or deferred?** |

### 40.12 · Proposed fix order (not implemented)

| # | Item | Why this position |
|---|---|---|
| 1 | **CUST-DEF-001** | P0 · all roles · backend-only · server deploy, no app release |
| 2 | **CUST-DEF-003** | financial trust boundary · backend-only |
| 3 | **D6 + D8** (+ D9 in its own commit) | same file (`custom.go`), backend-only; D9 is non-blocking |
| 4 | **CUST-DEF-002** | shared `ui` client; Kotlin tests |
| 5 | **CUST-DEF-004** | client session boundary; device witness required |
| 6 | **CUST-DEF-005** | cart pricing truth; device witness |
| 7 | §7 offline blocking (L1-019 feature build) | not a defect — required before CUST-16 can pass |

Each item is a separate atomic cycle (§3):

1. record;
2. regression test failing first;
3. smallest centralized fix;
4. impacted suites;
5. re-run.

Items 4–6 can ship in one Android debug build only after each passes its own cycle.
Non-blocking CONFIRMED items (D9, D11, CAF-04/06/07/10/11/12/14/17/18/19/21 and the
custom wallet option) are scheduled inside normal acceptance, in their groups.

### 40.13 · Cleanup truth (2026-09-19)

- **The Owner manually removed** the four RahalGo apps from Samsung Dual Messenger
  (user 95): **DONE**.
- **ADB verification: pending** until the phone is reachable. CUST-00-009 stays
  `NOT_TESTED` until then.
- Test residue on the phone (`/sdcard/def001`, `/data/local/tmp/def001.sh`,
  `/sdcard/rg_pre.xml`, `/sdcard/rg_ui.xml`) may remain until ADB is available. This
  is not a Customer acceptance blocker.

**Gate result:** 7 stop-class blockers confirmed (CUST-DEF-001…005, D6, D8). **Customer
acceptance execution stays closed until they are resolved through §3.**
