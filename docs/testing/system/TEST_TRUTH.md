# حقيقةُ الاختبار — التقرير

> **مولَّدٌ آليّاً** — `go run ./cmd/testtruth`.
> **ولا يُحرَّر بيد**، وحارسُ `TestTruthIsCurrent` يُسقط البناءَ إن شاخ.
> **والمصدرُ للآلة** [`TEST_TRUTH.json`](TEST_TRUTH.json).

---

# ١ · ما استُخرج من الشيفرة

**ولا رقمَ منه مكتوبٌ بيد.**

| ما هو | العدد |
|---|---|
| أبوابٌ في الموجّه | **348** |
| انتقالاتُ الطلب | **55** |
| أنواعُ قيدِ المحفظة | **14** — `adjustment` · `commission` · `compensation` · `driver_earning` · `merchant_earning` · `operating_expense` · `order_payment` · `payout` · `penalty` · `platform_expense` · `platform_profit` · `refund` · `reward` · `topup` |
| مواضعُ الإشعار | **49** — منها **12** موجَّهٌ بـ`Apps` |
| غرفُ البثّ | **5** — `customer` · `driver` · `merchant` · `ops` · `user` |
| تعريفاتُ الإعدادات | **143** — منها **115** مُغيِّرٌ للسلوك |
| حقولُ الطلب | **75** — من عقد `P-1` |
| ملفّاتُ اختبار | **355** |
| دوالُّ اختبار | **1522** |

---

# ٢ · الاختبارات

```
TOTAL      = 1522
MAPPED     = 438
INFRA      = 123
ORPHAN     = 961
```

**واليتيمُ اختبارٌ لا يعرف ماذا يحرس** — **ولا يُسقط البناءَ اليومَ**،
**ويُربَط مرحلةً بعد مرحلة.** وأكثرُها في:

| الحزمة | يتيمٌ |
|---|---|
| `qa` | 406 |
| `server` | 124 |
| `routing` | 76 |
| `orders_test` | 60 |
| `platform` | 58 |
| `orders` | 54 |
| `identity` | 23 |
| `settings` | 14 |
| `notify` | 13 |
| `push` | 11 |

---

# ٣ · التغطية

| السجلّ | العدد | مربوطٌ | بلا اختبار |
|---|---|---|---|
| **التدفّقات** | 36 | 33 | 3 |
| **العيوب** | 28 | 22 | 6 |
| **المخاطر** | 24 | 15 | 9 |
| **فجواتُ العقد** | 47 | 29 | 18 |
| **إعداداتُ السلوك** | 115 | 13 | 102 |

---

# ٤ · العيوبُ وحارسُها

| ID | العنوان | الحال | الاختبارات |
|---|---|---|---|
| **D1** | فكُّ الإسناد بلا حدثٍ في order_events | `FIXED_AND_PASSING` | `TestCENSUS_D1_ReleaseWritesEvent` |
| **D2** | convertLead بلا معاملةٍ واحدة · ٣ كتاباتٍ خطؤه… | `FIXED_AND_PASSING` | `TestATOMIC_LeadConversionIsOneUnit` · `TestFAIL_D2_ConvertLeadPartialStates` · `TestFIN_TargetRewardPrecedesCommit` · `TestFIN_TransactionBoundaries` · `TestUNIQ_ConcurrentConversionWithExistingOwner` · `TestUNIQ_ConcurrentLeadConversionMakesOneMerchant` · `TestUNIQ_DatabaseRefusesSecondMerchantForSameLead` · `TestUNIQ_FailureThenRetryMakesOneMerchant` · `TestRACE_DuplicateLeadConversion` · `TestXG29_FailureBeforeCommitLeavesNothing` · `TestXG29_FailureThenRetryGrantsRewardOnce` · `TestXG29_SuccessGrantsRewardExactlyOnce` |
| **D3** | «تذكّرني» تنقلب دائمةً بعد أوّل تجديد | `NO_REGRESSION_TEST_YET` | — |
| **D4** | سقفُ المفتوح داخلَ بوّابة واتساب | `FIXED_AND_PASSING` | `TestD4_BoundaryAtLimit` · `TestD4_CapHoldsWhenWhatsAppIsOff` · `TestD4_CapIndependentOfWhatsAppSetting` · `TestD4_ClosingAnOrderReleasesCapacity` · `TestD4_ConcurrentCreatesCannotExceedCap` · `TestD4_ConcurrentCustomCreatesCannotExceedCap` · `TestD4_CustomOrdersShareTheSameCap` · `TestD4_DifferentCustomersAreNotSerialized` · `TestD4_DistinctKeysAreCapped` · `TestD4_IdempotentRetryRecoversSameOrder` · `TestD4_LoweringCapKeepsExistingOrders` · `TestD4_RejectedCreateLeavesNothing` · `TestD4_StressReleaseAndIdempotency` · `TestD4_StressSequentialCap` · `TestCENSUS_D4_OpenLimitNestedInWhatsAppGate` |
| **D5** | المصروفُ والخزينةُ كتابتان بلا معاملة | `FIXED_AND_PASSING` | `TestATOMIC_ExpenseAndTreasuryAreOneUnit` · `TestFAIL_D5_ExpenseTreasuryPartial` · `TestFIN_ExpenseTreasuryInvariant` · `TestFIN_TransactionBoundaries` |
| **D6** | الطلبُ الخاصُّ لا ينادي cashBlocked | `EXPECTED_FAIL` | `TestCENSUS_D6_CustomOrderSkipsCashBan` |
| **D7** | سقفُ النقد يقيس المحصَّل لا المكشوف | `FIXED_AND_PASSING` | `TestD24_ComposesWithCashCeiling` · `TestD7_AtLimitExactlyIsAllowed` · `TestD7_CeilingUnderRepetition` · `TestD7_ConcurrentAssignmentsCannotOversubscribe` · `TestD7_CumulativeExposureIsCounted` · `TestD7_DriverAcceptObeysSameCeiling` · `TestD7_ExposureIsReleasedWhenOrderCloses` · `TestD7_OversizedCashOrderIsRefused` · `TestD7_WalletOrderIsNotBlocked` · `TestCENSUS_D7_CashCeilingCountsIncomingOrder` · `TestSampleFactory_DriverOnShiftWithCash` · `TestFIN_CashExposureContract` · `TestXG46_DriverAcceptNeedsOneConnection` |
| **D8** | الطلبُ الخاصُّ لا يطلب توثيقَ واتساب | `EXPECTED_FAIL` | `TestCENSUS_D6_D9_CustomOrderCreationGuards` |
| **D9** | الطلبُ الخاصُّ بلا حدثِ ''→pending | `EXPECTED_FAIL` | `TestCENSUS_D6_D9_CustomOrderCreationGuards` |
| **D10** | الاستعادةُ تُبطل نوعَ عميلٍ واحد | `FIXED_AND_PASSING` | `TestSessionClient_AdminLogoutAllStillGlobal` |
| **D11** | force_password_change بلا بوّابةٍ في أندرويد | `NO_REGRESSION_TEST_YET` | — |
| **D12** | Push.unregister بلا منادٍ | `FIXED_AND_PASSING` | `TestD12_AccountSwitchOnSameDevice` · `TestD12_ClientSendsDeviceTokenOnLogout` · `TestD12_DispatcherNoLongerTargetsLoggedOutDevice` · `TestD12_DispatcherSilentAfterTokenlessLogout` · `TestD12_FailureBetweenRevokeAndCleanupIsAtomic` · `TestD12_LogoutIsDeviceScoped` · `TestD12_LogoutIsIdempotent` · `TestD12_LogoutRemovesThisDeviceBinding` · `TestD12_LogoutWithoutDeviceTokenStillRemovesBinding` · `TestD12_SessionScopedCleanupKeepsOtherSession` · `TestD12_StressIdempotentLogout` · `TestD12_StressLoginRegisterLogout` · `TestD12_StressLogoutVsReRegisterRace` · `TestD12_StressTokenlessLogout` · `TestD12_StressTwoDevicesAndSwitch` · `TestD12_StressTwoSessionsAndClaim` · `TestD12_SuspensionIsNotLogout` · `TestD12_TokenRotationWithinSessionIsFullyCleaned` · `TestEV_PushTokenTargeting` |
| **D13** | سردُ /media/ مفتوحٌ — وإثباتُ التسليم فيه | `FIXED_AND_PASSING` | `TestD13_AvatarNeedsSignedURL` · `TestD13_DeliveryProofNeedsSignedURL` · `TestD13_MediaDirectoryIsNotListable` · `TestD13_PublicMediaStaysPublic` · `TestD13_SignedURLWorksAndForgeryDoesNot` · `TestFAIL_D13_MediaDirectoryListingOpen` |
| **D14** | handleWS لا يفحص ActiveStatus | `FIXED_AND_PASSING` | `TestCENSUS_D14_SuspendedCannotOpenSocket` · `TestSampleFactory_SuspendedIsRefused` · `TestD14_ActiveUserRealtimeUnaffected` · `TestD14_BlockedAndDeletedHaveNoRealtime` · `TestD14_EventScopeUnderRepetition` · `TestD14_ImpactSurfacesOnRealtimeAuthChange` · `TestD14_RealtimeEligibilityUnderRepetition` · `TestD14_SuspendedDriverScopedToItsOwnOrder` · `TestD14_SuspendedDriverScopedToOwnRoom` · `TestD14_SuspendedDriverWithoutActiveOrderGetsNoWork` · `TestD14_SuspendedHasNoBroadRealtimeAccess` |
| **D15** | AdminCreateUser في خطوتين | `FIXED_AND_PASSING` | `TestATOMIC_AdminUserCreationIsOneUnit` · `TestFAIL_D15_AdminCreateUserPartial` · `TestFAIL_D15_Reconciliation` |
| **D16** | طابورُ المواقع ملفٌّ بلا صاحب | `NO_REGRESSION_TEST_YET` | — |
| **D17** | الملاحةُ لا تعود بعد موت العمليّة | `NO_REGRESSION_TEST_YET` | — |
| **D18** | START_STICKY يعيد الخدمةَ بفترةِ الافتراض | `NO_REGRESSION_TEST_YET` | — |
| **D19** | LiveSocket يعيد الوصلَ بتوكنٍ منتهٍ | `FIXED_AND_PASSING` | `TestD19_ClientEvidenceIsRegistered` · `TestD19_ServerDeclaresAccessExpiry` |
| **D20** | البثُّ الحيُّ يتجاوز redactForMerchant | `FIXED_AND_PASSING` | `TestPublishOrderReachesEveryRoom` · `TestD22_OwnerPayloadObeysCustomerPrivacy` · `TestEV_MerchantDriverAssignment` · `TestEV_MerchantRealtimePrivacy` · `TestD20_BroadcastViewIsNotOverNarrow` · `TestD20_BroadcastViewObeysContract` · `TestD20_UnclassifiedFieldNeverReachesAnyAudience` · `TestD20_UnknownAudienceGetsNothing` · `TestD20_CustomerRealtimeVsREST` · `TestD20_MerchantRealtimeVsREST` · `TestOrderFieldsAllClassified` |
| **D21** | هاتفُ السائق يصل الزبون | `FIXED_AND_PASSING` | `TestEV_CustomerDriverAssignment` · `TestD21_CustomerRedactionAgainstContract` · `TestForbiddenFieldGuardCatchesLeak` · `TestOrderFieldsAllClassified` · `TestD21_NoRawOrderSerializationRemains` · `TestD21_RestOrderPrivacyMatrix` · `TestD21_RestPrivacyUnderRepetition` · `TestD23_CrossChannelPrivacyParity` |
| **D22** | الطلبُ الخاصُّ لا يُبثّ لصاحبه | `FIXED_AND_PASSING` | `TestD22_CustomOrderReachesItsOwner` · `TestD22_CustomOrderRoomSetIsComplete` · `TestD22_OwnerDeliveryUnderRepetition` · `TestD22_OwnerPayloadObeysCustomerPrivacy` · `TestEV_CustomOrderOwnerRealtime` · `TestD22_CustomOrderOwnerChannelContract` · `TestOrderFieldsAllClassified` |
| **D23** | حمولاتُ REST تكشف اقتصاداً داخليّاً | `FIXED_AND_PASSING` | `TestEV_CustomerDriverAssignment` · `TestD20_MerchantRealtimeVsREST` · `TestD21_CustomerRedactionAgainstContract` · `TestOrderFieldsAllClassified` · `TestD21_RestOrderPrivacyMatrix` · `TestD21_RestPrivacyUnderRepetition` · `TestD23_CrossChannelPrivacyParity` |
| **D24** | سقفُ الطلبات النشطة يُتجاوَز بالتزامن | `FIXED_AND_PASSING` | `TestD24_CapacityIsReleasedOnClose` · `TestD24_CapacityMatrix` · `TestD24_ComposesWithCashCeiling` · `TestD24_ConcurrentAcceptCannotExceedCap` · `TestD24_DifferentDriversAreNotSerialized` · `TestD24_LoweringLimitBlocksNewOnly` · `TestD24_ManualAssignmentObeysCap` · `TestD24_MixedPathRaceCannotExceedCap` · `TestD24_ReassignmentChecksTargetCapacity` · `TestFAIL_R7_DriverAcceptPartialState` · `TestRACE_MaxActiveOrders` |
| **D25** | هويّةُ متجرٍ واحدةٌ تصير متجرين بالتزامن | `FIXED_AND_PASSING` | `TestFAIL_D2_ConvertLeadPartialStates` · `TestUNIQ_ConcurrentConversionWithExistingOwner` · `TestUNIQ_ConcurrentLeadConversionMakesOneMerchant` · `TestUNIQ_DatabaseRefusesSecondMerchantForSameLead` · `TestUNIQ_FailureThenRetryMakesOneMerchant` · `TestRACE_DuplicateLeadConversion` |
| **D26** | إنذارُ الراصد يضيع بعد كتابة الوسم | `EXPECTED_FAIL` | `TestEV_R22WatchdogMarkerSuppressesRetry` |
| **D27** | سقوطُ الدفع بلا إعادةٍ دائمة | `FIXED_AND_PASSING` | `TestEV_R23PushFailureIsLost` |
| **D28** | لوحةُ الويب تعيد الوصلَ بالرمز المنتهي أبداً | `NO_REGRESSION_TEST_YET` | — |

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
| **XG-12** | `CRITICAL` | — | `COVERED` | `TestXG12_B3_LegacyOverReservationIsDetectedNotTruncated` · `TestXG12_C1_ConcurrentRequestsCannotOverReserve` · `TestXG12_C2_ReserveVsSpend` · `TestXG12_C4C5_ReleaseOnceAndNoDoubleDebit` · `TestXG12_F1F2_CreationIsOneUnit` · `TestXG12_F3F5_TerminalStateNeedsItsMoneyTruth` · `TestXG12_F4_AuditFailureRollsBackPayout` · `TestXG12_T1_RequestReservesAndSpendSeesAvailable` · `TestXG12_T2_PaidDebitsAndReleases` · `TestXG12_T3_RejectAndFailReleaseWithoutDebit` · `TestXG12_T4_ProcessingHoldsWithoutDebit` · `TestXG12_T5_ReversedCompensates` · `TestXG12_TreasuryCannotReserve` |
| **XG-13** | `HIGH` | — | `COVERED` | `TestXG13_DefaultIsPricingMargin` · `TestXG13_InvalidStoredValueFailsSafe` · `TestXG13_UnknownModeIsRejected` · `TestXG14_ReversalMirrorsSettlementInEveryMode` · `TestXG14_ThreeModesGiveTheirContract` · `TestFIN_CommissionSourceMatrix` · `TestFIN_SnapshotVsLiveEconomics` |
| **XG-14** | `CRITICAL` | — | `COVERED` | `TestXG13_DefaultIsPricingMargin` · `TestXG13_InvalidStoredValueFailsSafe` · `TestXG13_UnknownModeIsRejected` · `TestXG14_ReversalMirrorsSettlementInEveryMode` · `TestXG14_ThreeModesGiveTheirContract` |
| **XG-15** | `HIGH` | `sales.activation_orders` | `NOT_IMPLEMENTED` | — |
| **XG-16** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-17** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-18** | `BLOCKER` | — | `COVERED` | `TestFAIL_D2_ConvertLeadPartialStates` · `TestUNIQ_ConcurrentConversionWithExistingOwner` · `TestUNIQ_ConcurrentLeadConversionMakesOneMerchant` · `TestUNIQ_DatabaseRefusesSecondMerchantForSameLead` · `TestUNIQ_FailureThenRetryMakesOneMerchant` · `TestRACE_DuplicateLeadConversion` |
| **XG-19** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-20** | `CRITICAL` | — | `COVERED` | `TestAQ4_A1_SuccessCommitsBoth` · `TestAQ4_A2_AuditFailureRollsBackMoney` · `TestAQ4_A3_BusinessFailureLeavesNoAudit` · `TestAQ4_A5_RetryGivesOneOfEach` · `TestAQ4_A6_ConcurrentActionsKeepTheirOwnAudit` · `TestAQ4_A8_UncoveredActionsRemainBestEffort` · `TestAQ4_CriticalActionsUseTransactionalAudit` · `TestXG20_A2_AuditFailureRollsBackBusiness` · `TestXG20_A3_FailedBusinessLeavesNoAudit` · `TestXG20_C1C2_ConcurrentMutationsCorrelate` · `TestXG20_S1S2_RoleMutationAudited` · `TestXG20_S3_UserStatusAudited` · `TestXG20_S4_MerchantSuspendAudited` · `TestXG20_S5_SensitiveSettingAudited` · `TestXG20_S6_OpsOrderTransitionAudited` |
| **XG-21** | `BLOCKER` | — | `COVERED` | `TestD13_AvatarNeedsSignedURL` · `TestD13_DeliveryProofNeedsSignedURL` · `TestD13_MediaDirectoryIsNotListable` · `TestD13_PublicMediaStaysPublic` · `TestD13_SignedURLWorksAndForgeryDoesNot` |
| **XG-22** | `BLOCKER` | — | `COVERED` | `TestXG22_SuspendDuringTransitionIsDeterministic` · `TestXG22_T10_EnforcementIsServerSide` · `TestXG22_T1_SuspendedWithoutActiveOrderIsDenied` · `TestXG22_T2_SuspendedDriverCanFinishActiveOrder` · `TestXG22_T3_ExceptionDoesNotLeakToAnotherOrder` · `TestXG22_T4_ExceptionEndsAtTerminalState` · `TestXG22_T6_NormalActorUnchanged` · `TestXG22_T7_OpsCanStillResolveTheOrder` · `TestXG22_T8_BlockedHasNoException` · `TestXG22_T9_ExceptionDoesNotLeakAcrossRoles` |
| **XG-23** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-24** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-25** | `CRITICAL` | — | `COVERED` | `TestXQ2_C1_ConcurrentSettingChangeGivesNoHybridSnapshot` · `TestXQ2_F1F2_SnapshotIsAtomicWithTheOrder` · `TestXQ2_F3_UnreadableSettingBlocksCreation` · `TestXQ2_S1S2_OldOrderKeepsItsEconomicsNewOrderTakesTheNew` · `TestXQ2_S3_RefundUsesOriginalEconomics` · `TestXQ2_S4S5S6_SnapshotSurvivesSettingLoss` |
| **XG-26** | `CRITICAL` | — | `COVERED` | `TestXQ2_C1_ConcurrentSettingChangeGivesNoHybridSnapshot` · `TestXQ2_F1F2_SnapshotIsAtomicWithTheOrder` · `TestXQ2_F3_UnreadableSettingBlocksCreation` · `TestXQ2_S1S2_OldOrderKeepsItsEconomicsNewOrderTakesTheNew` · `TestXQ2_S3_RefundUsesOriginalEconomics` · `TestXQ2_S4S5S6_SnapshotSurvivesSettingLoss` |
| **XG-27** | `CRITICAL` | — | `COVERED` | `TestXQ2_C1_ConcurrentSettingChangeGivesNoHybridSnapshot` · `TestXQ2_F1F2_SnapshotIsAtomicWithTheOrder` · `TestXQ2_F3_UnreadableSettingBlocksCreation` · `TestXQ2_S1S2_OldOrderKeepsItsEconomicsNewOrderTakesTheNew` · `TestXQ2_S3_RefundUsesOriginalEconomics` · `TestXQ2_S4S5S6_SnapshotSurvivesSettingLoss` |
| **XG-28** | `CRITICAL` | — | `COVERED` | `TestXQ2_C1_ConcurrentSettingChangeGivesNoHybridSnapshot` · `TestXQ2_F1F2_SnapshotIsAtomicWithTheOrder` · `TestXQ2_F3_UnreadableSettingBlocksCreation` · `TestXQ2_S1S2_OldOrderKeepsItsEconomicsNewOrderTakesTheNew` · `TestXQ2_S3_RefundUsesOriginalEconomics` · `TestXQ2_S4S5S6_SnapshotSurvivesSettingLoss` |
| **XG-29** | `BLOCKER` | — | `COVERED` | `TestXG29_FailureBeforeCommitLeavesNothing` · `TestXG29_FailureThenRetryGrantsRewardOnce` · `TestXG29_SuccessGrantsRewardExactlyOnce` |
| **XG-30** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-32** | `CRITICAL` | — | `COVERED` | `TestXG29_FailureBeforeCommitLeavesNothing` · `TestXG29_FailureThenRetryGrantsRewardOnce` · `TestXG29_SuccessGrantsRewardExactlyOnce` · `TestXG32_ConcurrentTargetCrossingGrantsOnce` · `TestXG32_FailureRollbackThenRetryGrantsOnce` · `TestXG32_FirstConversionGrantsRewardImmediately` · `TestXG32_FurtherConversionsDoNotRepeatReward` · `TestXG32_PreviousMonthDoesNotSatisfyTarget` · `TestXG32_TargetTwoGrantsOnSecondOnly` |
| **XG-34** | `HIGH` | — | `EXPECTED_FAIL` | `TestDIAG_LocationWiderPool` · `TestXG34_DifferentKeysDoNotSerialize` · `TestXG34_NoTransactionIsLeftOpen` |
| **XG-35** | `CRITICAL` | — | `COVERED` | `TestAQ4_A1_SuccessCommitsBoth` · `TestAQ4_A2_AuditFailureRollsBackMoney` · `TestAQ4_A3_BusinessFailureLeavesNoAudit` · `TestAQ4_A5_RetryGivesOneOfEach` · `TestAQ4_A6_ConcurrentActionsKeepTheirOwnAudit` · `TestAQ4_A8_UncoveredActionsRemainBestEffort` · `TestAQ4_CriticalActionsUseTransactionalAudit` · `TestXG20_A2_AuditFailureRollsBackBusiness` · `TestXG20_A3_FailedBusinessLeavesNoAudit` · `TestXG20_C1C2_ConcurrentMutationsCorrelate` · `TestXG20_S1S2_RoleMutationAudited` · `TestXG20_S3_UserStatusAudited` · `TestXG20_S4_MerchantSuspendAudited` · `TestXG20_S5_SensitiveSettingAudited` · `TestXG20_S6_OpsOrderTransitionAudited` |
| **XG-41A** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-41B** | `MEDIUM` | — | `COVERED` | `TestXG41B_AuthUnavailableReachesTheReaderAsItself` · `TestXG41B_AuthUnavailableHasItsOwnMessage` |
| **XG-41C** | `HIGH` | — | `COVERED` | `TestXG41C_HarnessExclusivityIsEnforced` · `TestXG41C_PushQueueIsOwnedByItsProducer` |
| **XG-43** | `HIGH` | — | `COVERED` | `TestXG43_FinancialIdentityIsStableNotDisplayName` · `TestXG43_NoInvariantUsesDisplayIdentity` |
| **XG-44** | `MEDIUM` | — | `COVERED` | `TestXG44_SettleWaitsUntilTheTransferAttemptEnds` |
| **XG-47** | `MEDIUM` | — | `NOT_IMPLEMENTED` | — |
| **XG-48** | `HIGH` | — | `COVERED` | `TestXG48_CustomerCancelNeedsOneConnection` · `TestXG48_DeliverySettlementNeedsOneConnection` · `TestXG48_FailedDeliveryCompensationNeedsOneConnection` · `TestXG48_TransitionBothModesNeedOneConnection` · `TestXG48_TransitionUnderRepetitionOnOneConnection` |
| **XG-49** | `HIGH` | — | `COVERED` | `TestSEC7_SelfDeleteRevokesEverything` |
| **XG-46** | `HIGH` | — | `COVERED` | `TestXG46_DriverAcceptNeedsOneConnection` · `TestXG46_OrderCreateNeedsOneConnection` |
| **XG-45** | `MEDIUM` | — | `COVERED` | `TestXG45_EveryMobileReachableCodeHasArabic` |
| **XG-42** | `MEDIUM` | — | `COVERED` | `TestXG42_CapabilityUnionCustomRoleAndRevoke` · `TestXG42_ContactFieldsFollowCapabilityNotRoute` |
| **XG-36** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-38** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-40** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-39** | `HIGH` | — | `COVERED` | `TestXG39_C1_SuspendVsTransition` · `TestXG39_C2_BlockVsRefresh` · `TestXG39_S11S12_ReactivationDoesNotResurrect` · `TestXG39_S1S3S4_SuspensionKeepsSessionAndNarrowScope` · `TestXG39_S2_SuspendedWithoutOrderKeepsSessionButNoActivity` · `TestXG39_S5_SuspendedRefreshPreservesSameSession` · `TestXG39_S6S7_SuspendedCannotOpenNewSession` · `TestXG39_S8_RevokedSuspendedSessionStaysDenied` · `TestXG39_S9S10_BlockRevokesEverything` |
| **XG-37** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-33** | `CRITICAL` | — | `COVERED` | `TestIDEM_AllProtectedPathsUseCoordinator` · `TestIDEM_T10_CleanupSparesLiveClaim` · `TestIDEM_T11_SameKeyDifferentPayloadContractUnchanged` · `TestIDEM_T1_ConcurrentDuplicateExecutesOnce` · `TestIDEM_T2_OrphanBeforeTxIsReclaimed` · `TestIDEM_T3_StaleOwnerIsFenced` · `TestIDEM_T4_ActiveClaimCannotBeStolen` · `TestIDEM_T5_BusinessRollbackLeavesNothing` · `TestIDEM_T6_CommittedThenDeathReplaysWithoutDuplicate` · `TestIDEM_T8_TwoReclaimersExecuteOnce` · `TestIDEM_T9_StaleOwnerCannotDeleteNewerClaim` · `TestFAIL_C06_OrphanBeforeCommitBlocksOwner` · `TestFAIL_C06_TwoReclaimersExecuteNothing` · `TestIDEM_CleanupSparesLiveClaim` · `TestIDEM_CommittedBeforeResultDoesNotDuplicate` · `TestIDEM_LostResponseReplays` |
| **XG-31** | `CRITICAL` | — | `COVERED` | `TestOBL_BothInsufficient_AtomicOrigins` · `TestOBL_FutureEarningsSettleWithEvidence` · `TestOBL_MerchantInsufficient_OriginTraceable` · `TestOBL_MerchantSufficient_NoObligation` · `TestOBL_MultipleObligationsFIFO` · `TestOBL_RepAvailable_NoObligation` · `TestOBL_RepWithdrawn_OriginTraceable` · `TestOBL_ReplayCreatesNoDuplicate` · `TestFIN_XG10_CombinedMerchantAndRepInsufficiency` · `TestFIN_XG10_ConservationAcrossRefund` · `TestFIN_XG10_DebtSettlementArithmetic` · `TestFIN_XG10_RefundReplayDoesNotDoubleCharge` |

---

# ٦ · الشيخوخةُ والفجوات

```
STALE REFERENCES = 0
COVERAGE GAPS    = 18
```

## فجواتُ تغطية — **ما يحتاج اختباراً ولا اختبارَ له**

- D11 — لا اختبارَ انحدارٍ بعد
- D16 — لا اختبارَ انحدارٍ بعد
- D17 — لا اختبارَ انحدارٍ بعد
- D18 — لا اختبارَ انحدارٍ بعد
- D28 — لا اختبارَ انحدارٍ بعد
- D3 — لا اختبارَ انحدارٍ بعد
- F-06 (رفضُ المتجر) — لا اختبارَ مرتبطٌ به
- F-31 (شكوى أو بلاغٌ ثمّ حلٌّ بتعويض) — لا اختبارَ مرتبطٌ به
- F-32 (مراجعةُ صنفٍ معلَّق) — لا اختبارَ مرتبطٌ به
- R1 — لا اختبارَ يحسمه بعد
- R12 — لا اختبارَ يحسمه بعد
- R17 — لا اختبارَ يحسمه بعد
- R18 — لا اختبارَ يحسمه بعد
- R2 — لا اختبارَ يحسمه بعد
- R3 — لا اختبارَ يحسمه بعد
- R5 — لا اختبارَ يحسمه بعد
- R6 — لا اختبارَ يحسمه بعد
- R9 — لا اختبارَ يحسمه بعد

