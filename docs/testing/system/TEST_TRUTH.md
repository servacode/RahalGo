# حقيقةُ الاختبار — التقرير

> **مولَّدٌ آليّاً** — `go run ./cmd/testtruth`.
> **ولا يُحرَّر بيد**، وحارسُ `TestTruthIsCurrent` يُسقط البناءَ إن شاخ.
> **والمصدرُ للآلة** [`TEST_TRUTH.json`](TEST_TRUTH.json).

---

# ١ · ما استُخرج من الشيفرة

**ولا رقمَ منه مكتوبٌ بيد.**

| ما هو | العدد |
|---|---|
| أبوابٌ في الموجّه | **314** |
| انتقالاتُ الطلب | **55** |
| أنواعُ قيدِ المحفظة | **14** — `adjustment` · `commission` · `compensation` · `driver_earning` · `merchant_earning` · `operating_expense` · `order_payment` · `payout` · `penalty` · `platform_expense` · `platform_profit` · `refund` · `reward` · `topup` |
| مواضعُ الإشعار | **48** — منها **11** موجَّهٌ بـ`Apps` |
| غرفُ البثّ | **5** — `customer` · `driver` · `merchant` · `ops` · `user` |
| تعريفاتُ الإعدادات | **118** — منها **95** مُغيِّرٌ للسلوك |
| حقولُ الطلب | **75** — من عقد `P-1` |
| ملفّاتُ اختبار | **256** |
| دوالُّ اختبار | **919** |

---

# ٢ · الاختبارات

```
TOTAL      = 919
MAPPED     = 235
INFRA      = 109
ORPHAN     = 575
```

**واليتيمُ اختبارٌ لا يعرف ماذا يحرس** — **ولا يُسقط البناءَ اليومَ**،
**ويُربَط مرحلةً بعد مرحلة.** وأكثرُها في:

| الحزمة | يتيمٌ |
|---|---|
| `qa` | 126 |
| `server` | 116 |
| `routing` | 76 |
| `orders_test` | 60 |
| `orders` | 54 |
| `identity` | 22 |
| `settings` | 14 |
| `notify` | 13 |
| `push` | 10 |
| `catalog` | 9 |

---

# ٣ · التغطية

| السجلّ | العدد | مربوطٌ | بلا اختبار |
|---|---|---|---|
| **التدفّقات** | 35 | 29 | 6 |
| **العيوب** | 27 | 15 | 12 |
| **المخاطر** | 24 | 14 | 10 |
| **فجواتُ العقد** | 36 | 10 | 26 |
| **إعداداتُ السلوك** | 95 | 10 | 85 |

---

# ٤ · العيوبُ وحارسُها

| ID | العنوان | الحال | الاختبارات |
|---|---|---|---|
| **D1** | فكُّ الإسناد بلا حدثٍ في order_events | `NO_REGRESSION_TEST_YET` | — |
| **D2** | convertLead بلا معاملةٍ واحدة · ٣ كتاباتٍ خطؤه… | `FIXED_AND_PASSING` | `TestATOMIC_LeadConversionIsOneUnit` · `TestFAIL_D2_ConvertLeadPartialStates` · `TestFIN_TargetRewardPrecedesCommit` · `TestFIN_TransactionBoundaries` · `TestUNIQ_ConcurrentConversionWithExistingOwner` · `TestUNIQ_ConcurrentLeadConversionMakesOneMerchant` · `TestUNIQ_DatabaseRefusesSecondMerchantForSameLead` · `TestUNIQ_FailureThenRetryMakesOneMerchant` · `TestRACE_DuplicateLeadConversion` · `TestXG29_FailureBeforeCommitLeavesNothing` · `TestXG29_FailureThenRetryGrantsRewardOnce` · `TestXG29_SuccessGrantsRewardExactlyOnce` |
| **D3** | «تذكّرني» تنقلب دائمةً بعد أوّل تجديد | `NO_REGRESSION_TEST_YET` | — |
| **D4** | سقفُ المفتوح داخلَ بوّابة واتساب | `NO_REGRESSION_TEST_YET` | — |
| **D5** | المصروفُ والخزينةُ كتابتان بلا معاملة | `FIXED_AND_PASSING` | `TestATOMIC_ExpenseAndTreasuryAreOneUnit` · `TestFAIL_D5_ExpenseTreasuryPartial` · `TestFIN_ExpenseTreasuryInvariant` · `TestFIN_TransactionBoundaries` |
| **D6** | الطلبُ الخاصُّ لا ينادي cashBlocked | `NO_REGRESSION_TEST_YET` | — |
| **D7** | سقفُ النقد يقيس المحصَّل لا المكشوف | `EXPECTED_FAIL` | `TestSampleFactory_DriverOnShiftWithCash` · `TestFIN_CashExposureContract` |
| **D8** | الطلبُ الخاصُّ لا يطلب توثيقَ واتساب | `NO_REGRESSION_TEST_YET` | — |
| **D9** | الطلبُ الخاصُّ بلا حدثِ ''→pending | `NO_REGRESSION_TEST_YET` | — |
| **D10** | الاستعادةُ تُبطل نوعَ عميلٍ واحد | `NO_REGRESSION_TEST_YET` | — |
| **D11** | force_password_change بلا بوّابةٍ في أندرويد | `NO_REGRESSION_TEST_YET` | — |
| **D12** | Push.unregister بلا منادٍ | `EXPECTED_FAIL` | `TestEV_PushTokenTargeting` |
| **D13** | سردُ /media/ مفتوحٌ — وإثباتُ التسليم فيه | `FIXED_AND_PASSING` | `TestD13_AvatarNeedsSignedURL` · `TestD13_DeliveryProofNeedsSignedURL` · `TestD13_MediaDirectoryIsNotListable` · `TestD13_PublicMediaStaysPublic` · `TestD13_SignedURLWorksAndForgeryDoesNot` · `TestFAIL_D13_MediaDirectoryListingOpen` |
| **D14** | handleWS لا يفحص ActiveStatus | `EXPECTED_FAIL` | `TestSampleFactory_SuspendedIsRefused` |
| **D15** | AdminCreateUser في خطوتين | `FIXED_AND_PASSING` | `TestATOMIC_AdminUserCreationIsOneUnit` · `TestFAIL_D15_AdminCreateUserPartial` · `TestFAIL_D15_Reconciliation` |
| **D16** | طابورُ المواقع ملفٌّ بلا صاحب | `NO_REGRESSION_TEST_YET` | — |
| **D17** | الملاحةُ لا تعود بعد موت العمليّة | `NO_REGRESSION_TEST_YET` | — |
| **D18** | START_STICKY يعيد الخدمةَ بفترةِ الافتراض | `NO_REGRESSION_TEST_YET` | — |
| **D19** | LiveSocket يعيد الوصلَ بتوكنٍ منتهٍ | `NO_REGRESSION_TEST_YET` | — |
| **D20** | البثُّ الحيُّ يتجاوز redactForMerchant | `EXPECTED_FAIL` | `TestEV_MerchantDriverAssignment` · `TestEV_MerchantRealtimePrivacy` · `TestD20_CustomerRealtimeVsREST` · `TestD20_MerchantRealtimeVsREST` · `TestOrderFieldsAllClassified` |
| **D21** | هاتفُ السائق يصل الزبون | `EXPECTED_FAIL` | `TestEV_CustomerDriverAssignment` · `TestD21_CustomerRedactionAgainstContract` · `TestForbiddenFieldGuardCatchesLeak` · `TestOrderFieldsAllClassified` |
| **D22** | الطلبُ الخاصُّ لا يُبثّ لصاحبه | `EXPECTED_FAIL` | `TestEV_CustomOrderOwnerRealtime` · `TestD22_CustomOrderOwnerChannelContract` · `TestOrderFieldsAllClassified` |
| **D23** | حمولاتُ REST تكشف اقتصاداً داخليّاً | `EXPECTED_FAIL` | `TestEV_CustomerDriverAssignment` · `TestD20_MerchantRealtimeVsREST` · `TestD21_CustomerRedactionAgainstContract` · `TestOrderFieldsAllClassified` |
| **D24** | سقفُ الطلبات النشطة يُتجاوَز بالتزامن | `EXPECTED_FAIL` | `TestFAIL_R7_DriverAcceptPartialState` · `TestRACE_MaxActiveOrders` |
| **D25** | هويّةُ متجرٍ واحدةٌ تصير متجرين بالتزامن | `FIXED_AND_PASSING` | `TestFAIL_D2_ConvertLeadPartialStates` · `TestUNIQ_ConcurrentConversionWithExistingOwner` · `TestUNIQ_ConcurrentLeadConversionMakesOneMerchant` · `TestUNIQ_DatabaseRefusesSecondMerchantForSameLead` · `TestUNIQ_FailureThenRetryMakesOneMerchant` · `TestRACE_DuplicateLeadConversion` |
| **D26** | إنذارُ الراصد يضيع بعد كتابة الوسم | `EXPECTED_FAIL` | `TestEV_R22WatchdogMarkerSuppressesRetry` |
| **D27** | سقوطُ الدفع بلا إعادةٍ دائمة | `EXPECTED_FAIL` | `TestEV_R23PushFailureIsLost` |

---

# ٥ · فجواتُ العقد

| ID | الشدّة | يوقظه | الحال | الاختبارات |
|---|---|---|---|---|
| **XG-5** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-6** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-7** | `HIGH` | `orders.auto_accept_min` | `NOT_IMPLEMENTED` | — |
| **XG-8** | `MEDIUM` | — | `NOT_IMPLEMENTED` | — |
| **XG-9** | `MEDIUM` | — | `NOT_IMPLEMENTED` | — |
| **XG-10** | `BLOCKER` | — | `COVERED` | `TestFIN_RefundNotConditionedOnRepBalance` · `TestFIN_RepCommissionReversal` · `TestRACE_RefundVsPayout` · `TestFIN_XG10_DebtOffsetFromNextCommission` · `TestFIN_XG10_RepCommissionReversedOnRefund` · `TestFIN_XG10_RepWithdrewThenRefund` · `TestFIN_XG10_CombinedMerchantAndRepInsufficiency` · `TestFIN_XG10_ConservationAcrossRefund` · `TestFIN_XG10_DebtSettlementArithmetic` · `TestFIN_XG10_RefundReplayDoesNotDoubleCharge` |
| **XG-11** | `BLOCKER` | — | `COVERED` | `TestFIN_MerchantWithdrewThenRefund` · `TestFIN_RefundNotConditionedOnRepBalance` · `TestRACE_RefundVsPayout` · `TestFIN_XG11_RefundIndependentOfMerchantBalance` · `TestFIN_XG11_RefundUntouchedWhenMerchantSolvent` · `TestFIN_XG10_RepWithdrewThenRefund` |
| **XG-12** | `CRITICAL` | — | `NOT_IMPLEMENTED` | — |
| **XG-13** | `HIGH` | — | `EXPECTED_FAIL` | `TestFIN_CommissionSourceMatrix` · `TestFIN_SnapshotVsLiveEconomics` |
| **XG-14** | `CRITICAL` | — | `NOT_IMPLEMENTED` | — |
| **XG-15** | `HIGH` | `sales.activation_orders` | `NOT_IMPLEMENTED` | — |
| **XG-16** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-17** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-18** | `BLOCKER` | — | `COVERED` | `TestFAIL_D2_ConvertLeadPartialStates` · `TestUNIQ_ConcurrentConversionWithExistingOwner` · `TestUNIQ_ConcurrentLeadConversionMakesOneMerchant` · `TestUNIQ_DatabaseRefusesSecondMerchantForSameLead` · `TestUNIQ_FailureThenRetryMakesOneMerchant` · `TestRACE_DuplicateLeadConversion` |
| **XG-19** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-20** | `CRITICAL` | — | `NOT_IMPLEMENTED` | — |
| **XG-21** | `BLOCKER` | — | `COVERED` | `TestD13_AvatarNeedsSignedURL` · `TestD13_DeliveryProofNeedsSignedURL` · `TestD13_MediaDirectoryIsNotListable` · `TestD13_PublicMediaStaysPublic` · `TestD13_SignedURLWorksAndForgeryDoesNot` |
| **XG-22** | `BLOCKER` | — | `COVERED` | `TestXG22_SuspendDuringTransitionIsDeterministic` · `TestXG22_T10_EnforcementIsServerSide` · `TestXG22_T1_SuspendedWithoutActiveOrderIsDenied` · `TestXG22_T2_SuspendedDriverCanFinishActiveOrder` · `TestXG22_T3_ExceptionDoesNotLeakToAnotherOrder` · `TestXG22_T4_ExceptionEndsAtTerminalState` · `TestXG22_T6_NormalActorUnchanged` · `TestXG22_T7_OpsCanStillResolveTheOrder` · `TestXG22_T8_BlockedHasNoException` · `TestXG22_T9_ExceptionDoesNotLeakAcrossRoles` |
| **XG-23** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-24** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-25** | `CRITICAL` | — | `NOT_IMPLEMENTED` | — |
| **XG-26** | `CRITICAL` | — | `NOT_IMPLEMENTED` | — |
| **XG-27** | `CRITICAL` | — | `NOT_IMPLEMENTED` | — |
| **XG-28** | `CRITICAL` | — | `NOT_IMPLEMENTED` | — |
| **XG-29** | `BLOCKER` | — | `COVERED` | `TestXG29_FailureBeforeCommitLeavesNothing` · `TestXG29_FailureThenRetryGrantsRewardOnce` · `TestXG29_SuccessGrantsRewardExactlyOnce` |
| **XG-30** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-32** | `CRITICAL` | — | `COVERED` | `TestXG29_FailureBeforeCommitLeavesNothing` · `TestXG29_FailureThenRetryGrantsRewardOnce` · `TestXG29_SuccessGrantsRewardExactlyOnce` · `TestXG32_ConcurrentTargetCrossingGrantsOnce` · `TestXG32_FailureRollbackThenRetryGrantsOnce` · `TestXG32_FirstConversionGrantsRewardImmediately` · `TestXG32_FurtherConversionsDoNotRepeatReward` · `TestXG32_PreviousMonthDoesNotSatisfyTarget` · `TestXG32_TargetTwoGrantsOnSecondOnly` |
| **XG-34** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-35** | `CRITICAL` | — | `NOT_IMPLEMENTED` | — |
| **XG-36** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-38** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-40** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-39** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-37** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-33** | `CRITICAL` | — | `COVERED` | `TestIDEM_AllProtectedPathsUseCoordinator` · `TestIDEM_T10_CleanupSparesLiveClaim` · `TestIDEM_T11_SameKeyDifferentPayloadContractUnchanged` · `TestIDEM_T1_ConcurrentDuplicateExecutesOnce` · `TestIDEM_T2_OrphanBeforeTxIsReclaimed` · `TestIDEM_T3_StaleOwnerIsFenced` · `TestIDEM_T4_ActiveClaimCannotBeStolen` · `TestIDEM_T5_BusinessRollbackLeavesNothing` · `TestIDEM_T6_CommittedThenDeathReplaysWithoutDuplicate` · `TestIDEM_T8_TwoReclaimersExecuteOnce` · `TestIDEM_T9_StaleOwnerCannotDeleteNewerClaim` · `TestFAIL_C06_OrphanBeforeCommitBlocksOwner` · `TestFAIL_C06_TwoReclaimersExecuteNothing` · `TestIDEM_CleanupSparesLiveClaim` · `TestIDEM_CommittedBeforeResultDoesNotDuplicate` · `TestIDEM_LostResponseReplays` |
| **XG-31** | `CRITICAL` | — | `COVERED` | `TestOBL_BothInsufficient_AtomicOrigins` · `TestOBL_FutureEarningsSettleWithEvidence` · `TestOBL_MerchantInsufficient_OriginTraceable` · `TestOBL_MerchantSufficient_NoObligation` · `TestOBL_MultipleObligationsFIFO` · `TestOBL_RepAvailable_NoObligation` · `TestOBL_RepWithdrawn_OriginTraceable` · `TestOBL_ReplayCreatesNoDuplicate` · `TestFIN_XG10_CombinedMerchantAndRepInsufficiency` · `TestFIN_XG10_ConservationAcrossRefund` · `TestFIN_XG10_DebtSettlementArithmetic` · `TestFIN_XG10_RefundReplayDoesNotDoubleCharge` |

---

# ٦ · الشيخوخةُ والفجوات

```
STALE REFERENCES = 0
COVERAGE GAPS    = 28
```

## فجواتُ تغطية — **ما يحتاج اختباراً ولا اختبارَ له**

- D1 — لا اختبارَ انحدارٍ بعد
- D10 — لا اختبارَ انحدارٍ بعد
- D11 — لا اختبارَ انحدارٍ بعد
- D16 — لا اختبارَ انحدارٍ بعد
- D17 — لا اختبارَ انحدارٍ بعد
- D18 — لا اختبارَ انحدارٍ بعد
- D19 — لا اختبارَ انحدارٍ بعد
- D3 — لا اختبارَ انحدارٍ بعد
- D4 — لا اختبارَ انحدارٍ بعد
- D6 — لا اختبارَ انحدارٍ بعد
- D8 — لا اختبارَ انحدارٍ بعد
- D9 — لا اختبارَ انحدارٍ بعد
- F-06 (رفضُ المتجر) — لا اختبارَ مرتبطٌ به
- F-16 (تعذّرُ التسليم) — لا اختبارَ مرتبطٌ به
- F-20 (إرسالُ الطلب بواتساب) — لا اختبارَ مرتبطٌ به
- F-28 (تعليقُ متجر) — لا اختبارَ مرتبطٌ به
- F-31 (شكوى أو بلاغٌ ثمّ حلٌّ بتعويض) — لا اختبارَ مرتبطٌ به
- F-32 (مراجعةُ صنفٍ معلَّق) — لا اختبارَ مرتبطٌ به
- R1 — لا اختبارَ يحسمه بعد
- R12 — لا اختبارَ يحسمه بعد
- R17 — لا اختبارَ يحسمه بعد
- R18 — لا اختبارَ يحسمه بعد
- R2 — لا اختبارَ يحسمه بعد
- R24 — لا اختبارَ يحسمه بعد
- R3 — لا اختبارَ يحسمه بعد
- R5 — لا اختبارَ يحسمه بعد
- R6 — لا اختبارَ يحسمه بعد
- R9 — لا اختبارَ يحسمه بعد

