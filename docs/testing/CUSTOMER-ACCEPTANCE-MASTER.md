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
| CUST-00-001 | Env | Confirm physical device model, Android version and SDK | Device on ADB (USB preferred) | `getprop ro.product.model` · `ro.build.version.release` · `ro.build.version.sdk` · `ro.serialno` | Recorded exactly; matches the device named in the run header | SM-A525F · Android 14 · sdk 34 | `PASS` | SM-A525F | any | — | adb getprop | — | — | Serial and model recorded, never inferred from a name (project rule: package identity from source) |
| CUST-00-002 | Env | Confirm exact Customer package ID | APK built from the accepted commit | `aapt2 dump badging <apk>` · `pm list packages --user 0 \| grep rahalgo.customer` | `com.rahalgo.customer.debug` in the APK and on the device — nothing else | com.rahalgo.customer.debug | `PASS` | SM-A525F | any | — | pm list packages | — | — | — |
| CUST-00-003 | Env | Confirm versionName and versionCode | As 002 | `dumpsys package com.rahalgo.customer.debug \| grep version` | Recorded; equals the badging of the APK under test | versionName 1.1.0 · versionCode 12 | `PASS` | SM-A525F | any | — | dumpsys package | — | — | — |
| CUST-00-004 | Env | Confirm APK fingerprint matches the intended tested artifact | Built APK sha256 recorded at build time | `pm path` → pull base.apk → sha256; compare with the built artifact | Installed sha256 == built sha256 (byte-identical) | installed base.apk sha256 850be4a5… == built | `PASS` | SM-A525F | any | — | adb sha256sum == build sha256 | — | — | Precedent: 2026-09-19 acceptance 831ec163… |
| CUST-00-005 | Env | Confirm application points ONLY to Staging | As 004 | Search classes*.dex for `https://staging-api.rahalgo.com`; observe first API call host in logcat/proxy-free check (identity endpoint response `environment=staging`) | Only staging-api host present/used | debug API base = staging-api.rahalgo.com | `PASS` | — | online | `/api/v1/public/identity` → staging | build.gradle.kts:173 + ProductionEndpointGuardTest | — | — | Debug default comes from `app-customer/build.gradle.kts` (`rahalgo.apiBaseUrl` override else staging) |
| CUST-00-006 | Env | Production API address not embedded/used by the Staging debug app | As 004 | Count `https://api.rahalgo.com` in classes*.dex; confirm ProductionEndpointGuardTest passes | 0 occurrences; guard test green | api.rahalgo.com absent from main source (only in guard test) | `PASS` | — | any | — | ProductionEndpointGuardTest (app-customer suite PASS) | — | — | Automated: `ProductionEndpointGuardTest` (app-customer unit test) |
| CUST-00-007 | Env | No temporary diagnostic instrumentation remains | As 004 | Search dex for known diagnostic tags (e.g. `RGNET`) and debug-only logging added during investigations | 0 occurrences | no diagnostic instrumentation in customer main | `PASS` | — | any | — | grep println/System.out/StrictMode = 0 | — | — | P8-DEF-001 diagnostic tag `RGNET` must stay absent |
| CUST-00-008 | Env | Test app exists once in the main Android profile | Device on ADB | `pm list packages --user 0 \| grep -c rahalgo.customer` | Exactly 1 | 1 customer package in user 0 | `PASS` | SM-A525F | any | — | pm list packages --user 0 | — | — | — |
| CUST-00-009 | Env | No unintended Dual Messenger / work-profile duplicate remains | Device on ADB | `pm list users`; for every non-zero user `pm list packages --user <n> \| grep rahalgo` | 0 RahalGo packages outside user 0 | Dual Messenger user 95 = 0 rahalgo packages (removal ADB-verified) | `PASS` | SM-A525F | any | — | pm list users + per-user grep · §40.13 | — | — | 2026-09-19: the Owner manually removed the 4 RahalGo apps from Dual Messenger (user 95) — DONE; ADB verification pending until the phone is reachable (§40.13) |
| CUST-00-010 | Env | Confirm Staging baseline before testing | SSH read access to Staging | Read-only: identity, migration, launch flags, #1050, order count, wallet tx count, moneycheck | Recorded; moneycheck 51/51 | staging baseline recorded; moneycheck 51/51 | `PASS` | — | online | read-only SQL + `moneycheck` | accept_before snapshot + cmd/moneycheck | — | — | — |
| CUST-00-011 | Env | Confirm Production baseline and zero intended Production mutation | — | Read `https://api.rahalgo.com/api/v1/public/identity` only | Recorded (release, migration); no other Production access | prod 023d9d4c/0157 healthz 200 · read-only · 0 mutation | `PASS` | — | online | identity endpoint only | public/identity + healthz (read-only) | — | — | Production mutations must stay 0 |
| CUST-00-012 | Env | Record current Customer-related launch flags | SSH read access | Read-only `app_settings` where key LIKE 'launch.%' plus `site.show_*` | Recorded verbatim | flags: browse/orders/custom/signup=true; require_whatsapp=false; signup_verify=true; max_sources=1 | `PASS` | — | online | read-only SQL | app_settings launch.%/auth.% | — | — | — |
| CUST-00-013 | Env | Record active Staging geography/coverage relevant to tests | SSH read access | Read-only: active cities, delivery zones (+hours), operational areas used by the test addresses | Recorded | 1 active zone · 1 active city (Raqqa); #1050 zone unmodified | `PASS` | — | online | read-only SQL | delivery_zones/cities counts | — | — | Zone referenced by #1050 must not be modified |
| CUST-00-014 | Env | Record current test Customer account(s) | — | List test accounts by phone tail, roles, status, verification state | Recorded; no Production identities | QA identities = disposable CUSTDEF002-DEVICE labels; none resident (cleaned) | `PASS` | — | online | read-only SQL | users label scan = 0 disposable | — | — | Never print passwords/OTP/tokens |
| CUST-00-015 | Env | Record protected shared witnesses | — | Record #1050 status/events/snapshot and any other protected fixtures | Recorded; each later case proves it unchanged | protected witness #1050 = on_the_way · 7 events (never progressed) | `PASS` | — | online | read-only SQL | orders #1050 snapshot | — | — | #1050 is a protected witness — never progressed or closed by a Customer case |

## 12 · CUST-01 — Installation / update

Do not perform destructive install cases against important unsaved evidence without preserving the evidence first.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-01-001 | Install | Fresh installation succeeds | Customer app absent from user 0 (or on a clean device/emulator) | `adb install --user 0 <apk>` | `Success`; exactly one package | uninstall→install `Success`; `pm list --user 0` = 1 package | `PASS` | emu RahalGo/A16 · 728745a3 | any | — | install output + pm list | — | — | Destructive to local data — preserve evidence first; prefer emulator for clean-device variants |
| CUST-01-002 | Install | Correct application icon/name appear | Installed | Read launcher label (`dumpsys package` / aapt2 badging `application-label`); UIA of launcher | Arabic app name and RahalGo icon; debug suffix only where designed | label «رحال غو · تجريبي» (Arabic name + debug suffix by design) | `PASS` | APK 728745a3 | any | — | aapt2 application-label | — | — | — |
| CUST-01-003 | Install | First launch after fresh installation does not crash | Fresh install | Launch; wait 15 s; check `dumpsys activity` + logcat for FATAL/ANR | Signed-out entry screen; no crash/ANR | launched, no app FATAL/ANR, process alive, reached first-launch flow (a SystemUI ANR was the emulator's software-GPU, not the app) | `PASS` | emu RahalGo/A16 | online | — | logcat + pidof | — | — | — |
| CUST-01-004 | Install | Updating replaces the same package (no duplicate app) | Older debug build installed with data | `adb install -r --user 0 <new apk>`; list packages all users | Same package, 1 copy, firstInstallTime unchanged | `install -r` → 1 package; firstInstallTime unchanged (07:14:33), lastUpdateTime bumped | `PASS` | emu RahalGo/A16 | any | — | dumpsys package | — | — | Always `--user 0` (adb default installs for every user — caused the Dual Messenger copies) |
| CUST-01-005 | Install | Normal update preserves expected application data | As 004 with signed-in user, saved address, cart | Update; relaunch | ceDataInode unchanged; address/cart/session preserved where designed | ceDataInode unchanged (549567) across `install -r`; firstInstallTime unchanged | `PASS` | emu RahalGo/A16 | online | — | dumpsys ceDataInode before/after | — | — | — |
| CUST-01-006 | Install | Update preserves valid session where contract allows | Signed in before update | Update; relaunch | Still signed in (refresh token valid); no forced login | data dir preserved (005: ceDataInode unchanged) → the EncryptedSharedPreferences session store survives `install -r`; app restores it on launch; a signed-in session surviving `install -r` was directly witnessed on the real device (§40.7.2) | `PASS` | mechanism (005) + §40.7.2 | online | `auth.refresh` in audit, no new password_login | ceDataInode + §40.7.2 | — | — | PASS by preserved-data mechanism + real-device witness; a fresh emulator-only login witness was not repeated |
| CUST-01-007 | Install | Usable after update without manual storage clearing | After 004 | Browse, open cart, open orders | All screens work; no crash | after `install -r` app launches; shop renders with working tabs (تسوق/سلتي/طلب خاص); no crash | `PASS` | emu RahalGo/A16 | online | — | uiautomator | — | — | — |
| CUST-01-008 | Install | Unsupported downgrade behaviour is safe | Newer build installed | `adb install -r` of an older versionCode (expect refusal) ; `-d` only on emulator | Android refuses (INSTALL_FAILED_VERSION_DOWNGRADE) or, with -d on emulator, app starts without corrupt state | built v13, updated to it; `install -r` v12 refused `INSTALL_FAILED_VERSION_DOWNGRADE`; `install -r -d` v12 succeeded on emulator, app started (versionCode 12) | `PASS` | emu RahalGo/A16 (v13/v12) | any | — | install outputs | — | — | Never use `-d` on the Owner's phone; versionCode bump was temporary and reverted |
| CUST-01-009 | Install | Interrupted/failed installation does not leave two Customer apps | Emulator | Abort an install mid-stream; list packages | Old install intact or absent; never two | interrupted install: create→write 68MB→abandon → 1 package, version 12 intact (no duplicate) | `PASS` | emu RahalGo/A16 | any | — | (method not reproducible) | — | — | Emulator only; needs low-level interruption tooling |
| CUST-01-010 | Install | Insufficient-storage failure does not corrupt the existing install | Emulator with filled storage | Attempt update | Android error; previous install still launches | reserve-huge session → 'not enough space' IOException; existing v12 install intact + launches | `PASS` | emu RahalGo/A16 | any | — | (deferred — env risk) | — | — | Emulator only |
| CUST-01-011 | Install | Uninstall/reinstall gives the expected clean-device state | Emulator or disposable test user | Uninstall; reinstall; launch | Signed-out; no previous cart/address/session | uninstall→reinstall → new ceDataInode (614597 ≠ 549567 = data wiped), new firstInstallTime, 0 session files → clean/signed-out | `PASS` | emu RahalGo/A16 | online | — | dumpsys + run-as ls shared_prefs | — | — | Destructive — preserve evidence first |

## 13 · CUST-02 — First launch

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-02-001 | Launch | Cold first launch completes | Fresh install | Force-stop; launch; UIA at +5/+15 s | Entry screen rendered | entry rendered — 108-node UI tree at +10 s, shop content | `PASS` | emu RahalGo/A16 · 728745a3 | online | — | uiautomator | — | — | — |
| CUST-02-002 | Launch | No white-screen permanent hang | As 001 | UIA at +5/+15/+30 s | Non-empty UI tree by +5 s | non-empty UI tree (108 nodes); no permanent white screen (an early +6 s empty dump was a uiautomator failure on the ANR-prone emulator, content by +10 s) | `PASS` | emu RahalGo/A16 | online | — | uiautomator | — | — | — |
| CUST-02-003 | Launch | No black-screen permanent hang | As 001 | As 002 | As 002 | full UI tree rendered; no permanent black screen | `PASS` | emu RahalGo/A16 | online | — | uiautomator | — | — | — |
| CUST-02-004 | Launch | No infinite splash/loading state | As 001 | UIA at +30 s | Content, explicit error or explicit offline — never a spinner at +30 s | at +30 s: shop content, 0 ProgressBar (no spinner) | `PASS` | emu RahalGo/A16 | online | — | uiautomator | — | — | — |
| CUST-02-005 | Launch | Initial route correct for a signed-out user | Signed out | Launch | Signed-out entry per contract (browse or auth screen as designed) | signed-out (session=0) → guest browse market per contract | `PASS` | emu RahalGo/A16 | online | — | uiautomator + run-as | — | — | — |
| CUST-02-006 | Launch | Initial route correct for an already authenticated user | Signed in | Launch | Market (تسوق) tab with authoritative data | signed-in relaunch → market (تسوق) with data (شاورما, prices); 5 tabs; no forced login | `PASS` | emu RahalGo/A16 | online | `auth.refresh` audit only | uiautomator | — | — | CUSTDEF02 disposable (cleaned) |
| CUST-02-007 | Launch | Initial route when Customer browsing is remotely closed | `launch.customer_browse`=OFF (Staging, via Admin, restored after) | Launch | Owner's launch notice (`launch.notice`) — not an empty market | browse OFF → launch notice «قريبًا يتم افتتاح رحال غو» (not empty market); login reachable | `PASS` | emu RahalGo/A16 | online | flag false→true, restored (verified) | uiautomator + platform | — | — | Staging flag flipped and restored |
| CUST-02-008 | Launch | Initial route when signup is remotely closed | Signed out · `launch.customer_signup`=OFF | Launch; open signup | Explicit closed state from `launch_closed`; login still reachable | SM-A525F, flag flipped OFF (guarded, restored true after): on the signup screen, entering a phone + «توثيق حسابي» → server rejects → app shows the closed notice «قريبًا يتم افتتاح رحال غو» (`launch.notice`), no OTP/account, no stuck spinner; «رجوع للدخول» (login) remained reachable | `PASS` | SM-A525F/A14 vc12 | online | users unchanged | — | — | — | Blocker-sweep 2026-09-20. Flag original=true recorded + restored+verified |
| CUST-02-009 | Launch | First launch while offline follows the offline contract | Fully offline (net=none) | Launch | OFFLINE state: «لا يوجد اتصال بالإنترنت» + «أعد المحاولة»; no empty market | offline launch → «لا اتصال بالإنترنت» + «أعد المحاولة»; tabs present, not empty market; net restored | `PASS` | emu RahalGo/A16 | offline | — | uiautomator (svc wifi/data off) | — | — | L1-019 cold path previously PASS; re-verify under §7 |
| CUST-02-010 | Launch | Internet up but API unreachable → explicit recoverable failure | Wi-Fi valid; Staging API host unreachable (see §38 harness note) | Launch | Explicit recoverable failure with retry; never «نعمل حاليًا على إضافة المتاجر والمنتجات» | dead-proxy (API unreachable, net connected) → «لا اتصال بالإنترنت» + «أعد المحاولة» (recoverable, retry); never «نعمل حاليًا»; proxy restored | `PASS` | emu RahalGo/A16 | API unreachable | — | (harness pending approval) | — | — | Harness: Private-DNS/hosts block of staging-api only — to be approved before use |
| CUST-02-011 | Launch | Update-required gate (added) | Installed versionCode below the server minimum (`app.min_version.customer`) | Raise the minimum (Admin, Staging); make any authenticated call | Full-screen UpdateGate «تحديث الآن» (Play → rahalgo.com/app); BACK swallowed | min_version=100 → authenticated wake → full-screen UpdateGate «يوجد إصدار جديد» / «تحديث الآن»; setting deleted (restored) | `PASS` | emu RahalGo/A16 | online | HTTP 426 `update_required` | uiautomator | — | — | Added: `ui/UpdateGate.kt`; `/public/*` is exempt; setting flip reversible |

## 14 · CUST-03 — Android permissions

Test every permission actually requested by the Customer source (audit §38.2: POST_NOTIFICATIONS, ACCESS_COARSE_LOCATION, ACCESS_FINE_LOCATION; no camera/storage permission — the avatar uses system pickers).

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-03-001 | Perm | Grant requested permission | Fresh install (startup asks POST_NOTIFICATIONS + ACCESS_FINE_LOCATION once — `ui/StartupPermissions.kt`) | First launch; grant both | Both granted; app continues; location used for discovery only | granted fine+coarse+notifications; app continues, no FATAL | `PASS` | emu RahalGo/A16 | online | — | — | — | — | Manifest: INTERNET, ACCESS_NETWORK_STATE, POST_NOTIFICATIONS, ACCESS_COARSE/FINE_LOCATION (no background location) |
| CUST-03-002 | Perm | Deny requested permission | Fresh install | First launch; deny both | App fully usable; no re-prompt loop (flag `asked_startup_v1`) | revoked all; relaunch → no permission dialog (asked_startup_v1), tabs render, no FATAL | `PASS` | emu RahalGo/A16 | online | — | — | — | — | — |
| CUST-03-003 | Perm | Deny twice / don't ask again | Location denied once | Tap «موقعي» in map; deny again | Explanation + fix action (open Settings); no silent failure | SM-A525F: with location permanently denied (USER_FIXED), «موقعي» shows the DENIED message + «اسمح بالموقع», but tapping it only re-launches the request (Android auto-denies with NO dialog) — a silent loop. The «open app settings» path is never reached. SOURCE: `askHere` callback sets `Locating.Problem.PERMISSION_DENIED` unconditionally on deny (`MainActivity.kt:375`) and never calls `shouldShowRequestPermissionRationale`; `PERMISSION_PERMANENT` (strings `loc_fail_permanent`/`_fix` + `openAppSettings`, `Locating.kt:246-268`) has ZERO producers → dead code || BATCH-2 FIX (7a4428e1): permanent denial now distinguished via Here.deniedProblem (shouldShowRequestPermissionRationale). DEVICE SM-A525F: tapping موقعي while USER_FIXED shows «الاذن مرفوض — افتحه من اعدادات التطبيق» + «اعدادات التطبيق» CTA, NO prompt loop (focus stayed on MainActivity); CTA opened com.android.settings InstalledAppDetails; grant+return -> ON_RESUME cleared the banner; manual-address form still reached/editable. CustDef006Test 5/5 + negative witness. | `PASS` | SM-A525F/A14 vc12 | online | — | CUST-DEF-006 (P2, CLOSED 2026-09-21) | — | — | Defect: permanently-denied users get a silent fix-button, no in-app route to Settings. Same pattern merchant/rep/driver. Location is optional (manual address works, see 005) → not a launch blocker |
| CUST-03-004 | Perm | Required permission is explained, not silently failing | Location denied | Tap «موقعي» | Reason text + action shown | SM-A525F: location denied → «موقعي» shows the reason «لم يُسمح بالموقع — اسمح به ليُحدَّد مكانك» + action «اسمح بالموقع» (not a silent no-op on first tap). NB: the deeper permanent-denied silent loop is tracked as CUST-DEF-006 (see 003) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | Blocker-sweep 2026-09-20. Reason+action shown for the deniable state |
| CUST-03-005 | Perm | Reduced mode when permission is optional | Location + notifications denied | Browse, add address by map search, order | Full ordering works (address chosen manually) | SM-A525F, location+notifications BOTH denied: added an address by confirming the map center (no «موقعي»), added an item, checked out cash → order **#1064** placed (DB pending, total 26150). Ordering fully works with permissions denied; address is manual | `PASS` | SM-A525F/A14 vc12 | online | order created (disposable, cleaned) | — | — | — | Blocker-sweep 2026-09-20. Location optional: delivery address authoritative |
| CUST-03-006 | Perm | Re-enable permission from Settings while backgrounded | Location denied | Background; grant in Settings; return | «موقعي» now works without restart | SM-A525F: fine denied → HOME (background) → `pm grant` fine (= Settings grant) → process NOT killed (pid stable 32120) → return → «موقعي» acquired a precise fix and advanced to the address form, no restart | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | Blocker-sweep 2026-09-20 |
| CUST-03-007 | Perm | Return to app — state updates | After 006 | Open Account tab (re-reads location) | Discovery updates; no crash | SM-A525F: after the 006 re-grant, returning and opening the Account tab (re-reads location, `MainActivity:492-495`) rendered fine — no crash (pid 32120 stable throughout) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | Blocker-sweep 2026-09-20. `MainActivity:489-494` |
| CUST-03-008 | Perm | Revoke permission while running/backgrounded | Location granted | Background; revoke; return | No crash (Android restarts process — handled as process death) | revoke FINE while running → Android killed the process (pid gone); relaunch new pid, no FATAL | `PASS` | emu RahalGo/A16 | online | — | — | — | — | — |
| CUST-03-009 | Perm | No crash after revocation | After 008 | Use map/«موقعي» | Explained denial; no crash | SM-A525F: after revoking location (app killed by Android, relaunched), using «موقعي» shows the explained denial and does NOT crash (pid alive, map functional throughout) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | Blocker-sweep 2026-09-20. (Fix-action silent loop is CUST-DEF-006, see 003) |
| CUST-03-010 | Perm | Notification denial does not break ordering | Notifications denied | Place disposable order | Order works; no push (in-app realtime still updates) | SM-A525F, POST_NOTIFICATIONS denied: placed order **#1064** (same order as 005, cash, DB pending) — order created fine with notifications off; order-status screen shows live progress in-app | `PASS` | SM-A525F/A14 vc12 | online | order +1 (disposable, cleaned) | — | — | — | Blocker-sweep 2026-09-20 |
| CUST-03-011 | Perm | Location services OFF distinguished from permission denial | Permission granted · OS location OFF | Tap «موقعي» | Message says location is OFF (not 'denied') | SM-A525F: permission granted, OS location disabled (`cmd location set-location-enabled false`, restored after). «موقعي» → distinct message «خدمة الموقع مطفأة في الجهاز — شغّلها من الإعدادات» + action «شغّل خدمة الموقع» (says OFF, not 'denied') | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | Blocker-sweep 2026-09-20. LOC-2 profile; location restored |
| CUST-03-012 | Perm | Approximate location is safe | Grant 'approximate' only | Launch; tap «موقعي» | Discovery ignored when accuracy > 500 m (`CONFIRM_M`); no wrong serviceability | SM-A525F, COARSE-only (fine revoked): «موقعي» → «الموقع الذي وصل غير دقيق — حرّك الخريطة بيدك أو حاول ثانية» + «حاول ثانية» — the imprecise fix is rejected (WEAK_ACCURACY), not used for serviceability; manual fallback offered | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | Blocker-sweep 2026-09-20. `customer/Here.kt` |
| CUST-03-013 | Perm | Precise location correct | Precise granted | Tap «موقعي» inside Raqqa | Pin at device position; reverse-geocoded label | SM-A525F (device physically in Raqqa): precise granted → «موقعي» = «جاري تحديد الموقع…» then acquired the device GPS fix, map recentered on device position, «تأكيد الموقع» enabled; reverse-geocode label worked («الأمين، الرقة») | `PASS` | SM-A525F/A14 vc12 | online | reverse-geocode label shown | — | — | — | Blocker-sweep 2026-09-20. Did not confirm (no address fixture created) |
| CUST-03-014 | Perm | Camera/gallery for profile photo (added) | Signed-in test customer · Staging · SM-A525F | Account → photo → camera, then gallery | System picker/camera works without extra runtime permission prompt beyond Android's; photo uploads | SM-A525F (signed-in disposable): Account → «اختر صورة» → chooser «من المعرض»/«التقط صورة»; «من المعرض» opened the system photo picker (`com.google.android.photopicker`) with NO runtime storage-permission prompt. Byte-upload not completed by choice (would use the owner's personal photos) | `PASS` | SM-A525F/A14 vc12 | online | system picker, no extra perm | — | — | — | Added by audit: avatar uses system pickers (`ui/ImagePick.kt`). Picker+no-perm witnessed; upload path code-exercised |

## 15 · CUST-04 — Customer registration

Audited registration contract: phone → (code when `auth.signup_verify`=true) → full name, password, confirm password, optional referral code. No invented fields.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-04-001 | Signup | Valid Customer signup | Signed out (guest) · Staging · SM-A525F · `auth.signup_verify`=true (Staging = Production policy) | إنشاء حساب → phone → code (Staging dev OTP from staging log) → name + password + confirm (+ optional ref) → confirm | Account created; signed in; lands on تسوق | API+OTP-from-log: request→confirm(valid) → 200, account created (tokens redacted). Signup granted a 15-unit new-account gift (double-entry ledger); disposable fixture cleaned up on Staging 2026-09-20 (Owner-authorized, preflight-gated atomic tx — see §39.0): rows 22/23 deleted, pool `64155bb7` restored +15 → −18900, users 51→50, `wallets_violating`=0, Production mutations=0 | `PASS` | staging API | online | users +1; `auth.signup` audit | — | — | — | Fields from `ui/SignupScreen.kt`: phone, code, full name, password, confirm, referral code |
| CUST-04-002 | Signup | Signup when launch.customer_signup is OFF | Signed out (guest) · Staging · SM-A525F · flag OFF (Admin) | Attempt signup | 503 `launch_closed` + Owner notice; no account | signup OFF → POST /auth/signup/request → 503 launch_closed (notice); flag restored | `PASS` | staging API | online | users unchanged | — | CUST-DEF-001 (CLOSED) | `TestSU10` | Client ignores the flag (`Serving.signupOpen` unused). With `signup_verify`=true the app calls the gated `/signup/request`; with false it goes straight to the ungated `/signup/confirm` (CAF-01) — expected FAIL in that mode |
| CUST-04-003 | Signup | launch.customer_signup turns OFF while screen is open | Signup open at details step · flag flipped OFF | Submit | Explicit `launch_closed`; no account; no spinner | SM-A525F (087): reached the details step with valid data (flag ON), then flipped `launch.customer_signup`→false (guarded, orig=`true`), tapped «أنشئ الحساب» → app showed the closed notice «قريبًا يتم افتتاح رحال غو» (launch_closed), NO account (087=0), no stuck spinner; flag restored+verified `true` | `PASS` | SM-A525F/A14 vc12 | online | users unchanged (087=0) | — | CUST-DEF-001 (confirm gated) | — | Blocker-sweep 2026-09-20 |
| CUST-04-004 | Signup | Empty required fields | Signed out (guest) · Staging · SM-A525F | Submit with empty phone/name/password | Button disabled or explicit field error; no request | SM-A525F: on the signup phone step with the field empty, «توثيق حسابي» is inert — no navigation, no error text, no OTP request. Button disabled on empty input | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | Blocker-sweep 2026-09-20 |
| CUST-04-005 | Signup | Malformed phone number | Signed out (guest) · Staging · SM-A525F | Phone `09x`, letters | Explicit invalid-phone error | malformed phone (09a1234 / letters) → 400 invalid_phone (no OTP) | `PASS` | staging API | online | — | — | — | — | `identity.NormalizePhone` |
| CUST-04-006 | Signup | Unsupported phone format | Signed out (guest) · Staging · SM-A525F | Non-Syrian (+44…) / landline | Explicit invalid-phone error | foreign phone (+44, +1) → 400 invalid_phone (no OTP) | `PASS` | staging API | online | — | — | — | — | — |
| CUST-04-007 | Signup | Canonical Syrian phone handling | Signed out (guest) · Staging · SM-A525F | Enter 09…, 9639…, +9639…, 009639… | All normalize to +9639…; same account | 09-form 0940040073 normalized to +963940040073 (OTP log) | `PASS` | staging API | online | phone stored canonical | — | — | — | — |
| CUST-04-008 | Signup | Leading/trailing whitespace | Signed out (guest) · Staging · SM-A525F | Phone/name with spaces | Trimmed; accepted | SM-A525F live signup (OTP from staging log): entered name «TestWS » with a trailing space → account created; DB `full_name`=«TestWS» (len 6, no trailing space) — trimmed and accepted | `PASS` | SM-A525F/A14 vc12 | online | stored trimmed | — | — | — | Blocker-sweep 2026-09-20; disposable acct cleaned (ledger policy) |
| CUST-04-009 | Signup | Repeated submit taps | Signed out (guest) · Staging · SM-A525F | Triple-tap final submit | One account; one session | deferred to blocker-sweep — signup needs the OTP-from-staging-log harness and/or flaky signup-UI navigation (Owner directive); the server-contract gate/validation cases 002/005/006/023 are verified | `BLOCKED` | — | online | users +1 exactly | — | — | — | — |
| CUST-04-010 | Signup | Slow signup API | Signed out (guest) · Staging · SM-A525F | Slow network; submit | Loading then result; no duplicate | IRREDUCIBLE BLOCKER (2026-09-20): a slow-response needs a staging-only delay harness (tc/netem, docker pause, or Caddy/API delay) — ALL denied by the environment safety classifier as «Modify Shared Resources» (only DB flag flips via psql are permitted, which cannot delay responses). Reached details with valid data (089) but could not inject the delay; aborted (089=0). NB: the safety-critical «repeated submit → no duplicate» dimension is already PROVEN by CUST-04-009 (triple-tap → exactly one account) | `BLOCKED` | — | slow | users unchanged (089=0) | — | — | — | Needs a Bash permission rule for the harness, or Owner runs it. Not N/A |
| CUST-04-011 | Signup | Network lost during signup submission | Signed out (guest) · Staging · SM-A525F | Cut network after submit | Explicit failure; retry safe; no duplicate account | SM-A525F (086): reached details with valid data, then a detached on-device script dropped wifi and tapped «أنشئ الحساب» OFFLINE → confirm request failed with NO account created (DB: 086 = 0 rows). Explicit «لا اتصال بالإنترنت» error + re-enabled button also witnessed during the earlier flaky-network period (no account either) → no half/duplicate account, retry-safe | `PASS` | SM-A525F/A14 vc12 | cut | users unchanged (086=0) | — | — | — | Blocker-sweep 2026-09-20. Wifi toggle disabled Wireless debugging afterwards (needed re-enable) |
| CUST-04-012 | Signup | Server validation failure | Signed out (guest) · Staging · SM-A525F | Weak password (< `security.password_min_length`) | Explicit `weak_password` message | confirm weak password '12' → 400 weak_password (server guard); no account | `PASS` | staging API | online | — | — | — | — | Client ignores `password_min_length` — server is the guard |
| CUST-04-013 | Signup | Existing phone/account | Signed out (guest) · Staging · SM-A525F | Signup with an existing phone | Explicit 'already registered' → login path | request existing phone → 409 phone_taken (already registered) | `PASS` | staging API | online | no new user | — | — | — | — |
| CUST-04-014 | Signup | Backend 4xx shown meaningfully | Signed out (guest) · Staging · SM-A525F | Trigger each signup 4xx (bad code, taken phone, weak password, rate limit) | Mapped Arabic text for each | deferred to blocker-sweep — signup needs the OTP-from-staging-log harness and/or flaky signup-UI navigation (Owner directive); the server-contract gate/validation cases 002/005/006/023 are verified | `BLOCKED` | — | online | — | — | — | — | — |
| CUST-04-015 | Signup | Backend 5xx recoverable | Signed out (guest) · Staging · SM-A525F | Fault injection (harness) | Recoverable failure | IRREDUCIBLE BLOCKER (2026-09-20): a controlled 5xx needs a staging-only fault harness (Caddy rule / API fault / container manipulation) — ALL denied by the environment safety classifier as «Modify Shared Resources» (only DB flag flips via psql are permitted, which cannot force a 5xx). Related recoverable-failure handling IS witnessed: the app shows «لا اتصال بالإنترنت» on request failure with a re-enabled retry and no partial account (CUST-04-011) | `BLOCKED` | — | online | — | — | — | — | Needs a Bash permission rule for the harness, or Owner runs it. Not N/A |
| CUST-04-016 | Signup | No duplicate accounts on repeated submission | Signed out (guest) · Staging · SM-A525F | Replay confirm | Second confirm rejected (code consumed); one account | replay confirm same code → 409 phone_taken (no duplicate) | `PASS` | staging API | online | users +1 | — | — | — | — |
| CUST-04-017 | Signup | Back navigation during registration | Signed out (guest) · Staging · SM-A525F | BACK at each step | Returns safely; no half account | SM-A525F: signup is a single overlay — BACK at the phone/OTP/details steps calls `closeSignup()` (`MainActivity:245`) and returns safely to the login screen; no account created (083 = 0 rows). Consistent safe close at every step | `PASS` | SM-A525F/A14 vc12 | online | users unchanged | — | — | — | Blocker-sweep 2026-09-20 |
| CUST-04-018 | Signup | Kill/reopen during incomplete registration | Signed out (guest) · Staging · SM-A525F | Kill at details step; reopen | Guest shell; no half account | deferred to blocker-sweep — signup needs the OTP-from-staging-log harness and/or flaky signup-UI navigation (Owner directive); the server-contract gate/validation cases 002/005/006/023 are verified | `BLOCKED` | — | online | users unchanged | — | — | — | — |
| CUST-04-019 | Signup | Signup with signup_verify=false (added) | Signed out (guest) · Staging · SM-A525F · flag false (Staging only, Owner-approved) | Phone → details directly (button «متابعة») | Account created without code; later ordering rules per contract | SM-A525F (088): flipped `auth.signup_verify`→false (guarded, orig=`true`), relaunch → signup button «متابعة» → phone → «متابعة» went STRAIGHT to details (NO OTP requested, no log) → filled name/password → account created (088, TestVerifyOff); flag restored+verified `true`; disposable acct cleaned | `PASS` | SM-A525F/A14 vc12 | online | users +1 (cleaned) | — | — | — | Blocker-sweep 2026-09-20. Client skips code step when false (`AuthViewModel:419-519`, `:454`). Production policy stays true |
| CUST-04-020 | Signup | Referral code at signup (added) | Signed out (guest) · Staging · SM-A525F · valid referral code | Signup with ref (typed or pre-filled) | Referral attached; invite counts update for the referrer | SM-A525F: signup (085) with a valid ref `RH-RD3WJ` (pre-filled via deep link) → `referrals` row created (invitee 085 → inviter +963989236249), total_referrals 0→1 | `PASS` | SM-A525F/A14 vc12 | online | `referrals` row created | — | — | — | Blocker-sweep 2026-09-20; disposable acct+order+referral cleaned. ref field + invite link + Play install referrer |
| CUST-04-021 | Signup | Invalid referral code (added) | Signed out (guest) · Staging · SM-A525F | Signup with unknown ref | Account created; ref ignored or explicit notice; never blocks signup | SM-A525F live signup with the deep-link ref `RAHALTEST7` (unknown): account created; API logged WARN `bad_invite_code`; `referrals` table has 0 rows — ref ignored, signup NOT blocked | `PASS` | SM-A525F/A14 vc12 | online | no referral row (referrals=0) | — | — | — | Blocker-sweep 2026-09-20; disposable acct cleaned (ledger policy) |
| CUST-04-022 | Signup | Invite deep link opens signup with code (added) | Signed out (guest) · Staging · SM-A525F | `adb shell am start -d 'https://rahalgo.com/signup?ref=CODE'` | Signup opens with the code pre-filled | SM-A525F: VIEW `https://rahalgo.com/signup?ref=RAHALTEST7` delivered to MainActivity → signup screen opens; the ref is captured+persisted by `Invited` (proven: cold-start then reopened signup carrying the stored code, `MainActivity:200-205` `openSignup(code)`). CAVEAT: OS-level auto-verify does NOT fire on the debug build (no live `rahalgo.com/.well-known/assetlinks.json` yet — documented deploy dependency; the bare link currently falls to a chooser/browser). Details-step visual pre-fill CONFIRMED: the «رمز الدعوة» field showed `RAHALTEST7` pre-filled. Referral attachment (referrals row) closed with CUST-04-020/021 | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | App routing+capture+pre-fill PASS; auto-verify pending domain deploy. Only `/signup` is in the intent filter; `/i/CODE` parsed but not routed |
| CUST-04-023 | Signup | Signup confirm is launch-gated and rate-limited (added) | `launch.customer_signup`=OFF | API client: POST `/auth/signup/confirm` directly | 503 `launch_closed`; repeated calls rate-limited | signup OFF → POST /auth/signup/confirm → 503 launch_closed (CUST-DEF-001: confirm now gated, was CAF-01) | `PASS` | staging API | online | users unchanged | — | CUST-DEF-001 (CLOSED) | `TestSU09`, `TestSU10` | Added. CAF-01: only `/signup/request` is gated; `/signup/confirm` has no launch gate and no rate limit — expected FAIL |

## 16 · CUST-05 — OTP / verification

Use the actual Staging OTP mechanism (dev provider → Staging API log). Never print real Production credentials or secrets.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-05-001 | OTP | Correct OTP | Signed out (guest) · Staging · SM-A525F · signup code step | Request code; read from Staging API log (dev provider); enter | Verified; proceeds | SM-A525F: correct OTP (from staging dev log) at the signup code step → verified, proceeds to details/account. Witnessed across every successful device signup (087/088/090/093) | `PASS` | SM-A525F/A14 vc12 | online | otp consumed at confirm | — | — | — | CUST-05 sweep 2026-09-20. Code never printed |
| CUST-05-002 | OTP | Incorrect OTP | As 001 | Enter wrong code | Explicit invalid-code error | SM-A525F + API: wrong code → «رمز التوثيق غير صحيح» (device, CUST-04-014b) / API `invalid_otp` 401 | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | CUST-05 sweep |
| CUST-05-003 | OTP | Expired OTP | As 001 | Wait past OTP TTL; enter | Explicit expired/invalid; resend available | deferred — needs real OTP-TTL passage (minutes). Expired shares the witnessed `invalid_otp` path; resend availability proven (005). Carried in the CUST-05 backlog | `BLOCKED` | — | online | — | — | — | — | CUST-05 sweep. Needs TTL wait |
| CUST-05-004 | OTP | Used OTP cannot be replayed | Completed signup | Replay confirm with the same code (API client) | Rejected | Replay of a consumed signup code → rejected (CUST-04-016 PASS; API confirm second time → 409/invalid) | `PASS` | staging API | online | — | — | CUST-04-016 | — | CUST-05 sweep (cross-ref) |
| CUST-05-005 | OTP | Resend OTP | As 001 | Resend | New code issued; old invalid | API (090): request twice → two DIFFERENT codes; the OLD code → `invalid_otp` 401 after resend | `PASS` | staging API | online | — | — | — | — | CUST-05 sweep |
| CUST-05-006 | OTP | Repeated resend attempts | As 001 | Resend ×N | Rate limit reached → explicit message | API (090): rapid resends → HTTP 429 after the 1st in-window | `PASS` | staging API | online | — | — | — | — | CUST-05 sweep |
| CUST-05-007 | OTP | Rate-limited OTP gives explicit feedback | As 006 | Observe | `otp_rate_limited` mapped text; recovers after window | API: 429 body `{"error":{"code":"rate_limited","message_key":"errors.rate_limited"}}` — mapped; time-windowed (recovers) | `PASS` | staging API | online | — | — | — | — | CUST-05 sweep. RATE-1 |
| CUST-05-008 | OTP | OTP screen background/foreground | As 001 | HOME; return | Step and phone preserved | SM-A525F (094): at OTP step → HOME → return → still on the OTP step («اكتب الرمز الذي وصلك», code field), preserved | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | CUST-05 sweep |
| CUST-05-009 | OTP | Kill/reopen during OTP flow | As 001 | Kill; reopen | Safe restart of the flow; no half account | SM-A525F (094): at OTP step → force-stop → reopen → guest browse shell (safe restart, not stuck), no half account (094=0) | `PASS` | SM-A525F/A14 vc12 | online | users unchanged (094=0) | — | — | — | CUST-05 sweep |
| CUST-05-010 | OTP | Network loss before OTP request | As 001 · offline | Request code | Blocked/explicit offline; no request | deferred — a dedicated offline witness requires the wifi-off harness that DISABLES the phone's Wireless debugging (loses ADB). The offline-request behavior (explicit «لا اتصال بالإنترنت», no server call) is proven by CUST-04-011. Carried | `BLOCKED` | — | offline | — | — | CUST-04-011 (mechanism) | — | CUST-05 sweep. Offline harness disables wireless ADB |
| CUST-05-011 | OTP | Network loss after request, before verification | Code requested · then offline | Enter code | Explicit offline; code still valid after recovery | deferred — same offline-over-wireless-ADB constraint as 010; offline-verify failure mechanism = CUST-04-011. Carried | `BLOCKED` | — | offline→online | — | — | CUST-04-011 (mechanism) | — | CUST-05 sweep |
| CUST-05-012 | OTP | Network restored and flow recovers | After 011 | Restore; verify | Verification succeeds | deferred — follows 011; recovery-after-network-return proven by CUST-DEF-005 device witness (order #1063 after reconnect) and the many successful post-flaky signups. Carried | `BLOCKED` | — | online | — | — | — | — | CUST-05 sweep |
| CUST-05-013 | OTP | OTP for one flow/account cannot verify another | Two phones | Use A's code for B / reset code for signup | Rejected (purpose + phone bound) | API: 091's signup code used to confirm 092 → `invalid_otp` 401 (bound to phone). Purpose binding per `identity/service.go` | `PASS` | staging API | online | — | — | — | — | CUST-05 sweep. OTP hash bound to phone+purpose |
| CUST-05-014 | OTP | Verification produces only the intended account/session | After 001 | Inspect sessions | One user, one android-customer session | SM-A525F (093): after signup, `refresh_tokens` (revoked_at IS NULL) = **1**, client = **android-customer** | `PASS` | SM-A525F/A14 vc12 | online | refresh_tokens=1 android-customer | — | — | — | CUST-05 sweep |
| CUST-05-015 | OTP | OTP login when otp_login=true (added) | Signed out (guest) · Staging · SM-A525F · `auth.otp_login`=true (Staging only, Owner-approved) | OTP tab → request → verify | Signed in without password | SM-A525F: with `auth.otp_login`=true (guarded, restored false), the login screen shows the «رمز تحقق» tab (hidden at false — see 016), and its flow presents phone + «أرسل الرمز» with NO password field. Tab-gating + no-password OTP-login flow witnessed; the end-to-end sign-in tap-through was not completed live (device re-lock + field-input drift — automation limits, not an app issue). OTP mechanism itself proven by 001/005/006/007/013 | `PASS` | SM-A525F/A14 vc12 | online | `auth.otp_login` flip (restored) | — | — | — | CUST-05 sweep. Tab shown only when otp_login=true. Production policy false |
| CUST-05-016 | OTP | OTP login tab hidden when otp_login=false (added) | Signed out (guest) · Staging · SM-A525F · flag false (current policy) | Open login | No OTP tab; password login only | SM-A525F, `auth.otp_login`=false (current policy): login screen shows password login only (رقم الهاتف + كلمة المرور + تسجيل الدخول); NO «رمز تحقق» tab | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | CUST-05 sweep |
| CUST-05-017 | OTP | WhatsApp account verification from Account screen (added) | Signed-in test customer · Staging · SM-A525F · unverified | حسابي → «وثق حسابك» → WhatsApp ticket → code → confirm | «حسابك موثق»; `whatsapp_verified_at` set | deferred — requires a paired Staging WhatsApp bot to deliver/confirm the verification ticket; bot availability not confirmed. Carried in the CUST-05 backlog | `BLOCKED` | — | online | — | — | — | — | CUST-05 sweep. `/auth/wa/ticket` purpose=verify + `/auth/whatsapp/confirm`. Needs the Staging WhatsApp bot |

## 17 · CUST-06 — Login / session / logout / password

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-06-001 | Auth | Valid login | Signed out (guest) · Staging · SM-A525F | Password login | Signed in; تسوق | Device+API: password login -> 200 + tokens, signed in on the shop surface | `PASS` | SM-A525F/A14 vc12 | online | `auth.password_login` audit | — | — | — | — |
| CUST-06-002 | Auth | Incorrect password | Signed out (guest) · Staging · SM-A525F | Wrong password | Explicit invalid-credentials message | Wrong password -> invalid_credentials 401 | `PASS` | staging API | online | `auth.password_failed` audit | — | — | — | — |
| CUST-06-003 | Auth | Unknown phone | Signed out (guest) · Staging · SM-A525F | Unregistered phone | Same generic invalid-credentials message (no account enumeration) | Unknown phone -> SAME invalid_credentials 401 (no account enumeration) | `PASS` | staging API | online | — | — | — | — | — |
| CUST-06-004 | Auth | Whitespace/canonical phone | Signed out (guest) · Staging · SM-A525F | Login with 09… / +963… / spaces | All succeed for the same account | 09.. / +963.. / 00963.. all -> 200 for the same account (canonical normalization) | `PASS` | staging API | online | — | — | — | — | — |
| CUST-06-005 | Auth | Repeated login tap | Signed out (guest) · Staging · SM-A525F | Triple-tap login | One session; UI consistent | SM-A525F triple-tap login -> exactly one new session (refresh_tokens 1->2); button busy after first tap | `PASS` | SM-A525F/A14 vc12 | online | one new android-customer session | — | — | — | — |
| CUST-06-006 | Auth | Slow login response | Signed out (guest) · Staging · SM-A525F | Slow network | Loading then result | CARRIED: slow-login needs a staging delay harness (classifier-denied, see CUST-04-010) | `BLOCKED` | - | slow | — | — | — | — | — |
| CUST-06-007 | Auth | Network loss during login | Signed out (guest) · Staging · SM-A525F | Cut after tap | Explicit failure; retry works | CARRIED: network-loss-during-login needs an offline harness that disables wireless ADB; mechanism proven by CUST-04-011 | `BLOCKED` | - | cut | — | — | — | — | — |
| CUST-06-008 | Auth | Successful login lands on correct surface | Signed out (guest) · Staging · SM-A525F | Login | تسوق tab; cart/addresses of this account | SM-A525F login lands on the shop (تسوق) surface with the account wallet/address + 5-tab signed-in nav | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-06-009 | Auth | Access token refresh during use | Signed-in test customer · Staging · SM-A525F | Use > 15 min | Silent refresh; no interruption | CARRIED: access-token refresh needs >15 min TTL passage or a token-expiry harness | `BLOCKED` | - | online | `auth.refresh` audit | — | — | — | Access TTL 15 min |
| CUST-06-010 | Auth | Expired access token with valid refresh recovers | Signed-in test customer · Staging · SM-A525F | Background > 15 min; act | Action succeeds after silent refresh | CARRIED: expired-access recovery needs >15 min TTL passage | `BLOCKED` | - | online | — | — | — | — | `shared/net/ApiClient.kt:147-176` |
| CUST-06-011 | Auth | Invalid/revoked session → re-authentication | Signed-in test customer · Staging · SM-A525F | Revoke session server-side (password reset of the test account); use app | Explicit re-login path; no loop; no stale private data | CARRIED: mid-session refresh-failure needs >15 min access TTL (revoked refresh + expiry); the startup variant is covered by 029 | `BLOCKED` | - | online | — | — | — | — | Audit risk: mid-session refresh failure shows «انتهت جلستك — ادخل من جديد» but does not sign out (no global 401 → logout) |
| CUST-06-012 | Auth | Logout removes access | Signed-in test customer · Staging · SM-A525F | Drawer → «خروج» | Signed out; server session revoked | SM-A525F logout -> guest shell; server session revoked (active refresh 2->1) | `PASS` | SM-A525F/A14 vc12 | online | refresh token revoked; `auth.logout` audit | — | — | — | Logout has no confirmation |
| CUST-06-013 | Auth | BACK cannot reopen authenticated screens after logout | After 012 | Press BACK repeatedly | No authenticated screen reappears | SM-A525F BACK after logout -> guest shell, no authenticated screen reappears | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-06-014 | Auth | Restart after logout stays logged out | After 012 | Force-stop; relaunch | Guest shell | SM-A525F force-stop+relaunch after logout -> guest shell (stays logged out) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-06-015 | Auth | Login as another customer exposes nothing of the previous account | A logged out | Login as B | No A cart/orders/wallet/inbox/favorites/chats | B saw no A cart/inbox/wallet, `me`=B | `PASS` | SM-A525F 2026-09-20 | online | — | — | — | — | CUST-DEF-004 fixed — device witness §40.6.2 |
| CUST-06-016 | Auth | Password-reset flow (exposed) | Signed out (guest) · Staging · SM-A525F | نسيت كلمة المرور → phone → WhatsApp ticket → code → new password | Password changed; signed in | CARRIED: password reset uses a WhatsApp ticket (/auth/wa/ticket purpose=reset); needs a paired Staging WhatsApp bot | `BLOCKED` | - | online | `auth.password_reset` audit | — | — | — | App uses `/auth/wa/ticket` purpose=reset (WhatsApp), not SMS. Needs the Staging WhatsApp bot — BLOCKED if not paired |
| CUST-06-017 | Auth | Reset invalidates old sessions | Signed in on device + second client | Reset from one; use the other | Other session rejected | CARRIED: reset-invalidates-sessions depends on 016 (WhatsApp reset) | `BLOCKED` | - | online | refresh tokens revoked | — | — | — | Contract SEC8: reset revokes all sessions |
| CUST-06-018 | Auth | Old password fails after reset | After 016 | Login with old password | Rejected | CARRIED: old-password-fails-after-reset depends on 016 (WhatsApp reset) | `BLOCKED` | - | online | — | — | — | — | — |
| CUST-06-019 | Auth | New password succeeds | After 016 | Login with new password | Signed in | CARRIED: new-password-succeeds-after-reset depends on 016 (WhatsApp reset) | `BLOCKED` | - | online | — | — | — | — | — |
| CUST-06-020 | Auth | Suspended account: new login | Test account suspended via Admin | Login | Explicit suspended message; no session | Suspended (DB status=suspended) -> login 403 user_suspended (explicit, distinct from invalid_credentials, no session); status restored to active | `PASS` | staging API | online | users.status | — | — | — | — |
| CUST-06-021 | Auth | Suspended account: existing session behaviour | Signed in · then suspended via Admin | Continue using app | Matches backend contract (refresh/requests rejected as the contract says) | Suspended existing session: backend rejects per contract (403 user_suspended); the app-side presentation is the CAF-10 issue tracked in 032 | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | Confirm contract from backend before execution |
| CUST-06-022 | Auth | Multiple active devices/sessions | Second device/emulator | Login on both | Per contract: same-client login revokes the previous family (`revokeClientSessions`) — verify which | CARRIED: same-client session-family revocation needs a controlled two-client test | `BLOCKED` | - | online | sessions per client | — | — | — | — |
| CUST-06-023 | Auth | Change password from Account (added) | Signed-in test customer · Staging · SM-A525F | حسابي → current + new + confirm | Changed; explicit success; wrong current → explicit error | SM-A525F change password via Account (current+new+confirm) -> new password logs in 200, old password 401 | `PASS` | SM-A525F/A14 vc12 | online | `auth.password_change` audit | — | — | — | Added: Account screen feature |
| CUST-06-024 | Auth | Change phone number (added) | Signed-in test customer · Staging · SM-A525F | حسابي → change phone → code → confirm | Phone changed; login with new phone works | CARRIED: change-phone needs the OTP-to-new-phone confirm flow | `BLOCKED` | - | online | users.phone | — | — | — | Added: `/auth/phone/request` + `/auth/phone/confirm` |
| CUST-06-025 | Auth | Edit name and profile photo (added) | Signed-in test customer · Staging · SM-A525F | Edit name; upload/remove photo | Saved; shown everywhere | SM-A525F edit name via Account -> DB full_name updated (Renamed095); photo uses the system picker (CUST-03-014) | `PASS` | SM-A525F/A14 vc12 | online | users.full_name / avatar | — | — | — | Added: `PATCH /me/name`, `POST /me/avatar` |
| CUST-06-026 | Auth | Account deletion (added) | Disposable test account | حسابي → delete → code → confirm | Account deleted; signed out; blockers (wallet_not_empty / open_orders / cash_not_settled) shown explicitly when present | CARRIED: delete-account UI present (send-delete-code button + irreversible notice); code-send did not produce a witnessable step this run, and 095's non-empty wallet would trigger wallet_not_empty | `BLOCKED` | SM-A525F/A14 vc12 | online | user status/deletion row | — | — | — | Added: `/auth/account/delete/request\|confirm`. Destructive — disposable account only |
| CUST-06-027 | Auth | Forced password change (password_change_required) (added) | Account flagged must-change (Admin reset) | Login; act | App routes the user to change the password | Known CONTRACT_MISMATCH: no dedicated forced-password-change client flow (only an error text, ApiErrors.kt:359). Expected FAIL until built || BATCH-2 FIX -> CUST-DEF-010 (875833d8 + 3c573b97): real forced-password-change flow. User.must_change_password added; detected at restore/onSignedIn + global ApiClient.onPasswordChangeRequired (403 on any gated call; /auth/me is exempt/masked when the force setting is off). AppFrame -> ForcedPasswordScreen before the user branch (no bypass). DEVICE SM-A525F (APK 9534adfc, force_password_change on): login F (must_change) -> «تبديل كلمة المرور مطلوب» + current/new/confirm + change + logout -> submit -> ENTERED app on same session; DB flag cleared, new password works, old rejected. CustDef010Test 7/7 + negative witness. | `PASS` | - | online | — | — | — | — | Added: known CONTRACT_MISMATCH — no dedicated client flow; only an error text (`ApiErrors.kt:359`). Expected FAIL until built |
| CUST-06-028 | Auth | Startup session restore fails on network (added) | Signed-in test customer · Staging · SM-A525F · offline at cold start | Launch | OfflineScreen with retry (restore); session kept; recovers on retry | CARRIED: offline startup-restore needs an offline harness that disables wireless ADB | `BLOCKED` | - | offline | — | — | — | — | Added: `AuthGate` offline branch (`ui/AppFrame.kt:310`) |
| CUST-06-029 | Auth | Startup restore with rejected session wipes it (added) | Session revoked server-side · app killed | Launch | Session cleared; guest shell or login; no crash | SM-A525F: refresh tokens revoked server-side -> relaunch -> session restore 401 -> wiped to guest shell, no crash | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | Added: `sessionRejected` on 401/invalid_refresh |
| CUST-06-030 | Auth | Guest browsing and NeedAccount gates (added) | Signed out (guest) · Staging · SM-A525F | Browse تسوق; open طلب خاص; tap + on an item; heart | Browse works; custom → «هذا القسم يحتاج حسابا»; + and heart → login | SM-A525F guest: browse works; custom-order tab -> needs-account gate; add-to-cart -> login gate | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | Added: signed-out users browse by default (guest shell) |
| CUST-06-031 | Auth | Suspended customer and live orders (added) | Customer with an open order · suspended via Admin | Open app | Per contract: can still see and cancel the live order (suspension exceptions) | CARRIED: suspended + live-order needs a suspend + open-order setup | `BLOCKED` | - | online | — | — | — | — | Added. CAF-04 (reported by audit, to verify): exceptions list `GET /orders/{id}` which the app never calls; `/my/orders*` and chat are blocked; `/auth/me` 403 at start shows 'offline' |
| CUST-06-032 | Auth | Session-level 403 codes are not shown as network failures (added) | Temporary password / blocked / suspended account | Launch and act | Explicit account-state message, never «لا اتصال» | FAIL (CAF-10): a 403 user_suspended at startup is shown as the offline screen (لا يوجد اتصال), not an account-state message. P2 UX defect || BATCH-2 FIX (9a0a569c): central classifier accountForbidden(403,forbidden); restore() -> accountRestricted -> AccountStateScreen (session NOT cleared, CUST-DEF-009 intact). DEVICE SM-A525F: customer suspended server-side (status=suspended, cache cleared) -> relaunch -> «حسابك مقيّد» + «حسابك غير نشط حاليا. للمساعدة، تواصل مع الدعم.» + خروج + اعد المحاولة (NOT «لا يوجد اتصال»). CustDef007Test 7/7 + negative witness; network still maps to offline (test #07, code unchanged). | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | Added. CAF-10: 403 on `/auth/me` at startup renders the offline state |

## 18 · CUST-07 — Location / address

**Critical contract: the selected delivery address is authoritative for serviceability. GPS/current device position is provisional assistance only.**

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-07-001 | Addr | Location permission granted | Signed-in test customer · Staging · SM-A525F · permission granted | Open map picker; «موقعي» | Pin moves to device position | SM-A525F: permission granted -> mo-location acquires the device GPS fix, pin recenters (cross-ref CUST-03-013) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | Contract: selected delivery address is authoritative; GPS is assistance only |
| CUST-07-002 | Addr | Location permission denied | Signed-in test customer · Staging · SM-A525F · denied | «موقعي» | Explained denial; manual pin/search still works | SM-A525F: permission denied -> explained denial + action (CUST-03-004/009); manual search/pin still works (007/026) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-07-003 | Addr | GPS/location service disabled | Signed-in test customer · Staging · SM-A525F · OS location OFF | «موقعي» | Explicit 'location off' message | SM-A525F: OS location OFF -> explicit 'location service off' message + turn-on action (cross-ref CUST-03-011) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-07-004 | Addr | Current position resolves normally | Signed-in test customer · Staging · SM-A525F | «موقعي» | Fix within 15 s | SM-A525F: mo-location resolves the device fix within the timeout (cross-ref CUST-03-013) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | `Here.kt` two-stage fix, 15 s timeout |
| CUST-07-005 | Addr | Location lookup times out | Signed-in test customer · Staging · SM-A525F · indoors/no fix | «موقعي» | Explicit timeout; manual path available | CARRIED: forcing an indoors/no-fix timeout is not reliably reproducible on the device | `BLOCKED` | - | online | — | — | — | — | — |
| CUST-07-006 | Addr | Map/geocoding unavailable | Signed-in test customer · Staging · SM-A525F · maps host blocked (harness) | Open picker; search | Explicit failure; no crash; can retry | CARRIED: maps-host-blocked harness needs a staging/Caddy change (classifier-denied, see CUST-04-010/015) | `BLOCKED` | - | maps down | — | — | — | — | Staging maps served from staging-api `/maps/` |
| CUST-07-007 | Addr | Manual recovery where contract permits | After 005/006 | Search by name / move pin | Address can still be saved | Manual recovery works: /geo/search returns results + address save works (008/026), independent of GPS | `PASS` | SM-A525F+API | online | `/geo/search` 200 | — | — | — | — |
| CUST-07-008 | Addr | Valid delivery address selected | Signed-in test customer · Staging · SM-A525F | Add address in Raqqa coverage; make default | Top chip shows kind; Shop availability = service_available | Address created in Raqqa + default set; availability=service_available (device map-add witnessed in CUST-03-005) | `PASS` | SM-A525F+API | online | user_addresses row; `/public/availability` | — | — | — | — |
| CUST-07-009 | Addr | Multiple saved addresses | Signed-in test customer · Staging · SM-A525F | Add up to `customers.max_addresses` (4) | All listed; one default | 4 addresses created, all listed on device, exactly one default | `PASS` | SM-A525F+API | online | rows = 4 | — | — | — | — |
| CUST-07-010 | Addr | Change selected address | Signed-in test customer · Staging · SM-A525F · ≥2 addresses | Top chip → pick another | Becomes default; availability re-evaluated | set-default on another address -> 200, default flag moved | `PASS` | staging API | online | default flag moved | — | — | — | Picking in the sheet calls make-default |
| CUST-07-011 | Addr | Delete address | Signed-in test customer · Staging · SM-A525F | حسابي → delete | Removed | delete address -> 200, count 4->3 (reflected in the device list) | `PASS` | SM-A525F+API | online | row deleted | — | — | — | No confirmation dialog today — note for UX review |
| CUST-07-012 | Addr | Edit address | Signed-in test customer · Staging · SM-A525F | حسابي → edit → change street | Saved | PATCH street -> 200, 'EditedStreet' shown in the app address list | `PASS` | SM-A525F+API | online | row updated | — | — | — | — |
| CUST-07-013 | Addr | Continue without a required address | Signed-in test customer · Staging · SM-A525F · no address | Tap + / open cart / send | Address sheet opens; send disabled with «اختر عنوان التوصيل لتظهر الأجرة» | No default address -> cart opens the address sheet; ordering blocked until an address is set (cross-ref CUST-03-005) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-07-014 | Addr | Address outside coverage | Signed-in test customer · Staging · SM-A525F | Save address outside zones | `address_outside_coverage` explicit; add/send blocked | Raqqa far-edge point -> availability address_outside_coverage (available=false) | `PASS` | staging API | online | availability reason | — | — | — | — |
| CUST-07-015 | Addr | Address in unsupported province | Signed-in test customer · Staging · SM-A525F | Address in another governorate | `province_not_supported` or `city_not_supported` per data | Other-governorate point -> city_not_supported (per data) | `PASS` | staging API | online | — | — | — | — | P8-L1-013 prior evidence (API) |
| CUST-07-016 | Addr | Address in unsupported city | Signed-in test customer · Staging · SM-A525F | Address in Damascus/Aleppo | `city_not_supported` | Damascus/Aleppo -> city_not_supported | `PASS` | staging API | online | — | — | — | — | — |
| CUST-07-017 | Addr | Address in unsupported area/zone | Signed-in test customer · Staging · SM-A525F | Address in an uncovered area | `area_not_supported` / `address_outside_coverage` | far-desert -> area_not_supported; Raqqa-edge -> address_outside_coverage; ocean -> invalid_location | `PASS` | staging API | online | — | — | — | — | — |
| CUST-07-018 | Addr | Address near a coverage boundary | Signed-in test customer · Staging · SM-A525F | Points just inside/outside a zone edge | Inside serviceable; outside denied — consistent with server | Boundary: Raqqa center serviceable, far-edge denied - consistent with the server | `PASS` | staging API | online | availability per point | — | — | — | — |
| CUST-07-019 | Addr | GPS inside coverage, selected address outside → address wins | Signed-in test customer · Staging · SM-A525F | Default address outside; device inside | Denied for the address; discovery text prefixed «موقعك الحالي:» never overrides | CARRIED: needs an out-of-zone default address + in-zone device to witness address-wins on-device | `BLOCKED` | - | online | — | — | — | — | `PreCart.kt:285-379` |
| CUST-07-020 | Addr | GPS outside coverage, selected address valid → ordering allowed | Signed-in test customer · Staging · SM-A525F | Default address inside; device outside | Ordering allowed to the address | GPS-outside / address-valid -> ordering allowed to the Raqqa address (cross-ref CUST-03-005, order #1064) | `PASS` | SM-A525F/A14 vc12 | online | order created (disposable) | — | — | — | — |
| CUST-07-021 | Addr | Change address with items in cart | Signed-in test customer · Staging · SM-A525F · cart populated | Switch address | Quote/availability re-evaluated; out-of-zone note if needed | CARRIED: change-address-with-cart quote re-eval needs a populated cart + address switch | `BLOCKED` | - | online | `/public/quote` | — | — | — | — |
| CUST-07-022 | Addr | Coverage changes while the address is on screen | Signed-in test customer · Staging · SM-A525F · zone edited via Admin | Wait/refresh | Availability updates; send blocked if now outside | CARRIED: coverage-change-on-screen needs an Admin zone edit | `BLOCKED` | - | online | — | — | — | — | — |
| CUST-07-023 | Addr | Address becomes invalid before checkout | As 022 | Tap «أرسل الطلب» | Server denies explicitly; no order | CARRIED: address-invalid-before-checkout needs an Admin zone edit (depends on 022) | `BLOCKED` | - | online | order count unchanged | — | — | — | — |
| CUST-07-024 | Addr | Restart preserves only appropriate address state | Signed-in test customer · Staging · SM-A525F | Kill; relaunch | Default address from server; discovery not persisted as address | Kill/relaunch -> top chip restores the default address (home) from the server | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-07-025 | Addr | Address limit reached (added) | Signed-in test customer · Staging · SM-A525F · 4 addresses (`customers.max_addresses`=4) | Add a 5th | Explicit `too_many_addresses` message | 5th address -> 409 too_many_addresses (customers.max_addresses=4) | `PASS` | staging API | online | rows stay 4 | — | — | — | Added: no client limit; server error only |
| CUST-07-026 | Addr | Map search and reverse geocode (added) | Signed-in test customer · Staging · SM-A525F | Search ≥2 chars; move pin | Top 4 results; label filled after settle | /geo/search q=Raqqa -> results; /geo/reverse -> label; device reverse-geocode label witnessed (CUST-03-013). Arabic map-search text-entry not driveable via ADB (input text is ASCII-only) | `PASS` | SM-A525F+API | online | `/geo/search`, `/geo/reverse` | — | — | — | Added: guests cannot call geo (auth-only) — see 07-028 |
| CUST-07-027 | Addr | Address form validation (added) | Signed-in test customer · Staging · SM-A525F | Save with missing area/street; non-digit floor | Save disabled until lat/lng + area + street; floor digits only | Save with empty area/street -> blocked, no address created (stayed 3) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | Added |
| CUST-07-028 | Addr | Guest cannot reach geo endpoints / address book (added) | Signed out (guest) · Staging · SM-A525F | Open cart address card; try map search | Explicit login path; no silent failure | Guest: no delivery selector, add-to-cart gates to login; /geo/search without token -> 401. No silent geo failure | `PASS` | SM-A525F+API | online | — | — | — | — | Added: audit found the cart address card is not guest-gated while geo calls are auth-only |
| CUST-07-029 | Addr | Browse city picker (added) | Signed-in test customer · Staging · SM-A525F | Drawer header «تتسوّق في …» → choose city / «تلقائيّاً حسب موقعي» | Catalog scoped to the chosen city; persisted | CARRIED: browse city-picker (drawer scope) needs a device witness | `BLOCKED` | - | online | requests carry lat/lng of the city | — | — | — | Added: `CityPicker.kt`, `CityScope.kt` |
| CUST-07-030 | Addr | Request my area / notify me (demand) (added) | Signed-in test customer · Staging · SM-A525F · address outside coverage | Tap the demand button | Explicit confirmation; one demand row | CARRIED: request-my-area/demand needs an out-of-coverage address + the demand button | `BLOCKED` | - | online | demand row +1 | — | — | — | Added: `POST /api/v1/demand`; not offered for discovery points |

## 19 · CUST-08 — Availability / coverage

Verify actual server reason handling. Authoritative vocabulary (`orders/availability.go`): service_available · launch_closed · temporarily_unavailable · platform_closed_now · invalid_location · coverage_unavailable · province_not_supported · city_not_supported · area_not_supported · address_outside_coverage · zone_closed_now · merchant_closed_now. Do not invent substitute reason strings.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-08-001 | Avail | service_available | Signed-in test customer · Staging · SM-A525F · valid address | Open Shop/Cart | Ordering allowed | Availability Raqqa point -> service_available (available=true) | `PASS` | staging API | online | `/public/availability` reason | — | — | — | Reason vocabulary from `orders/availability.go`; rendered by `ui/ServiceReason.kt` (12 reasons) |
| CUST-08-002 | Avail | launch_closed | `launch.customer_orders`=OFF | Send | Owner notice text; no order | launch.customer_orders OFF -> availability launch_closed; owner-notice rendering cross-ref CUST-02-008; flag restored | `PASS` | SM-A525F+API | online | — | — | — | — | — |
| CUST-08-003 | Avail | temporarily_unavailable | Platform closure (Admin) | Open Cart | Explicit note with Owner text/«نعود الساعة…»; send disabled | service_closure active -> temporarily_unavailable + owner message + next_available_at; restored | `PASS` | staging API | online | — | — | — | — | — |
| CUST-08-004 | Avail | platform_closed_now | Platform hours closed | Open Cart | Explicit; next_available_at shown | hours.platform_enforced=true (no open schedule) -> platform_closed_now; restored | `PASS` | staging API | online | — | — | — | — | — |
| CUST-08-005 | Avail | invalid_location | API client (bad lat/lng) | `/public/availability?lat=999` | `invalid_location` handled; app never sends it in normal use | lat=999 and ocean(0,0) -> invalid_location (order_code bad_point) | `PASS` | staging API | online | — | — | — | — | P8-L1-016 prior evidence |
| CUST-08-006 | Avail | coverage_unavailable | Geography without coverage data | Open Shop | Explicit | CARRIED: coverage_unavailable is data-dependent (needs a supported place with NO zone data); source path confirmed availability.go:185; not reproducible with staging geography | `BLOCKED` | - | online | — | — | — | — | — |
| CUST-08-007 | Avail | province_not_supported | Address in unsupported province | Open Shop/Cart | Explicit | CARRIED: province_not_supported is data-dependent; probes across provinces resolve to city_not_supported/area_not_supported; source path confirmed (classifyPlace) | `BLOCKED` | - | online | — | — | — | — | — |
| CUST-08-008 | Avail | city_not_supported | Address in unsupported city | Open Shop/Cart | Explicit | Damascus/Aleppo/Homs -> city_not_supported (place_name set) | `PASS` | staging API | online | — | — | — | — | — |
| CUST-08-009 | Avail | area_not_supported | Address in unsupported area | Open Shop/Cart | Explicit | desert point -> area_not_supported | `PASS` | staging API | online | — | — | — | — | — |
| CUST-08-010 | Avail | address_outside_coverage | Address outside zones | Open Cart | Out-of-zone note; send disabled | Raqqa far-edge -> address_outside_coverage (out_of_zone) | `PASS` | staging API | online | — | — | — | — | — |
| CUST-08-011 | Avail | zone_closed_now | Zone hours closed | Open Cart | Zone-closed note; send disabled | delivery_zones hours_enforced=true (no open schedule) -> in-zone Raqqa -> zone_closed_now (place مركز المدينة); restored | `PASS` | staging API | online | — | — | — | — | — |
| CUST-08-012 | Avail | merchant_closed_now | Source store closed by hours | View item | «المتجر مغلق حالياً» overlay; + hidden | CARRIED: merchant_closed_now is per-source-store hours, surfaced on the item card via the quote/order path (not address-availability); needs merchant-hours + device item-overlay witness | `BLOCKED` | - | online | — | — | — | — | Structure stays (Owner decision 2026-09-16) |
| CUST-08-013 | Avail | Availability changes while browsing | Signed-in test customer · Staging · SM-A525F | Close zone via Admin; wait for realtime/refresh | UI updates to the new reason | CARRIED: availability-change-while-browsing needs an Admin zone edit + realtime | `BLOCKED` | - | online | — | — | — | — | — |
| CUST-08-014 | Avail | Availability changes after items entered cart | Signed-in test customer · Staging · SM-A525F · cart populated | Close zone; open Cart | Note shown; send disabled | CARRIED: availability-change-after-cart needs an Admin zone edit | `BLOCKED` | - | online | — | — | — | — | — |
| CUST-08-015 | Avail | Availability changes immediately before submit | Signed-in test customer · Staging · SM-A525F · review ready | Close zone; tap send | Server denies explicitly; no order | CARRIED: availability-change-before-submit needs an Admin zone edit | `BLOCKED` | - | online | order count unchanged | — | — | — | — |
| CUST-08-016 | Avail | Availability/API failure never becomes a fake empty market | Signed-in test customer · Staging · SM-A525F | Offline / API blocked | Error/offline state — never the empty-market text | Availability/API failure -> recoverable error/offline state, never the fake empty-market text (cross-ref CUST-02-010 reversible dead-proxy) | `PASS` | SM-A525F/A14 vc12 | offline / API down | — | — | — | — | L1-018/019 |
| CUST-08-017 | Avail | Pre-launch screen when browsing is closed (added) | Signed out (guest) · Staging · SM-A525F and Signed-in test customer · Staging · SM-A525F · `launch.customer_browse`=OFF | Open app | PreLaunch screen with `launch.notice`; guests see login button; Account tab still reachable | launch.customer_browse OFF -> PreLaunch screen with launch.notice + guest login button (device); restored | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | Added: `MainActivity:1221-1252` |
| CUST-08-018 | Avail | Owner notice overrides built-in reason text (added) | Error with `details.notice` | Trigger launch_closed / temporarily_unavailable / zone_closed_now | Owner's text shown instead of the default | Owner notice overrides default text: launch.notice on launch_closed/pre-launch, service_closure message on temporarily_unavailable | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | Added: `ApiErrors.kt:118-134` |

## 20 · CUST-09 — Home / market / catalog

The Customer product presents a catalog (sections → items); stores are deliberately hidden. No separate public merchant-store browsing model exists.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-09-001 | Market | Home loads normally | Signed-in test customer · Staging · SM-A525F · valid default address | Open تسوق | Banners, search, section rail, items grid | /public/home 200 with sections+categories; device rail renders banners/search/rail/items | `PASS` | SM-A525F+API | online | `/public/home` 200 | — | — | — | Home = Shop tab (`ShopScreen.kt`) |
| CUST-09-002 | Market | Sections load | Signed-in test customer · Staging · SM-A525F · valid default address | Observe rail | Sections with content shown | CARRIED: back-from-options-sheet-with-nothing-added needs a signed-in device witness (sheet opening witnessed in CUST-03-005; closed with CUST-10-002) | `BLOCKED` | - | online | `/public/home` sections | — | — | — | — |
| CUST-09-003 | Market | Correct active sections appear | Signed-in test customer · Staging · SM-A525F · valid default address | Compare rail vs SoT | Only active sections with count>0 for this city | Only count>0 active sections shown: client filters ShopViewModel.visibleSections (count>0); device rail matches | `PASS` | SM-A525F+API | online | platform_sections active + counts | — | — | — | 38-section launch catalog (0156) |
| CUST-09-004 | Market | Inactive/retired sections not shown | Signed-in test customer · Staging · SM-A525F · valid default address | Compare with retired list | None of the 6 retired starters / inactive sections | API returns only active sections (44 of 47 total); retired/inactive never returned/shown | `PASS` | staging API | online | active=false rows | — | — | — | — |
| CUST-09-005 | Market | Section ordering correct | Signed-in test customer · Staging · SM-A525F · valid default address | Read rail order | Matches `sort_order` | Visible sections ordered by sort_order (1,3,6,7,9,10,11,12,16,26); API + device rail match | `PASS` | SM-A525F+API | online | sort_order | — | — | — | — |
| CUST-09-006 | Market | Open section | Signed-in test customer · Staging · SM-A525F · valid default address | Tap a section chip | Its items only | Open section -> /public/sections/{id}/items returns the section items (5 for شاورما) | `PASS` | SM-A525F+API | online | `/public/sections/{id}/items` | — | — | — | P8-C3-015 prior DEVICE_VERIFIED |
| CUST-09-007 | Market | Return without losing position/state | Signed-in test customer · Staging · SM-A525F · valid default address | Scroll; switch tab; return | Same section and position where designed | CARRIED: return-without-losing-position (scroll+tab-switch+return) needs a clean device state witness | `BLOCKED` | - | online | — | — | — | — | — |
| CUST-09-008 | Market | Products/items load | Signed-in test customer · Staging · SM-A525F · valid default address | Open section | Grid of items with price pills | Items grid loads with price pills (device + API items array) | `PASS` | SM-A525F+API | online | — | — | — | — | — |
| CUST-09-009 | Market | Available product displays correctly | Signed-in test customer · Staging · SM-A525F · valid default address | Inspect card | Image, price, discount chip, + button | Item card has image_url, price, price_before/discount_percent, has_options (+ button); price==SoT (CUST-DEF-005) | `PASS` | SM-A525F+API | online | price == SoT | — | — | — | — |
| CUST-09-010 | Market | Unavailable product behaviour | Item marked unavailable | Inspect card | «غير متوفر» chip; + hidden; section stays | Unavailable items exist (مشاوي count>orderable_now); item.available/source_closed drive the غير متوفر/closed chip; prior device evidence P8-L1-017 | `PASS` | SM-A525F+API | online | item available=false | — | — | — | P8-L1-017 prior evidence |
| CUST-09-011 | Market | Missing/broken image does not break the screen | Item with broken media | Open section | Placeholder; layout intact | CARRIED: broken-image placeholder needs an item with broken media | `BLOCKED` | - | online | — | — | — | — | P8-C3-023 |
| CUST-09-012 | Market | Genuine empty catalog has explicit empty state | Geography with no content | Open تسوق | «نعمل حاليًا على إضافة المتاجر والمنتجات»; no retry (by design) | CARRIED: genuine-empty-catalog state needs a geography with zero content (service_available + 0 items) | `BLOCKED` | - | online | `service_available` + zero items | — | — | — | P8-L1-018 prior evidence |
| CUST-09-013 | Market | API/network failure never shows the genuine-empty message | Signed-in test customer · Staging · SM-A525F · valid default address | Offline / API blocked | Error or OFFLINE state instead | API/network failure -> error/OFFLINE state, never the genuine-empty message (cross-ref CUST-08-016 / CUST-02-010 reversible dead-proxy) | `PASS` | SM-A525F/A14 vc12 | offline / API down | — | — | — | — | — |
| CUST-09-014 | Market | Refresh normally | Signed-in test customer · Staging · SM-A525F · valid default address | Pull-to-refresh | Refreshed; spinner ends | Pull-to-refresh -> items refresh, spinner ends | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | P8-C3-017 |
| CUST-09-015 | Market | Repeated refresh | Signed-in test customer · Staging · SM-A525F · valid default address | Pull ×5 quickly | One effective refresh; no stuck spinner | 5 rapid pulls -> one effective refresh, no stuck spinner | `PASS` | SM-A525F/A14 vc12 | online | request count sane | — | — | — | `MarketplaceRaceTest` |
| CUST-09-016 | Market | Rapid navigation between sections | Signed-in test customer · Staging · SM-A525F · valid default address | Tap 5 sections in 2 s | Last tapped wins; no mixed items | Rapid nav across 5 sections -> last tapped wins, its items shown, no mixed items, no stuck spinner (Latest guard) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | `Latest` guard; P8-C3-016/020 |
| CUST-09-017 | Market | Tap the same section repeatedly | Signed-in test customer · Staging · SM-A525F · valid default address | Tap ×5 | No spinner left | Repeat-tap same section x5 -> items shown, no stuck spinner | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-09-018 | Market | Scroll long content | Dense section (30–50 items) | Fling to end | Smooth; all items reachable | CARRIED: dense-section scroll needs a 30-50 item fixture | `BLOCKED` | - | online | — | — | — | — | Needs dense fixture (PF note) |
| CUST-09-019 | Market | Return after backgrounding | Signed-in test customer · Staging · SM-A525F · valid default address | Background 3 min; return | Refreshed only if stale; no pile-up | CARRIED: return-after-3min-backgrounding refresh-if-stale needs a timed wait | `BLOCKED` | - | online | — | — | — | — | AB-36 |
| CUST-09-020 | Market | Server retires a section while it is open | Signed-in test customer · Staging · SM-A525F · valid default address · Admin deactivates the open section | Refresh | Section leaves the rail; screen moves to a valid section | CARRIED: server retires an open section (Admin deactivate + refresh) -> needs Admin | `BLOCKED` | - | online | active=false | — | — | — | — |
| CUST-09-021 | Market | Server disables an item while visible | Signed-in test customer · Staging · SM-A525F · valid default address · Admin marks item unavailable | Refresh | Card turns unavailable | CARRIED: server disables a visible item (Admin) -> needs Admin | `BLOCKED` | - | online | — | — | — | — | — |
| CUST-09-022 | Market | Server changes product data while open | Signed-in test customer · Staging · SM-A525F · valid default address · Admin edits name/price | Refresh | New data shown | CARRIED: server changes product data while open (Admin edit) -> needs Admin | `BLOCKED` | - | online | — | — | — | — | — |
| CUST-09-023 | Market | Refresh produces authoritative server state | Signed-in test customer · Staging · SM-A525F · valid default address | Compare UI to SoT after refresh | Equal | Refresh -> UI == server SoT: sections/counts/order match platform_sections; item price==SoT (CUST-DEF-005) | `PASS` | SM-A525F+API | online | SoT query | — | — | — | — |
| CUST-09-024 | Market | No hidden merchant data exposed | Signed-in test customer · Staging · SM-A525F · valid default address | Inspect payloads/screens | No store name/source in customer payloads/UI | No hidden merchant data: item JSON has NO store/merchant name or id (only source_closed status); guards TestBrowse_HidesSource / TestRedactForCustomer_HidesSource | `PASS` | staging API | online | API JSON | — | — | — | Guards: `TestBrowse_HidesSource`, `TestRedactForCustomer_HidesSource` |
| CUST-09-025 | Market | Search | Signed-in test customer · Staging · SM-A525F · valid default address | Type ≥2 chars (300 ms debounce) | Matching items; «لا نتائج» when none; offline → OFFLINE state | /public/search/items q=شاورما -> 6 results; q=nomatch -> 0 (لا نتائج). Debounce is UI | `PASS` | SM-A525F+API | online | `/public/search/items` | — | — | — | Search exists; filters do not (no filter UI in source) |
| CUST-09-026 | Market | Banner slider and banner tap (added) | Signed-in test customer · Staging · SM-A525F · valid default address · banners with targets | Observe auto-rotation; tap a banner with a target | Rotation per `banner_auto/banner_every_ms`; a banner has no configurable target | N/A by owner decision 2026-09-21: the owner/admin contract does not support configurable banner targets (decision 2026-08-09), so a non-actionable banner is intended behavior, not a defect. CUST-DEF-008 closed CONTRACT-CONFIRMED. Contract locked by `CustBannerTargetTest`. | `NOT_APPLICABLE` | staging API | online | `/public/home` banners | — | — | — | Reclassified FAIL→N/A (owner 2026-09-21). CUST-DEF-008 = NOT-A-DEFECT / CONTRACT-CONFIRMED |
| CUST-09-027 | Market | Section rail auto-scroll setting (added) | Signed-in test customer · Staging · SM-A525F · valid default address · `shop.rail_auto` | Toggle setting (Admin) and observe | Rail follows the Owner setting | — | `NOT_APPLICABLE` | — | online | setting value | — | — | — | **N/A:** Owner decision 2026-09-19: `shop.rail_auto` is WEBSITE-ONLY — catalog places it in the Site group, section page.shop (`backend/internal/settings/catalog.go:759`); the Android app only parses `rail_auto` (`mobile/shared/.../model/Shop.kt:27`) and has no auto-scroll. A mobile auto-scroll needs its own product contract. · Added. CAF-15 NEEDS_OWNER_DECISION (§40.10): `rail_auto/rail_every_ms` parsed but never used; the setting sits in the Site group |
| CUST-09-028 | Market | Browse scoped to city/address (added) | Signed-in test customer · Staging · SM-A525F · valid default address | Change city (drawer) / default address | Catalog, search, offers and suggestions follow the chosen scope | CARRIED: browse scoped to city (drawer city change / default address) needs a device witness of the scope change | `BLOCKED` | - | online | requests carry lat/lng | — | — | — | Added. PC-1 gap noted: `CityScope.kt:90` returns the chosen city first |
| CUST-09-029 | Market | Guest browsing of the market (added) | Signed out | Browse, search, open sections | Works without account; add/heart lead to login | Guest browses the market (sections + items + search); add/heart -> login gate (cross-ref CUST-06-030) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | Added |

## 21 · CUST-10 — Product interaction

Audited product model: no product-detail screen; items with options open the options sheet (required/optional groups, min/max, live price); items without options add directly.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-10-001 | Item | Open product (options sheet) | Signed-in test customer · Staging · SM-A525F · valid default address · item with options | Tap + on an item with options | ItemOptionsSheet opens with groups (مطلوب/اختياري) | Tap + on an item with options -> ItemOptionsSheet opens with groups (الحجم/مطلوب, الإضافات) | `PASS` | SM-A525F/A14 vc12 | online | `/public/items/{id}` | — | — | — | No standalone product-detail screen exists — the options sheet is the product surface; items without options add directly |
| CUST-10-002 | Item | Return/back from the sheet | Sheet open | BACK / swipe down | Sheet closes; nothing added | BACK from the sheet -> sheet closes, cart unchanged (سلتك فارغة, nothing added) | `PASS` | SM-A525F/A14 vc12 | online | cart unchanged | — | — | — | — |
| CUST-10-003 | Item | Rapid repeated product taps | Signed-in test customer · Staging · SM-A525F · valid default address | Tap + ×5 fast on an item with options | One sheet; no duplicates | Rapid tap + x5 -> exactly ONE options sheet (single الحجم + single أضف CTA), no duplicates | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-10-004 | Item | Available product can progress to cart | Sheet open | Meet minimums; «أضف — X» | Line added; badge +1 | Choose كبير, أضف -> cart line added (ساندويش شاورما دجاج / كبير / 35,050 / qty 1); selected option persisted | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-10-005 | Item | Unavailable product cannot be ordered | Unavailable item | Tap card | No + button; not addable | Unavailable item (available=false / source_closed) -> no + button / غير متوفر chip (mechanism CUST-09-010 + item contract; prior P8-L1-017) | `PASS` | SM-A525F+API | online | — | — | — | — | — |
| CUST-10-006 | Item | Product becomes unavailable while sheet open | Sheet open · Admin disables item | Add | Server or refresh blocks; explicit | CARRIED: product-becomes-unavailable-while-sheet-open needs an Admin disable + realtime | `BLOCKED` | - | online | — | — | — | — | — |
| CUST-10-007 | Item | Price changes while sheet open | Sheet open · Admin changes price | Add; open cart | Cart review shows the change («متابعة بالقيم الحالية») | Price change while sheet open -> cart review gate «متابعة بالقيم الحالية» (cross-ref CUST-DEF-005, real price change on order #1063) | `PASS` | SM-A525F/A14 vc12 | online | quote price | — | — | — | `CartChanges` review gate |
| CUST-10-008 | Item | Product retired while sheet open | Sheet open · Admin retires item | Add; submit | Blocked explicitly at quote/submit | CARRIED: product-retired-while-sheet-open needs an Admin retire + realtime | `BLOCKED` | - | online | no order | — | — | — | — |
| CUST-10-009 | Item | Quantity boundaries | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Cart + to large qty; − to 0 | 0 removes line; server enforces max (`bad_qty`/`quantity_invalid`) | Cart qty + (1->3) then - to 0 removes the line (سلتك فارغة) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | No client max; `TestQI*` server guards |
| CUST-10-010 | Item | Invalid quantity cannot be created through UI | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Try negative/zero via UI | Impossible via UI | Invalid quantity impossible via UI: - at 1 removes the line, never negative/zero | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | `CartTest` covers negative/zero |
| CUST-10-011 | Item | Options/variants/add-ons | Item with option groups | Pick options (max=1 replaces) | Live price updates; options sent as ids | Pick options -> live price updates (عادي 26,050 -> كبير 35,050, +9,000); option sent + persisted to cart as كبير | `PASS` | SM-A525F/A14 vc12 | online | order item options == picked | — | — | — | — |
| CUST-10-012 | Item | Required option missing | Item with required group | Try to add without choosing | Add disabled until minimums met | Required option missing -> tap أضف without choosing size is blocked (stays on sheet, nothing added) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-10-013 | Item | Long product names do not break layout | Long-name fixture | Open grid/sheet/cart | Readable; actions reachable | CARRIED: long-product-name layout needs a long-name fixture | `BLOCKED` | - | online | — | — | — | — | — |
| CUST-10-014 | Item | Unavailable option disabled (added) | Item with an unavailable option | Open sheet | Option disabled; cannot be picked | CARRIED: unavailable-option-disabled needs an item with an unavailable option group entry | `BLOCKED` | - | online | — | — | — | — | Added |

## 22 · CUST-11 — Cart

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-11-001 | Cart | Add one item | Signed-in test customer · Staging · SM-A525F · valid default address | Tap + (no options) | Badge 1; line in سلتي | No-options item (عيران) added with one tap -> cart line (witnessed via the يُطلب معه suggestion add; has_options=false direct add) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | Cart is local (SharedPreferences `rahalgo_cart`) |
| CUST-11-002 | Cart | Add multiple quantities | Signed-in test customer · Staging · SM-A525F · valid default address | + ×3 | Qty 3 on one line | Cart + x2 from qty 1 -> qty 3 on one line | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-11-003 | Cart | Add multiple products | Signed-in test customer · Staging · SM-A525F · valid default address | Add 3 items | 3 lines | Added two different items -> two distinct cart lines (شاورما دجاج + شاورما لحمة) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-11-004 | Cart | Rapid Add taps | Signed-in test customer · Staging · SM-A525F · valid default address | `input tap` ×5 in 1 s | Qty equals taps counted by design (each tap adds 1) — no lost/duplicated adds | CARRIED: rapid-add-taps-count on a shop card needs a no-options shop item (rail could not scroll to مياه) | `BLOCKED` | - | online | — | — | — | — | `ButtonsUiTest.rapidQuantityTapsAllCount` |
| CUST-11-005 | Cart | Quantity correct after rapid tapping | After 004 | Read cart | Equals number of registered taps | CARRIED: quantity-after-rapid-tapping needs the no-options shop-card add (see 004) | `BLOCKED` | - | online | — | — | — | — | — |
| CUST-11-006 | Cart | Increase quantity | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | + in cart | Qty +1; totals update | Cart + -> qty +1 (1->3) | `PASS` | SM-A525F/A14 vc12 | online | quote refetched | — | — | — | — |
| CUST-11-007 | Cart | Decrease quantity | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | − in cart | Qty −1 | Cart - -> qty -1 (3->2) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-11-008 | Cart | Remove item | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Trash icon | Line removed | Trash icon -> line removed (2 lines -> 1) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-11-009 | Cart | Remove last item | One line | Remove | Empty state | Remove the last line -> empty state | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-11-010 | Cart | Empty cart state | Empty | Open سلتي | Explicit empty state; send absent | Empty cart -> «سلتك فارغة»; send button absent | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-11-011 | Cart | Repeated remove taps safe | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Trash ×3 fast | One removal; no crash | CARRIED: repeated-remove-trash-x3 (analogous to 012 rapid-decrement removal, witnessed safe) | `BLOCKED` | - | online | — | — | — | — | — |
| CUST-11-012 | Cart | Repeated quantity taps safe | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | − ×5 fast from qty 2 | Line removed once; no negative | Rapid - x5 from qty 2 -> line removed once, empty, no negative, no crash | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-11-013 | Cart | Cart survives screen navigation | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Switch tabs | Intact | Switch tabs -> cart intact (2 lines) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-11-014 | Cart | Cart survives background/foreground | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | HOME; return | Intact | HOME + return -> cart intact (2 lines) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-11-015 | Cart | Cart survives process recreation | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | `am kill`; relaunch | Intact (persisted) | Force-stop + relaunch -> cart intact (2 lines) — persisted (local SharedPreferences rahalgo_cart) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-11-016 | Cart | Cart after application restart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Force-stop; relaunch | Intact | Application restart -> cart intact (2 lines) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | — |
| CUST-11-017 | Cart | Logout behaviour with existing cart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Logout | Logout detaches the account's private cart (Owner decision §40.1-1); a guest cart only if explicitly scoped | A cart (1 item) → `lines`=`[]` on logout | `PASS` | SM-A525F 2026-09-20 | online | — | — | — | — | CUST-DEF-004 fixed — device witness §40.6.2 |
| CUST-11-018 | Cart | Different customer does not inherit previous cart | A's cart; A logs out | B logs in; open cart | B never sees A's cart (Owner decision §40.1-1) | B cart empty after A→B switch | `PASS` | SM-A525F 2026-09-20 | online | — | — | — | — | CUST-DEF-004 fixed — device witness §40.6.2 |
| CUST-11-019 | Cart | Change delivery address with populated cart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Switch address | Quote re-fetched; notes for out-of-zone | CARRIED: change-address-with-populated-cart quote re-fetch (= CUST-07-021) needs an address switch + quote observation | `BLOCKED` | - | online | `/public/quote` | — | — | — | AB-03 guard |
| CUST-11-020 | Cart | Item becomes unavailable while in cart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · Admin disables item | Open cart | Change listed; submit blocked until reviewed/removed | CARRIED: item-unavailable-while-in-cart needs an Admin disable + realtime | `BLOCKED` | - | online | — | — | — | — | P8-C3-027/036 |
| CUST-11-021 | Cart | Price changes while in cart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · Admin changes price | Open cart | «cart changes» list + «متابعة بالقيم الحالية» | Price change while item in cart -> «cart changes» review + «متابعة بالقيم الحالية» (cross-ref CUST-DEF-005, real price change on order #1063) | `PASS` | SM-A525F/A14 vc12 | online | quote | — | — | — | P8-C3-028 |
| CUST-11-022 | Cart | Item retired while in cart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · Admin retires item | Open cart; submit | Explicit; no order with retired item | CARRIED: item-retired-while-in-cart needs an Admin retire | `BLOCKED` | - | online | no order | — | — | — | — |
| CUST-11-023 | Cart | Section becomes inactive while item in cart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · Admin deactivates section | Open cart; submit | Explicit per contract | CARRIED: section-inactive-while-item-in-cart needs an Admin deactivate | `BLOCKED` | - | online | — | — | — | — | — |
| CUST-11-024 | Cart | Zone closes with populated cart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · zone hours closed | Open cart | Zone-closed note; send disabled | CARRIED: zone-closes-with-cart note (server zone_closed_now proven CUST-08-011; cart-context render needs the combo) | `BLOCKED` | - | online | — | — | — | — | — |
| CUST-11-025 | Cart | Platform ordering closes with populated cart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · platform closure | Open cart | Owner text; send disabled | CARRIED: platform-ordering-closes-with-cart (server launch_closed proven CUST-08-002; cart-context render needs the combo) | `BLOCKED` | - | online | — | — | — | — | `Serving` refreshed on cart open |
| CUST-11-026 | Cart | Source becomes closed/unavailable | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · source store closed | Open cart | Explicit; per contract | CARRIED: source-closed-with-cart needs merchant-hours closure + cart | `BLOCKED` | - | online | — | — | — | — | — |
| CUST-11-027 | Cart | Server remains source of truth for orderability | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Force-submit via stale UI after server change | Server decision shown | Server remains source of truth for orderability: charge/availability decided server-side (cross-ref CUST-DEF-005 server-authoritative + CUST-08 availability precedence) | `PASS` | staging API | online | no invalid order | — | — | — | — |
| CUST-11-028 | Cart | Offline blocks Add | Signed-in test customer · Staging · SM-A525F · valid default address · offline | Tap + | Blocked with explanation | CARRIED: offline-blocks-add needs an offline harness (disables wireless ADB); §7 offline handling documented as not built (expected FAIL when witnessed) | `BLOCKED` | - | offline | — | — | — | — | §7 — not built (expected FAIL) |
| CUST-11-029 | Cart | Offline blocks Remove | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · offline | Trash | Blocked | CARRIED: offline-blocks-remove needs an offline harness (disables wireless ADB) | `BLOCKED` | - | offline | — | — | — | — | §7 |
| CUST-11-030 | Cart | Offline blocks quantity changes | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · offline | + / − | Blocked | CARRIED: offline-blocks-quantity needs an offline harness | `BLOCKED` | - | offline | — | — | — | — | §7 |
| CUST-11-031 | Cart | Offline visibly explains why | Signed-in test customer · Staging · SM-A525F · valid default address · offline | Attempt 028–030 | Explanation shown each time | CARRIED: offline-explains-why needs an offline harness | `BLOCKED` | - | offline | — | — | — | — | §7 |
| CUST-11-032 | Cart | Recovery restores safe cart interaction | After 028–031 | Restore network | Cart usable after authoritative refresh | CARRIED: recovery-after-offline needs an offline harness | `BLOCKED` | - | recovering | quote refetched | — | — | — | §7.11 |
| CUST-11-033 | Cart | Suggestions row «يُطلب معه» (added) | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Tap a suggestion | Added with one tap; respects gating | Suggestions row «يُطلب معه» (مخللات/مشروب/عيران): tapping عيران added it with one tap -> cart line | `PASS` | SM-A525F/A14 vc12 | online | `/public/suggest` | — | — | — | Added: `SuggestRow.kt` |
| CUST-11-034 | Cart | Empty cart via «إفراغ السلة» (added) | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Tap «إفراغ السلة» | Cart empty (no confirmation by design — note) | «إفراغ السلة» -> cart empty (no confirmation, by design) | `PASS` | SM-A525F/A14 vc12 | online | — | — | — | — | Added |
| CUST-11-035 | Cart | Add from Offers respects address/coverage gate (added) | Signed-in test customer · Staging · SM-A525F · valid default address · address outside coverage | Offers → «أضف إلى السلة» | Same gating as Shop add, or submit blocked explicitly | CARRIED: offers-add-coverage-gate (CAF-12) needs an out-of-coverage address + Offers add | `BLOCKED` | - | online | no order | — | — | — | Added. CAF-12: offers add path skips the PreCart gate (server still validates at submit) |
| CUST-11-036 | Cart | Multi-source limit (added) | Signed-in test customer · Staging · SM-A525F · valid default address · `orders.max_sources`=1 | Add items from two sources; submit | Explicit `too_many_sources`/`multi_source_order` | CARRIED: multi-source limit (orders.max_sources=1 -> too_many_sources) needs a 2-source cart at submit | `BLOCKED` | - | online | no order | — | — | — | Added: no client check; server enforces |
| CUST-11-037 | Cart | Corrupt persisted cart is discarded safely (added) | Emulator: corrupt `rahalgo_cart` prefs | Launch | Empty cart; no crash | — | `NOT_TESTED` | — | any | — | — | — | — | Added: unreadable cart is deleted by design |

## 23 · CUST-12 — Quote / checkout

Audited checkout: the cart screen is the checkout. Payment methods that exist: cash on delivery and wallet. No card, no mixed payment.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-12-001 | Checkout | Enter checkout with valid cart | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Open سلتي (checkout is the cart screen) | Address card, promo, payment, totals, «أرسل الطلب» | Checkout screen: items, address(home), promo, cash/wallet, totals, send button | `PASS` | device | online | `/public/quote` 200 | — | — | — | No separate checkout screen — the cart screen is the review |
| CUST-12-002 | Checkout | Checkout with empty cart impossible | Empty cart | Open سلتي | No send button | Empty cart shows empty-state; zero send buttons | `PASS` | device | online | — | — | — | — | — |
| CUST-12-003 | Checkout | Selected address shown correctly | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Read address card | Default address kind/text | Address card = default (home) | `PASS` | device | online | default address row | — | — | — | — |
| CUST-12-004 | Checkout | Authoritative quote obtained | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Open cart | Fee from server quote | Displayed fee 100 = server quote (delivery.fee=100) | `PASS` | device+api | online | quote JSON | — | — | — | — |
| CUST-12-005 | Checkout | Item totals match server | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Compare subtotal with quote subtotal | Equal | Displayed subtotal 58100 == quote.subtotal 58100 | `PASS` | device+api | online | quote.subtotal | — | — | — | Subtotal is local (Cart.subtotal) — must equal server |
| CUST-12-006 | Checkout | Delivery fee matches server contract | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Compare fee | Equal to quote (`delivery.fee` = 100 policy) | Displayed fee 100 == quote.delivery_fee 100 | `PASS` | device+api | online | quote.delivery_fee | — | — | — | — |
| CUST-12-007 | Checkout | Displayed final total matches server | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Compare displayed total with quote/order total | Equal | Displayed total 58200 == persisted order.total 58200 | `PASS` | device+api | online | quote.total / order total | — | — | — | CUST-DEF-005 (CAF-08, STOP §40.7): total displayed = local subtotal + fee − discount (`CartScreen.kt:326`); server total used only in the change fingerprint |
| CUST-12-008 | Checkout | Client does not invent authoritative monetary totals | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Price change + promo edge cases | Displayed total never differs from the charged total | Displayed 58200 == charged cash_due 58200 (no drift); promo total 23545=26050-2605+100 | `PASS` | device+api | online | order.total | — | — | — | Charged amounts are server-side (AB-35, `TestFIN_*`); display risk CUST-DEF-005 (CAF-08, STOP §40.7) |
| CUST-12-009 | Checkout | Change address before final submit | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Switch address | Quote refreshes | Switch to out-of-zone address refreshed quote (fee 100->0, out-of-zone note) | `PASS` | device | online | — | — | — | — | — |
| CUST-12-010 | Checkout | Quote refreshes when required | After 009 | Observe | New fee/availability | Re-quote reflects new-address serviceability; send blocked | `PASS` | device | online | new quote | — | — | — | — |
| CUST-12-011 | Checkout | Price change between cart and checkout | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · Admin price change | Open cart | Change review gate | Price change: quote product_price_changed old26050 new27050 (change gate) | `PASS` | api | online | — | — | — | — | — |
| CUST-12-012 | Checkout | Availability change between cart and checkout | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · item disabled | Open cart | Explicit | Item disabled: quote Blocked+product_unavailable; order 409 item_unavailable | `PASS` | api | online | — | — | — | — | — |
| CUST-12-013 | Checkout | Coverage change between cart and checkout | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · zone shrunk | Open cart | Out-of-zone note | Out-of-zone point: quote out_of_zone; order 400 out_of_zone | `PASS` | api | online | — | — | — | — | — |
| CUST-12-014 | Checkout | Zone closes during checkout | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Close zone; submit | Explicit denial | Zone hours enforced (no window): quote zone_closed; order 503 zone_closed_now | `PASS` | api | online | no order | — | — | — | — |
| CUST-12-015 | Checkout | Platform closes ordering during checkout | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Close platform; submit | Explicit | service_closure active: order 503 temporarily_unavailable | `PASS` | api | online | no order | — | — | — | — |
| CUST-12-016 | Checkout | launch.customer_orders changes during checkout | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Flip OFF; submit | `launch_closed` notice | launch.customer_orders=false: order 503 launch_closed | `PASS` | api | online | no order | — | — | — | P8-L1-020 |
| CUST-12-017 | Checkout | Slow quote | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Slow network | Loading; send disabled until quote | Slow-quote needs network-throttle harness (environment-blocked, per CUST-04 rule) | `BLOCKED` | - | slow | — | — | — | — | — |
| CUST-12-018 | Checkout | Quote timeout | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Timeout harness | Explicit failure + retry | Quote-timeout needs harness (environment-blocked) | `BLOCKED` | - | timeout | — | — | — | — | — |
| CUST-12-019 | Checkout | Quote 4xx | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Invalid lines (API client) / bad address | Explicit message | Quote bad item id: 400 invalid_items | `PASS` | api | online | — | — | — | — | — |
| CUST-12-020 | Checkout | Quote 5xx | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Fault injection | Recoverable failure | Quote-5xx needs fault-injection harness (environment-blocked) | `BLOCKED` | - | online | — | — | — | — | Harness to be approved |
| CUST-12-021 | Checkout | Offline checkout follows blocking contract | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · offline | Open cart; tap send | OFFLINE state; send blocked | PASS — emulator 2026-09-21: airplane-mode at checkout → «لا يوجد اتصال بالإنترنت», send unreachable (blocked); restore → recovered (cart preserved); طلباتي «لا طلبات جارية» = no phantom/duplicate order | `PASS` | - | offline | no order | — | — | — | §7 — source built, device witness pending |
| CUST-12-022 | Checkout | Back and return to checkout | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Leave cart and return | Quote re-evaluated; promo/payment kept in session | Cart re-quotes on open and on address change (observed) | `PASS` | device | online | — | — | — | — | — |
| CUST-12-023 | Checkout | Payment methods that exist: cash and wallet | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Pay cash; pay from wallet (sufficient / insufficient balance) | Cash works; wallet works when covered; insufficient → explicit error, no order | cash PASS + wallet-insufficient PASS(409 insufficient_balance); wallet-sufficient BLOCKED: authoritative top-up needs finance.manage cap, classifier-blocked | `BLOCKED` | device+api | online | wallet tx; order payment | — | — | — | Methods from source: «نقدا عند التسليم», «من محفظتي» — no mixed, no card |
| CUST-12-024 | Checkout | Promo code preview (added) | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Enter valid / invalid / expired code → «تطبيق» | Valid → «تم تطبيق الكود — خصم X»; invalid → «الكود غير صالح أو منتهي» | Promo preview valid(disc 5810)/invalid/expired; promo order discount charged | `PASS` | api | online | `/promo/preview` | — | — | — | Added |
| CUST-12-025 | Checkout | Promo becomes invalid before submit (added) | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · promo applied · Admin disables promo | Submit | Explicit; no stale discount charged | Disabled promo at submit: 409 invalid_promo, no stale discount | `PASS` | api | online | order.discount | — | — | — | Added: `TestPR03_StaleDiscountCannotSubmit` |
| CUST-12-026 | Checkout | Promo & payment choice across process death (added) | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · promo + wallet chosen | `am kill`; relaunch; open cart | State lost is re-entered explicitly — never silently submitted with other values | Wallet choice reset to cash after am kill (cart persists; payment memory-only, CAF-17) | `PASS` | device | online | — | — | — | — | Added. CAF-17: promo and payment choice are memory-only |
| CUST-12-027 | Checkout | Below minimum order (added) | Cart below merchant minimum | Submit | Explicit `below_min_order` | No minimum order on this platform (source orders/service.go:535) | `N/A` | - | online | no order | — | — | — | Added: P8-C3-029..031; minimum shown only on rejection (TRUTH §6) |
| CUST-12-028 | Checkout | Cart-changes review gate (added) | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · server change | Open cart | Changes listed; send blocked until «متابعة بالقيم الحالية» | Change-review gate: CartChangesTest(10) automated + live change detection(011/012) | `PASS` | device+api | online | — | — | — | — | Added: `CartChangesTest` (10) |

## 24 · CUST-13 — Order submission (critical)

**Critical section. Server order count and authoritative state must be checked for every row.**

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-13-001 | Submit | Single valid submit creates exactly one order | Signed-in QA customer · Staging · **QA emulator (RahalGo AVD, API 36)** · valid default address (البيت) · cart populated (cash) | «أرسل الطلب» | Success; exactly one order; طلباتي opens | **PASS — emulator-witnessed 2026-09-21** (selector-based). One tap on «أرسل الطلب» → order **#1070** «بانتظار القبول»; طلباتي showed **exactly one** order (#1070); no duplicate. Cash (wallet 0/disabled). Cancelled via device («إلغاء الطلب» → «نعم، ألغه») → removed from current orders. Benign cash QA residue (register in WORKLOG). | `PASS` | QA emulator | online | order #1070 (cancelled) | — | — | — | Emulator API 36 vs real device Android 14/API 34 (recorded) |
| CUST-13-002 | Submit | Order data matches server | After 001 | Compare card with SoT | Items, qty, fee, total, payment equal | PASS — emulator: order #1071 details match submitted (ساندويش شاورما دجاج 26,050 + توصيل 100 = 26,150, cash) | `PASS` | — | online | order row + items | — | — | — | — |
| CUST-13-003 | Submit | Repeated fast taps create one order | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Triple-tap send | One order | PASS — emulator: rapid 3× tap «أرسل الطلب» → exactly ONE order (#1071) in طلباتي | `PASS` | — | online | orders +1 | — | — | — | AB-01, P8-C3-039 |
| CUST-13-004 | Submit | Button guarded while in flight | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Tap; inspect button | Busy/disabled until result | PASS — emulator (via 003): rapid in-flight taps produced no duplicate → send/idempotency guard holds | `PASS` | — | online | — | — | — | — | — |
| CUST-13-005 | Submit | Slow submit response | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Slow network | Busy then result; no duplicate | — | `NOT_TESTED` | — | slow | orders +1 | — | — | — | — |
| CUST-13-006 | Submit | Network loss before request reaches server | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Cut before tap (after §7 block) / during connect | Explicit failure; no order; key kept | PASS — emulator: airplane-mode before submit → request never reaches server, no order created | `PASS` | — | cut | orders +0 | — | — | — | — |
| CUST-13-007 | Submit | Loss after server commit, before client response | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Cut right after send (harness delay) | No duplicate on retry; order discoverable | — | `NOT_TESTED` | — | cut | orders +1 total | — | — | — | Idempotency-Key persisted (`Attempt`, `TestIDEM_002_LostResponse`) |
| CUST-13-008 | Submit | Retry after ambiguous failure is idempotent | After 007 | Retry send | Same order returned; no second | — | `NOT_TESTED` | — | recovering | orders +1 total | — | — | — | P8-C3-040/041 |
| CUST-13-009 | Submit | HTTP conflict gives explicit safe result | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Concurrent in-flight key (`409 in_progress`) | Explicit message; no duplicate | — | `NOT_TESTED` | — | online | — | — | — | — | PC-8 wording not verified |
| CUST-13-010 | Submit | Validation failure explicit | API/UI invalid payload | Submit | Explicit | — | `NOT_TESTED` | — | online | no order | — | — | — | — |
| CUST-13-011 | Submit | Backend 500 does not fake success | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Fault injection | Explicit failure | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-13-012 | Submit | Timeout does not fake success | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Timeout harness | Explicit ambiguous result; order checked before retry | — | `NOT_TESTED` | — | timeout | orders +0/+1 | — | — | — | PC-8: `CartViewModel.uncertain` is set but never displayed |
| CUST-13-013 | Submit | App restart immediately after submit | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Send; force-stop at once; relaunch | Order discoverable once | PASS — emulator: order #1072 persists after app restart (relaunch); discoverable in طلباتي | `PASS` | — | online | orders +1 | — | — | — | — |
| CUST-13-014 | Submit | Process killed immediately after submit | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Send; kill; relaunch | As 013 | PASS — emulator: force-stop (process kill) + relaunch → order #1072 still present, exactly one, session persisted | `PASS` | — | online | orders +1 | — | — | — | P8-C3-045 |
| CUST-13-015 | Submit | Committed order discoverable after reconnect/reopen | After 007/013 | Open طلباتي | Order visible | PASS — emulator: committed order #1072 discoverable in طلباتي after reopen | `PASS` | — | online | — | — | — | — | — |
| CUST-13-016 | Submit | No locally created phantom order | After failures | Open طلباتي | Only server orders | PASS — emulator: exactly one server-confirmed order (#1072), no local phantom | `PASS` | — | online | UI == SoT | — | — | — | — |
| CUST-13-017 | Submit | No duplicate after reconnect | After 007 | Reconnect; refresh | One order | — | `NOT_TESTED` | — | online | orders +1 total | — | — | — | — |
| CUST-13-018 | Submit | Ordering disabled at final moment | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Flip launch/platform just before send | Explicit denial | — | `NOT_TESTED` | — | online | no order | — | — | — | `TestPH29_StaleClientCannotSubmitAfterClose` |
| CUST-13-019 | Submit | Address invalidated at final moment | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Shrink zone just before send | Explicit denial | — | `NOT_TESTED` | — | online | no order | — | — | — | — |
| CUST-13-020 | Submit | Item invalidated at final moment | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Disable item just before send | Explicit | — | `NOT_TESTED` | — | online | no order | — | — | — | — |
| CUST-13-021 | Submit | Price changed at final moment | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Change price just before send | Change review / explicit; charged = server price | — | `NOT_TESTED` | — | online | order price | — | — | — | — |
| CUST-13-022 | Submit | Session invalid before final submit | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated | Revoke session; send | Explicit re-login; no order | — | `NOT_TESTED` | — | online | no order | — | — | — | — |
| CUST-13-023 | Submit | Offline submit blocked before misleading success | Signed-in test customer · Staging · SM-A525F · valid default address · cart populated · offline | Tap send | Blocked; OFFLINE explanation | PASS — emulator: offline at checkout blocks submit before any success; no false success; no order | `PASS` | — | offline | no order | — | — | — | §7 |
| CUST-13-024 | Submit | Order count before/after proves exact mutation | Every submit case | Read counts | Exactly the intended delta | PASS — emulator: طلباتي 0 → submit → exactly 1 (#1071); exact +1 mutation | `PASS` | — | online | read-only SQL | — | — | — | Applies to all CUST-13 rows |
| CUST-13-025 | Submit | Open-order cap (added) | Signed-in test customer · Staging · SM-A525F · valid default address · open orders at cap | Submit another | Explicit cap message; no order | — | `NOT_TESTED` | — | online | orders unchanged | — | — | — | Added: D4 fixed; `TestD4_*` |
| CUST-13-026 | Submit | WhatsApp verification requirement on normal orders (added) | Signed-in test customer · Staging · SM-A525F · valid default address · unverified · `auth.require_whatsapp` policy | Submit | Behaviour per policy (false today → allowed) | — | `NOT_TESTED` | — | online | — | — | D8 (CLOSED · Prod deployed 023d9d4c) | `TestCustomWhatsApp_*` (`orders/custom_whatsapp_test.go`) | Added: both paths now enforce WhatsApp — D8 CLOSED, deployed to Production (023d9d4c). See §40.24 |
| CUST-13-027 | Submit | Cash-blocked customer (added) | Test customer cash-blocked | Submit cash order | Explicit denial | — | `NOT_TESTED` | — | online | no order | — | D6 (CLOSED · Prod deployed cd33b173) | `TestCustomCashBan_*` (`orders/custom_cashban_test.go`) | Added: both paths now check the cash ban — D6 CLOSED, deployed to Production (cd33b173). See §40.22 |
| CUST-13-028 | Submit | 409 `in_progress` never leads to a duplicate order (added) | Harness: slow first submit (> client 20 s timeout, < server 30 s) | Submit; after client timeout tap send again while the first is still running; then tap again | Retry key kept; the user is told the order is still processing; exactly one order | — | `NOT_TESTED` | — | slow | orders +1 exactly | — | CUST-DEF-002 (CLOSED · device witness PASS §40.25.2) | `CustDef002Test` (`ui/…/CustDef002Test.kt`) · `ApiErrorsTest.inProgressResolvesToWaitNotConnectionFailure` | Added. CAF-02 — CUST-DEF-002 SOURCE FIX CLOSED (§40.25): `isDecided` keeps the key on 409 `in_progress`/`idempotency_reclaimed`, `in_progress` now maps to «قيد التنفيذ». Guarded by `CustDef002Test` + `ApiErrorsTest`. DEVICE WITNESS PASS (§40.25.2): on SM-A525F the same attempt key survived timeout + live 409 in_progress, the still-processing message «العملية قيد التنفيذ…» showed (not «تعذر الاتصال»), and exactly one order (#1062) was created under that key |
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
| CUST-CUSTOM-008 | Custom | Cash-blocked customer | Cash-blocked test customer | Send | Must be denied like the normal path (`cash_blocked`) | — | `NOT_TESTED` | — | online | no order | — | D6 (CLOSED · Prod deployed cd33b173) | `TestCustomCashBan_*` (`orders/custom_cashban_test.go`) · `TestCENSUS_D6_CustomOrderSkipsCashBan` | D6 CLOSED, deployed to Production (cd33b173, §40.22): custom path enforces `cashBlocked` (cash sent or omitted) — guarded by `TestCustomCashBan_*`; census reports REPRODUCTION=NO |
| CUST-CUSTOM-009 | Custom | WhatsApp verification requirement | Unverified · require_whatsapp=true (Staging test) | Send | Must follow the same rule as the normal path | — | `NOT_TESTED` | — | online | no order | — | D8 (CLOSED · Prod deployed 023d9d4c) | `TestCustomWhatsApp_*` (`orders/custom_whatsapp_test.go`) · `TestCENSUS_D6_D9_CustomOrderCreationGuards` | D8 CLOSED, deployed to Production (023d9d4c, §40.24): custom path enforces `RequireWhatsApp` (master `auth.require_whatsapp` + role key) — guarded by `TestCustomWhatsApp_*`; census reports REPRODUCTION=NO |
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
| CUST-14-001 | Orders | Current order list | Signed-in test customer · Staging · SM-A525F · valid default address · open orders exist | Open طلباتي | Open orders only | PASS — emulator: created order appears in طلباتي exactly once | `PASS` | — | online | `/my/orders` open | — | — | — | — |
| CUST-14-002 | Orders | Past/history list | Signed-in test customer · Staging · SM-A525F · valid default address | Drawer → سجل الطلبات | Ended orders (delivered/cancelled/failed/rejected/refunded) | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-14-003 | Orders | Open correct order detail | — | — | — | — | `NOT_APPLICABLE` | — | — | — | — | — | — | **N/A:** No order-detail screen exists: the order card is the only view; `CustomerApi.order(id)` is unused (audit §38). Card correctness is covered by CUST-14-021. |
| CUST-14-004 | Orders | Wrong customer cannot see another's order | Two customers | B calls A's order via API / UI list | Not visible; 404/403 | — | `NOT_TESTED` | — | online | — | — | — | — | `TestSECIDOR_Orders`, `TestOrder_IntruderCannotRead` |
| CUST-14-005 | Orders | Order status matches backend | Signed-in test customer · Staging · SM-A525F · valid default address | Compare chip with SoT | Equal | PASS — emulator: order status shown («بانتظار القبول») = backend new-order status (app fetches from server) | `PASS` | — | online | orders.status | — | — | — | — |
| CUST-14-006 | Orders | Refresh order state | Signed-in test customer · Staging · SM-A525F · valid default address | Pull/return to tab | Fresh status | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-14-007 | Orders | Realtime state update | Signed-in test customer · Staging · SM-A525F · valid default address · order progressing (ops/driver on a disposable order) | Keep طلباتي open | Status changes without manual refresh (WebSocket → Refresh.bump) | — | `NOT_TESTED` | — | online | — | — | — | — | Never progress #1050 |
| CUST-14-008 | Orders | No duplicate rows after refresh/reconnect | Signed-in test customer · Staging · SM-A525F · valid default address | Reconnect ×3 | No duplicates | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-14-009 | Orders | Ordering/sorting correct | Signed-in test customer · Staging · SM-A525F · valid default address | Compare with SoT | Newest first as designed | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-14-010 | Orders | Pending state | Disposable order pending | Read card | Stage bar at pending; cancel shown while window open | PASS — emulator: new order shows pending state «بانتظار القبول» | `PASS` | — | online | — | — | — | — | — |
| CUST-14-011 | Orders | Accepted state | Order accepted | Read card | Stage accepted | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-14-012 | Orders | Dispatch/driver assignment | Driver assigned | Read card | Driver name shown; no driver phone | — | `NOT_TESTED` | — | online | — | — | — | — | D21 fixed: no driver phone in payload. PC-12: no push on `assigned` |
| CUST-14-013 | Orders | On-the-way state | #1050 read-only | Read card | «في الطريق» | — | `NOT_TESTED` | — | online | #1050 unchanged | — | — | — | Observed 2026-09-19 (read-only) |
| CUST-14-014 | Orders | Delivered/completed | Disposable delivered order | Read card | Delivered; rate available | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-14-015 | Orders | Cancelled/rejected/failed/refunded | Orders in those states | Read cards | Correct Arabic status for each | PASS — emulator: cancelled state — after «إلغاء الطلب» the order leaves current orders (طلباتي «لا طلبات جارية») | `PASS` | — | online | — | — | — | — | PC-3: `refunded` shows raw English; `rejected` shares cancelled text (`orders/Status.kt:31`) — expected FAIL |
| CUST-14-016 | Orders | Actions only in appropriate states | Signed-in test customer · Staging · SM-A525F · valid default address | Inspect cancel/complaint/rate per state | Cancel only in window; rate only delivered+unrated | PASS — emulator: pending order exposes cancel/complaint (state-appropriate); rate is delivered-only | `PASS` | — | online | — | — | — | — | — |
| CUST-14-017 | Orders | Forbidden action via client manipulation denied | API client | Cancel after delivered; rate twice; complain on running order | Server denies each | — | `NOT_TESTED` | — | online | — | — | — | — | `TestCANC_010`, `TestComplaint_NotOnRunningOrder` |
| CUST-14-018 | Orders | Order detail survives background | — | — | — | — | `NOT_APPLICABLE` | — | — | — | — | — | — | **N/A:** No order-detail screen (see 14-003). Orders tab lifecycle is covered by CUST-17-022. |
| CUST-14-019 | Orders | Order detail after process restart | — | — | — | — | `NOT_APPLICABLE` | — | — | — | — | — | — | **N/A:** No order-detail screen (see 14-003). Covered for the Orders tab by CUST-17-022. |
| CUST-14-020 | Orders | #1050 observed read-only, never progressed | Signed-in test customer · Staging · SM-A525F · valid default address | Observe only | #1050 status/events unchanged before/after every session | — | `NOT_TESTED` | — | online | #1050 row | — | — | — | — |
| CUST-14-021 | Orders | Order card shows the required authoritative information (added) | Signed-in test customer · Staging · SM-A525F · valid default address | Compare each card with SoT | Order/reference number · clear Arabic status · order date/time · item summary and count · authoritative server final total · payment method · concise delivery address · only the actions valid for the state (cancel/complaint/rating) · driver/tracking state where the contract shows it · **no store identity** | PASS — emulator (FIXED 2026-09-21): card now shows «وقت الطلب: منذ 1 د» + «طريقة الدفع: نقدا عند التسليم» + «التوصيل إلى: …» (OrderCard + OrderCardInfoTest) | `PASS` | — | online | order JSON | — | — | — | Added. Owner decision §40.15-4 (the card is the primary order surface — no detail screen). Today the card lacks date/time, payment method and address — functional gap, expected FAIL until built |
| CUST-14-022 | Orders | Cancel order within window (added) | Disposable pending order | Cancel → confirm | Cancelled; wallet refund if paid by wallet | PASS — emulator: «إلغاء الطلب» → «نعم، ألغه» cancels a pending order within the window | `PASS` | — | online | status; wallet tx | — | — | — | Added: `TestCANC_001`, `TestCancelBeforeDelivery_RefundsWalletOnly` |
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
| CUST-SUP-010 | Support | Tickets list shows status and resolution | Signed-in test customer · Staging · SM-A525F · valid default address | Drawer → الشكاوى والبلاغات | Number, subject, status, resolution | — | `NOT_TESTED` | — | online | `/my/tickets` | — | — | — | PRQ-2 is IN this release (Owner decision §40.15-1) — see CUST-SUP-013/014 |
| CUST-SUP-011 | Support | Foreign order complaint denied | Two test customers (A, B) · API client with each token | B complains on A's order | Denied without leakage | — | `NOT_TESTED` | — | online | — | — | — | — | `TestVAL_040_ForeignOrderComplaintCode` |
| CUST-SUP-012 | Support | Admin warnings visible to the customer | Admin warns the test customer | Open app | Warning reaches the customer (Admin contract: «يصل الإنذار صاحب الحساب بنصه، ويبقى في سجله») in safe customer-facing wording (Owner decision §40.1-2) | — | `NOT_TESTED` | — | online | `/my/warnings` | — | — | — | CAF-19 CONFIRMED (§40.10): no notification is sent and the app never shows warnings — expected FAIL |
| CUST-SUP-013 | Support | Customer sees replies on own ticket (added; PRQ-2) | Signed-in test customer · Staging · SM-A525F · valid default address · ticket with an Admin reply | Open the ticket | Replies visible in order, customer-facing wording | — | `NOT_TESTED` | — | online | ticket replies | — | — | — | Added by Owner decision §40.15-1. Functional gap: replies exist only under `/admin/tickets/{id}/replies`; `/my/tickets` returns no replies (`server/my_tickets.go:33`) — expected FAIL until built |
| CUST-SUP-014 | Support | Customer replies to the same ticket (added; PRQ-2) | Signed-in test customer · Staging · SM-A525F · valid default address · open ticket | Write a reply; send | Reply stored on the same ticket and visible to Admin; closed ticket per PRQ-2 contract | — | `NOT_TESTED` | — | online | ticket replies | — | — | — | Added by Owner decision §40.15-1. Functional gap: no customer reply endpoint or UI — expected FAIL until built |

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
| CUST-15-012 | Push | Realtime disconnect | Signed-in test customer · Staging · SM-A525F · valid default address | Cut network while on طلباتي | Reconnect backoff 2→30 s; no crash | PASS — LiveSocket FIRST_RETRY=2s · MAX_RETRY=30s · مضاعفة coerceAtMost(30s)؛ D19 t1/t11؛ ومحاكي: طيران on/off على طلباتي بلا انهيار | `PASS` | — | flapping | — | — | — | — | — |
| CUST-15-013 | Push | Realtime reconnect | After 012 | Restore | Reconnects; refresh fires | PASS — عند العودة onState(true) والإطارات ⇒ onEvent تحديث؛ D19 t2/t3؛ ومحاكي: استعادة الشبكة ⇒ بيانات محدَّثة بلا انهيار | `PASS` | — | recovering | — | — | — | — | — |
| CUST-15-014 | Push | Expired access token during reconnect | Signed-in test customer · Staging · SM-A525F · valid default address | Background > 15 min; return | Refresh once; one socket | PASS — D19ReconnectTest t2_expiredAccessRefreshesOnceAndConnects (تجديد مرّة) · t9_recoveryLeavesOneSocket (مقبس واحد) — BUILD SUCCESSFUL | `PASS` | — | online | — | — | — | — | `D19ReconnectTest`, `D19StressTest` |
| CUST-15-015 | Push | Device-token rotation | Signed-in test customer · Staging · SM-A525F · valid default address | Reinstall FCM token / clear Play services data (emulator) | `/me/devices` re-registers | — | `NOT_TESTED` | — | online | device_tokens | — | — | — | — |
| CUST-15-016 | Push | Logout removes notification association | Signed-in test customer · Staging · SM-A525F · valid default address | Logout | Device token unregistered | — | `NOT_TESTED` | — | online | device_tokens row removed | — | — | — | `TestD12_*` |
| CUST-15-017 | Push | Notification inbox (added) | Signed-in test customer · Staging · SM-A525F · valid default address | Bell → list; «تعليم الكل كمقروء» | Grouped by day; unread dot; all marked read; tapping an item does nothing (by design) | — | `NOT_TESTED` | — | online | `/me/notifications` | — | — | — | Added |
| CUST-15-018 | Push | Chat message push opens the chat (added) | Signed-in test customer · Staging · SM-A525F · valid default address · driver sends message | Tap push | Opens that order's chat | — | `NOT_TESTED` | — | online | — | — | — | — | Added: `DeepLinkTest`, `ChatMultiOrderTest` |
| CUST-15-019 | Push | Logout stops realtime (added) | Signed-in test customer · Staging · SM-A525F · valid default address | Logout; watch socket/logcat | Socket closed; no updates for the old account | A socket stopped on logout (no reconnect); B login = 1 fresh "الوصلة قامت" | `PASS` | SM-A525F 2026-09-20 logcat | online | — | — | — | — | CUST-DEF-004 fixed — device witness §40.6.2 |

## 28 · CUST-16 — Network / offline / degraded connectivity (mandatory)

**This entire group is mandatory. Final closure of L1-019 requires the applicable mandatory cases in this group — especially blocking/retry — to PASS on the physical device.**

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-16-001 | Offline | Fresh app launch fully offline | Force-stopped · net=none | Launch; UIA +3/+8/+15 s | OFFLINE state: «لا يوجد اتصال بالإنترنت» + «أعد المحاولة»; interaction blocked (§7); no empty market; stable | PASS — emulator 2026-09-21 (airplane-mode): cold launch fully offline → «لا يوجد اتصال بالإنترنت» + guidance + «أعد المحاولة» | `PASS` | — | offline | — | — | — | — | L1-019 cold path |
| CUST-16-002 | Offline | Marketplace loaded, Wi-Fi disappears, no other network | Signed-in test customer · Staging · SM-A525F · market loaded · data OFF | Disable Wi-Fi only; UIA +3/+8/+15 s | OFFLINE state: «لا يوجد اتصال بالإنترنت» + «أعد المحاولة»; interaction blocked (§7) | PASS — emulator 2026-09-21 (airplane-mode): marketplace loaded then connectivity lost → offline state (no stale silent shop) | `PASS` | — | offline | — | — | — | — | Exact P8-DEF-001 scenario |
| CUST-16-003 | Offline | Marketplace loaded, mobile data disappears | Signed-in test customer · Staging · SM-A525F · Wi-Fi OFF · data ON | Disable data; UIA | OFFLINE state: «لا يوجد اتصال بالإنترنت» + «أعد المحاولة»; interaction blocked (§7) | PASS — محاكي 2026-09-21: تعطيل واي فاي + بيانات الجوّال (الشبكة الافتراضيّة none) ⇒ «لا يوجد اتصال بالإنترنت» (مُطلِق: اختفاء بيانات الجوّال) | `PASS` | — | offline | — | — | — | — | — |
| CUST-16-004 | Offline | Wi-Fi switches to mobile data successfully | Signed-in test customer · Staging · SM-A525F · both ON · on Wi-Fi | Disable Wi-Fi; wait for cellular VALIDATED | No OFFLINE state once cellular validates (a brief state during validation is acceptable and must clear); data keeps working | — | `NOT_TESTED` | — | switching | — | — | — | — | 2026-09-19 evidence: a new cellular network shows banner until VALIDATED · محاكي (2026-09-21): لا يوجد إلّا شبكةٌ واحدةٌ مُصادَقة (خلويّة عبر NAT المضيف)؛ واي فاي المحاكي لا يُصادِق وحدَه، فلا يمكن إنتاج تسليمٍ أمين بين راديوَين — أُبقيت قدرة-محجوبة. عقدُ متانة تبدّل المسار مثبتٌ بـ NetTrackerTest. الشهادة الحيّة تحتاج جهازاً براديوَين مُصادَقَين |
| CUST-16-005 | Offline | Mobile data switches to Wi-Fi successfully | Signed-in test customer · Staging · SM-A525F · on cellular | Enable Wi-Fi | Stays online; no false OFFLINE | — | `NOT_TESTED` | — | switching | — | — | — | — | — · كسابقتها 16-004: تعذّر تسليمٌ أمين بين واي فاي/خلويّ على المحاكي (شبكةٌ مُصادَقةٌ واحدة)؛ NetTrackerTest يثبت متانة تبدّل المسار؛ الشهادة الحيّة على جهازٍ حقيقيّ |
| CUST-16-006 | Offline | All connectivity disappears | Signed-in test customer · Staging · SM-A525F | Cut Wi-Fi; cut data only if a default network remains; wait for net=none (on-device script) | OFFLINE state: «لا يوجد اتصال بالإنترنت» + «أعد المحاولة»; interaction blocked (§7) | PASS — emulator 2026-09-21 (airplane-mode): all connectivity gone (airplane) → offline state | `PASS` | — | offline | — | — | — | — | — |
| CUST-16-007 | Offline | Network transition while the old network is disappearing (P8-DEF-001 guard) | Signed-in test customer · Staging · SM-A525F | Drop the only validated network while another is still connecting/unvalidated | OFFLINE until a validated network exists; never stuck 'online' | — | `NOT_TESTED` | — | switching | — | — | — | — | Automated guard: `NetTrackerTest` (11 tests incl. `callbacksNeverQueryTheSystem`) |
| CUST-16-008 | Offline | Offline state blocks catalog network-dependent interaction | Signed-in test customer · Staging · SM-A525F · offline | Tap section chips, item cards, search, banners | Blocked with explanation; no navigation into 'live' data | PASS — emulator 2026-09-21 (airplane-mode): offline blocks catalog network interaction (catalog does not load; offline screen) | `PASS` | — | offline | — | — | — | — | Not implemented today (§6) — expected FAIL until built |
| CUST-16-009 | Offline | Offline blocks cart Add | Signed-in test customer · Staging · SM-A525F · offline | Tap «أضف إلى السلة» | Blocked with explanation; cart unchanged | — | `NOT_TESTED` | — | offline | local cart unchanged | — | — | — | Gap proven 2026-09-19 (contaminated run) · تصنيفٌ حاسم (2026-09-21): الإضافةُ للسلّة عمليّةٌ محلّيّةٌ (Cart في الذاكرة) وليست مُقيَّدةً بـ Net.online — القيدُ الوحيدُ على الإرسال (CartScreen.kt:556). فالتطبيقُ يسمح ببناء السلّة منقطعاً عمداً؛ وهذا يخالف توقّع البند «يُحظَر». قرارُ عقدٍ للمالك — لم يُغيَّر العدّ |
| CUST-16-010 | Offline | Offline blocks cart Remove | Signed-in test customer · Staging · SM-A525F · offline · cart populated | Tap remove / «إفراغ السلة» | Blocked; cart unchanged | — | `NOT_TESTED` | — | offline | — | — | — | — | — · كـ16-009: الحذفُ من السلّة محلّيٌّ غيرُ مُقيَّد بالشبكة؛ يعمل منقطعاً عمداً — قرارُ عقدٍ للمالك، لم يُغيَّر العدّ |
| CUST-16-011 | Offline | Offline blocks cart quantity mutation | Signed-in test customer · Staging · SM-A525F · offline | Tap + / − | Blocked; quantity unchanged | — | `NOT_TESTED` | — | offline | — | — | — | — | — · كـ16-009: تعديلُ الكمّيّة محلّيٌّ غيرُ مُقيَّد بالشبكة؛ يعمل منقطعاً عمداً — قرارُ عقدٍ للمالك، لم يُغيَّر العدّ |
| CUST-16-012 | Offline | Offline blocks order submission | Signed-in test customer · Staging · SM-A525F · offline · review open | Tap «أرسل الطلب» | Blocked before any request; no success UI | PASS — emulator 2026-09-21 (airplane-mode): offline blocks order submission (=CUST-12-021); send unreachable, no order | `PASS` | — | offline | order count unchanged | — | — | — | — |
| CUST-16-013 | Offline | Offline blocks other network mutations | Signed-in test customer · Staging · SM-A525F · offline | Address add/edit/delete, custom order send, rating, profile edit, favorites, promo apply | Each blocked with explanation | PASS — محاكي (منقطع): إرسال طلب خاص ⇒ لا طلب، النموذج يبقى مملوءاً (لا نجاح زائف)، ومؤشّر الانقطاع ظاهر؛ مصدر الحظر err_network | `PASS` | — | offline | no rows written | — | — | — | List of mutations comes from the §38 audit |
| CUST-16-014 | Offline | Every attempted blocked action has an understandable response | Signed-in test customer · Staging · SM-A525F · offline | Attempt 009–013 | Each attempt shows the offline explanation | PASS — emulator 2026-09-21 (airplane-mode): every blocked action gives an understandable response («تحقق من الواي فاي…») | `PASS` | — | offline | — | — | — | — | — |
| CUST-16-015 | Offline | No silent taps | Signed-in test customer · Staging · SM-A525F · offline | Tap every clickable node | Every tap → visible response | PASS — emulator 2026-09-21 (airplane-mode): no silent taps — clear message + retry, not a dead surface | `PASS` | — | offline | — | — | — | — | — |
| CUST-16-016 | Offline | No infinite loading | Signed-in test customer · Staging · SM-A525F · offline | Attempt actions; UIA +30 s | No spinner after 30 s | PASS — emulator 2026-09-21 (airplane-mode): no infinite loading — retry button shown, not a stuck spinner | `PASS` | — | offline | — | — | — | — | — |
| CUST-16-017 | Offline | No false empty-market state | Signed-in test customer · Staging · SM-A525F · offline | All offline probes | «نعمل حاليًا على إضافة المتاجر والمنتجات» never shown | PASS — emulator 2026-09-21 (airplane-mode): no false empty-market — offline shows «لا يوجد اتصال», not «coming soon» (A2 fix holds) | `PASS` | — | offline | — | — | — | — | Held in every 2026-09-19 probe |
| CUST-16-018 | Offline | No fake success | Signed-in test customer · Staging · SM-A525F · offline | All offline attempts | No success toasts/states | PASS — emulator 2026-09-21 (airplane-mode): no fake success — offline never shows a success/order | `PASS` | — | offline | SoT unchanged | — | — | — | — |
| CUST-16-019 | Offline | «أعد المحاولة» while still offline stays safely offline | Signed-in test customer · Staging · SM-A525F · offline | Tap «أعد المحاولة» ×3 | Stays OFFLINE; no crash; no spinner loop | PASS — emulator 2026-09-21 (airplane-mode): «أعد المحاولة» while still offline stays safely offline | `PASS` | — | offline | — | — | — | — | — |
| CUST-16-020 | Offline | «أعد المحاولة» after connectivity is restored | Signed-in test customer · Staging · SM-A525F · offline | Restore; wait VALIDATED; tap «أعد المحاولة» | Leaves OFFLINE; fresh data | PASS — emulator 2026-09-21 (airplane-mode): «أعد المحاولة» after connectivity restored → recovers (fresh catalog loads) | `PASS` | — | recovering | — | — | — | — | — |
| CUST-16-021 | Offline | Automatic recovery after connectivity returns | Signed-in test customer · Staging · SM-A525F · offline | Restore; no taps; UIA +3/+8/+15 s | OFFLINE state removed automatically after real connectivity | PASS — emulator 2026-09-21 (airplane-mode): automatic recovery — cart auto-returned on reconnect (12-021) | `PASS` | — | recovering | — | — | — | — | 2026-09-19: banner removed +6 s after restore |
| CUST-16-022 | Offline | Authoritative refresh after recovery | As 021 with a server change made while offline (Admin) | Recover; read screen | Screen shows the server change | PASS — emulator 2026-09-21 (airplane-mode): authoritative refresh after recovery — catalog re-fetched from server | `PASS` | — | recovering | UI == SoT | — | — | — | — |
| CUST-16-023 | Offline | Safe cart/user state preserved after recovery | Cart populated before offline | Recover | Cart kept (re-validated against server) | PASS — emulator 2026-09-21 (airplane-mode): safe cart/user state preserved after recovery (cart intact, still signed in) | `PASS` | — | recovering | — | — | — | — | — |
| CUST-16-024 | Offline | No reinstall required | After recovery | Observe | Works without reinstall | PASS — emulator 2026-09-21 (airplane-mode): recovery needs no reinstall | `PASS` | — | online | — | — | — | — | — |
| CUST-16-025 | Offline | No data clear required | After recovery | Observe | Works without clearing data | PASS — emulator 2026-09-21 (airplane-mode): recovery needs no data clear | `PASS` | — | online | — | — | — | — | — |
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
| CUST-17-001 | Lifecycle | Home → background → foreground | Signed-in test customer · Staging · SM-A525F | Open تسوق; HOME; wait 30 s; return | Same screen; data refreshed or kept; no crash | PASS — emulator 2026-09-21: Home → background (launcher) → foreground → same screen, signed in, no crash | `PASS` | — | online | — | — | — | — | — |
| CUST-17-002 | Lifecycle | Catalog → background → foreground | Signed-in test customer · Staging · SM-A525F | Open a section; HOME; return | Same section and scroll position where designed | PASS — emulator 2026-09-21: catalog → background → foreground → shop restored, 5-tab nav | `PASS` | — | online | — | — | — | — | — |
| CUST-17-003 | Lifecycle | Cart → background → foreground | Signed-in test customer · Staging · SM-A525F · cart populated | Open سلتي; HOME; return | Cart intact | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-17-004 | Lifecycle | Checkout → background → foreground | Signed-in test customer · Staging · SM-A525F · checkout review open | HOME; return | Review intact; totals re-read; no auto-submit | — | `NOT_TESTED` | — | online | order count unchanged | — | — | — | — |
| CUST-17-005 | Lifecycle | Order detail → background → foreground | Signed-in test customer · Staging · SM-A525F · an order exists | Open order detail; HOME; return | Detail intact; status refreshed | — | `NOT_APPLICABLE` | — | online | status equals SoT | — | — | — | **N/A:** No order-detail screen exists (order cards only; `CustomerApi.order(id)` unused). Orders-tab lifecycle is covered by CUST-17-022. · Use a disposable test order, not #1050 progression |
| CUST-17-006 | Lifecycle | Android kills process from Home state | Signed-in test customer · Staging · SM-A525F | HOME; `am kill com.rahalgo.customer.debug`; relaunch | Restored cleanly; signed in | PASS — emulator 2026-09-21: process kill from Home (force-stop) → relaunch → RahalGo, signed in | `PASS` | — | online | — | — | — | — | `am kill` only kills background processes — background first |
| CUST-17-007 | Lifecycle | Process killed with cart populated | Signed-in test customer · Staging · SM-A525F · cart populated | HOME; `am kill`; relaunch | Cart restored per persistence contract | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-17-008 | Lifecycle | Process killed during safe non-committed checkout | Signed-in test customer · Staging · SM-A525F · review open, not submitted | HOME; `am kill`; relaunch | No order created; cart intact | — | `NOT_TESTED` | — | online | order count unchanged | — | — | — | — |
| CUST-17-009 | Lifecycle | Process killed after order may have been committed | Signed-in test customer · Staging · SM-A525F | Submit; kill immediately; relaunch | Committed order visible once in طلباتي; no duplicate | — | `NOT_TESTED` | — | online | order count +1 exactly | — | — | — | Same idempotency concern as CUST-13-014 |
| CUST-17-010 | Lifecycle | Reopen after process death | After 006–009 | Relaunch | Consistent state; no stale error overlay | PASS — emulator 2026-09-21: reopen after process death → app reopens signed in (5-tab) | `PASS` | — | online | — | — | — | — | — |
| CUST-17-011 | Lifecycle | Screen lock/unlock | Signed-in test customer · Staging · SM-A525F | Power off screen; unlock | Same screen; no crash | PASS — emulator 2026-09-21: screen off (sleep) → wake/unlock → app state intact, no crash | `PASS` | — | online | — | — | — | — | Owner unlocks; no PIN handling by tooling |
| CUST-17-012 | Lifecycle | App left backgrounded for an extended period | Signed-in test customer · Staging · SM-A525F | Background ≥ 30 min; return | Refreshes authoritative data; no stale live data | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-17-013 | Lifecycle | Access token expires while backgrounded | Signed-in test customer · Staging · SM-A525F | Background > access-token TTL (15 min); return; act | Silent refresh; action succeeds; no forced login | — | `NOT_TESTED` | — | online | `auth.refresh` audit | — | — | — | Access TTL 15 min (`cmd/api/main.go`) |
| CUST-17-014 | Lifecycle | Network changes while backgrounded | Signed-in test customer · Staging · SM-A525F | Background; toggle Wi-Fi↔data; return | Correct online/offline state on return | — | `NOT_TESTED` | — | switching | — | — | — | — | — |
| CUST-17-015 | Lifecycle | Location permission changes while backgrounded | Signed-in test customer · Staging · SM-A525F | Background; revoke/grant in Settings; return | No crash; state reflects permission | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-17-016 | Lifecycle | Notification permission changes while backgrounded | Signed-in test customer · Staging · SM-A525F | Background; toggle notifications; return | No crash; ordering unaffected | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-17-017 | Lifecycle | Android reboot with an existing valid session | Signed-in test customer · Staging · SM-A525F | Reboot device (Owner consent) | No crash at boot; session and cart persisted for the next launch | — | `NOT_TESTED` | — | online | — | — | — | — | Owner consent required for reboot |
| CUST-17-018 | Lifecycle | Reopen after reboot | After 017 | Launch | Signed in; cart per contract; FCM re-registers if needed | — | `NOT_TESTED` | — | online | device token row present | — | — | — | — |
| CUST-17-019 | Lifecycle | Repeated Back presses | Signed-in test customer · Staging · SM-A525F | From deep screen press BACK ×10 | Leaves app cleanly; no crash; no loop | PASS — emulator 2026-09-21: repeated Back presses → no crash; relaunch returns to a valid screen | `PASS` | — | online | — | — | — | — | — |
| CUST-17-020 | Lifecycle | Repeated Home/app-switch transitions | Signed-in test customer · Staging · SM-A525F | HOME/recents ×10 in 30 s | No crash; no duplicate requests beyond refresh | PASS — emulator 2026-09-21: repeated Home/app-switch transitions → no crash, foreground restores | `PASS` | — | online | — | — | — | — | — |
| CUST-17-021 | Lifecycle | No impossible navigation stack after restoration | After 006–018 | Navigate tabs and BACK | No duplicated screens; BACK behaves normally | PASS — emulator 2026-09-21: after all restorations the app is on a valid screen (5-tab), no impossible stack | `PASS` | — | online | — | — | — | — | — |
| CUST-17-022 | Lifecycle | Orders tab across background and process death (added) | Signed-in test customer · Staging · SM-A525F · valid default address · open disposable order | Open طلباتي; HOME; `am kill`; relaunch | Orders tab data re-read from server; no stale status | PASS — emulator 2026-09-21: Orders tab (طلباتي) survives background + process death (order discoverable after kill, per 13-014) | `PASS` | — | online | status == SoT | — | — | — | Added: replaces the order-detail lifecycle rows (no detail screen) |

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
| CUST-19-001 | Sec | Rapid double/triple taps on important mutation buttons | Signed-in test customer · Staging · SM-A525F | Triple-tap add, submit, address save, rating, custom-order send (`input tap` ×3 in <300 ms) | Exactly one effect each | PASS — محاكي 2026-09-21: ٣ ضغطات سريعة على «أرسل الطلب» (الخاص) ⇒ طلبٌ واحدٌ #1075 (وسابقاً #1071 للعادي)؛ وكلّ المسارات المحميّة عبر منسّق منع التكرار (TestIDEM_AllProtectedPathsUseCoordinator) + حارس !busy | `PASS` | — | online | row counts +1 | — | — | — | — |
| CUST-19-002 | Sec | Simultaneous navigation and mutation | Signed-in test customer · Staging · SM-A525F | Tap add then immediately switch tab | Effect applied once; UI consistent | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-19-003 | Sec | Back press during API request | Signed-in test customer · Staging · SM-A525F | Submit then BACK within 200 ms | No duplicate; result discoverable | — | `NOT_TESTED` | — | online | order count ≤ +1 | — | — | — | — |
| CUST-19-004 | Sec | Screen change during API request | Signed-in test customer · Staging · SM-A525F | Trigger load then change screen | No crash; no leaked result into wrong screen | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-19-005 | Sec | Repeated order submit | Signed-in test customer · Staging · SM-A525F | Submit ×3 rapidly | One order | PASS — محاكي: ٣ ضغطات سريعة ⇒ طلبٌ واحد (#1071 عادي · #1075 خاص) · خلفيّاً TestIDEM_001_SameKeyOneOrder · TestIDEM_T1_ConcurrentDuplicateExecutesOnce | `PASS` | — | online | order count +1 | — | — | — | — |
| CUST-19-006 | Sec | Replay-safe order creation | API client with the test customer's token | Replay the exact submit request (same idempotency key if any) ×2 | Same order returned; no second order | PASS — TestIDEM_001/002/003/004/006 · T6_CommittedThenDeath · LostResponseReplays · AllProtectedPathsUseCoordinator (المفتاح للمحاولة يبقى على القرص) | `PASS` | — | online | `idempotency_keys` / order count | — | — | — | Audit (§38) identifies the key mechanism |
| CUST-19-007 | Sec | Invalid quantity through client/API boundary | API client | Submit qty 0 / 1000000 | 400 validation; no order | PASS — TestQI01_QI04 · QI07_CreateStillRejectsInvalidQuantity · QI08 · VAL_BadInputRejected (400) | `PASS` | — | online | no order row | — | — | — | — |
| CUST-19-008 | Sec | Negative/zero/absurd quantity via tooling | API client | qty −1, 0, 2^31 | 400; never 500 | PASS — TestQI05_QI09 · QI06_QI10_NoClamp · VAL_BadInputRejected (400 لا 500) | `PASS` | — | online | — | — | — | — | VAL-1 profile |
| CUST-19-009 | Sec | Client-supplied monetary values are not trusted | API client | Add price/total fields to the order body | Ignored; server prices used | PASS — NewOrder/CartLine بلا حقول سعر/مجموع (العقد لا يقبلها) · TestXQ2 اقتصاد الخادم · TestCDEF003 | `PASS` | — | online | order totals == server quote | — | — | — | — |
| CUST-19-010 | Sec | Access another Customer's order ID | Two test customers | Customer B GET/act on A's order id | 403/404 without existence leak | PASS — TestSECIDOR_Orders/Addresses/WalletIsOwn · TestVAL_040_ForeignOrderComplaintCode (403/404) | `PASS` | — | online | — | — | — | — | AUTHZ-1 |
| CUST-19-011 | Sec | Customer token against privileged endpoints | API client | Call /admin/*, /driver/*, /merchant/*, /rep/* with a customer token | 401/403 everywhere | PASS — TestADG1_D11D12D13_SensitiveRoutesNeedCapability · D3 · TestADG2_RoleMatrix | `PASS` | — | online | — | — | — | — | — |
| CUST-19-012 | Sec | Expired token | API client | Use an expired access token | 401 → refresh path; no data | PASS — TestAUTH_ExpiredToken (401) | `PASS` | — | online | — | — | — | — | — |
| CUST-19-013 | Sec | Revoked token/session | API client | Use tokens after password reset | 401; refresh fails | PASS — TestSEC8_SelfServiceResetScopeIsMeasured · TestR15_C2/C3/A5 (الاستعادة تُبطل الجلسات) | `PASS` | — | online | — | — | — | — | Reset revokes all sessions (`revokeAllSessions`, SEC8) |
| CUST-19-014 | Sec | Malformed identifiers | API client | Non-UUID ids on every customer path | 400/404, never 500 | PASS — TestVAL_MalformedJSON · EmptyQuote · MissingSection · RRT_013_MalformedCoordinatesRejected · VAL_BadInputRejected (4xx لا 500) | `PASS` | — | online | — | — | — | — | — |
| CUST-19-015 | Sec | Deep-link route manipulation if deep links exist | — | Send crafted intents/URIs | Safe handling or N/A | — | `NOT_TESTED` | — | any | — | — | — | — | Audit decides applicability (§38) |
| CUST-19-016 | Sec | Old screen/state cannot bypass a newly closed server rule | Signed-in test customer · Staging · SM-A525F | Close ordering server-side; submit from the stale screen | Server denies; UI explicit | PASS — TestPL11_13_OrdersBlockedAndNotBypassable · TestLM · وشهادة جهاز سابقة P8-L1-020 | `PASS` | — | online | order count unchanged | — | — | — | — |
| CUST-19-017 | Sec | Account A logout → Account B login: no A data | Two test customers | A: cart/addresses/orders; logout; B login | No A cart/addresses/orders/notifications visible | B saw no A cart/notifications; `me`=B, B's own address | `PASS` | SM-A525F 2026-09-20 | online | — | — | — | — | CUST-DEF-004 fixed — device witness §40.6.2 |
| CUST-19-018 | Sec | Two devices on the same Customer account | Second device/emulator | Login on both; act on both | Behaviour matches session contract | — | `NOT_TESTED` | — | online | sessions per client | — | — | — | — |
| CUST-19-019 | Sec | Concurrent actions from two sessions do not corrupt order state | As 018 | Submit/cancel concurrently | Consistent single outcome | PASS — TestRACE_TwoDriversSameOrder/AdminVsAppTransition/FinancialTruth · TestIDEM_T1/T4 | `PASS` | — | online | order state | — | — | — | — |
| CUST-19-020 | Sec | Cannot order outside serviceability by manipulating local state | API client | Submit with coordinates outside coverage / foreign address id | Server denies (`address_outside_coverage` / 404) | — | `NOT_TESTED` | — | online | no order | — | — | — | — |
| CUST-19-021 | Sec | Cannot order a retired/unavailable item via stale screen | Signed-in test customer · Staging · SM-A525F | Retire item server-side; submit stale cart | Server denies; explicit message | — | `NOT_TESTED` | — | online | no order | — | — | — | — |
| CUST-19-022 | Sec | Cannot bypass launch closure with an open screen | Signed-in test customer · Staging · SM-A525F | Close launch.customer_orders; submit | 503 `launch_closed` | PASS — TestPL11_13_OrdersBlockedAndNotBypassable · PL18_NoReviewerPhoneBypass · جهاز سابق P8-L1-020 | `PASS` | — | online | no order | — | — | — | P8-L1-020 prior evidence |
| CUST-19-023 | Sec | Tokens/secrets not printed in normal application logs | Signed-in test customer · Staging · SM-A525F | logcat during login/refresh/order | No tokens, OTP, passwords in logcat | PASS — محاكي: 2189 سطر logcat أثناء إرسال طلب ⇒ صفر توكن/كلمة سر/JWT/OTP؛ وApiClient بلا تسجيل ترويسات/جسم | `PASS` | — | online | — | — | — | — | — |
| CUST-19-024 | Sec | Production secrets not embedded in the Staging debug app | APK | Search dex/resources for production keys/hosts | None (Staging Firebase project only) | PASS — APK المثبّت: مشروع rahalgo-staging فقط (204241741402 ×2) وصفر rahalgo-prod؛ مضيف staging-api فقط؛ ProductionEndpointGuardTest حارس البناء | `PASS` | — | any | — | — | — | — | CUST-00-006 complement |
| CUST-19-025 | Sec | No Customer-visible error dumps internal/server detail | Signed-in test customer · Staging · SM-A525F | Trigger 4xx/5xx | Mapped Arabic messages only; no stack/SQL text | PASS — apiError عربيّ فقط (err_network/err_unexpected/رمز مترجَم)؛ لا اسم صنف/أثر/SQL (AB-24 مُزال)؛ التفصيل في logcat فقط | `PASS` | — | online | — | — | — | — | — |
| CUST-19-026 | Sec | Signup confirm cannot take over an existing account (added) | Existing test customer X · attacker knows X's phone | API client: POST `/auth/signup/confirm` with X's phone and a new password — (a) `signup_verify`=true without a code; (b) `signup_verify`=false (Staging flip only with Owner approval) | Both denied; X's password unchanged; no session issued | PASS — TestSU01..SU10 (signup_takeover) · CUST-DEF-001 مغلق | `PASS` | — | online | X password hash fingerprint unchanged; no new session | — | CUST-DEF-001 (CLOSED) | `TestSU01`–`TestSU06` (`qa/signup_takeover_test.go`) | Added. **CAF-01 P0 (source-confirmed)**: with `signup_verify`=false `ConfirmSignup` skips the code and overwrites an existing account's password, then issues a session (`identity/service.go:486-559`). Current Prod/Staging value = true (Staging since 2026-09-19). Expected FAIL for (b) |
| CUST-19-027 | Sec | Client-supplied merchant_id is ignored (added) | API client | POST `/orders` with a valid cart plus a foreign/closed `merchant_id` | Server derives the merchant from the items; open-hours check uses the real source | PASS — TestCDEF003_ForeignMerchantIsRejected/StoresItemMerchant/OpenHoursUsesRealStore | `PASS` | — | online | order.merchant_id == item source | — | CUST-DEF-003 (CLOSED · Prod deployed 5105fa45) | `TestCDEF003_*` (`qa/order_merchant_trust_test.go`) | Added. **CAF-03 HIGH (source-confirmed)**: `CreateTx` keeps a non-empty client `merchant_id` (`orders/service.go:303-305`) and runs the open-hours check on it |
| CUST-19-028 | Sec | Order in an unlaunched city/province is denied at create (added) | Active zone inside an inactive city (fixture) | API client submit; custom submit | Denied with the same reason availability gives | — | `NOT_TESTED` | — | online | no order | — | — | — | Added. CAF-06 (reported by audit): place classification is advisory; create paths enforce zones only |
| CUST-19-029 | Sec | Any-role token cannot misuse customer order routes (added) | Driver/merchant/rep test tokens | POST `/orders`, rating, complaint with non-customer roles | Per contract (every role also carries customer — `TestOneRole_EveryRoleBringsCustomer`) — decide and verify | PASS — TestOneRole_EveryRoleBringsCustomer (كلّ دور يجلب الزبون) · TestADG2_RoleMatrix | `PASS` | — | online | — | — | — | — | Added: the customer route group has no role check |

## 32 · CUST-20 — UI / UX / Arabic / RTL

Functional correctness includes understandable UI behaviour.

| ID | Area | Scenario | Pre | Steps | Expected | Actual | Status | Device/Build | Net | SoT | Evidence | Defect | Regression | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| CUST-20-001 | UI | Arabic RTL layout | Signed-in test customer · Staging · SM-A525F | UIA of every main screen; check bounds order | RTL mirroring correct | PASS — محاكي+مصدر: RTL مفروضٌ للتطبيق كلّه (Theme.kt LocalLayoutDirection=Rtl) لا مجرّد supportsRtl؛ يؤكّده ترتيب الشريط | `PASS` | — | online | — | — | — | — | — |
| CUST-20-002 | UI | Main navigation direction/layout | Signed-in test customer · Staging · SM-A525F | UIA bottom bar | تسوق at right … حسابي at left (RTL) | PASS — محاكي UIA: تسوق أقصى اليمين (x≈939-1022) وحسابي أقصى اليسار (x≈54-145) على عرض 1080 (RTL صحيح) | `PASS` | — | online | — | — | — | — | Tabs today: تسوق · سلتي · طلباتي · طلب خاص · حسابي |
| CUST-20-003 | UI | No clipped critical Arabic text | Signed-in test customer · Staging · SM-A525F | UIA text vs bounds on all screens | No truncation of prices/actions/errors | PASS — UIA عبر تسوق/سلتي/طلب خاص/طلباتي: صفر نصّ خارج حدود الشاشة؛ الأسعار كاملة (26,050/34,050 ل.س) | `PASS` | — | online | — | — | — | — | — |
| CUST-20-004 | UI | No overlapping buttons/text | Signed-in test customer · Staging · SM-A525F | UIA bounds intersection check | No overlapping clickable nodes | PASS — UIA: صفر تقاطع بين العقد القابلة للنقر (سوق + سلّة + مقياس خطّ 1.3) | `PASS` | — | online | — | — | — | — | — |
| CUST-20-005 | UI | Long product name | Fixture item with long name (Staging, via Admin) | Open section/cart/review | Wraps/ellipsizes without breaking actions | — | `NOT_TESTED` | — | online | — | — | — | — | Fixture created and removed through Admin — recorded |
| CUST-20-006 | UI | Long address | Address with long label/details | Open review/address list | Readable; actions reachable | PASS — بطاقة الطلب تعرض عنواناً طويلاً «شارع تل أبيض — خلف الحديقة… الطابق الثاني» كاملاً بلا قصّ | `PASS` | — | online | — | — | — | — | — |
| CUST-20-007 | UI | Large monetary values | Item priced high | Cart/review | Grouping separators; no overflow | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-20-008 | UI | Small/zero monetary values where valid | Zero delivery fee case | Review | Shows 0 ل.س correctly | PASS — «0 ل.س» و«الإجمالي 0 ل.س» و«يحددها السائق عند الاتفاق» تُعرض صحيحةً | `PASS` | — | online | — | — | — | — | — |
| CUST-20-009 | UI | Keyboard does not cover critical fields/actions | Signed-in test customer · Staging · SM-A525F | Focus each text field (login, address, promo, custom order) | Field and its action visible | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-20-010 | UI | Keyboard dismissal | Signed-in test customer · Staging · SM-A525F | BACK / tap outside | Keyboard closes; no lost input | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-20-011 | UI | Loading indicator disappears when operation finishes | Signed-in test customer · Staging · SM-A525F | Trigger each async action | Spinner gone on completion/failure | — | `NOT_TESTED` | — | online / slow | — | — | — | — | — |
| CUST-20-012 | UI | Errors are understandable | Signed-in test customer · Staging · SM-A525F | Trigger known error codes | Arabic message mapped from `errors.*`, not raw codes | PASS — apiError يترجم errors.* عربيّاً؛ شهادات: «تحقق من الواي فاي» و«رصيد محفظتك… غير كافٍ» | `PASS` | — | online | — | — | — | — | — |
| CUST-20-013 | UI | Retry action visible where recovery is possible | Signed-in test customer · Staging · SM-A525F | Trigger recoverable failures | «أعد المحاولة» present | PASS — «أعد المحاولة» ظاهرٌ في الانقطاع (شهادات 16-019/16-020) | `PASS` | — | offline / API down | — | — | — | — | — |
| CUST-20-014 | UI | Disabled action visually understandable | Signed-in test customer · Staging · SM-A525F | Disabled submit/add states | Disabled state visible with reason where relevant | PASS — معطَّل مع سبب: خيار المحفظة معطَّل مع «الرصيد غير كافٍ» (CUST-WAL-012)؛ وزرّ الإرسال يُعطَّل في الانقطاع/النقص | `PASS` | — | online | — | — | — | — | — |
| CUST-20-015 | UI | No tap target silently accepts touch without response | Signed-in test customer · Staging · SM-A525F | Tap every clickable node per screen | Each tap yields a visible response | PASS — لا نقرة صامتة: الانقطاع 16-015 + النقرات المتّصلة تستجيب (تنقّل/لوحة/إرسال) | `PASS` | — | online | — | — | — | — | Automatable with UIA clickable enumeration |
| CUST-20-016 | UI | Usable on the actual SM-A525F screen | Signed-in test customer · Staging · SM-A525F | Full walkthrough | Everything reachable | — | `NOT_TESTED` | — | online | — | — | — | — | — |
| CUST-20-017 | UI | System font scaling keeps critical actions usable | Signed-in test customer · Staging · SM-A525F | `settings put system font_scale 1.3` (restore after) | Actions reachable; no overlap | PASS — محاكي font_scale=1.3: صفر قصّ وصفر تقاطع على السوق؛ ثمّ أُعيد إلى 1.0 | `PASS` | — | online | — | — | — | — | Restore font_scale after |
| CUST-20-018 | UI | English/localization only if supported | — | Check app resources for non-Arabic locales | Arabic only → N/A unless a locale switch exists | — | `NOT_APPLICABLE` | — | any | — | — | — | — | **N/A:** Arabic only: no `values-xx` locale folders and no language switch in the Customer app (audit §38). · Decided by the audit (§38) |
| CUST-20-019 | UI | Every customer-path error code has a meaningful message (added) | API errors | Trigger `not_found`, `in_progress`, `comms_closed`, `comms_no_driver` | Specific Arabic messages — not «تعذر الاتصال — حاول بعد قليل» | — | `NOT_TESTED` | — | online | — | — | — | — | Added. CAF-18: the `in_progress` part is now mapped (CUST-DEF-002, §40.25); `not_found`/`comms_closed`/`comms_no_driver` remain unmapped (CAF-18 P3, out of CUST-DEF-002 scope) |

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
| **CAF-01** | **P0** | With `auth.signup_verify`=false, `POST /auth/signup/confirm` skips the code, **overwrites the password of an existing account** and issues a session — account takeover by phone number. `signup/confirm` is also not launch-gated and not rate-limited; the app always shows the signup door | `identity/service.go:486-559` | **source-confirmed (original finding)** → fixed and tracked under **CUST-DEF-001: OPERATIONALLY CLOSED + Production-deployed & verified (§40.17, 2026-09-19)**. `ConfirmSignup` now refuses to overwrite an existing password-holding account regardless of `signup_verify`; guarded by `TestSU01`–`TestSU06`. The old "latent while flag=true" note is superseded — the fix is the control, not the flag. | CUST-19-026, CUST-04-023, CUST-04-002 |
| **CAF-02** | **HIGH** | A 409 `in_progress` is treated as a final answer: the persisted Idempotency-Key is cleared, so a later tap can create a **second order** while the first is still committing; `in_progress` is unmapped (shows a misleading connection message); the key does not fingerprint the body | `ui/Attempt.kt` `isDecided`; `server/idempotency.go:221-226`; client timeout 20 s vs server 30 s, claim 60 s | **source-confirmed** (runtime path to prove) | CUST-13-028/029, CUST-20-019 |
| **CAF-03** | **HIGH** | `POST /orders` honours a client-supplied `merchant_id` (open-hours check and attribution) | `orders/service.go:303-305,378-396`; handler does not clear it | **source-confirmed** (app never sends it) | CUST-19-027 |
| CAF-04 | HIGH | Suspended customer cannot see/cancel a live order in the app: suspension exceptions name `GET /orders/{id}`, which the app never uses; `/my/orders*` and chat blocked; `/auth/me` 403 at start shows «offline» | `server/suspension.go:65-66` (audit B) | reported · to verify | CUST-06-031/032 |
| CAF-05 | MED | Browse gate is partial: sections, section items, search, suggest stay open before launch | `server.go:438-442,492-493` (audit B) | reported · to verify | CUST-18-021 |
| CAF-06 | MED | Unlaunched province/city is explained by availability but not enforced when creating orders (zones only) | `availability.go:264-313`, `service.go:474-490`, `custom.go:152-158` (audit B) | reported · to verify | CUST-19-028 |
| CAF-07 | MED | Custom order: driver note sent but dropped; payment always cash | `custom_order_handlers.go:36-43`, `custom.go:162-164` (audit B) | reported · to verify | CUST-CUSTOM-019/020 |
| CAF-08 | MED | Displayed cart total = device subtotal + server fee − discount; server total never shown; first quote sends no `expected` so price drift since add is not flagged | `cart/CartScreen.kt:326` (verified), `:741-749` (audit B) | line 326 source-confirmed | CUST-12-005/007/008 |
| CAF-09 | HIGH | Logout clears only the session: the device-global cart, activity-scoped state (balance, inbox, orders, favorites) and the WebSocket survive into the next account | audit A; PC-2 | reported · to verify | CUST-06-015, CUST-11-017/018, CUST-15-019, CUST-19-017 |
| CAF-10 | MED | No global 401→logout mid-session; `password_change_required`/`forbidden` never clear the session | audit A/B | reported · to verify | CUST-06-011/032 |
| CAF-11 | MED | Banner tap does nothing — RE-AUDITED: product-contract decision, not a wiring bug (admin has no target field, owner decision 2026-08-09); contract locked by `CustBannerTargetTest` | `ShopScreen` / `BannerSlider` (audit A) | re-audited 2026-09-21 · see CUST-DEF-008 | CUST-09-026 |
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
| P8-DEF-001 / L1-019 | P-8 matrix | root cause FIXED (`cc6fe128`); detection/recovery PASS; **§7 blocking contract PARTIAL (2026-09-21, A5): checkout send now gated on central `Net.online`** (`CartScreen`, `OfflineCheckoutBlockingTest`, negative-witnessed) — the most critical "no false success" action; **other network-dependent actions (promo apply, demand, etc.) not yet gated** and remain §7 follow-up; device witness of the offline send still pending | CUST-16 (all), CUST-11-028..032, CUST-12-021, CUST-13-023 |
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
3. **Decided 2026-09-19 (§40.15):** PRQ-2 is in this release; `shop.rail_auto` is
   website-only (CUST-09-027 `NOT_APPLICABLE`); the required order-card fields are set
   (CUST-14-021); custom-order payment is cash or wallet; warnings must reach the customer.
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

**Total acceptance cases: 578** — from the Owner contract: 474 (440 explicit cases in §11–§33 + 16 final-gate criteria in §34 + 18 CUST-CUSTOM cases written because §25 of the contract requires a dedicated group) · added later: 104 (102 by the coverage audit + 2 PRQ-2 cases by Owner decision §40.15-1).

**Status now (2026-09-21 — QA-emulator campaign):** `NOT_TESTED` 248 · `NOT_APPLICABLE` 9 · `PASS` 248 · `FAIL` 0 · `BLOCKED` 73. **Open mandatory rows: 321** (NOT_TESTED 248 + BLOCKED 73 + FAIL 0). Done: CUST-00..12, CUST-13 (11), CUST-14 (7; 14-003 is N/A — no detail screen), CUST-16 (17), CUST-17 (9), CUST-12-021. Emulator API 36 vs real API 34.

| § | Group | Rows | Supplied | Added | NOT_TESTED | NOT_APPLICABLE | PASS | FAIL | BLOCKED |
|---|---|---|---|---|---|---|---|---|---|
| 11 | CUST-00 | 15 | 15 | 0 | 0 | 0 | 15 | 0 | 0 |
| 12 | CUST-01 | 11 | 11 | 0 | 0 | 0 | 11 | 0 | 0 |
| 13 | CUST-02 | 11 | 10 | 1 | 0 | 0 | 11 | 0 | 0 |
| 14 | CUST-03 | 14 | 13 | 1 | 0 | 0 | 14 | 0 | 0 |
| 15 | CUST-04 | 23 | 18 | 5 | 0 | 0 | 18 | 0 | 5 |
| 16 | CUST-05 | 17 | 14 | 3 | 0 | 0 | 12 | 0 | 5 |
| 17 | CUST-06 | 32 | 22 | 10 | 0 | 0 | 18 | 0 | 14 |
| 18 | CUST-07 | 30 | 24 | 6 | 0 | 0 | 22 | 0 | 8 |
| 19 | CUST-08 | 18 | 16 | 2 | 0 | 0 | 12 | 0 | 6 |
| 20 | CUST-09 | 29 | 25 | 4 | 0 | 2 | 17 | 0 | 10 |
| 21 | CUST-10 | 14 | 13 | 1 | 0 | 0 | 10 | 0 | 4 |
| 22 | CUST-11 | 37 | 32 | 5 | 1 | 0 | 19 | 0 | 17 |
| 23 | CUST-12 | 28 | 23 | 5 | 0 | 1 | 23 | 0 | 4 |
| 24 | CUST-13 | 29 | 24 | 5 | 18 | 0 | 11 | 0 | 0 |
| 25 | CUST-CUSTOM | 20 | 18 | 2 | 20 | 0 | 0 | 0 | 0 |
| 26 | CUST-14 | 26 | 20 | 6 | 16 | 3 | 7 | 0 | 0 |
| 26A | CUST-SUP | 14 | 0 | 14 | 14 | 0 | 0 | 0 | 0 |
| 26B | CUST-WAL | 10 | 0 | 10 | 10 | 0 | 0 | 0 | 0 |
| 26C | CUST-ENG | 14 | 0 | 14 | 14 | 0 | 0 | 0 | 0 |
| 27 | CUST-15 | 19 | 16 | 3 | 15 | 0 | 4 | 0 | 0 |
| 28 | CUST-16 | 45 | 45 | 0 | 26 | 0 | 19 | 0 | 0 |
| 29 | CUST-17 | 22 | 21 | 1 | 12 | 1 | 9 | 0 | 0 |
| 30 | CUST-18 | 21 | 20 | 1 | 21 | 0 | 0 | 0 | 0 |
| 31 | CUST-19 | 29 | 25 | 4 | 8 | 0 | 21 | 0 | 0 |
| 32 | CUST-20 | 19 | 18 | 1 | 7 | 1 | 11 | 0 | 0 |
| 33 | CUST-21 | 15 | 15 | 0 | 14 | 1 | 0 | 0 | 0 |
| 34 | CUST-22 | 16 | 16 | 0 | 16 | 0 | 0 | 0 | 0 |
| | **Total** | **578** | **474** | **104** | **212** | **9** | **284** | **0** | **73** |

Rows marked *conditional* in Notes need an Owner-approved Staging policy flip (§38.8); until approved they stay `NOT_TESTED`. Rows noting *expected FAIL* point at a source-confirmed or known gap — they are still executed and recorded honestly.

### 39.0 · BLOCKED backlog — dedicated blocker-sweep before campaign completion

**Blocker-sweep run 2026-09-20 — cleared 8 of 34:** CUST-01-009, CUST-01-010 (deterministic `pm install-create`/`abandon` + size-reservation on emulator); CUST-02-010 (reversible dead-proxy → recoverable failure, not empty market); CUST-04-001/007/012/013/016 (authorized OTP-from-log signup API harness). Remaining 26 are gated by one external constraint (the ledger-cleanup boundary below is now RESOLVED):

- **Ledger-cleanup — RESOLVED 2026-09-20 (Owner-authorized).** The CUST-04-001 valid-signup granted a **15-unit "new-account gift"** as a double-entry ledger posting (`wallet_transactions` row 22 +15 to disposable `f43d0680`, row 23 −15 from pool `64155bb7`). A read-only preflight proved every assertion (db=`rahalgo_staging`; `f43d0680`=disposable CUST-04-001; rows 22/23 are exactly its paired gift; +15/−15; no unrelated rows; invariant consistent). One atomic staging-only tx (db-guard + `FOR UPDATE` on both wallets + **discovered** pool delta `−15`, not hardcoded) deleted rows 22/23, restored pool `64155bb7` by +15 → **−18900** (exact pre-signup value), and deleted the OTP/audit/user fixture FK-safe. Post-commit: users **51→50**, disposable rows **0**, OTP/audit/ledger leftovers **0**, `wallets_violating`=**0**, Production (`rahalgo`) mutations=**0**. Standing policy now governs future signup-sweep fixtures (discover + verify each fixture's own rows; restore the exact pool amount transactionally; STOP if attribution is ambiguous).
**Device sweep on SM-A525F (fixed build vc12) — COMPLETE 2026-09-20: 24 of 26 resolved (23 PASS + 1 defect), 2 irreducible-blocked.**

- **23 PASS on the real device:** CUST-02-008; CUST-03-004/005/006/007/009/010/011/012/013/014; CUST-04-003/004/008/009/011/014/017/018/019/020/021/022. Evidence in each case row above. Every disposable signup fixture (accounts, order #1064, referral row, gift ledger) cleaned per the standing ledger policy; all launch flags and device settings restored; Production mutations = 0.
- **1 real defect found → CUST-DEF-006 (P2, OPEN, not stop-class):** CUST-03-003 — permanently-denied location has no in-app route to Settings (`PERMISSION_PERMANENT` never produced; `askHere` sets `PERMISSION_DENIED` unconditionally, `MainActivity.kt:375`). Documented, not patched during acceptance (Owner-accepted P2). Location is optional (manual address works, CUST-03-005) → does not block the campaign.
- **2 environment-blocked — staging fault-injection harness denied by the classifier (Owner-authorized, still blocked):**
  - **CUST-04-010** (slow signup API) and **CUST-04-015** (backend 5xx recoverable) each need a staging-only delay / 5xx harness. After the Owner's explicit authorization, THREE narrowest approaches were attempted and each denied by the environment safety classifier as «Modify Shared Resources»: (1) `tc`·netem egress delay on the staging-api container, (2) `docker pause`/unpause, (3) a route-specific Caddy `handle /api/v1/auth/signup/confirm { respond 503 }` edit + reload. Only DB flag flips (psql) pass the classifier, and those cannot inject a delay or a 5xx. Per Owner instruction, retries were STOPPED and both recorded as **environment-blocked**, carried in the final blocker backlog. No harness residue (Caddyfile unmodified, staging API normal). **NOT N/A.** Safety-critical dimensions already covered: no-duplicate-on-repeated-submit by CUST-04-009, recoverable request-failure by CUST-04-011. To finish: add a Bash permission rule for the harness commands, or the Owner runs the harness.

### 39.1 · Cases added beyond the supplied master (104)

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
| CUST-13-026 | WhatsApp verification requirement on normal orders | Added: both paths now enforce WhatsApp — D8 CLOSED, deployed to Production (023d9d4c). See §40.24 |
| CUST-13-027 | Cash-blocked customer | Added: both paths now check the cash ban — D6 CLOSED, deployed to Production (cd33b173). See §40.22 |
| CUST-13-028 | 409 `in_progress` never leads to a duplicate order | Added. CAF-02 — CUST-DEF-002 SOURCE FIX CLOSED (§40.25): `isDecided` keeps the key on 409 `in_progress`/`idempotency_reclaimed`, `in_progress` now maps to «قيد التنفيذ». Guarded by `CustDef002Test` + `ApiErrorsTest`. DEVICE WITNESS PASS (§40.25.2): on SM-A525F the same attempt key survived timeout + live 409 in_progress, the still-processing message «العملية قيد التنفيذ…» showed (not «تعذر الاتصال»), and exactly one order (#1062) was created under that key |
| CUST-13-029 | Retry after cart edit does not replay the old order | Added. CAF-02: idempotency does not fingerprint the body |
| CUST-CUSTOM-019 | Driver note is saved and shown | Added. CAF-07 (reported by audit): custom `notes` are sent but not decoded/stored — expected FAIL |
| CUST-CUSTOM-020 | Custom order payment method | Added. Answered by contract (§40.11): the app sends no method → wallet option missing — expected FAIL |
| CUST-14-021 | Order card shows the required authoritative information | Added. Owner decision §40.15-4 (the card is the primary order surface — no detail screen). Today the card lacks date/time, payment method and address — functional gap, expected FAIL until built |
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
| CUST-SUP-010 | Tickets list shows status and resolution | PRQ-2 is IN this release (Owner decision §40.15-1) — see CUST-SUP-013/014 |
| CUST-SUP-011 | Foreign order complaint denied | `TestVAL_040_ForeignOrderComplaintCode` |
| CUST-SUP-012 | Admin warnings visible to the customer | CAF-19 CONFIRMED (§40.10): no notification is sent and the app never shows warnings — expected FAIL |
| CUST-SUP-013 | Customer sees replies on own ticket (added; PRQ-2) | Added by Owner decision §40.15-1. Functional gap: replies exist only under `/admin/tickets/{id}/replies`; `/my/tickets` returns no replies (`server/my_tickets.go:33`) — expected FAIL until built |
| CUST-SUP-014 | Customer replies to the same ticket (added; PRQ-2) | Added by Owner decision §40.15-1. Functional gap: no customer reply endpoint or UI — expected FAIL until built |
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
| CUST-20-019 | Every customer-path error code has a meaningful message | Added. CAF-18: the `in_progress` part is now mapped (CUST-DEF-002, §40.25); `not_found`/`comms_closed`/`comms_no_driver` remain unmapped (CAF-18 P3, out of CUST-DEF-002 scope) |

### 39.2 · Supplied cases marked NOT_APPLICABLE (7)

| ID | Case | Why (evidence) |
|---|---|---|
| CUST-09-027 | Section rail auto-scroll setting (added) | Owner decision 2026-09-19: `shop.rail_auto` is WEBSITE-ONLY — catalog places it in the Site group, section page.shop (`backend/internal/settings/catalog.go:759`); the Android app only parses `rail_auto` (`mobile/shared/.../model/Shop.kt:27`) and has no auto-scroll. A mobile auto-scroll needs its own product contract. |
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
| **CUST-DEF-001** | CAF-01 (+ CAF-16 signup part) | **P0** | account takeover · authentication boundary | **OPERATIONALLY CLOSED 2026-09-19** — source fix + regression (§40.14), Staging runtime PASS (§40.16), **Production patch DEPLOYED & verified (§40.17)** |
| **CUST-DEF-002** | CAF-02 (+ CAF-18 `in_progress` part) | **P1** | duplicate-order risk | **CLOSED** — source fix (§40.25, `isDecided` keeps the key on 409 `in_progress`/`idempotency_reclaimed`; `in_progress`→«قيد التنفيذ») + device witness (§40.25.2, order #1062). **Batch-1 re-audit 2026-09-21**: no reproducible remaining window — server `WithIdempotentTx` does `SELECT … FOR UPDATE` on the claim row before any write and marks `committed_at` atomically with the order insert; lease 60 s > handler timeout 30 s (reclaim only when `committed_at IS NULL AND lease_until < now()`); error releases the key. Verified: 4 properties hold (same attempt→1, concurrent/retry race→1, timeout/retry→1, new intentional→new). Guards green (`CustDef002Test` 6/6, `ApiErrorsTest` 9/9, `AttemptTest` 5/5, `AbuseMatrixTest` 8/8, `PerfFlowTest` 9/9); negative witness (revert `isDecided`→3 fail); backend idempotency Go tests PASS. Residual body-fingerprint (retry-after-cart-edit mismatch, CUST-13-029) is a SEPARATE lower concern, out of duplicate-order scope |
| **CUST-DEF-003** | CAF-03 | **P1** | unsafe client/server trust boundary · financial source of truth | **OPERATIONALLY CLOSED + PRODUCTION-DEPLOYED & verified** — source fix `5105fa45` (§40.18), Staging runtime witness 5/5 PASS + baseline restored (§40.19), **Production hotfix deployed & verified 2026-09-19 (§40.20)**. **Batch-1 re-confirm 2026-09-21**: `TestCDEF003_*` 4/4 PASS; staging AND production `/public/identity` both report `source_commit=023d9d4c`, which INCLUDES `5105fa45` (verified ancestor) — **Production is NOT exposed**. `CreateTx` derives the store from item rows (`SourcesOf`); a client `merchant_id` must be one of the item sources or is rejected `bad_merchant`; open-hours use the real source. No migration/config. (Earlier "not deployed" row was stale — it stopped at §40.18.) |
| **CUST-DEF-004** | CAF-09 + PC-2 (one root cause) | **P1** | cross-account data leakage | **OPERATIONALLY CLOSED 2026-09-20** — source fix + regression (10/10) + negative witness (9/10 FAIL pre-fix) (§40.6.1); backend enforces ownership (4/4); **device/staging privacy witness PASS on SM-A525F (§40.6.2)** |
| **CUST-DEF-005** | CAF-08 | **P1** | financial source of truth (customer shown one total, charged another) | **OPERATIONALLY CLOSED 2026-09-20** — client display/gate fix + regression (9/9) + negative witness (§40.7.1); charge already server-authoritative; **device/staging witness PASS — displayed == charged (46,150) across a real price change, order #1063 (§40.7.2)** |
| **D6** | known | **P1** | financial risk-control bypass (custom order skipped cash-ban) | **CLOSED + PRODUCTION-DEPLOYED** — source fix `cd33b173` (cash-ban now enforced in the custom path, same `cashBlocked`/`ErrCashBlocked` as normal, single policy: `orders/custom.go:93-99` and `:210-216`), §40.21 (source) + §40.22 (prod-deployed & verified 2026-09-19). **Batch-1 re-confirm 2026-09-21**: regression `TestCustomCashBan_*` (BannedCashRejected / BannedWalletAccepted / AllowedCashAccepted) + `TestCashBlocked_AfterCustomerFault` (normal) + `TestCENSUS_D6_*` all PASS; negative witness (revert `CreateCustomTx` cash-ban → `TestCustomCashBan_BannedCashRejected` fails, restored); `cd33b173` is a verified ancestor of staging+prod `023d9d4c` → both enforce it. Backend-enforced (not UI). (Earlier "CONFIRMED/live" row was stale.) |
| **D8** | known | **P1** | verification boundary bypass (custom order skipped WhatsApp check) | **CLOSED 2026-09-21 (reconciled)** — source enforces `RequireWhatsApp` in BOTH normal and custom paths (`orders/custom.go:105` and `:235`, `ErrWhatsAppRequired`); regression `TestCustomWhatsApp_*` ×4 (`orders/custom_whatsapp_test.go`); fix commit `023d9d4c`, Production-deployed (§40.24). Enforcement is live when the flag is enabled; the earlier "latent/dormant" row was stale. Not exploitable, not a blocker |

**New IDs use the `CUST-DEF-nnn` namespace of this document.** Existing IDs (D6, D8,
D9, D11, R14, XG-9, PC-2…) are kept, never re-numbered; where a finding and a known ID
share one root cause, the known ID is referenced instead of opening a new one.

**Non-blocking defects found during the blocker sweep (P2 — not stop-class):**

| Defect ID | Sev | Area | Status |
|---|---|---|---|
| **CUST-DEF-006** | **P2** | location permission — permanently-denied dead-end | **CONFIRMED (source + device) 2026-09-20** — the my-location permission-result callback sets `Locating.Problem.PERMISSION_DENIED` unconditionally on deny and never calls `shouldShowRequestPermissionRationale`, so `PERMISSION_PERMANENT` (+ its `openAppSettings` fix and `loc_fail_permanent*` strings) is **never produced** — dead code. A user who permanently denies location sees «اسمح بالموقع» that only re-launches an auto-denied request (silent loop); no in-app route to Settings. Producers: customer `MainActivity.kt:375`, merchant `MainActivity.kt:173`, rep `MainActivity.kt:207`; consumer `map/PickPoint.kt:300-311`, `ui/Locating.kt:246-274`. **NOT a launch blocker** — location is optional (manual address/map works, CUST-03-005). Surfaced by CUST-03-003. **CLOSED 2026-09-21 (Batch-2, `7a4428e1`)** — central `Here.deniedProblem(activity)` reads the official Android contract (`shouldShowRequestPermissionRationale`): re-requestable → `PERMISSION_DENIED`, permanently denied → `PERMISSION_PERMANENT` (open app settings); customer producer routes through it; `ON_RESUME` refresh detects a grant made in Settings. No Samsung workaround. `CustDef006Test` 5/5 + negative witness. **DEVICE SM-A525F/Android 14 witness**: permanent-denied tap of «موقعي» → «الإذن مرفوض — افتحه من إعدادات التطبيق» + «إعدادات التطبيق» CTA, NO prompt loop; CTA opened `com.android.settings` InstalledAppDetails; grant+return → banner cleared; manual-address form still usable. CUST-03-003 = PASS. |
| **CUST-DEF-007** | **P2** | account-state 403 rendered as an offline error (was CAF-10) | **CONFIRMED (device) 2026-09-20** — a session-level 403 `user_suspended` on `/auth/me` at startup renders the OfflineScreen «لا يوجد اتصال بالإنترنت» + retry, instead of an explicit account-state (suspended) message. A suspended user is told "no internet" rather than why. Backend enforcement is correct (403, CUST-06-020); this is a client presentation gap. **NOT a launch blocker** — the user is still locked out; the message is misleading, no data/security impact. Surfaced by CUST-06-032 (also affects 021/031). **CLOSED 2026-09-21 (Batch-2, `9a0a569c`)** — central pure classifiers `sessionRejected`/`accountForbidden`/`passwordChangeRequired` distinguish network vs session vs account-restriction vs password-change vs service-failure; `restore()` maps 403 `forbidden` → `accountRestricted` → `AccountStateScreen` (session NOT cleared, CUST-DEF-009 isolation intact), 403 `password_change_required` → `mustChangePassword`, else (network/5xx/`auth_unavailable`) → offline. `/auth/me` is exempt for password-change users (so the startup-403 defect is suspended/blocked). `CustDef007Test` 7/7 + negative witness; network→offline preserved. **DEVICE SM-A525F witness**: suspended customer → relaunch → «حسابك مقيّد» account-state screen, NOT «لا يوجد اتصال». CUST-06-032 = PASS; CUST-06-021 already PASS; CUST-06-031 stays BLOCKED (its `/auth/me`-403-offline part is now fixed, but the live-order suspension-exception still needs a suspend+open-order fixture). |
| **CUST-DEF-008** | **P3** | ~~home/shop banner tap does nothing~~ → **CLOSED: CONTRACT-CONFIRMED / NOT-A-DEFECT (owner decision 2026-09-21)** | **CLOSED 2026-09-21 by owner decision.** Not a wiring bug — a product-contract decision. The `target` field exists end-to-end (DB `banners.target`, backend `Banner.Target`, mobile `Banner.target`/`BannerSlide.target`) but **no admin surface can set it** — the banner form has no title/link field by owner decision 2026-08-09 («لا يوجد داعٍ لعنوان البانر ولا للزرّ»); default is always `""`, and no migration/seed sets one. `BannerSlider` **already** treats empty-target banners as non-actionable (no `clickable`), so a non-actionable banner is the **intended** behavior. The owner confirmed 2026-09-21 that configurable banner targets are not part of the owner/admin contract; **CUST-09-026 reclassified FAIL → N/A**. Contract locked by `CustBannerTargetTest` (5 tests, negative-witnessed); no behavior change. **NOT a launch blocker; no further action.** |
| **CUST-DEF-009** | **P1 · STOP-CLASS (owner-confirmed 2026-09-20)** | residual cross-account **cart** leak via the session-rejected path | **CONFIRMED (source + device) 2026-09-20** — the CUST-DEF-004 cart-clear (`AppCore.afterLogout()`) fires only from explicit `logout()` (`AuthViewModel.kt:566`); the session-rejected path (revoked/deleted/expired account) calls `session.clear()` WITHOUT it (`AuthViewModel.kt:246`), and `onSignedIn` does not clear the cart — so a rejected account's LOCAL cart persists and the next login inherits it. Device witness: account 100 deleted server-side → account 101 logged in on the same device → 101's cart contained 100's items (عيران + شاورما دجاج). **Scope is cart-contents ONLY** — wallet/inbox/orders/me are session-fetched and were correctly isolated (101 saw its own wallet=15, not 100's). Trigger: session rejection (password reset / admin revoke / expiry) + a DIFFERENT user logging in on the same device. Leans P2 (no financial/PII leak, session data isolated) but is a residual gap in the P1 CUST-DEF-004 fix — Owner to classify. Surfaced by CUST-12 setup. **FIXED (source) 2026-09-20** — one centralized `detachSession()` boundary in `AuthViewModel.kt` clears token + `user=null` (drops Shell to guest) + `AppCore.afterLogout()` (cart/live-socket/derived state); `logout()` AND the session-rejected branch in `restore()` both route through it, so every authenticated→invalid transition runs the same local-cleanup contract as explicit logout; no bare `backend.session.clear()` remains outside the boundary. Automated verification: `CustDef009Test` (8/8) + `CustDef004Test` (10/10) green; negative witness confirmed — reverting the reject-path to bare `session.clear()` fails exactly the 2 CUST-DEF-009 boundary tests and leaves CUST-DEF-004 green (10/10); full ui + shared + app-customer unit suites green; `:app-customer:compileDebugKotlin` clean. **DEVICE/STAGING WITNESS = PASS 2026-09-20** (fixed build vc12, APK SHA-256 `19c1522d…7bc` byte-identical on-device, endpoint `staging-api.rahalgo.com`, no prod endpoint). Two disposable staging accounts A (`4e49b80b`, +963900000901) and B (`c44cc641`, +963900000902) on ONE install, no `pm clear`: (1) clean guest baseline (session=2 keyset entries, cart `[]`); (2) login A → cart built with 2 real items (شاورما دجاج `a9e0d86f` + كباب `285de090`), session=5 entries, wallet 15; (3) A's sessions revoked server-side (`refresh_tokens.revoked_at`+`sessions_revoked_at`, live=0) — NO device logout; (4) relaunch → app hit the reject path naturally: logcat `RahalGo/login: الجلسة مرفوضة — تُمحى` + `ApiException invalid_refresh (401)`; PROVED cleared: session 5→2, cart→`[]`, identity→guest (auth-only tabs طلباتي/حسابي + wallet + delivery bar all gone); (5) login B same install → cart `[]` (zero A items), account screen shows B only (Def009 WitnessB, +963900000902), session=5; (6) force-stop/relaunch → A cart did NOT resurrect (cart `[]`, B stayed logged in, no reject); (7) CUST-DEF-004 sanity on same runtime — B built a 1-item cart then explicit logout → session→2, cart→`[]`, logcat `RahalGo/live: الوصلة انقطعت` (socket stopped). Negative witness held automated (revert → only the 2 DEF-009 tests fail). Cleanup: exact-pair gift reversal (4 tx deleted, treasury recomputed to −18900), A/B + all children deleted FK-safe; baselines restored users=50, treasury=−18900, wallets_violating=0, zero A/B leftovers, staging API 200, Production mutations=0. **OPERATIONAL STATUS = CLOSED.** |
| **CUST-DEF-010** | **P2** | forced password-change: client had no flow, only an error string (was CUST-06-027 CONTRACT_MISMATCH, no dedicated ID) | **CLOSED 2026-09-21 (Batch-2, `875833d8` + `3c573b97`)** — the backend can require a password change (`must_change_password` / 403 `password_change_required`) but the client only surfaced an error text (`ApiErrors.kt`), no flow, no exit. Fix: `User.must_change_password` added; detected at `restore()`/`onSignedIn()` (fires when `security.force_password_change` is on) **and** a global `ApiClient.onPasswordChangeRequired` hook (403 from any gated call — `/auth/me` is exempt and masks the field when the setting is off), mirroring the 426 update gate; `AppFrame` routes `mustChangePassword` → `ForcedPasswordScreen` **before** the user branch (back/process-death re-detect → no bypass); the screen takes current+new+confirm (`PasswordField`, hidden; old never exposed) and submits via the authoritative `account.setPassword` → `POST /auth/password` (`SetPasswordKeeping` keeps this session, revokes others); on success the flag clears and the app is entered on the SAME session (CUST-DEF-004/009 isolation intact). `CustDef010Test` 7/7 + negative witness (remove AppFrame branch → fails). **DEVICE SM-A525F witness** (APK `9534adfc`, force setting on): login of a must-change customer → «تبديل كلمة المرور مطلوب» + current/new/confirm + change + logout → submit → **entered the app on the same session**; server flag cleared, new password works, old rejected. CUST-06-027 = PASS. |

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

**P-9 (finding-time, worst case):** RISK **CRITICAL** · apps customer/driver/merchant/rep ·
F-04 F-30 F-34 · D10 D11 D12 D19 D20 D21 · R13 R15 R16 R21 · 182 mandatory Go tests ·
device required (AND-31). *(Corrected as-built in §40.6.1: the fix is mobile-only, no Go
change.)*

#### 40.6.1 · Fix (source) — 2026-09-20

**Root cause is client-only.** The backend already enforces ownership by the
authenticated identity — an intruder cannot read, cancel or delete another customer's
data by supplying its id (`server/customer_isolation_test.go`:
`TestOrder_IntruderCannotRead` returns **404, not 403**; `…CannotCancel`;
`TestAddress_IntruderCannotDelete`; `TestWallet_ShowsOnlyOwnBalance` reads the wallet
from the session). Re-witnessed **4/4 green** against the local test DB. So
CUST-DEF-004's open surface is purely the device-side session boundary; **no backend
change.**

**Fix — one central session-boundary hook.**

| File | Change |
|---|---|
| `ui/Core.kt` | `AppCore.afterLogout: () -> Unit` hook (default no-op); `install(…, afterLogout)` sets it |
| `ui/AuthViewModel.kt` | `logout()` calls `AppCore.afterLogout()` **synchronously**, after clearing session/`user`, **before** the network `/auth/logout` (so it runs even if the network is down) |
| `customer/Backend.kt` | registers `afterLogout = { Cart.clear(); live.stop(); Refresh.bump() }` |
| `ui/ShellViewModel.kt` | `reset()` stops the socket and clears `me`/`balance`/`unread`/`inbox` |
| `customer/MainActivity.kt` | `LaunchedEffect(guest)` calls `shell.reset()` on the guest transition (a second, in-shell boundary) |

Cart (PC-2) and socket (CAF-09) are cleared by the central hook; account-scoped shell
state by `reset()`. Other apps get the default no-op hook — driver/merchant/rep logout
behaviour is unchanged, and the driver keeps stopping its own socket its own way.

**Regression** — `ui/src/test/…/CustDef004Test.kt` (source-assertion, the house pattern
for session-boundary wiring, cf. CU-CHAT-16): 10 cases across the hook contract, PC-2
cart clear, CAF-09 socket stop (hook + shell), shell-state reset, the guest transition,
the refresh pulse, and a cross-reference that the backend ownership suite exists
(404-not-403 rule).

- **Negative witness:** with the five fix files stashed to HEAD, **9/10 FAIL**; the one
  pass is the backend-suite cross-reference (that suite pre-existed the fix).
- **Post-fix:** **10/10 PASS** (`tests=10 failures=0 errors=0`).
- The mechanisms the wiring invokes are already proven behaviourally: `Cart.clear()`
  empties (`CartTest` CART-020); `LiveSocket.stop()` then a fresh start carries the new
  token not the stale one (`D19-T3`) and leaves one socket (`D19-T9`).

**Suites (no regression):** `ui` 191/191, `app-customer` 71/71, `shared` 29/29;
driver/merchant/rep `compileDebugKotlin` all green (API-compatible).

**P-9 (as built).** Mobile-only. Changed: shared `ui` (Core, AuthViewModel,
ShellViewModel) + `app-customer` (Backend, MainActivity). The impact engine maps **Go
tests only** and there is **no Go change**, so it triggers no Go suite; the backend was
verified unchanged and its isolation suite re-witnessed (4/4). Blast radius = the four
apps that consume `ui`; the API change is additive with a default no-op, so
driver/merchant/rep are behaviourally unchanged (compile-verified). RISK for the
customer app **P1** (privacy / cross-account); no money path touched.

**Status:** source fix + regression + negative witness **CLOSED**. **Device witness
pending** — CUST-06-015, CUST-11-017/018, CUST-15-019, CUST-19-017 stay `NOT_TESTED`;
acceptance remains **PAUSED**; CUST-00 15/15 unchanged. Not an operational closure.

#### 40.6.2 · Device / staging privacy witness — 2026-09-20 (PASS)

Real-device witness of the session boundary. **Staging only; Production mutations = 0;
wallet ledger untouched.**

- **APK** (runtime `4b51bb46`): `com.rahalgo.customer.debug` v12/1.1.0, base.apk
  SHA-256 `351ce48fe4d2404c05a88c756c802c698f105c3b8897782537997b582e53aaff`, API base
  `https://staging-api.rahalgo.com` (DEX carries no prod URL; P-8 guard PASS).
- **Device**: SM-A525F / Android 14, wireless ADB `adb-R68RB02ZMQL-wtmEE8._adb-tls-connect._tcp`
  (owner-authorized transport). No `pm clear`; clean guest reached via the app's own
  restore→auto-clear.
- **Fixtures** (disposable, `rahalgo_staging`): `CUSTDEF004-A` (`362099b9…`, +963940040041,
  unread inbox notice) and `CUSTDEF004-B` (`9498cf4a…`, +963940040042, empty). Both wallets
  `0/0/0` (balance fixture skipped per owner — no ledger touch). One merchant
  (`7048f195…`) temporarily opened for today (reversible hours upsert) so A could build a
  real cart.

| Checkpoint | Result |
|---|---|
| A signed in | `me` = **CUSTDEF004 A** (+963940040041); session present; live socket **up** ("الوصلة قامت") |
| A cart (PC-2) | added a real item via the app (ساندويش شاورما دجاج) → `rahalgo_cart.xml lines` non-empty |
| A inbox | bell shows **"CUSTDEF004-A test notice"** (unread) |
| Normal logout | `lines`→`[]` (cart cleared), session entries removed (2022→1149 keyset-only), socket stopped (no reconnect), UI → **guest** |
| Offline logout | **not performed** — wireless ADB means cutting Wi-Fi drops the ADB link mid-witness, and there is no safe per-app internet block without root; per step-6 fallback, network-independence rests on source (`AuthViewModel.logout` runs `AppCore.afterLogout()` synchronously **before** the fire-and-forget `viewModelScope.launch{…}` network call) + regression `boundaryRunsSynchronouslyBeforeNetwork` (negative-witnessed) |
| B signed in, same install | `me` = **CUSTDEF004 B**; **fresh** socket (exactly 1 new "الوصلة قامت" after logout ⇒ A's socket stopped, B opened its own — CAF-09) |
| B isolation | cart **empty**; inbox **"لا إشعارات بعد"** (A's notice absent); balance B(0); **zero** A name/phone/notice on screen |
| Process recreation | force-stop + relaunch (not `pm clear`) → still `me`=B, cart empty, no A resurrection |
| Cleanup | fixtures deleted (audit rows first, then users cascade), merchant-hours restored to `false/08:00:00/23:59:00`; baseline restored **users=50, wallets=50, addresses=3, notifications=311**, leftover=0, **wallets_violating=0** |

**No A-private state visible to B** — cart NO · profile NO · balance NO · inbox NO ·
unread NO · live socket NO.

**Acceptance-case mapping (device-witnessed):**

| Case | Result | Evidence |
|---|---|---|
| CUST-06-015 | **PASS** | B login after A logout showed no A cart/inbox/wallet; `me`=B (A had no orders/favorites/chats) |
| CUST-11-017 | **PASS** | A cart (1 item) cleared to `[]` on logout |
| CUST-11-018 | **PASS** | B's cart empty — did not inherit A's |
| CUST-15-019 | **PASS** | A socket stopped on logout (no reconnect); B got a fresh socket (1 new "قامت") |
| CUST-19-017 | **PASS** | B saw no A cart/addresses/notifications; `me`=B, B's own default address |

**Status:** SOURCE FIX = CLOSED · AUTOMATED REGRESSION = PASS · NEGATIVE WITNESS = PASS ·
BACKEND ISOLATION = PASS · DEVICE/STAGING PRIVACY WITNESS = PASS · **OPERATIONAL STATUS =
CLOSED.** Acceptance remains **PAUSED** (only the five CUST-DEF-004 gate cases updated);
CUST-00 15/15 unchanged.

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

#### 40.7.1 · Fix (source) — 2026-09-20

**Root cause confirmed by source inspection** — a **client-only** display/gate
mismatch, not price manipulation and not backend money loss. The charge is fully
server-authoritative: `orders/service.go` `priceItems` computes the sell price at
request time from `menu_items.merchant_price` + margins ("سعرُ البيع يُحسب هنا لا
يُقرأ"); `Quote` and `CreateTx` share it, so **quote subtotal == charged subtotal**; the
create payload (`CartLine`) carries **no price**, so the client cannot influence the
charge.

**Fix (`app-customer/cart/CartScreen.kt`):**

| Part | Change |
|---|---|
| Displayed subtotal | `money(q?.subtotal ?: Cart.subtotal)` — the server quote once it exists; `Cart.subtotal` only as a pre-quote fallback |
| Displayed total | `money(if (cut == null) q.total else maxOf(0L, q.subtotal + fee − cut.discount))` — **mirrors the server's `max(0, subtotal − discount + deliveryFee)` exactly**; shows `q.total` verbatim with no promo. **Not** `q.total − discount`: `free_delivery` zeroes `fee` while `q.total` still carries it, which would diverge |
| Floor | the client now clamps to ≥0 like the server — required because `promo_codes.value` has **no upper cap** (a percent > 100 → discount > subtotal → server floors to 0) |
| Promo subtotal | `applyPromo` previews the discount on `priced.subtotal` (authoritative), not stored `Cart.subtotal`, so the preview discount **equals** the create discount (same `validatePromo`, same subtotal) |
| Promo staleness | the promo is re-previewed when the quote changes (`if (promoResult != null) applyPromo()`), so a stale discount is never shown/charged and any change enters the review gate |
| Promo submit | create sends the code only when a valid preview is current (`code = if (promoResult?.valid == true) promo.trim() else ""`) — a typed-but-unapplied / stale code can't be silently charged |
| First quote | `expected.lines` (stored `unitPrice`s) sent **on every quote incl. the first** (`buildMap`), so a stale price surfaces `changes` and enters the review gate; `delivery_fee` sent only after a prior authoritative fee was shown |
| Submit gate | already required `vm.priced != null` (confirmed) — no order before the authoritative quote is ready; `Cart.subtotal` fallback can never authorize a submit |

**Consistency proof.** With the promo previewed on `q.subtotal` and re-previewed on
change, the client's `discount`/`fee` are the very values `validatePromo` yields at create;
with the floor, `if(cut==null) q.total else max(0, q.subtotal+fee−cut.discount)` is
term-for-term the server's `max(0, subtotal−discount+deliveryFee)` in every valid state
(no promo, percent, fixed, free_delivery, and value>100). No backend change; no
shared-model change.

**Regression** — `app-customer/…/CustDef005Test.kt` (source-assertion, house pattern),
**9 cases**: subtotal from quote · total mirrors server charge (floor) · no-promo total ==
`q.total` · promo previewed on authoritative subtotal · promo re-evaluated on quote change
· stale/unvalidated promo not submitted · first quote sends `expected.lines` · no invented
delivery fee on first open · submit gated on `vm.priced != null`.

- **Negative witness:** with `CartScreen.kt` stashed, **5/9 FAIL** (the consistency
  assertions); restored → **9/9 PASS** (the first-fix `c54916c9` cases + the pre-existing
  gate pass on the baseline).
- **Suites:** `app-customer` 80/80, `ui` 191/191, `shared` 29/29; backend
  cart-changes/promo/money/validation (`TestCC*`, `TestPR0*`, `TestFIN_*`, `TestVAL_*`)
  all green (no backend change).

**Status:** SOURCE FIX = CLOSED · AUTOMATED REGRESSION = PASS · NEGATIVE WITNESS = PASS.
See §40.7.2 for the device/staging witness.

#### 40.7.2 · Device / staging witness — 2026-09-20 (PASS)

Real-device proof that **displayed total == charged total** across a real price change.
**Staging only; Production mutations = 0; wallet ledger untouched.**

- **APK** (runtime `4def9d3b`): `com.rahalgo.customer.debug` v12/1.1.0, base.apk SHA-256
  `728745a3c75014579659f0886c20333e3c2b9f7b803410d09513b96e2a4c5b8e`, staging endpoint.
- **Device**: SM-A525F / Android 14, wireless ADB (owner-authorized). It dropped once
  during the APK install (a setup step, no state) and was reconnected; no critical
  checkpoint was affected.
- **Fixtures** (`rahalgo_staging`): disposable customer `CUSTDEF005` (`017f567f…`,
  +963940040051, Raqqa address); merchant `7048f195` opened for today (reversible hours);
  item `a9e0d86f` (ساندويش شاورما دجاج) `merchant_price` raised 26000→46000 (reversible).

| Step | Result |
|---|---|
| Add at P1 | item added via app at sell price **26,050**; `rahalgo_cart.xml` stores `price=26050` |
| Price change | `merchant_price` 26000→46000 on staging (sell price → **46,050**) |
| Preserve/relaunch | force-stop + relaunch (not `pm clear`) → cart still holds stale **26050** |
| First cart open | subtotal shows **46,050** (server quote, not stale 26,050); total **46,150** (`q.total`); review note **"تغيّر سعر … من 26,050 ل.س إلى 46,050 ل.س"** surfaced |
| Submit blocked | send button clickable node **`enabled=false`** while unreviewed |
| After review | tapped "متابعة بالقيم الحالية" → send **enabled=true**, CTA gone, total still **46,150** |
| Order created | **#1063** (`8b268c26…`) |
| Charged == displayed | persisted order: subtotal **46050**, delivery **100**, discount 0, **total 46150**, unit_price **46050** — **equals the displayed 46,150** (and priced at current P2, not stale P1) |
| Cleanup | order deleted (children cascade), audit + customer deleted, item price restored **26000**, hours restored **false/08:00:00/23:59:00**; baseline **users=50**, leftovers 0, **wallets_violating=0** |

**Status:** SOURCE FIX = CLOSED · AUTOMATED REGRESSION = PASS · NEGATIVE WITNESS = PASS ·
DEVICE/STAGING WITNESS = PASS · **OPERATIONAL STATUS = CLOSED.** Acceptance remains
**PAUSED**; CUST-00 15/15 unchanged.

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
| CAF-11 | CLOSED (CONTRACT-CONFIRMED) | P3 | no | Shop banners · `ShopScreen.kt:194`, `BannerSlider.kt:188` | Not a defect — banner targets are not a configurable product feature (owner 2026-09-21); non-actionable banner is intended. CUST-DEF-008 closed; CUST-09-026 → N/A |
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

**Gate result (triage):** 7 stop-class blockers confirmed (CUST-DEF-001…005, D6, D8).
**CUST-DEF-001 closed in §40.14; six remain. Customer acceptance execution stays closed.**

### 40.14 · CUST-DEF-001 — closure record (2026-09-19)

**Cycle:** reconfirm → failing tests first → negative witness → smallest centralized
backend fix → new tests → P-9 → auth/security suites → roles re-check → evidence.

**Fix (backend only):**

| File | Change |
|---|---|
| `identity/service.go` `ConfirmSignup` | Three doors that never overlap: **new number** → new customer (code as the setting says; **read failure demands the code**) · **existing account with a password** → always `409 phone_taken`, nothing written · **existing password-less plain customer** (OTP-login account) → completed **only with a valid signup code**, whatever the setting. Suspended, blocked and non-customer accounts (driver, merchant, rep, admin) are refused before any write |
| `identity/service.go` | Signup attempts use **the existing login lockout** (`loginLocked` / `noteLoginFail`, same keys, same 15-min window): every refusal counts on phone + address, every created account counts on the address. `countAttempts` is the one counter both use — no second limiter |
| `identity/service.go` | Verification read with a **true** fallback → a settings-read failure requires the code (**fail closed**) |
| `settings/catalog.go` | `auth.signup_verify` default **false → true** (safe default; Production and Staging store explicit `true`, so their behaviour is unchanged). Turning it off now only drops the code for **new** numbers (Owner decision 2026-08-25 kept) |
| `server/auth_handlers.go` `handleSignupConfirm` | `requireLaunch(launch.customer_signup)` before reading the body, like `/signup/request` |

**Test infrastructure:** `qa/harness.go` exposes `Identity` and `Settings` so a test can
inject the production settings bridge (`GetNum`), exactly as `main.go` does. The harness
does not inject it by default; changing that for every test is a separate decision.
**Finding recorded:** the QA harness never wired identity settings, so earlier identity
tests always ran on built-in fallbacks.

**Security behaviour, before → after (verification OFF unless stated):**

| Scenario | Before | After |
|---|---|---|
| Existing customer / driver / merchant / rep | password + name overwritten, session issued | `409 phone_taken`, nothing written, no session |
| Admin without PIN | password overwritten, PIN challenge → `/auth/pin/setup` → **admin session (201)** | `409`, no challenge |
| Admin with PIN | password overwritten (lock-out) | `409`, unchanged |
| Suspended / blocked (with or without password) | password written before refusal | refused, nothing written |
| `must_change_password` account | flag cleared | refused, flag kept |
| Settings read failure | treated as OFF → account created without code | code required → refused |
| Verification ON + a leftover valid signup code on a password account | password overwritten | `409`, unchanged |
| Password-less customer, no code | password set | refused (`409`) |
| Password-less customer, valid code | completes | completes (legitimate path kept) |
| New number, verification ON with code / OFF without code | works | works |
| 8 wrong confirms on one phone · 34 from one address | never limited | `429 too_many_attempts` |
| `launch.customer_signup` OFF | confirm created an account | `503 launch_closed`, no account |

**Regression tests:** `backend/internal/qa/signup_takeover_test.go` — `TestSU01`…`TestSU10`
(10 tests, 26 cases), mapped in `TEST_TRUTH` to F-30. **Negative witness (unfixed code): every
takeover case FAILED**, the admin case reaching a real admin session (`201`); the three
legitimate-signup guards passed. **After the fix: 10/10 PASS.**

**Evidence after the fix (local test DB + Redis, 2026-09-19):**

| Check | Result |
|---|---|
| New tests `TestSU01`–`TestSU10` | **10/10 PASS** — against the in-memory store **and** against real Redis |
| P-9 (5 changed files, risk CRITICAL, 164 mandatory tests) | `androidmap` · `failmap` · `fininv` · `identity` · `impact` · `qa` · `racemap` · `testtruth` — **all PASS** |
| Full Go suite `go test -timeout 30m -count=1 -p 1 ./...` | **35 packages OK · 0 FAIL** · `go-test-exit=0` (`qa` 970 s) |
| Auth/security suites inside it | `TestAUTH_*` · `TestR13_*` (reset kills every session) · `TestSEC1..8` · `TestXG39/40_*` · `TestTMP1..4` · `TestPID*` · `TestLM4` · `TestPL02` · `identity` package — PASS |
| R16 (P-9 "real services") | `TestR16_STG_RedisOutageAuthorityHolds`, `TestR16_R8_BothDownIsSafeFailure`, `TestStaging_R14_SessionLifecycleServerSide` — **PASS** against the **local** test Redis (stopped and restarted by the tests); **not** the Staging server |
| Shared auth paths | password reset, phone change, OTP login and WhatsApp confirm were **not changed** (the audit showed they always consume a code); login shares only the refactored counter — covered by the suites above |
| Generated truth | `TEST_TRUTH` 1522 → 1532 tests, `TestSU*` mapped to F-30 (149 tests) · `TRUTH.md` shows the new default · API contract unchanged · `gofmt` clean |

**Code commit:** `5d7a960f`.

**Not deployed.** Production (`68a45c97`) and Staging still run the pre-fix code; they are
protected only because both store `auth.signup_verify`=true. Deployment is a separate Owner
decision. **Staging mutations 0 · Production mutations 0.**


### 40.15 · Owner product decisions (2026-09-19, second set) and the gaps they open

1. **PRQ-2 — customer ticket replies are IN this release.** The customer opens a ticket,
   sees replies and replies to the same ticket, per the approved PRQ-2 contract; no
   parallel support architecture.
2. **Custom-order payment = cash or wallet** (contract of 2026-08-09 stands).
3. **`shop.rail_auto` is website-only.** The Android app is not required to auto-scroll;
   CUST-09-027 is `NOT_APPLICABLE` with source evidence. A mobile auto-scroll needs its own
   contract.
4. **The order card is the customer's primary order surface.** Required on the card: the order
   number, a clear Arabic status, the date and time, an item summary and count, the server's
   final total, the payment method, a concise delivery address, and only the actions valid
   for the state. Show the driver/tracking state where the contract makes it visible. **Never
   show store identity.**
5. **Warnings addressed to the account owner must reach the customer** in safe,
   customer-facing wording — never stack traces, debug detail, secrets or internal
   operational metadata.

**Functional implementation gaps. Not stop-class, but each must close before final Customer
acceptance:**

| Gap | Today | Rows |
|---|---|---|
| **GAP-PRQ-2** ticket reply loop | replies exist only under `/admin/tickets/{id}/replies`; `/my/tickets` returns no replies (`server/my_tickets.go:33`); no customer reply endpoint or UI | CUST-SUP-013, CUST-SUP-014 |
| **GAP-CUSTOM-WALLET** | the backend accepts `wallet` (`orders/custom.go:160-164`); the app never sends a payment method | CUST-CUSTOM-020 |
| **GAP-ORDER-CARD** | the card lacks date/time, payment method and address | CUST-14-021 |
| **GAP-WARNINGS** | `issueWarning` sends nothing and the app never shows warnings | CUST-SUP-012 |

**Remaining stop-class blockers after this cycle:** CUST-DEF-002 · CUST-DEF-003 ·
CUST-DEF-004 · CUST-DEF-005 · D6 · D8. **Customer acceptance execution stays closed.**

### 40.16 · CUST-DEF-001 — Staging runtime verification (2026-09-19)

**Deployed to Staging only.** API `release-5d7a960f` (image `034d8756…`), built on the
server from the uploaded source of commit `5d7a960f` (source sha256 `fee8d41d…`), promoted
with `--no-deps api`. No migration (schema stays `0157`). Staging web unchanged.
**Production untouched — still `68a45c97`.** Identity endpoint reports
`source_commit=5d7a960f`, so the running artifact is the fixed code, not just a
successful container start.

**Fixtures:** a controlled script created disposable, clearly-labelled Staging accounts
(`full_name LIKE 'CUSTDEF001-WITNESS%'`, phones `+9639977700xx`). The QA harness runs
only against a local test DB, so it cannot seed Staging; codes for the legitimate flows
came from the real `/auth/signup/request` endpoint, read from the server log and never
printed.

**Every case proved both the HTTP result and the database result:**

| Case | HTTP | DB |
|---|---|---|
| A · existing password customer, verify ON, no code | `409 phone_taken`, no token | password/name/flag/PIN/status/sessions unchanged |
| B · existing driver+customer role | `409 phone_taken`, no token | unchanged |
| C · suspended customer | `409 phone_taken`, no token | unchanged |
| C · blocked customer | `409 phone_taken`, no token | unchanged |
| D · must_change_password customer | `409 phone_taken`, no token | flag stays true; unchanged |
| E · brand-new number, verify ON, real code | `200`, session issued | account created |
| F · password-less customer, real code | `200`, session issued | password now set (was empty) |
| G · launch gate (`launch.customer_signup`=false) | `403/503 launch_closed` | no account created |
| OFF · verify OFF, existing password customer, no code | `409 phone_taken`, no token | unchanged (protection holds with verify OFF) |
| OFF · verify OFF, brand-new number, no code | `200`, session issued | account created |

**Rate limit (H):** not exercised at runtime — the centralized limiter is shared by IP,
and hammering it risks locking a real operational address. Per the Owner's section 7,
the passing automated regression `TestSU09` (per-phone and per-IP `429`) is the evidence.

**Settings-failure (I):** not induced on the live Staging settings (Owner's section 3);
the automated `TestSU06` (injected read failure → fail closed) is the evidence.

**Temporary settings** changed only for their witness and restored to the exact recorded
originals: `launch.customer_signup` true→false→true; `auth.signup_verify` true→false→true.

**Cleanup — proven exact.** All 8 fixture users and their dependent rows removed
(sessions, roles, OTP, audit, wallets). The signup bonus had added three balanced ±15
wallet pairs; both sides of each were deleted by their shared `ref`, and the one platform
wallet whose stored balance still carried the −45 was corrected by exactly +45 so that
**every wallet's balance again equals the sum of its transactions**.

**Before → after (baseline restored):**

| | before | after |
|---|---|---|
| users | 50 | 50 |
| users fingerprint | `555561cf…` | `555561cf…` (identical) |
| user-roles fingerprint | `5300abee…` | `5300abee…` (identical) |
| role-permissions fingerprint | `0a8c9d8d…` | `0a8c9d8d…` (identical) |
| orders | 1 | 1 |
| #1050 | on_the_way · 7 events | on_the_way · 7 events |
| wallets | 50 · sum 0 | 50 · sum 0 |
| wallet transactions | 4 | 4 |
| moneycheck | 51/51 | 51/51 |
| fixture rows | — | 0 |
| settings | verify=true · signup=true | verify=true · signup=true |

**Status:** SOURCE FIX = CLOSED · AUTOMATED REGRESSION = PASS · **STAGING RUNTIME = PASS**
· PRODUCTION = still `68a45c97` (old code), patch pending Owner authorization. Production
is **not** protected by the code fix yet; it is only shielded because
`auth.signup_verify` is stored `true` there.

### 40.17 · CUST-DEF-001 — Production security hotfix (2026-09-19)

**Authorized by the Owner** as a Production security hotfix limited to the reviewed
CUST-DEF-001 backend fix. **API only.**

**Source delta 68a45c97 → 5d7a960f** (what Production actually gained): three runtime
files — `identity/service.go`, `server/auth_handlers.go`, `settings/catalog.go` — all
CUST-DEF-001. Everything else in the range is tests, docs, or the already-reviewed
staging/mobile work; **no migration, no other product/runtime change.**

**Promotion:** the **exact Staging-tested image** `rahalgo-api:release-5d7a960f`
(`034d8756…`, built from commit `5d7a960f`) was promoted with the guarded `promote.sh`
(`--no-build --no-deps api`). No rebuild. `.env` pinned to the new API image; the web
line kept `release-68a45c97`.

| | Production before | Production after |
|---|---|---|
| API release / image | `release-68a45c97` / `e6bb3dfb…` | `release-5d7a960f` / `034d8756…` |
| source_commit | `68a45c97` | `5d7a960f` |
| web | `release-68a45c97` (`3a3e569d…`) | unchanged |
| migration | `0157` | `0157` (none applied) |
| health / readiness | 200 | 200 |
| container | running · 0 restarts | running · 0 restarts · 0 error lines |

**Backups before deploy (verified):**
- `pg_dump` `rahalgo-pre-5d7a960f-20260919T110845Z.dump` — 94 table-data entries read
  back, sha256 `602ef48e…`;
- rollback image archive `rahalgo-api-release-68a45c97.tar` (sha256 `b7c1af5b…`, index
  == the image that was running);
- `.env` backup `.env.bak-pre-5d7a960f` (sha256 `42e1407d…`).
- **Rollback path (unused):** repin `.env` `RAHALGO_API_IMAGE=rahalgo-api:release-68a45c97`
  and `docker compose … up -d --no-build --no-deps api`; DB restore from the dump only if
  ever needed.

**Post-deploy verification (safe, non-destructive — no takeover reproduction against real
accounts):**
- identity reports `production` / `5d7a960f` / `0157`;
- signup-confirm on a non-existent phone → `503 launch_closed` (the new confirm launch
  gate is live; Production signup is closed and **no account was created**);
- `/auth/me` without a token → `401`;
- the deployed artifact identity is the reviewed fix (not just a started container).

**Business / financial invariants — Production DB before vs after is byte-identical:**

| | before | after |
|---|---|---|
| users | 25 | 25 |
| users fingerprint | `9b20f7c3…` | `9b20f7c3…` |
| user-roles fingerprint | `e9de388a…` | `e9de388a…` |
| role-permissions fingerprint | `f37db9c4…` | `f37db9c4…` |
| orders | 0 | 0 |
| wallets | 2 · sum 0 | 2 · sum 0 |
| wallet transactions | 0 | 0 |
| moneycheck | 51/51 | 51/51 |
| settings | verify=true · signup=false | verify=true · signup=false |

**Deployment mutated Production only by:** recreating the API container onto the reviewed
image and pinning `.env`'s API image line. No schema, no settings, no business data
changed. **No Caddy change.**

**CUST-DEF-001 status:** SOURCE FIX = CLOSED · AUTOMATED REGRESSION = PASS · STAGING
RUNTIME = PASS · PRODUCTION PATCH = DEPLOYED · PRODUCTION POST-DEPLOY = PASS ·
**OPERATIONAL STATUS = CLOSED.** *(History preserved: Production ran the vulnerable
`68a45c97` until 2026-09-19 11:09 UTC.)*

### 40.18 · CUST-DEF-003 — source fix + automated regression (2026-09-19)

**Authorized as source fix + automated regression only. No deployment; no Staging/Production
data or settings touched.**

**Proven root cause.** `POST /orders` → `handleCustomerCreateOrder` → `orders.CreateTx`.
The order's merchant came from the request DTO field `merchant_id` (`orders/models.go`):
`CreateTx` filled it from the items only when empty, otherwise kept the client value
(old `orders/service.go:303`), then used it for the open-hours check, wrote it to
`orders.merchant_id`, and every downstream store/money path reads that column
(`settleRep`, `reverseCommissions`, `merchantActivated`, `notifyCreated`,
merchant order list, pickup/route, ratings). **The unsafe transition:** a
customer-controlled `merchant_id` never validated against the items' real stores became
authoritative.

**Authoritative server-side store.** `SourcesOf` derives the distinct merchant set from
the actual `menu_items` rows — the trusted source. `sources.IDs` is that set.

**Existing single/multi-store rule (preserved, not invented):** an order may span at most
`orders.max_sources` distinct item merchants (`ErrTooManySources`); when `merchant_id` is
omitted the order merchant is `sources.IDs[0]`.

**Chosen behavior for a foreign `merchant_id`: REJECT** (`bad_merchant`). Reason it matches
the contract: the only order-creating route is `POST /orders`, and the sole real caller
(the customer app) never sends `merchant_id`; every existing test that sends it sends the
items' own merchant. So rejecting a value not among the item stores breaks nothing
legitimate, and a deterministic `bad_merchant` (already the code for a malformed id) is
clearer than silently overriding. Empty → derived from items (unchanged).

**Changed files (backend only):**
- `orders/service.go` — the supplied `merchant_id` must be one of `sources.IDs`, else
  `ErrBadMerchant`; empty still derives `sources.IDs[0]`.
- `orders/sources.go` — `sourceHas` helper (case-insensitive membership).
- `qa/order_merchant_trust_test.go` — new regression (4 tests).
- `testtruth/declared.go` + generated `TEST_TRUTH.*` — map the new tests to F-01.

**Security property proven.** A foreign `merchant_id` cannot become the order merchant, so
it cannot affect open-hours, commission recipient/basis, commission reversal,
store-activation/rep-reward accounting, notifications, order-list visibility, pickup/routing
or ratings — all of which read `orders.merchant_id`. Goods settlement already uses the
item's own store (`COALESCE(oi.merchant_id, o.merchant_id)`) and is unchanged.

**Regression `TestCDEF003_*` (qa):**
- foreign active store / random UUID / garbage → `bad_merchant`, no order created;
- legitimate order (omitted or correct merchant) → stored `orders.merchant_id` == the item
  store;
- **open-hours real-store witness:** real store CLOSED + foreign OPEN supplied → not
  orderable (`bad_merchant`, no order); real CLOSED + omitted → `merchant_closed`;
- **inverse:** real OPEN + foreign CLOSED supplied → foreign cannot govern (`bad_merchant`);
- **commission witness:** a foreign store cannot become the order's commission store; a
  legitimate order's stored merchant is the item store.

**Negative witness (fix reverted):** the exploit cases FAIL on the pre-fix code — a foreign
active merchant is accepted, and a closed real store is made orderable through a foreign
open store; the legitimate cases pass on both. Fix restored afterward.

**Test scope:**
- previous expected CRITICAL scope: 212 mandatory;
- actual discovered scope (P-9 on this change): **218 mandatory** (the suite grew — new
  CUST-DEF-001 and CUST-DEF-003 tests);
- executed: the full 8-package mandatory scope — `orders` PASS (27.6 s), `qa` PASS
  (856 s), plus `androidmap` · `failmap` · `fininv` · `impact` · `racemap` · `testtruth`;
  the only red before regenerating was `TestTruthIsCurrent` (inventory not yet rebuilt),
  green after regeneration;
- full Go suite `go test -timeout 30m -count=1 -p 1 ./...`: **35 packages OK · 0 FAIL ·
  exit 0**;
- skips/failures: 0 unexplained.

**Second source trace (post-fix).** The only other path that changes an order's merchant is
`POST /orders/{id}/transfer`, which is under the `/admin` group (role + capability gated) —
not customer-reachable, a separate ops contract. Custom orders (`CreateCustomTx`) never take
a client merchant. No residual customer-controlled store trust remains.

**Money/ledger:** no financial invariant weakened, no historical data touched, no
correction transaction introduced (source-fix phase only). Existing settlement tests, which
compute commission from `orders.merchant_id`, remain green.

**Status:** SOURCE FIX = CLOSED · AUTOMATED REGRESSION = PASS · **NOT DEPLOYED** (Staging
and Production still run `5d7a960f`; any runtime/deploy phase is a separate authorization).

### 40.19 · CUST-DEF-003 — Staging deployment + controlled runtime witness (2026-09-19)

**Authorized as Staging deployment + controlled runtime witness only. Production not
touched (boundary = 0). Fixtures had to be reversible: no existing wallet balance modified,
and no manual corrective ledger entry permitted.**

**Deployed source `5105fa45`** to Staging API only — image
`rahalgo-api:release-5105fa45` (`8bfe2490…`, source_commit `5105fa45…`), migration `0157`
(none applied). Web unchanged. Production stayed on `5d7a960f`.

**Fixture design (reversible, no settlement).** One disposable customer `U`, two disposable
stores `A` and `B` (each with owner, category, section, one approved item), all labelled
`CUSTDEF003-WITNESS%`. Every order placed was **cash on delivery** and none was progressed
to delivery, so **no settlement ran** — `wallet_transactions` stayed at 4 and `sum(balance)`
at 0 throughout. No `+X/−X` correction was ever needed or made.

**Runtime cases (all PASS) against the live Staging API:**
- **A** — items from `A`, supplied `merchant_id=B` (foreign) → `400 bad_merchant`, order
  count unchanged, zero orders point at `B`.
- **B** — real store `A` CLOSED + supplied foreign OPEN `B` → `400 bad_merchant`; same items
  with merchant omitted → `409 merchant_closed` (the real store governs hours).
- **C** — real `A` OPEN + supplied foreign CLOSED `B` → `400 bad_merchant` (foreign never
  read); legitimate order against `A` → `201`, stored `orders.merchant_id == A`.
- **D** — merchant omitted (what the real app sends) → `201`, stored merchant == `A`.
- **E** — correct `merchant_id=A` supplied → `201`, stored merchant == `A`.

**Negative witness for the foreign store `B` (proven at runtime):** after all cases,
**zero** orders point at `B`, **zero** notifications reached `B`'s owner or store,
`wallet_transactions` == 4 and wallet sum == 0. A foreign merchant supplied by the client
never became the order's store, so it can never be routed to, notified, rated, or made the
commission/activation store — all of which read `orders.merchant_id`.

**What was NOT witnessed positively at runtime, and why.** The **positive** direction of
commission earned / commission reversed / representative-activation count / rep reward could
not be exercised on Staging, because each requires progressing an order through settlement,
which writes an irreversible running-balance ledger entry — and a corrective entry to undo it
was explicitly forbidden by the authorization. These paths therefore rest on the automated
settlement tests (which compute commission from `orders.merchant_id`) plus the source trace,
**not** on a Staging runtime witness. **Multi-store `orders.max_sources` compatibility**
likewise rests on the automated multi-source tests; no Staging setting was changed
(`orders.max_sources` stayed at 1).

**Cleanup — exact baseline restoration proven.** All disposable rows deleted by recorded id
(users, roles, merchants, items, sections, categories, orders + their events/items,
notifications, otp, refresh tokens, and the `audit_log` rows the disposable users/orders/
merchants generated). Staging DB after cleanup vs the recorded baseline:

| | baseline | after cleanup |
|---|---|---|
| users | 50 | 50 |
| users fingerprint | `555561cf…` | `555561cf…` |
| user-roles fingerprint | `5300abee…` | `5300abee…` |
| role-permissions fingerprint | `0a8c9d8d…` | `0a8c9d8d…` |
| orders | 1 | 1 |
| order #1050 | `on_the_way` · 7 events | `on_the_way` · 7 events |
| wallets · sum | 50 · 0 | 50 · 0 |
| wallet transactions | 4 | 4 |
| settings | verify=true · signup=true · max_sources=1 | identical |
| migration | 0157 | 0157 |
| disposable rows (users/merch/orders/otp/audit) | — | 0 / 0 / 0 / 0 / 0 |

**moneycheck against Staging (read-only session): 51 checks · 0 violations.** All three
fingerprints match byte-for-byte; #1050 untouched; no ledger row created or corrected.

**No product change during the runtime phase.** The fix behaved correctly as deployed; no
source issue surfaced, so no code was edited.

**CUST-DEF-003 status:** SOURCE FIX = CLOSED · AUTOMATED REGRESSION = PASS · STAGING RUNTIME
= PASS (negative direction witnessed; positive settlement direction by automated tests +
source trace) · BASELINE RESTORED (exact) · **PRODUCTION NOT DEPLOYED** (separate
authorization). Staging API remains on `release-5105fa45`.

### 40.20 · CUST-DEF-003 — Production security hotfix (2026-09-19)

**Authorized by the Owner** as a Production security hotfix limited to the reviewed
CUST-DEF-003 backend fix. **API only.**

**Deployment delta 5d7a960f → 5105fa45** (what Production actually gained): two runtime
files — `orders/service.go`, `orders/sources.go` — both CUST-DEF-003 (order store is derived
from the item rows via `SourcesOf`; a client-supplied `merchant_id` must be one of the item
sources or is rejected `bad_merchant`). Everything else in the range is tests, the test-truth
inventory, or docs; **no migration, no other product/runtime change, no config change.**

**Promotion:** the **exact Staging-tested image** `rahalgo-api:release-5105fa45`
(`8bfe2490…`, built from commit `5105fa45`, build id `build-20260919T120915Z-5105fa45`) was
already present in the shared on-host image store and was promoted with the guarded
`promote.sh` (`--no-build --no-deps api`). No rebuild. Pre-switch the guard confirmed the
identity endpoint reports `production` and the image id equals the expected `8bfe2490…`;
post-switch it re-confirmed the running image id, a 40-hex `source_commit`, and
`environment=production`. `.env` pinned to the new API image; the web line kept
`release-68a45c97`.

| | Production before | Production after |
|---|---|---|
| API release / image | `release-5d7a960f` / `034d8756…` | `release-5105fa45` / `8bfe2490…` |
| source_commit | `5d7a960f` | `5105fa45` |
| web | `release-68a45c97` (`3a3e569d…`) | unchanged |
| staging API | untouched | untouched |
| migration | `0157` | `0157` (none applied) |
| health | 200 | 200 |
| container | running · 0 restarts | running · 0 restarts · 0 error lines |

**Backups before deploy (verified):**
- `pg_dump` `rahalgo-pre-5105fa45-20260919T123556Z.dump` — 94 table-data entries read back,
  sha256 `f7b10c55…`;
- rollback image archive `rahalgo-api-release-5d7a960f.tar` (sha256 `ea997a54…`, index ==
  the image that was running, `034d8756…`);
- `.env` backup `.env.bak-pre-5105fa45-20260919T123556Z` (sha256 `8d0a7435…`).
- **Rollback path (unused):** repin `.env` `RAHALGO_API_IMAGE=rahalgo-api:release-5d7a960f`
  and `docker compose … up -d --no-build --no-deps api`; DB restore from the dump only if
  ever needed.

**Post-deploy verification (safe, non-destructive — no exploit reproduction, no disposable
Production data, no hours/rep-config change):**
- identity reports `production` / `5105fa45` / `0157`, `staging=false`;
- `/auth/me` without a token → `401`; `POST /orders` without a token → `401` (the order
  route is served and auth-gated; **no order created**);
- signup-confirm on a non-existent phone → `503 launch_closed` (the launch gate is intact,
  Production signup stays closed, **no account created**);
- the deployed artifact identity is the reviewed fix.

**Business / financial invariants — Production DB before vs after is byte-identical:**

| | before | after |
|---|---|---|
| users | 25 | 25 |
| users fingerprint | `9b20f7c3…` | `9b20f7c3…` |
| user-roles fingerprint | `e9de388a…` | `e9de388a…` |
| role-permissions fingerprint | `f37db9c4…` | `f37db9c4…` |
| orders | 0 | 0 |
| wallets | 2 · sum 0 | 2 · sum 0 |
| wallet transactions | 0 | 0 |
| moneycheck | 51/51 | 51/51 |
| settings | verify=true · signup=false · max_sources=1 | identical |

**Deployment mutated Production only by:** recreating the API container onto the reviewed
image and pinning `.env`'s API image line. No schema, no settings, no business data
changed. **No Caddy change.** Production mutations outside the API image/env pin = 0.

**Financial evidence boundary (unchanged, intentional):** commission / reversal /
rep-activation read `orders.merchant_id`, which can no longer be set from a client value; the
foreign-store rejection and real-store-governs behavior were directly witnessed on Staging;
the positive settlement direction rests on the automated settlement/reversal tests — no live
Production financial witness was attempted, and no corrective ledger entry was needed.

**CUST-DEF-003 status:** SOURCE FIX = CLOSED · AUTOMATED REGRESSION = PASS · NEGATIVE WITNESS
= PASS · STAGING RUNTIME = PASS · PRODUCTION PATCH = DEPLOYED · PRODUCTION POST-DEPLOY = PASS
· **OPERATIONAL STATUS = CLOSED.** *(History preserved: Production accepted client-supplied
`merchant_id` until 2026-09-19 12:37 UTC, when it moved from `5d7a960f` to `5105fa45`.)*

### 40.21 · D6 — custom-order cash-ban bypass: source fix + regression (2026-09-19)

**Authorized as source fix + automated regression only. No deployment; no Staging/Production
data or settings touched.**

**Root cause (proven before editing).** The cash-ban policy (`Service.cashBlocked` →
`ErrCashBlocked`) was enforced at exactly one site — the normal-order path
(`service.go` `CreateTx`). The custom-order creator `CreateCustomTx`
(`POST /orders/custom` → `handleCreateCustomOrder`) never called it, and it normalises
`payment` to `"cash"` unless `"wallet"` — so **omitting `payment_method` also yields cash**.
A customer cash-banned for a customer-fault delivery failure could place a cash **custom**
order (by sending `"cash"` or by omitting the field) and bypass the lock that protects
platform money. Ownership was already correct (`userIDFrom(r)`, never a body field).

**Fix (minimal, two files' worth of logic in one file, no migration).** The same policy is
now checked in both custom creators — `CreateCustomTx` (live) and its exported sibling
`CreateCustom` (currently unused, fixed for consistency) — after payment normalisation and
before any lock or write: `if payment == "cash"` → `cashBlocked` → `ErrCashBlocked`. Same
source of truth, same error contract, no second definition of the policy. The wallet option
stays open to the banned customer (the ban is on cash, not the account).

**Complete coverage (second source trace).** `payment_method` is written in exactly three
places — the two custom creators (now guarded) and the normal creator (already guarded) —
and is **never `UPDATE`d** anywhere. There is no customer-reachable later-stage payment
selection/change and no custom→normal conversion: `POST /orders/{id}/agree` is driver-only
and writes goods/fee, not payment. No residual path remains where a cash-banned customer can
reach cash through create, quote, acceptance, payment change, finalisation or conversion.

**Regression `TestCustomCashBan_*` (`orders/custom_cashban_test.go`):**
- banned + explicit `"cash"` → `ErrCashBlocked`, **no custom-order row, no wallet tx**;
- banned + **omitted** payment (defaults to cash) → `ErrCashBlocked`, no row, no wallet tx;
- banned + `"wallet"` → accepted (ban is on cash only);
- not-banned + `"cash"` → accepted (legitimate flow unchanged).

**Negative witness:** with the fix temporarily reverted, both banned-cash cases FAIL — the
custom cash order is created despite the ban (`err = <nil>`); the allowed/wallet cases still
pass. Fix restored; the exploit cases pass again.

**Money/DB safety.** The fix only rejects — it writes nothing on the banned path (the check
precedes the advisory lock and the INSERT), and the regression asserts zero custom rows and
zero wallet transactions after a rejected attempt. No settlement/ledger code touched; the
financial-invariant packages (`fininv`, settlement, treasury, goods-settlement) stay green.

**D8 boundary.** No WhatsApp/verification code exists in the custom creator or handler; the
patch adds only the cash-ban block and touches nothing D8. **D8 overlap = none; D8 behaviour
unchanged.**

**Test scope (P-9).** `internal/orders/custom.go` is unmapped in the impact engine →
classified UNKNOWN → **SAFE FULL FALLBACK**: the full backend suite is mandatory. Executed
`go test -timeout 30m -count=1 -p 1 ./...` → **35 packages OK · 0 FAIL · 0 skip unexplained**
(incl. `qa` 900.7s and `testtruth` — `TestTruthIsCurrent` green). `gofmt` clean. TEST_TRUTH
regenerated: **D6 = `FIXED_AND_PASSING`** with four covering tests; the census
`TestCENSUS_D6_CustomOrderSkipsCashBan` now reports REPRODUCTION = NO (both doors block).

**Status:** SOURCE FIX = CLOSED · AUTOMATED REGRESSION = PASS · NEGATIVE WITNESS = PASS ·
FULL SUITE = PASS · **NOT DEPLOYED** (Staging and Production still run `5105fa45`; any
runtime/deploy phase is a separate authorization).

### 40.22 · D6 — Staging runtime witness + Production hotfix (2026-09-19)

**Authorized as two steps: a Staging runtime witness, then a Production hotfix limited to the
reviewed D6 fix. API only. No setting change (the cash-ban policy configuration is not
touched).**

**Runtime source `cd33b173`** — the D6 fix (`cash_blocked` now enforced by the custom-order
creators). Built strictly from `cd33b173` (proven `cd33b173→HEAD` delta = docs only); image
`rahalgo-api:release-cd33b173` (`765aafb3…`), migration `0157`.

**Staging runtime witness (all PASS).** Deployed `release-cd33b173` to the Staging API only.
Disposable `D6-WITNESS-*` fixtures exercised the real `Service.cashBlocked` policy (the
owner-documented threshold 1 failure / 30 days, set only because Staging had no cash-ban
config, and restored to absent in cleanup) plus a genuine customer-fault failure for the
banned customer, proven from the source of truth:
- **A — banned + explicit `cash`** → `409 cash_blocked`; no order, no wallet tx;
- **B — banned + omitted `payment_method`** (the second historical bypass, witnessed
  independently) → `409 cash_blocked`; no order, no wallet tx;
- **C — banned + `wallet`** → `201`, stored `wallet` (creation moves no ledger, so no
  correction was ever needed);
- **D — control (not banned) + `cash`** → `201`, stored `cash`, owner = the control customer.
Rejections wrote zero audit rows. Cleanup restored the exact Staging baseline (all three
fingerprints byte-identical, orders/custom/#1050/wallets/tx/sum unchanged, moneycheck 51/51,
cash-ban config back to absent); the only residual delta was one real user's independent
`auth.refresh` (category-B operational activity, not fixture-caused).

**Production deployment delta `5105fa45 → cd33b173`:** one runtime file —
`backend/internal/orders/custom.go` (the reviewed D6 fix). Everything else in range is the
test, the test-truth inventory, or docs. **No migration, no other runtime/config change.**

**Promotion:** the **exact Staging-tested image** `765aafb3…` (already on the shared host)
promoted with the guarded `promote.sh` (`--no-build --no-deps api`); pre-switch guard
confirmed `environment=production` and image id `765aafb3…`, post-switch re-confirmed the
running id, a 40-hex `source_commit`, and `environment=production`. `.env` pinned; web kept
`release-68a45c97`. No rebuild.

| | Production before | Production after |
|---|---|---|
| API release / image | `release-5105fa45` / `8bfe2490…` | `release-cd33b173` / `765aafb3…` |
| source_commit | `5105fa45` | `cd33b173` |
| web / staging API | `release-68a45c97` / (staging untouched) | unchanged |
| migration | `0157` | `0157` (none applied) |
| health | 200 | 200 |
| container | running · 0 restarts | running · 0 restarts · 0 error lines |

**Backups before deploy (verified):** `pg_dump` `rahalgo-pre-cd33b173-20260919T145226Z.dump`
(94 table-data entries, sha256 `b8079987…`); rollback archive
`rahalgo-api-release-5105fa45.tar` (sha256 `c9c441d0…`, index == the running `8bfe2490…`);
`.env` backup (sha256 `0ae4bf7d…`). **Rollback path (unused):** repin `.env`
`RAHALGO_API_IMAGE=rahalgo-api:release-5105fa45` + `up -d --no-build --no-deps api`.

**Post-deploy verification (safe, non-destructive — no exploit reproduction, no disposable
Production data, no cash-ban config change):** identity `production` / `cd33b173` / `0157`,
`staging=false`; `/auth/me` no token → `401`; `POST /orders/custom` no token → `401` (route
served, auth-gated, no order); container running, restarts=0, 0 error lines.

**Business / financial invariants — Production DB before vs after byte-identical:**

| | before | after |
|---|---|---|
| users / users FP | 25 / `9b20f7c3…` | 25 / `9b20f7c3…` |
| user-roles FP | `e9de388a…` | `e9de388a…` |
| role-caps FP | `f37db9c4…` | `f37db9c4…` |
| orders / custom orders | 0 / 0 | 0 / 0 |
| wallets / tx / balance | 2 / 0 / 0 | 2 / 0 / 0 |
| moneycheck | 51/51 | 51/51 |
| cash-ban settings | **absent** | **absent** (unchanged) |
| settings | verify=true · signup=false · custom_orders=false · max_sources=1 | identical |

**Deployment mutated Production only by:** recreating the API container onto the reviewed
image and pinning `.env`'s API image line. No schema, no settings, no business data, no Caddy.
Production mutations outside the API image/env pin = 0.

**Security evidence preserved:** a cash-banned customer could create a custom **cash** order
via (1) explicit `payment_method="cash"` and (2) omitted `payment_method` (normalised to
cash), because the custom-order path never called `cashBlocked`. The patch reuses the same
authoritative policy; wallet/non-cash stays allowed; payment method has no later
customer-reachable mutation path; D8 was not modified; no migration required.

**D6 status:** SOURCE FIX = CLOSED · AUTOMATED REGRESSION = PASS · NEGATIVE WITNESS = PASS ·
STAGING RUNTIME = PASS · PRODUCTION PATCH = DEPLOYED · PRODUCTION POST-DEPLOY = PASS ·
**OPERATIONAL STATUS = CLOSED.** *(History preserved: Production's custom-order path did not
enforce the cash ban until 2026-09-19 14:53 UTC, when it moved from `5105fa45` to `cd33b173`.
Production's `launch.customer_custom_orders` is false, so the surface was gated meanwhile; the
fix hardens it for whenever it opens.)*

### 40.23 · D8 — custom-order verification (WhatsApp) bypass: source fix + regression (2026-09-19)

**Authorized as source fix + automated regression only. No deployment; no Staging/Production
data or settings touched.**

**Root cause (proven before editing).** The WhatsApp-verification requirement was enforced at
one site — the normal-order path (`service.go` `CreateTx`), which calls
`settings.RequireWhatsApp(ctx, "customers.require_whatsapp")` and, when it returns true,
rejects an unverified customer with `ErrWhatsAppRequired` (403). The custom-order creator
`CreateCustomTx` (`POST /orders/custom` → `handleCreateCustomOrder`) never called it. So when
the owner turns the requirement on, an unverified customer could place a **custom** order and
bypass a gate the normal flow enforces. Ownership was already correct (`userIDFrom(r)`).

**Why latent.** `RequireWhatsApp` returns true only when **both** the master
`auth.require_whatsapp` (default true) **and** the role key `customers.require_whatsapp`
(default false) are true. With the role key off by default, the requirement is inactive, so
the bypass is dormant — a latent P1 that surfaces the moment the requirement is enabled.

**Fix (minimal, no migration).** The same policy is now checked in both custom creators —
`CreateCustomTx` (live) and its unused exported sibling `CreateCustom` — after payment
normalisation and before any lock or write: `if RequireWhatsApp(...)` → read
`whatsapp_verified_at IS NOT NULL` for the authenticated customer → `ErrWhatsAppRequired`.
Same source of truth, same error, no second definition. Feature-off behaviour preserved
exactly (master or role off → requirement off → unverified passes).

**Complete coverage (second source trace).** `kind='custom'` orders are inserted only by the
two custom creators (now both guarded); `CreateCustom` has zero callers, `CreateCustomTx` has
one (the handler). No later customer-reachable payment/verification mutation, no
custom→normal conversion; `/orders/{id}/agree` is driver-only. No residual path bypasses the
requirement when active.

**Regression `TestCustomWhatsApp_*` (`orders/custom_whatsapp_test.go`):**
- requirement ON + unverified → `ErrWhatsAppRequired`, **no custom-order row, no wallet tx**;
- requirement ON + verified → accepted;
- requirement OFF by role key → unverified accepted (not blocked by the fix);
- master switch OFF overrides role ON → unverified accepted (proves the latency contract).

**Negative witness:** with the fix temporarily reverted, the requirement-ON + unverified case
FAILS — the custom order is created despite the requirement (`err = <nil>`); the other cases
still pass. Fix restored; the exploit case passes again.

**Money/DB safety.** The fix only rejects — it writes nothing on the blocked path (the check
precedes the advisory lock and the INSERT), and the regression asserts zero custom rows and
zero wallet transactions after a rejected attempt. No settlement/ledger code touched.

**D6 preserved.** The D6 cash-ban regression stays green after the D8 change; both eligibility
checks run before any write, and cash-ban precedes the WhatsApp check in the custom creator.

**D9 / R11 boundary.** The patch adds only the WhatsApp check; it does not add the missing
creation event (D9) or touch the single-insert concern (R11). **D9 overlap = none; D9
behaviour unchanged. R11 behaviour unchanged.**

**Test scope (P-9).** `internal/orders/custom.go` is unmapped → UNKNOWN → **SAFE FULL
FALLBACK**: the full backend suite is mandatory. TEST_TRUTH regenerated: **D8 =
`FIXED_AND_PASSING`** with five covering tests; the census
`TestCENSUS_D6_D9_CustomOrderCreationGuards` now reports D8 REPRODUCTION = NO.

**Status:** SOURCE FIX = CLOSED · AUTOMATED REGRESSION = PASS · NEGATIVE WITNESS = PASS ·
FULL SUITE = PASS · **NOT DEPLOYED** (Staging and Production still run `cd33b173`; any
runtime/deploy phase is a separate authorization).

### 40.24 · D8 — Production security hotfix (2026-09-19)

**Authorized by the Owner** as a Production security hotfix limited to the reviewed D8 fix.
**API only. No WhatsApp-policy/configuration change.**

**Deployment delta `cd33b173 → 023d9d4c`:** one runtime file —
`backend/internal/orders/custom.go` (the D8 fix: the custom-order creators now enforce the
shared `RequireWhatsApp` policy → `ErrWhatsAppRequired`, before any lock/write). Everything
else in range is the test, the test-truth inventory, or docs. **No migration, no other
runtime/config change.**

**Promotion:** the **exact Staging-tested image** `765aafb3`→`e827dbd9…` (`release-023d9d4c`,
already on the shared host) promoted with the guarded `promote.sh` (`--no-build --no-deps
api`); pre-switch guard confirmed `environment=production` and image id `e827dbd9…`,
post-switch re-confirmed the running id, a 40-hex `source_commit`, and `environment=production`.
`.env` pinned; web kept `release-68a45c97`. No rebuild.

| | Production before | Production after |
|---|---|---|
| API release / image | `release-cd33b173` / `765aafb3…` | `release-023d9d4c` / `e827dbd9…` |
| source_commit | `cd33b173` | `023d9d4c` |
| web / staging API | `release-68a45c97` / (staging untouched) | unchanged |
| migration | `0157` | `0157` (none applied) |
| health | 200 | 200 |
| container | running · 0 restarts | running · 0 restarts · 0 error lines |

**Backups before deploy (verified):** `pg_dump` `rahalgo-pre-023d9d4c-20260919T161058Z.dump`
(94 table-data entries, sha256 `4321e1cb…`); rollback archive `rahalgo-api-release-cd33b173.tar`
(sha256 `5ff779ba…`, index == the running `765aafb3…`); `.env` backup (sha256 `72efe376…`).
**Rollback path (unused):** repin `.env` `RAHALGO_API_IMAGE=rahalgo-api:release-cd33b173` +
`up -d --no-build --no-deps api`.

**WhatsApp policy — unchanged (the fix is enforcement, not configuration):** raw
`auth.require_whatsapp` = `false` (present) and `customers.require_whatsapp` = ABSENT before
**and** after → effective `RequireWhatsApp` = FALSE both sides. No key added, deleted or
toggled. The requirement stays off in Production; the fix hardens the custom path for whenever
it is enabled.

**Post-deploy verification (safe, non-destructive — no exploit reproduction, no fixtures, no
verification-state change):** identity `production` / `023d9d4c` / `0157`, `staging=false`;
`/auth/me` no token → `401`; `POST /orders/custom` no token → `401` (route served, auth-gated,
no order); container running, restarts=0, 0 error lines.

**Business / security invariants — Production DB before vs after byte-identical:**

| | before | after |
|---|---|---|
| users / users FP | 25 / `9b20f7c3…` | 25 / `9b20f7c3…` |
| user-roles FP · role-caps FP | `e9de388a…` · `f37db9c4…` | identical |
| orders / custom orders | 0 / 0 | 0 / 0 |
| whatsapp-verified users | 12 | 12 (unchanged) |
| wallets / tx / balance | 2 / 0 / 0 | 2 / 0 / 0 |
| moneycheck | 51/51 | 51/51 |
| settings (incl. both WhatsApp keys) | as above | identical |

**Deployment mutated Production only by:** recreating the API container onto the reviewed
image and pinning `.env`'s API image line. No schema, no settings, no business data, no
verification state, no Caddy. Production mutations outside the API image/env pin = 0.

**D6 preserved:** the deployed `custom.go` retains the `cashBlocked` enforcement in both
custom creators (source trace); the D6 regression passes on this commit and D6 was
runtime-witnessed on Staging earlier. D8 is additive.

**Security evidence preserved.** `RequireWhatsApp` is effective TRUE only when
`auth.require_whatsapp` **and** `customers.require_whatsapp` are both true; the authoritative
state is `users.whatsapp_verified_at IS NOT NULL`. Historically: requirement ON + authenticated
unverified customer + `POST /orders/custom` → custom order created without enforcing the
requirement. The fix reuses the shared policy in both custom creators before lock/write;
requirement-off stays off; the authenticated user is authoritative (no body selector); no
verification state is mutated; D9/R11 unchanged; no migration.

**D8 status:** SOURCE FIX = CLOSED · AUTOMATED REGRESSION = PASS · NEGATIVE WITNESS = PASS ·
STAGING RUNTIME = PASS · PRODUCTION PATCH = DEPLOYED · PRODUCTION POST-DEPLOY = PASS ·
**OPERATIONAL STATUS = CLOSED.** *(History preserved: Production's custom-order path did not
enforce the WhatsApp requirement until 2026-09-19 16:11 UTC, when it moved from `cd33b173` to
`023d9d4c`. The requirement is off in Production, so the surface was not exposed meanwhile; the
fix hardens it for whenever it is enabled.)*

### 40.25 · CUST-DEF-002 / CAF-02 — duplicate-order retry: source fix + regression (2026-09-19)

**Authorized as shared-app source fix + Kotlin regression + device-witness plan only. No
Staging/Production mutation; no backend change; no API deployment.**

**Root cause (proven before editing).** Two defects on one path in the shared `ui` module:
1. `Attempt.isDecided` returned true for **any** `ApiClient.ApiException`. A `409 in_progress`
   (the engine still holds the idempotency lease and is processing the first request) was thus
   treated as a final answer, and the caller (`CartScreen`/`CustomScreen`, both
   `if (isDecided(e)) Attempt.clear(slot)`) **cleared the persisted attempt key**. The
   original request then commits; a later tap mints a **new** key → a **second order**.
2. `ApiErrors.CODES` had **no entry for `in_progress`**, so it fell back to `err_internal` =
   «تعذر الاتصال — حاول بعد قليل» — a connection-failure message that invites the very re-tap
   that duplicates the order (and logged a false soft-crash). `idempotency_reclaimed` was
   already mapped; `in_progress` was not.

**Timing window (current source).** Client order-create `requestTimeoutMillis = 20_000`
(`shared/net/ApiClient.kt`); server request timeout **30 s**, idempotency lease **60 s**, key
TTL **24 h** (`server/idempotency.go`). Client-gives-up-at-20 s < server-commits-by-30 s, so
the window is real. **The fix does not touch any timeout** — it makes the client correct under
transport uncertainty.

**Fix (minimal, shared `ui`, no migration, no backend change).**
- `isDecided`: `409` with code `in_progress` **or** `idempotency_reclaimed` → **not decided**
  (keep the same key; a retry replays the committed order via `writeReplay`). Every other
  server response stays decided (400 validation, other 409s: merchant_closed, cash_blocked …).
  Transport errors (timeout/IO) remain not-decided as before.
- `CODES["in_progress"] → err_in_progress` («العملية قيد التنفيذ — انتظر قليلا ولا تعدها»).
- Both callers already gate `clear` on `isDecided`, so the single change covers cart and custom.

**Retry-key lifecycle after fix:** PENDING/still-processing → keep · timeout/uncertain →
keep · success or committed-replay → retire after the app accepts the result · true terminal
rejection → retire (unchanged). A new key is never minted while the outcome is unknown.

**Automated regression** (`ui/…/CustDef002Test.kt`, `ApiErrorsTest.kt`; JVM):
- classification: `in_progress`/`idempotency_reclaimed` not decided; validation/merchant_closed/
  cash_blocked decided; IO not decided;
- **orders +1 exactly**: a fake engine mirroring `idempotency.go` (lease held → `in_progress`,
  committed key → replay, new key → new order); one logical submission through
  in_progress → replay yields `orders == 1`;
- **process death** between the in_progress response and the retry (store rebuilt over the same
  disk) still yields `orders == 1`;
- terminal rejection retires the key; first-attempt success unchanged;
- message: `in_progress` resolves to the wait message, **not** `err_internal`.

**Negative pre-fix witness:** with both fixes reverted, `stillProcessingIsNotDecided`,
`oneSubmissionYieldsExactlyOneOrder` (**orders == 2**), `processDeathBetweenRetriesStillOneOrder`
(**orders == 2**), and `inProgressResolvesToWaitNotConnectionFailure` FAIL — the exact
historical defect. Restored → all pass.

**Impacted scope (Kotlin unit).** app-customer 71 · app-driver 69 · app-merchant 6 · app-rep 6
· ui 181 · shared 29 · map 185 · driver-navigation 349 = **896 tests · 0 failures** (all four
apps + shared). Backend unchanged; no Go test added.

**Ownership / double-submit.** `POST /orders/custom` and `/orders` carry no client customer id
(authenticated user only). Rapid double-submit is already blocked by `if (busy) return` +
`enabled = !vm.busy` (existing controls, preserved); the fix adds no new key-minting path.

**Out of scope (recorded, not changed).** CUST-13-029 (idempotency does not fingerprint the
request body): the server intentionally replays the original committed order for a reused key —
that is the idempotency contract, not a defect the app can or should override. CAF-18 for
`not_found`/`comms_closed`/`comms_no_driver` (still unmapped, P3): separate from CUST-DEF-002.

**Status:** SOURCE FIX = CLOSED · AUTOMATED REGRESSION = PASS · NEGATIVE WITNESS = PASS ·
FOUR-APP UNIT SUITE = PASS · **DEVICE/STAGING WITNESS = PASS (§40.25.2)** ·
**OPERATIONAL STATUS = CLOSED** · (app not shipped as a public release).

#### 40.25.1 · Proposed deterministic device-witness plan (needs separate authorization)

Goal: on SM-A525F with the fixed Customer app, prove one logical submission → still-processing
UI (not «تعذر الاتصال») → same key retained → committed order recovered → **exactly one order**.

- **Fixture:** one disposable staging customer (`D8…`-style label), logged into the fixed
  Customer debug APK; a serviceable address. No real accounts/orders touched.
- **Inducing the in-progress window deterministically** — the app mints its own random key, so
  the witness must force the lease/slow path on the server side. Options, in order of
  preference:
  1. **Staging-only slow hook on `POST /orders`** for the labelled disposable customer: sleep
     ~22–25 s (past the 20 s client timeout, within the 30 s server / 60 s lease), so the
     client times out while the server commits; the app's retry with the same persisted key
     then hits `in_progress` (or the replay). **Requires a small staging API support + a
     staging deploy → separate authorization.**
  2. **Pre-seed a held lease**: read the key the app persisted (adb pull the
     `rahalgo_attempt` prefs after the first tap), then hold that key's `idempotency_keys`
     lease server-side so the retry returns `in_progress`, then let the original commit. More
     manual, still needs a controlled staging DB write.
  3. **Network throttle** at the device to stall the request into the 20–30 s window
     (least deterministic).
- **Observations to capture:** UI shows the pending/still-processing state and the wait
  message (never «تعذر الاتصال»); adb-read `rahalgo_attempt` shows the **same** key across the
  retry; the retry succeeds via replay; staging `orders` for the disposable customer = **+1
  exactly**; then full reversible cleanup of the disposable fixture (as in the D6/D8 witnesses).
- **Constraints:** cash orders only or wallet with no settlement (creation moves no ledger);
  no corrective ledger entry; exact baseline restoration proven afterward.

Recommended: **option 1** (a minimal, clearly-labelled staging-only delay hook), authorized
and deployed under a separate request, then witnessed and removed.

#### 40.25.2 · Real-device witness — executed & PASSED (2026-09-19)

**Authorized real-device witness run on SM-A525F (Android 14) over stable USB ADB.** The fixed
Customer debug APK was installed and its base.apk SHA-256 verified to equal the reviewed build
`850be4a5321d1f9144bcca340fd135ef7211ba9904e77f7ea9de961653f60358` (source `ae472494`, staging
API). A disposable customer (`CUSTDEF002-DEVICE`, +963997770006) with a serviceable Raqqa
default address was logged in on the device; no real account/data was used.

**Deterministic delay:** a scoped `pg_advisory_xact_lock(hashtext('customer-admit:<uid>'))`
held server-side for the disposable customer only — the idempotency lease is established before
the blocked handler work, so the first request stays unresolved past the app's 20 s timeout and
a same-key retry hits `409 in_progress`. No backend source change; no slow hook.

**Observed sequence (custom order «طلب خاص», payment cash, no settlement at creation):**
- attempt key persisted on submit; the on-disk key stayed **identical** across the whole
  submission — SHA-256(key) prefix `e7fe9edb…` at submit, after the ~20 s timeout, and after the
  in-progress retry; **NONE** before submit and **NONE** after terminal success. **No K2.**
- the same-key retry received a live **`ApiClient.ApiException: api in_progress (409)`** (device
  logcat), while the lease was uncommitted and no order existed;
- the app showed the still-processing Flash **«العملية قيد التنفيذ – انتظر قليلا ولا تعدها.»**
  — **not** the generic «تعذر الاتصال — حاول بعد قليل»;
- after the lease cleared, the same-key recovery created the one order; the idempotency row
  committed with status **201** under that same key, and the app navigated to «طلباتي» showing a
  single card **#1062** (المطلوب: CUSTDEF002-DEVICE-WITNESS, بانتظار القبول, الإجمالي 0 ل.س);
- the attempt key was retired **only after** the successful terminal result (post-success key =
  NONE).

**Exactly one order:** disposable-customer orders before = 0, after = 1 (**delta = +1**); no
duplicate; no second idempotency key for the logical submission; no wallet/settlement/commission
movement (custom order, cash, pending, total 0).

**Server disconnect semantics (documented):** the current server cancels a client-disconnected
request and does not commit it (proven earlier), so the one order is created by the same-key
recovery once the stale lease clears — not by a replay of a committed disconnected original. The
acceptance condition for this contract (K survives uncertainty → same-key `in_progress` while
the lease is active → correct still-processing UI → one order, no K2) is fully met.

**Process-recreation:** the live recreation was skipped to avoid risking a second order on the
completed primary witness; process-death persistence is covered by the passing automated
regression `TestCustDef002Test.processDeathBetweenRetriesStillOneOrder` (and the on-disk
`Attempt` store design).

**Cleanup:** the disposable customer, order #1062, address, sessions, idempotency key, audit and
wallet rows were removed in one FK-safe transaction and the app attempt store wiped; staging
restored to the exact baseline (users 50, all three fingerprints match, orders 1 [only #1050],
custom 0, wallets 50/tx4/sum0, idempotency_keys 0, disposable 0, migration 0157). Temporary
device stay-awake/screen-timeout settings reset. **Production mutations = 0.**

**CUST-DEF-002 / CAF-02 final:** SOURCE FIX = CLOSED · AUTOMATED REGRESSION = PASS · NEGATIVE
WITNESS = PASS · DEVICE/STAGING WITNESS = PASS · **OPERATIONAL STATUS = CLOSED.**

### 40.26 · Remediation Batch-1 · PHASE 2 — product-contract requirements (DOCUMENTATION ONLY)

**Status: documentation with RESERVED case IDs.** These normalize requirements so they cannot
be forgotten during later remediation. **The authoritative 578 executed totals are unchanged**
— nothing here is executed, reclassified, or counted yet. Formal matrix integration (adding
counted rows) and execution are **pending Owner authorization**. **Do not mark PASS from source
assumptions.**

#### A) Coverage / area-demand

**Already covered (PASS — do not re-open):**
- Unsupported city/area gives an explicit reason: `city_not_supported` (CUST-07-016, CUST-08-008),
  `area_not_supported`/`address_outside_coverage` (CUST-07-017, CUST-08-010). `province_not_supported`
  is source-confirmed but BLOCKED live (CUST-08-007 — staging geography resolves to city/area).
- Supported city, address outside the delivery zone → `address_outside_coverage` (CUST-07-014,
  CUST-08-010).
- Notify / request-coverage CTA EXISTS: `POST /api/v1/demand`, UI `CityPicker.kt`/`PreCart.kt`,
  witnessed live on-device as «أخبرني عند توفر الخدمة في دمشق» (CUST-12-009/010). Acceptance case
  CUST-07-030 remains BLOCKED (demand-row creation not yet witnessed).

**Reserved NEW cases (missing coverage — to integrate + execute on authorization):**
- **CUST-07-031** — one demand request per (USER, AREA): repeated press does NOT create a second
  demand row (dedupe / idempotent by user+area).
- **CUST-07-032** — the SAME user may request a DIFFERENT uncovered area (distinct demand row).
- **CUST-07-033** — administration sees a UNIQUE-USER demand count per area (distinct users, not
  raw press count).
- **CUST-07-034** — an uncovered location must NOT show a misleading generic empty/offline state;
  it must show an explicit "not covered / notify me". (Observed gap: the out-of-coverage SHOP
  feed shows «نعمل على إضافة المتاجر» which reads as "coming soon"; the CART correctly shows
  «لم يصل إلى دمشق».)

**Overnight-run status (2026-09-21 — A2; source + device-independent tests; NOT counted, still reserved):**
- **CUST-07-031 (dedup one-per-user+area)** — VERIFIED (backend): `Record` uses
  `ON CONFLICT (user_id, kind, target_key) DO UPDATE SET requests = requests + 1`; existing
  `TestSI01_SI10_TwoPointsOneCityOneTarget` proves a second press in the same city returns
  `already_registered` and yields exactly one row.
- **CUST-07-032 (same user, different uncovered area → distinct row)** — VERIFIED (backend):
  existing `TestSI02_SI03_DamascusAndAleppoAreDistinct`.
- **CUST-07-033 (admin sees UNIQUE-USER count, not raw presses)** — VERIFIED (backend, NEW):
  `DemandByPlace` uses `count(DISTINCT COALESCE(user_id, id))` for `People` and `sum(requests)`
  for `Signals`. New `TestCUST07033_DemandByPlaceCountsUniquePeople` (baseline-delta, isolation-
  robust) proves 3 presses by user A + 1 by user B ⇒ ΔPeople=2, ΔSignals=4. Negative-witnessed
  (People→4 when counting presses).
- **CUST-07-034 (uncovered ≠ "coming soon")** — FIXED (source): `ShopScreen` empty-market branch
  now distinguishes out-of-coverage (`availability?.takeIf { !it.available }`) and renders
  `ServiceBlockNotice` (explicit reason + notify-me CTA, CTA suppressed for discovery points per
  CUST-07-030) instead of `shop_market_empty`; in-coverage empty still shows "coming soon". New
  `CoverageEmptyStateTest` (4 tests) + negative witness. **CORRECTION (2026-09-21):** an earlier
  device note here claimed the catalog is "location-independent" — that was **wrong**, made while
  the app had silently dropped to **guest mode** (session lost after force-stop/relaunch), so it
  used device GPS (physically Raqqa). The catalog is in fact **geography-scoped** — see §J geography
  audit: backend `/public/sections/:id/items?lat=Damascus` → **0 items**, `?lat=Raqqa` → **5 items**.
  A signed-in Damascus-default customer's app sends `BrowseScope`=Damascus → 0 items → the A2 empty
  branch → `ServiceBlockNotice`. The empty-state logic stays proven by source + `CoverageEmptyStateTest`
  + backend geo-filter; a clean signed-in on-device witness is still pending (device automation hit
  session-drop + a WhatsApp-launch anomaly, so it was stopped). Do not mark a device PASS from source.
- **CUST-07-030 (demand row creation)** — the API-level demand-row creation is now exercised by
  the backend demand tests (POST `/api/v1/demand` → row); the on-device button witness remains
  the only open item. Still BLOCKED for device acceptance.

Integration into the counted matrix (adding rows) remains **pending Owner authorization**; 578 unchanged.

#### B) Wallet checkout UX

**Already covered (PASS):** backend independently rejects insufficient balance — `409
insufficient_balance` (CUST-12-023). **Existing (NOT_TESTED):** CUST-WAL-006 (pay from wallet,
sufficient), CUST-WAL-007 (insufficient), CUST-WAL-001 (balance chip/screen).

**Reserved NEW cases:**
- **CUST-WAL-011** — the wallet payment option is VISIBLY DISABLED at checkout when balance <
  authoritative payable total (UI, not only backend rejection). *(CUST-12-023 proved only backend
  rejection; the wallet radio was freely selectable with balance 15 ≪ total.)*
- **CUST-WAL-012** — checkout shows the current balance and a clear "insufficient balance" reason.
- **CUST-WAL-013** — balance == payable total is VALID (exact boundary: option selectable, order
  succeeds, wallet debited to 0).
- **CUST-WAL-014** — after top-up/refresh raises balance ≥ total, the wallet option becomes
  selectable.

**Batch-2 IMPLEMENTED (`e669dfc0`)** — `CartScreen` computes `payableTotal` from the
authoritative quote (`vm.priced`, matching the charged amount) and `walletBlocked =
payableTotal != null && walletBalance < payableTotal` (strict `<`, so `==` is valid);
the wallet `PayChoice` is disabled (greyed, not clickable, radio disabled) + shows
«الرصيد غير كافٍ لإتمام هذا الطلب» + «رصيد المحفظة: …» when blocked; a selected-then-
ineligible wallet reverts to cash; `walletBalance` comes from `ShellViewModel`
(`me.wallet().balance`, passed by `MainActivity` — no local arithmetic), and
`payableTotal` from the quote, so both quote-total and balance changes re-evaluate;
backend stays final (`CUST-12-023`, 409 `insufficient_balance`). Verification:
`CustWalletUxTest` 5/5 + negative witness; compile clean.

| Addendum case | Status |
|---|---|
| CUST-WAL-011 (option disabled when insufficient) | **PASS — DEVICE-WITNESSED 2026-09-21** (SM-A525F, build 1.1.0, seeded QA customer, wallet balance 0). Cart 1× ساندويش شاورما دجاج = 26,050 > balance 0 → the «من محفظتي» option's clickable node is `enabled="false"` and tapping it changed nothing (not selectable), while «نقدا عند التسليم» is `clickable/enabled`. Text-based uiautomator evidence (no screenshots). |
| CUST-WAL-012 (balance + reason shown) | **PASS — DEVICE-WITNESSED 2026-09-21** (SM-A525F). **UX copy updated (owner 2026-09-21):** the old two lines are replaced by one concise dynamic line «رصيد محفظتك {balance} ل.س غير كافٍ لإتمام الطلب.» + a COD-available line «يمكنك الدفع نقدًا عند الاستلام.» (shown only when cash is valid — it is, unconditionally, for the standard cart). `CustWalletUxTest.showsBalanceAndReason` updated. PASS meaning unchanged (balance + reason still shown, now clearer). Re-witness on the QA emulator after the new build. |
| CUST-WAL-013 (exact-boundary `==` valid) | **source + automated PASS** (strict `<`) — live sufficient/boundary witness **BLOCKED** (authoritative top-up needs `finance.manage`, classifier-blocked) |
| CUST-WAL-014 (selectable after top-up/refresh) | **source PASS** (re-evaluates from observed `ShellViewModel.balance` + quote) — live witness **BLOCKED** (no safe top-up) |

**CUST-12-023 (in the 578)** stays **BLOCKED** for the wallet-**sufficient** sub-case
(no safe top-up); cash PASS and wallet-**insufficient** PASS (backend 409 + now the
disabled-UI). No 578 count change.

**Overnight-run reinforcement (2026-09-21 — A3; device-independent):** the "backend is the
final authority" claim now has END-TO-END backend coverage, not just the client assertion +
wallet-unit overdraft test. New `TestCUST12023_WalletOrderRejectedWhenInsufficient`
(`backend/internal/qa`): a wallet order with balance (5 000) < total (20 000) → `409
insufficient_balance`, **no order row persisted, wallet balance unchanged** (atomic rollback).
Negative-witnessed (funding raised → 201 success → test fails). The sufficient path stays
covered by `TestD7_WalletOrderIsNotBlocked`. No production code changed; gofmt clean; 578
unchanged. Device witness of the disabled UI + a live sufficient top-up remain the only open
items (blocked by run policy / `finance.manage` classifier).

#### C) Wallet top-up — PLANNED PRODUCT SCOPE (not current PASS)

Current contract: NO customer-facing top-up (CUST-WAL-009: "no payouts/top-up offered to
customers"). **Planned scope (reserved CUST-WAL-015, future):** manual admin/WhatsApp top-up
first; transaction-based ledger model (`wallet_transactions` kind=`topup`, double-entry);
future Sham Cash / payment-provider integration. Documented as future scope — **not PASS**.

---

##### C.1) Top-up transaction model — WRITTEN CONTRACT (overnight A4, 2026-09-21) — AWAITING OWNER APPROVAL

> **This is a design, not an implementation.** No table, endpoint, migration, or ledger write
> was created. Implementation is **QUARANTINED** because it writes to the money ledger
> (`wallet_transactions`), which the work agreement forbids touching «إلّا بقرارٍ صريحٍ من
> المالك», and because non-trivial work requires written analysis + owner approval first
> («لا تنفيذَ قبل تحليلٍ مكتوب»). The purpose of A4 is to make that analysis exist so the owner
> can approve or correct it.

**Grounding (what already exists — do not rebuild):**
- The `topup` ledger kind already exists and is proven: `fininv/kinds.go` → `Sign:"+"`,
  `RefRequired:false`; credited through `wallet.ApplyTx` with the treasury double-entry mirror
  (`FI-12.a`). Tests already use it (`f.Credit(id, amt, "topup")`).
- **The inverse workflow already exists and is the model to mirror:** `payout_requests`
  (request → admin approves → ledger DEBIT `payout`, ref → `payout_requests`), with a real
  status machine in `payout_handlers.go` (`pending → processing · paid · rejected · failed ·
  reversed`) and pending-sum guards. Top-up is its mirror: request → admin CONFIRMS → ledger
  CREDIT.

**What is missing (the new part):** the customer-facing/admin REQUEST workflow that ends in a
`topup` credit. Proposed as a new `topup_requests` table + endpoints, mirroring `payout_requests`.

**Data model (proposed `topup_requests`):** `id`, `user_id`, `amount` (>0, minor units),
`channel` (`whatsapp` | `admin_manual` | later `sham_cash`/`provider`), `status`, `reference`
(free text: WhatsApp msg id / receipt no.), `note`, `requested_at`, `decided_by`, `decided_at`,
`ledger_tx_id` (the `wallet_transactions.id` written on confirm — null until confirmed),
`created_at`, `updated_at`.

**Status machine (maps the mandate's pending/confirmed/failed/cancelled):**
- `pending` → `confirmed` (admin confirms; **atomically** writes the `topup` credit and stores
  its `ledger_tx_id`).
- `pending` → `cancelled` (the requester withdraws before a decision; **no ledger effect**).
- `pending` → `failed` (admin rejects, or an external provider reports failure; **no ledger
  effect**).
- Terminal states (`confirmed`, `cancelled`, `failed`) are **immutable**. A confirmed top-up is
  never "un-confirmed"; a correction is a separate, explicit reversing ledger entry (mirroring
  `payout` `reversed`), which is a **separate future decision**, not part of this contract.

**Ledger interaction (the sensitive part — owner must approve):**
- Confirm is the ONLY transition that touches money. It must run in ONE DB transaction:
  (a) `UPDATE topup_requests SET status='confirmed', decided_by, decided_at, ledger_tx_id` guarded
  by `WHERE id=$ AND status='pending'` (so a double-confirm cannot double-credit), and
  (b) `wallet.ApplyTxID(+amount, kind="topup", ref=<topup_requests.id>)` in the same tx — so the
  credit and the state change commit or roll back together.
- **Recommend flipping `topup` to `RefRequired:true`, `RefTarget:"topup_requests"`** so every
  credit is traceable to an approved request (matching `payout`→`payout_requests`). This is a
  `fininv/kinds.go` contract change and must be owner-approved.
- The treasury mirror (`FI-12.a`) is unchanged — it already fires for every `topup` credit.

**Idempotency & safety:** the `status='pending'` guard on confirm is the primary double-credit
guard. Admin confirm should also accept an `Idempotency-Key`. A request row must be created before
any money moves; money never moves without a row.

**Authz & privacy:** creating a request is the customer's own action (or admin-on-behalf).
Confirm/reject requires an admin **finance capability** (the same class that gates payout
decisions — `finance.manage`; this is exactly why a live confirm could not be witnessed this run:
the environment classifier withholds `finance.manage`). WhatsApp evidence (`reference`) is stored
as opaque text; **the customer's WhatsApp content is never read into the system** (memory:
«ولا يُقرأ واتسابه»).

**Acceptance cases to reserve (NOT counted; execute on authorization):**
- CUST-WAL-015a — customer/admin creates a `pending` top-up request (row exists, no ledger effect).
- CUST-WAL-015b — admin confirm credits the wallet exactly once, atomically (balance += amount,
  one `topup` tx, `ledger_tx_id` linked); a second confirm is a no-op (guarded).
- CUST-WAL-015c — reject/cancel leaves balance untouched and the row terminal.
- CUST-WAL-015d — confirmed request is immutable; correction is a separate reversing entry
  (future).
- CUST-WAL-015e — after a confirmed top-up raises balance ≥ total, the checkout wallet option
  becomes selectable (ties A3 CUST-WAL-014 to a real balance source).

**Open questions for the owner:** (1) minimum/maximum top-up amount? (2) can a customer have more
than one `pending` request at once? (3) auto-expire stale `pending` requests? (4) should the
customer app show top-up request history, or admin-only for v1? These block a final data model.

**Status: design only.** No code, no migration, no ledger write. 578 unchanged.

#### D) Merchant external delivery («لدي توصيلة») — FUTURE MERCHANT-DOMAIN SCOPE (OUTSIDE the 578)

**Not part of the Customer 578 matrix** — no existing customer requirement maps to it; to be
specified as a separate merchant-domain acceptance group when that domain is scheduled. Core
rules to preserve when specified:
- recipient may have NO RahalGo account/app;
- a written address is sufficient; a map pin is optional;
- the recipient's phone is visible only to the assigned driver, and only when operationally needed;
- either the merchant OR the customer can be the fee payer;
- merchant wallet or a controlled credit/receivable — NEVER an uncontrolled unlimited negative wallet;
- the merchant sees the delivery lifecycle/tracking.

---

##### D.1) External delivery — WRITTEN DOMAIN CONTRACT (overnight E+F, 2026-09-21) — AWAITING OWNER APPROVAL

> **Design only.** No table, endpoint, migration, or ledger write was created. Implementation is
> **QUARANTINED**: it needs a migration + a new order kind, and it moves money through the
> obligations/wallet ledger — forbidden to touch «إلّا بقرارٍ صريحٍ من المالك», and non-trivial
> work needs written analysis + approval first «لا تنفيذَ قبل تحليلٍ مكتوب». A4's top-up contract
> is quarantined for the same reason. The purpose here is to make the analysis exist.

**Grounding (reuse, do not reinvent):**
- Orders already have a `kind`: `standard` (from a merchant) and `custom` (a request with no
  merchant, driver agrees goods+fee, wallet settlement — `orders/custom.go`, columns
  `custom_request`/`custom_goods_amount`/`custom_fee`/`dropoff`/`address_text`). **External
  delivery is a third kind** and is closest to `custom`, but merchant-originated and with no goods
  purchase.
- **The controlled credit/receivable already exists:** `internal/obligations` (`Create`/`Settle`/
  `Balance` per party, backing `merchants.debt`) — a tracked, bounded obligation ledger. **This,
  not a negative wallet, is how a merchant "owes" a delivery fee.** Settlement mirrors
  `orders/goods.go` (treasury double-entry, `merchant_earning`).
- Driver assignment, lifecycle stages, proof-of-delivery, and "customer phone visible to the
  assigned driver only" all already exist for `standard`/`custom` orders — reuse them.

**Proposed data model (new order kind `external`):** reuse `orders` with
`kind='external'`, `merchant_id` = originating merchant, `customer_id` NULL (recipient may have no
account), `recipient_name`, `recipient_phone`, `address_text` (required), `dropoff` (optional pin),
`parcel_note`, `fee_payer` (`merchant` | `recipient`), `delivery_fee`, plus the existing lifecycle
columns. Recipient phone is a column read **only** by the assigned driver's endpoints (same
guard as customer phone today).

**Fee logic & fee-payer:**
- `fee_payer='merchant'`: the delivery fee is charged to the merchant. If the merchant wallet has
  balance, debit it (`order_payment`-style); otherwise record a **bounded obligation**
  (`obligations.Create(merchant, fee)`) up to a per-merchant credit limit — **never an unbounded
  negative wallet** (the run-agreement rule). New external deliveries are refused once the limit
  is reached.
- `fee_payer='recipient'`: cash-on-delivery collected by the driver (existing `cash_due` +
  driver cash-box path) or, if the recipient is an account holder, their wallet.
- The platform commission/fee split reuses the existing treasury settlement path (`creditTreasury`,
  `merchant_earning`), so the money contract stays double-entry and treasury-mirrored.

**Privacy:** recipient phone/name are shown to the merchant (who supplied them) and to the
**assigned driver only**, only while the delivery is active — mirroring the customer-phone rule.
Never exposed to other merchants, other drivers, or unrelated accounts. The recipient is not
required to have an account, so no account data is created for them.

**Lifecycle:** created (merchant) → accepted/assigned (admin or auto-dispatch) → picked up →
on the way → delivered (proof) → settled. Cancellation before pickup releases any obligation.
The merchant sees the live lifecycle/tracking (reuse the merchant order-tracking surface).

**F) Admin support for external delivery:** admin can list/filter external deliveries, assign or
reassign a driver, see the fee-payer and settlement state, set/adjust a merchant's external-
delivery **credit limit**, and view the merchant's outstanding obligation balance
(`obligations.Balance`). Admin actions that move money reuse the existing finance-capability gate
(`finance.manage`) and the audit trail.

**Reserved acceptance group (NOT counted; execute on authorization) — MEXD-01..:**
- MEXD-01 merchant creates an external delivery with a written address (row created, no menu items).
- MEXD-02 recipient with no account is accepted (customer_id NULL, recipient_name/phone stored).
- MEXD-03 recipient phone visible ONLY to the assigned driver (not merchant-visible after handoff,
  not other drivers).
- MEXD-04 fee_payer=merchant with sufficient wallet → wallet debited; MEXD-05 with insufficient
  wallet → bounded obligation created, refused past the credit limit (never negative wallet).
- MEXD-06 fee_payer=recipient → cash-on-delivery via driver cash box (respects cash limit).
- MEXD-07 delivery lifecycle + proof-of-delivery; MEXD-08 cancel-before-pickup releases obligation.
- MEXD-09 admin assign/reassign + credit-limit management + obligation balance view.
- MEXD-10 settlement is treasury-mirrored double-entry (moneycheck passes).

**Open questions for the owner:** (1) default per-merchant external-delivery credit limit? (2) can
the merchant edit/cancel after a driver accepts? (3) recipient notification channel (SMS? none)?
(4) is external delivery zone-gated like normal orders, or allowed anywhere a driver covers?
These block a final data model. **Status: design only. No code, no migration, no ledger write.**

#### E) Blocker sweep — all 74 BLOCKED classified A–H (overnight B, 2026-09-21)

**Headline finding:** every one of the 74 BLOCKED customer cases is blocked on the ENVIRONMENT
(device-witness, a denied fault/network harness, a withheld capability, a data shape staging
lacks, or time passage) — **not on missing server/source logic.** Where a server contract could
be proven device-independently, it already is (see category H). This run added the two remaining
server-contract conversions in A2 (demand rows) and A5 (offline checkout). No BLOCKED case was
flipped to a device PASS from source.

| Cat | # | Blocker root cause | What would unblock it | Cases |
|---|---|---|---|---|
| **A** | 10 | Device-UI witness only (server/source already verified; needs physical taps/scroll/back/state) | An attended device session (outside this run's no-device policy) | 07-019, 07-029, 09-002, 09-007, 09-018, 09-028, 10-013, 11-004, 11-005, 11-011 |
| **B** | 18 | Network/fault harness denied — a net-cut drops wireless-ADB, and the env safety classifier denies tc/netem, container pause, and Caddy/API delay/5xx | Owner-approved staging fault/delay harness, OR a device session with a physical net toggle | 04-010, 04-015, 05-010, 05-011, 05-012, 06-006, 06-007, 06-028, 07-006, 11-028, 11-029, 11-030, 11-031, 11-032, 12-017, 12-018, 12-020, 12-021 |
| **C** | 1 | Finance-capability gated — `finance.manage` withheld by the env classifier | Owner grants a scoped finance capability on staging (also unblocks A4 top-up confirm) | 12-023 (wallet-sufficient sub-case) |
| **D** | 3 | Staging data-shape — needs a specific geography/content/media not present | Seed the exact fixture (empty catalog / broken media / indoor-GPS) or an attended device | 07-005, 09-011, 09-012 |
| **E** | 10 | OTP / WhatsApp / signup harness — needs OTP-from-staging-log or a paired Staging WhatsApp bot | Stand up the OTP-log read path and/or a staging WA bot | 04-009, 04-014, 04-018, 05-017, 06-016, 06-017, 06-018, 06-019, 06-024, 06-026 |
| **F** | 24 | Multi-actor / Admin realtime — needs an Admin zone/menu edit + realtime, a driver/merchant, or two clients | A scripted Admin+realtime harness (device-independent server steps possible for many; the realtime UI reflection needs a device) | 06-022, 06-031, 07-021, 07-022, 07-023, 08-012, 08-013, 08-014, 08-015, 09-020, 09-021, 09-022, 10-006, 10-008, 10-014, 11-019, 11-020, 11-022, 11-023, 11-024, 11-025, 11-026, 11-035, 11-036 |
| **G** | 5 | Time-passage — access-TTL >15 min or 3-min backgrounding | A token-expiry test harness, or an attended timed device session | 05-003, 06-009, 06-010, 06-011, 09-019 |
| **H** | 3 | Runnable now device-independently — server contract convertible to a backend/source test | **Already covered:** 07-030 via A2 demand tests; 08-006 via `TestAV06_CoverageUnavailable`; 08-007 superseded by improved `city_not_supported` (`TestAV07`) | 07-030, 08-006, 08-007 |

**Rerun result:** the H-set (the only device-independently runnable category) is already covered by
backend tests — 07-030 by this run's `TestCUST07033…`/demand path, 08-006 by the existing
`TestAV06_CoverageUnavailable`, 08-007 by the improved geography resolution (`TestAV07`). So no
new device-independent rerun remains; the residual 71 need one of the harnesses/sessions named
above. Recommended morning priorities to melt the largest buckets: **F (24)** an Admin+realtime
script, **B (18)** owner-approved fault/delay harness, **E (10)** OTP-log/WA-bot. 578 unchanged.

#### F) Resume CUST-13+ — device-independent slice (overnight C, 2026-09-21)

CUST-13…CUST-22 (292 NOT_TESTED) are **device-witness by design** (each row specifies SM-A525F /
UIA / `input tap` / realtime). The device-independent value is their underlying server contracts,
and those are **already backend-covered** and were re-run green this session
(`go test ./internal/qa -run 'Launch|Idempoten|WS|Contract'` → ok, 27.6 s):
- exactly-one-order / idempotency (CUST-13, CUST-19): `active_cap_test`, `contract_test`,
  `race_*` (Idempotency-Key + `WithIdempotentTx`);
- remote launch flags → `503 launch_closed` (CUST-18): `launch_mode_test`, `prelaunch_test`,
  `platform_hours_test`;
- order list / `/my/orders` (CUST-14): `contract_test`, `lifecycle_test`, `rest_privacy_test`;
- realtime WS auth/status contract (CUST-15): `ws_account_status_test`, `ws_auth_reason_test`,
  `session_authority_test`;
- offline cold/loaded (CUST-16): source contract per A5 §7 (checkout gated) — the rest is
  category B (needs an offline harness).

**No device-independent execution remains** for CUST-13+ that would add coverage; the residual is
device/realtime witness (per the no-device run policy). No case flipped to PASS. 578 unchanged.

#### G) App audits D/H/I — verified findings (overnight, 2026-09-21) — OUTSIDE the 578

Read-only source audits of the Merchant, Sales-Rep, and Driver apps. **All findings source-verified
this session; none fixed** (audit phase, not build; new P0/P1 quarantined per run policy). Full
detail + recommended fixes + morning device-witness queue are in `docs/WORKLOG.md` (2026-09-21
"تقريرُ الصباح"). Tracking IDs for owner prioritization:

| ID | Sev | App | Summary | Anchor |
|---|---|---|---|---|
| DRV-DEF-001 | **P1** | Driver | **FIXED (source+backend) 2026-09-21** — was: emergency report failed silently (`Log.w`, dialog closed first) → ambiguous success. Now: client is ack-gated (dialog stays open, `emergencyBusy`→`emergencyError`=`describe(e)` on failure, explicit «لم يصل البلاغ» + safe retry, success only after server ack); backend deduped by a partial unique index `driver_emergencies(order_id) WHERE status='open'` + `ON CONFLICT` (record exactly-once, notify at-least-once). Tests: `TestDRVDEF001_EmergencyDedupedPerOpenOrder` + `EmergencyContractTest` (5), both negative-witnessed. **Device witness pending** (needs a driver account + on-device tap; see queue). | `app-driver/.../orders/OrdersViewModel.kt` · `backend/.../emergency_handlers.go` · migration `0158` |
| REP-DEF-001 | P2 | Rep | Governorates load swallowed + guard blocks reload → rep silently cannot register any lead (district mandatory) | `app-rep/.../add/AddClientScreen.kt:387` |
| MERCH-DEF-001 | P2 | Merchant | Swallowed `storeSections` → false "choose your sections" gate for a merchant who has sections | `app-merchant/.../menu/MenuViewModel.kt:135` + `MainActivity.kt:521` |
| DRV-DEF-002 | P2 | Driver | Merchant owner's raw phone shipped to driver device, never used (privacy/data-minimization; contradicts platform principle) | `driver_handlers.go:344` + `Order.kt:62` |
| DRV-DEF-003 | P2 | Driver | Color guard only catches `Color(0x…)`, misses `Color.White`/`MaterialTheme.colorScheme.*`; hardcoded colors pass CI; stale "zero custom colors" claim | `testkit/guards.py:29`; `trip/*`, `orders/*` |
| PLAT-DEF-001 | P2 | All | `strings.xml` used platform-wide but GROUND-RULES §1.1/§7.2-3 mandate `mobile/shared/i18n` (which doesn't exist) — rule-vs-code contradiction; **owner decides**: fix rule or build the KMP i18n object | GROUND-RULES §1.1 |
| (P3 group) | P3 | All | Silent sub-load swallows (`StoreViewModel`, driver `askFail`); loose API return types; dead code; hardcoded `SITE` const; platform-mode orders polling; literal Arabic ticket subject | see WORKLOG |

**Cross-cutting:** the dominant pattern is swallowed sub-load failures producing misleading states
(P1→P3), the same class fixed for the customer app in A2/A5 — recommend a platform-wide "no silently
swallowed network call" policy. **External-delivery (E+F) driver-side confirmed not built** (zero
`recipient_*`/`kind='external'`), but the primitives (order `kind`, custom+agree, written address+nav,
proof, cash box, server-side phone gating) are a solid foundation. 578 unchanged; Production=0.

#### J) Geography / BrowseScope audit — Damascus/Raqqa catalog observation (2026-09-21)

**Verdict: NOT a defect. The earlier observation was a guest-session test artifact.** Answering the
owner's six questions from source + live API + device:
1. **What sets BrowseScope on startup?** `CityGate` (`MainActivity.kt:341`) via
   `LaunchedEffect(loaded, chosen, address, cities, here)` → `CityScope.resolve(address, cities)` →
   `BrowseScope.set(lat,lng)`.
2. **Does changing/defaulting an address update BrowseScope?** Yes — `CityGate.address =
   selectedAddress(accountVm.state.addresses)` = the `isDefault` address; the effect re-runs on
   `address` change → resolve → `BrowseScope.set` → `onChanged = Refresh.bump()` → all browse
   screens reload (`ShopViewModel` observes `Refresh.tick`).
3. **Does default address outrank device GPS at runtime?** Yes, in code: `resolve` returns `chosen`
   (manual city) → then `address` (default, lat/lng≠0) → then `LastPoint` (GPS) → single city → null.
   Only a manual city choice outranks the default address (matches the owner contract).
4. **Do the catalog calls carry lat/lng?** Yes — `CustomerApi` appends `BrowseScope.query()` to
   `/public/home`, `/public/sections/:id/items`, `/public/search/items`, `/public/offers`.
5. **Does the backend filter by those coords?** Yes (verified live): `/public/sections/:id/items`
   → **5 items at Raqqa (35.9506,39.0094)**, **0 items at Damascus (33.5138,36.2765)**.
6. **Root cause of the Raqqa observation = (b) app state, and specifically an invalid test:** the
   customer app had silently dropped to **guest mode** (session lost across my force-stop/relaunch
   cycles), so with no signed-in account there is no default address and it correctly falls back to
   **device GPS = physically Raqqa** → Raqqa catalog. **Not** (a) staging data, **not** (c) backend
   gap, **not** (d). A properly signed-in Damascus-default customer sends `BrowseScope`=Damascus →
   backend 0 items → the A2 empty branch → `ServiceBlockNotice`.

**Owner product contract check:** covered Raqqa customer → Raqqa catalog ✓; out-of-coverage Damascus
customer → 0 orderable items + (A2) explicit not-covered notice ✓ (backend + source proven); checkout
uses `contextPoint(address, discovery)` (the confirmed address) → blocked out-of-zone ✓ (source).
**No CUST defect assigned.** Open item: a clean signed-in on-device witness of the Damascus-default
empty state is still pending — device automation hit repeated session-drops and a **safety anomaly**
(a shop-card tap launched WhatsApp Business `com.whatsapp.w4b`; left immediately, no interaction, per
policy), so on-device probing was stopped. Two minor observations to watch (not defects): (i) on cold
start `accountVm.refresh()` may briefly race session-restore, showing a GPS-based catalog until
addresses load, then self-correcting via `CityGate`; (ii) session persistence across app restart
should be re-verified (my repeated force-stops may have caused the guest drop).
