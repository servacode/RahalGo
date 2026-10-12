-- ══════════════════════════════════════════════════════════════════════
-- **من وصلته رسالةُ الترحيب على الواتساب حسابُه موثَّق** (قرارُ المالك ٢٠٢٦-١٠-١٢)
-- ══════════════════════════════════════════════════════════════════════
--
-- المتجرُ والمندوبُ والسائقُ يُفتح حسابُهم ويُرسَل إليهم رقمُه وكلمتُه على الواتساب —
-- **فالرقمُ رقمُهم.** ومن الآن يُوسَم ذلك عند الإرسال (`MarkWhatsAppFromWelcome`)،
-- **وهذه للقائمين قبلَه**: من وصلته الرسالةُ (`welcome_sent_at`) ولم يُوثَّق بعد.
UPDATE users
   SET whatsapp_phone = phone, whatsapp_verified_at = welcome_sent_at, updated_at = now()
 WHERE welcome_sent_at IS NOT NULL
   AND whatsapp_verified_at IS NULL
   AND phone IS NOT NULL;
