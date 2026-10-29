package com.rahalgo.shared.merchant

import com.rahalgo.shared.model.Envelope
import com.rahalgo.shared.model.WalletStatement
import com.rahalgo.shared.net.ApiClient
import io.ktor.http.HttpMethod
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أبوابُ المتجر — كما تناديها بوّابتُه في الويب حرفاً بحرف**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (`backend/internal/server/server.go` — كتلةُ `/merchant`، قِيست نداءاتُها
 *  بنداً بنداً ٢٠٢٦-٠٨-٢٣.)
 *
 * **وكلُّها تحت `RequireRoles("merchant")`** — يفرضه المحرّكُ لا الشاشة.
 *
 * # ولماذا وُجد هذا الملفّ
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-٢٣: «بناء تطبيق المتجر بشكل كامل ومتكامل».)
 *
 * **وكانت بوّابةُ الويب هي بيتَ المتجر الوحيد** — يفتح متصفّحاً ليقبل
 * طلباً. **وصاحبُ مطعمٍ يداه في العجين لا يفتح متصفّحاً**: يحتاج جرساً
 * يرنّ وزرّاً واحداً.
 *
 * # ولا منطقَ هنا
 *
 * **نداءاتٌ وأنواعٌ لا غير** (`GROUND-RULES.md` §7.1) — والقرارُ في
 * المحرّك، والعرضُ في الشاشة. **وطبقةٌ وسطى تقرّر تصير محرّكاً ثانياً
 * يفترق عن الأوّل.**
 */
class MerchantApi(private val api: ApiClient) {

    // ══════════════════════════════════════════════════════════════════
    // **متاجرُه** — واحدٌ في الغالب، وقد يملك اثنين
    // ══════════════════════════════════════════════════════════════════
    //
    // **ولا يُفترض أنّه واحد**: من ملك فرعين ثمّ رأى فرعاً واحداً ظنّ
    // الآخرَ ضاع. **والقائمةُ تُقرأ كما تأتي.**

    // ══════════════════════════════════════════════════════════════════
    // **ومغلَّفٌ لا مصفوفة** — كشفه أوّلُ تشغيلٍ على الجهاز ٢٠٢٦-٠٨-٢٣
    // ══════════════════════════════════════════════════════════════════
    //
    // **كتبتُ العقدَ تخميناً فردّ المحرّكُ كائناً** — والشاشةُ عرضت
    // `Expected start of the array '[' but had '{'`.
    //
    // **و`self_manage_orders` يأتي معها عمداً**: إعدادُ منصّةٍ لا يملك
    // المتجرُ قراءتَه من بابه، **ويحتاجه ليعرف أيَعرض أزرارَ القبول** —
    // فالجوابُ مع السؤال لا نداءٌ ثانٍ لسطرٍ واحد.
    suspend fun stores(): StoresPage = api.call("/api/v1/merchant/stores")

    /** **إنذاراتُه** — ومن أُنذر يرى إنذارَه قبل أن يُحظر. */
    suspend fun warnings(): WarningsPage = api.call("/api/v1/merchant/warnings")

    // ══════════════════════════════════════════════════════════════════
    // **الطلبات — قلبُ التطبيق**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **طلباتُ متجرٍ** — صفحةٌ بعدّادات، كما يردّها المحرّك.
     *
     * **و`openOnly` الجاريةُ وحدَها** (تدقيقُ ٢٠٢٦-١٠-٢٧) — شاشةُ العمل لا
     * تخلط «تم التسليم» بالطلب الجديد، **والمنتهي في السجلّ.** وافتراضُه كما كان.
     */
    suspend fun orders(storeId: String, openOnly: Boolean = false): OrdersPage =
        api.call(
            "/api/v1/merchant/stores/$storeId/orders" + if (openOnly) "?open_only=true" else "",
        )

    /**
     * **سجلُّ ما انتهى** — قسمٌ منفصلٌ كما في الويب.
     *
     * (قرارُ المالك ٢٠٢٦-٠٨-٢٣.)
     *
     * **و«الطلبات» لاستقبال الجديد وحدَه** — **وخلطُ ما يُعمل بما مضى
     * يجعل صاحبَ المتجر يبحث عن طلبٍ جارٍ بين مئةٍ سُلّمت.**
     *
     * **والمحرّكُ يفرض `closed_only` من نفسه في وضع المنصّة** — وهذا
     * النداءُ يطلبها صراحةً في الوضعين.
     */
    suspend fun history(storeId: String, page: Int = 1): OrdersPage =
        api.call("/api/v1/merchant/stores/$storeId/orders?with_active=true&page=$page")

    /** **طلبٌ بتفصيله** — أصنافُه وخياراتُه وملاحظتُه. */
    suspend fun order(orderId: String): MerchantOrder =
        api.call("/api/v1/merchant/orders/$orderId")

    /**
     * **ينقل حالَ الطلب** — `accepted` أو `preparing` أو `rejected`.
     *
     * **والمحرّكُ يحرس الانتقال** (`statuses.go`): ما لا يجوز يُردّ،
     * **فلا تحرسه الشاشةُ مرّتين** — تعرضه وتقرأ الجواب.
     */
    suspend fun transition(orderId: String, to: String, note: String = ""): MerchantOrder =
        api.call(
            "/api/v1/merchant/orders/$orderId/transition",
            HttpMethod.Post,
            TransitionInput(to, note),
        )

    /**
     * **«جاهزٌ للاستلام»** — علامةٌ يرفعها من يعرف.
     *
     * **وبابٌ مستقلٌّ عن `transition`** لأنّه ليس انتقالَ حالٍ بل
     * إشارةُ مطبخ: **الطلبُ يبقى `preparing` ويصير السائقُ مدعوّاً.**
     */
    suspend fun markReady(orderId: String): MerchantOrder =
        api.call("/api/v1/merchant/orders/$orderId/ready", HttpMethod.Post, Unit)

    // ══════════════════════════════════════════════════════════════════
    // **القائمة — بضاعتُه بيده**
    // ══════════════════════════════════════════════════════════════════

    suspend fun menu(storeId: String): List<MenuSection> =
        api.call("/api/v1/merchant/stores/$storeId/menu")

    /**
     * **يبدّل توفّرَ صنفٍ بضغطة** — أكثرُ فعلٍ يتكرّر في يومه.
     *
     * **ونفد الصنفُ فعلٌ من ثانية**: لو احتاج فتحَ شاشةِ تعديلٍ كاملةٍ
     * **لَتركه معروضاً ونفد** — فيُطلب ما ليس عنده.
     */
    suspend fun setAvailability(itemId: String, available: Boolean): Ack =
        api.call(
            "/api/v1/merchant/menu/items/$itemId/availability",
            HttpMethod.Patch,
            AvailabilityInput(available),
        )

    /**
     * **أقسامُ السوق بصورها** — ليختار الصنفُ موضعَه، **ولتُبنى شبكةُ
     * الأصناف بوجوهها لا بحروف.**
     */
    suspend fun platformSections(): SectionsPage =
        api.call("/api/v1/merchant/platform-sections")

    /**
     * ══════════════════════════════════════════════════════════════════
     * **أقسامُ هذا المتجر — ما يبيعه هو**
     * ══════════════════════════════════════════════════════════════════
     *
     * (بلاغُ المالك ٢٠٢٦-٠٨-٢٦: «مو معقول كلُّ الأقسام تطلع عند كلّ
     *  المتاجر… مطعمٌ شو علاقتُه بالأحذية؟»)
     *
     * **و`platformSections` تبقى للاختيار** — منها يعلن ما يبيع.
     * **وهذه تُقرأ في نموذج الصنف**، فلا يرى إلّا ما أعلن.
     *
     * **وفارغةٌ تعني الكلّ** — فمتجرٌ لم يختر بعدُ لا يُحبس بلا أقسام.
     */
    suspend fun storeSections(storeId: String): SectionsPage =
        api.call("/api/v1/merchant/stores/$storeId/sections")

    /** **يستبدل القائمةَ كلَّها** — فمن أزال قسماً يزول فعلاً. */
    suspend fun setStoreSections(storeId: String, ids: List<String>): Ack =
        api.call(
            "/api/v1/merchant/stores/$storeId/sections",
            HttpMethod.Put,
            mapOf("sections" to ids),
        )

    /** **بمفتاح منع التكرار** — صنفٌ ضاع ردُّه فأُعيد لا يُدرج ثانيةً. */
    suspend fun createItem(storeId: String, input: MenuItemInput, idempotencyKey: String? = null): MenuItem =
        api.call(
            "/api/v1/merchant/stores/$storeId/menu/items",
            HttpMethod.Post,
            input,
            idempotencyKey = idempotencyKey,
        )

    suspend fun updateItem(itemId: String, input: MenuItemInput): MenuItem =
        api.call("/api/v1/merchant/menu/items/$itemId", HttpMethod.Patch, input)

    suspend fun deleteItem(itemId: String): Ack =
        api.call("/api/v1/merchant/menu/items/$itemId", HttpMethod.Delete, Unit)

    suspend fun createSection(storeId: String, name: String): MenuSection =
        api.call(
            "/api/v1/merchant/stores/$storeId/menu/sections",
            HttpMethod.Post,
            SectionInput(name),
        )

    // ══════════════════════════════════════════════════════════════════
    // **الساعاتُ والإعدادات**
    // ══════════════════════════════════════════════════════════════════

    suspend fun hours(storeId: String): List<DayHours> =
        api.call("/api/v1/merchant/stores/$storeId/hours")

    suspend fun setHours(storeId: String, days: List<DayHours>): Ack =
        api.call("/api/v1/merchant/stores/$storeId/hours", HttpMethod.Put, HoursInput(days))

    /**
     * **يحفظ إعداداتِه** — ويردّ `{"updated": true}` لا المتجرَ.
     *
     * **فالشاشةُ تُعيد القراءةَ بعده** — **ومن بنى الحالَ من ردٍّ لا
     * يحمله قرأ حقولاً فارغةً فمحا ما على الشاشة.**
     */
    suspend fun settings(storeId: String, input: StoreSettingsInput): Updated =
        api.call("/api/v1/merchant/stores/$storeId/settings", HttpMethod.Patch, input)

    /**
     * **سعرُ صرف المتجر** (طلبُ المالك ٢٠٢٦-١٠-١٢) — وتتبعه أصنافُه الدولاريّة.
     * **وقفزةٌ فوق ١٠٪ تُردّ** `exchange_rate_jump` بمثالٍ حتّى يُرسَل `confirm`.
     */
    suspend fun setExchangeRate(storeId: String, rate: Double, confirm: Boolean = false): ExchangeRateResult =
        api.call("/api/v1/merchant/stores/$storeId/exchange-rate", HttpMethod.Patch,
            ExchangeRateInput(rate, confirm))

    /**
     * **يبدّل طريقةَ مستحقّاته** — `cash` أو `wallet`، للطلبات الجديدة وحدَها
     * (قرارُ المالك ٢٠٢٦-١٠-٠١: «تكون بإعدادات المتجر ويقدر يبدّلها»).
     */
    suspend fun setSettlementMethod(storeId: String, method: String): SettlementMethodAck =
        api.call(
            "/api/v1/merchant/stores/$storeId/settlement-method",
            HttpMethod.Patch,
            mapOf("method" to method),
        )

    /**
     * ══════════════════════════════════════════════════════════════════
     * **يفتح المتجرَ أو يغلقه**
     * ══════════════════════════════════════════════════════════════════
     *
     * (تصحيحُ المالك ٢٠٢٦-٠٨-٢٣: «المتجرُ لا يستقبل طلبات — إذا كان
     *  مفتوحاً يستقبل وإذا كان مغلقاً لا يستقبل».)
     *
     * **وهذا هو البابُ الوحيدُ للإغلاق** — `/settings` لا يقبل حقلاً
     * له إطلاقاً (قِيس من `merchant_ops_handlers.go`: يقبل مدّةَ
     * التحضير والحدَّ الأدنى والعنوانَ والدبّوس لا غير).
     *
     * **وكنتُ أرسل `accepts_orders` إلى `/settings`** — **فيُقرأ حقلاً
     * مجهولاً ويُطرح صامتاً**: يقلب المفتاحُ نفسَه في الشاشة ولا يتبدّل
     * شيءٌ في القاعدة، **ويظنّ صاحبُ المتجر أنّه أغلق وهو مفتوح.**
     *
     * **و`emergency_closed` جزءٌ من معادلة «مفتوح» نفسِها**
     * (`orders/hours.go`) — فليس إيقافَ استقبالٍ بل إغلاقاً حقيقيّاً.
     */
    suspend fun setClosed(storeId: String, closed: Boolean): ClosedAck =
        api.call(
            "/api/v1/merchant/stores/$storeId/emergency",
            HttpMethod.Post,
            ClosedInput(closed),
        )

    // ══════════════════════════════════════════════════════════════════
    // **المالُ والتقارير**
    // ══════════════════════════════════════════════════════════════════

    /** **محفظتُه** — والكشفُ عامٌّ لكلّ الأدوار، فلا نداءَ خاصّ. */
    suspend fun wallet(): WalletStatement = api.call("/api/v1/me/wallet")

    /**
     * **تقريرُ مدًى** — و`from`/`to` بصيغة `YYYY-MM-DD`.
     *
     * **واليومُ وحدَه يُطلب بمساواتهما** — **ومدًى من سبعة أيّامٍ يُقرأ
     * «اليوم» فيظنّ صاحبُ المتجر يومَه أكبرَ ممّا هو.**
     */
    suspend fun reports(storeId: String, from: String = "", to: String = ""): StoreReport {
        val q = if (from.isEmpty()) "" else "?from=$from&to=$to"
        return api.call("/api/v1/merchant/stores/$storeId/reports$q")
    }

    // ══════════════════════════════════════════════════════════════════
    // **البلاغُ على السائق — وما صار به**
    // ══════════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-٠٨-٢٣: «بسجلّ الطلبات يجب أن يكون هناك زرُّ إبلاغٍ
    //  عن السائق»، و«بالقائمة الجانبيّة قسمُ الشكاوي والبلاغات».)
    //
    // **وشروطُ القبول ثلاثةٌ في المحرّك** (`support/merchant_report.go`)،
    // **تُقرأ في الشاشة قبل أن يُعرض الزرّ** — وإلّا ردّ المحرّكُ خطأً
    // لا يفهمه صاحبُ المتجر:
    //
    //   ١ · الطلبُ مُغلقٌ فعلاً   (`closed_at` ليس فارغاً)
    //   ٢ · له سائقٌ مُسنَد      (بلاغٌ بلا مشكوٍّ عليه يقف بلا طرف)
    //   ٣ · ضمن مهلة الشكوى      (تحدّدها الإعدادات)

    /** **أسبابُه أربعة** — والمحرّكُ يرفض ما سواها. */
    suspend fun reportReasons(): ReasonsPage = api.call("/api/v1/merchant/report-reasons")

    suspend fun reportDriver(orderId: String, reason: String, note: String): Ack =
        api.call("/api/v1/merchant/orders/$orderId/report", HttpMethod.Post,
            ReportInput(reason, note))

    /**
     * **بلاغاتُه هو لا شكاوى الزبائن عليه.**
     *
     * **وليس `/my/tickets`** — ذاك يفلتر بـ`customer_id`، **وبلاغُ المتجر
     * يحمل في تلك الخانة زبونَ الطلب.** فلو رُبط به لرأى صاحبُ المتجر
     * **ما شُكي به عليه**. البابُ الصحيح `/merchant/my-reports` يفلتر
     * بـ`created_by`.
     */
    /**
     * **الشكاوى التي رُفعت على متجره** — (بلاغُ المالك ٢٠٢٦-٠٨-٢٦:
     * «الشكاوي يلي عليه ويلي اله»).
     *
     * **ولا يُذكر من اشتكى** — اسمُ الزبون ليس من حقّ المتجر، **ومن
     * عرفه قد يعاقبه في طلبه القادم.** ورقمُ الطلب يكفي للفهم.
     */
    suspend fun reportsAgainst(storeId: String): ReportsPage =
        api.call("/api/v1/merchant/stores/$storeId/reports-against")

    suspend fun myReports(): ReportsPage = api.call("/api/v1/merchant/my-reports")

    // ══════════════════════════════════════════════════════════════════
    // **وصورةُ الصنف يرفعها بنفسه**
    // ══════════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-٠٨-٠٧: «كلُّ منتجٍ بالإضافة إلى تفاصيله يكون له
    //  صورة».)
    //
    // **وبابُه أضيقُ من باب الإدارة**: `merchantKinds` في المحرّك يحصر
    // الأنواعَ — **ولا يرفع تاجرٌ شعارَ المنصّة.**
    /**
     * **يرفع صورةَ صنفٍ ويعيد معرّفَها ومصغّرتَها.**
     *
     * **وكانت تعيد المعرّفَ وحدَه وترمي `thumb_url`** — والمحرّكُ
     * يرسلهما معاً. **فبعد الرفع لا يجد الحقلُ عنواناً يعرضه**، فيرسم
     * الحرفَ الاحتياطيّ، **ويظنّ صاحبُ المتجر أنّ الصورةَ لم تُحفظ**
     * (بلاغُ المالك ٢٠٢٦-٠٨-٢٩).
     *
     * **وهي محفوظةٌ فعلاً** — المعرّفُ يُرسَل مع الصنف ويُخزَّن.
     */
    suspend fun uploadItemImage(fileName: String, bytes: ByteArray): MediaRef {
        val raw = api.upload(
            "/api/v1/merchant/media", fileName, bytes, mapOf("kind" to "menu_item"),
        )
        return api.json.decodeFromString<Envelope<MediaRef>>(raw).data ?: MediaRef()
    }

    /**
     * **قائمةُ المشروبات** (قرارُ المالك ٢٠٢٦-١٠-١٩) — تُكتب مرّةً وينسخها المحرّكُ
     * إلى كلّ الأصناف، **ويتبع الدولاريُّ سعرَ الصرف.**
     */
    suspend fun drinks(storeId: String): List<Drink> =
        api.call<DrinksBody>("/api/v1/merchant/stores/$storeId/drinks").drinks

    suspend fun setDrinks(storeId: String, drinks: List<Drink>): List<Drink> =
        api.call<DrinksBody>("/api/v1/merchant/stores/$storeId/drinks", HttpMethod.Put, DrinksBody(drinks)).drinks

    /** **خياراتُ الصنف الجاهزة** (٢٠٢٦-١٠-٢٧) — كلُّها معلَّمةً بما يطلع عنده. */
    suspend fun optionPresets(storeId: String): OptionPresetsBody =
        api.call("/api/v1/merchant/stores/$storeId/option-presets")

    suspend fun setOptionPresets(storeId: String, body: SetOptionPresets): OptionPresetsBody =
        api.call("/api/v1/merchant/stores/$storeId/option-presets", HttpMethod.Put, body)

    /** **نسخُ صنفٍ بصورته وخياراته** — ويُرجع معرّفَ النسخة. */
    suspend fun duplicateItem(itemId: String): String =
        api.call<DuplicatedItem>("/api/v1/merchant/menu/items/$itemId/duplicate", HttpMethod.Post, emptyMap<String, String>()).id

    /** **الأسعارُ دفعةً واحدة** — معاينةٌ ثمّ تطبيق. */
    suspend fun bulkPrice(storeId: String, input: BulkPriceInput): BulkPriceResult =
        api.call("/api/v1/merchant/stores/$storeId/menu/prices", HttpMethod.Post, input)

    /** **يرفع شعارَ متجره** (طلبُ المالك ٢٠٢٦-١٠-١٨) — ثمّ يُربط بـ`settings`. */
    suspend fun uploadLogo(fileName: String, bytes: ByteArray): MediaRef {
        val raw = api.upload(
            "/api/v1/merchant/media", fileName, bytes, mapOf("kind" to "merchant_logo"),
        )
        return api.json.decodeFromString<Envelope<MediaRef>>(raw).data ?: MediaRef()
    }

    // **والقائمةُ والساعاتُ مصفوفتان خامّتان** — لا مغلَّفَ لهما،
    // **قِيس من `merchant_handlers.go` و`merchant_ops_handlers.go`.**

    // ══════════════════════════════════════════════════════════════════
    // **رقمُ الطلبات حسب الدوام وموظّفو المتجر** (قرارُ المالك ٢٠٢٦-١٠-٢٠)
    // ══════════════════════════════════════════════════════════════════
    //
    // كما تناديها لوحةُ المتجر بالويب (`StoreStaffAdmin.tsx`). **والموظّفُ لا يدير
    // الموظّفين** — المحرّكُ يردّه `staff_not_allowed` (`notStaff`). والأنواعُ في `Staff.kt`.

    suspend fun orderPhones(storeId: String): OrderPhones =
        api.call("/api/v1/merchant/stores/$storeId/order-phones")

    // **والحقولُ كلُّها تُرسَل صراحةً** — المُرمِّزُ يُسقط ما يساوي قيمتَه الافتراضيّة،
    // **فـ«دوام كامل» (`full`) كان لا يُرسَل أصلاً** فيردّه المحرّكُ «تحقق من البيانات».
    suspend fun setOrderPhones(storeId: String, phones: OrderPhones): OrderPhones =
        api.call(
            "/api/v1/merchant/stores/$storeId/order-phones", HttpMethod.Put,
            mapOf(
                "mode" to phones.mode,
                "morning_phone" to phones.morning,
                "evening_phone" to phones.evening,
                "evening_from" to phones.eveningFrom,
            ),
        )

    suspend fun staff(): List<StaffMember> =
        api.call<StaffPage>("/api/v1/merchant/staff").staff

    suspend fun addStaff(phone: String, name: String): StaffAdded =
        api.call("/api/v1/merchant/staff", HttpMethod.Post, StaffInput(phone, name))

    suspend fun removeStaff(staffId: String): StaffRemoved =
        api.call("/api/v1/merchant/staff/$staffId", HttpMethod.Delete, Unit)
}

// ══════════════════════════════════════════════════════════════════════
// **الأنواع — كما يكتبها المحرّك**
// ══════════════════════════════════════════════════════════════════════

/**
 * **متاجرُه ومعها إعدادُ إدارة الطلبات** — مغلَّفٌ واحدٌ من المحرّك.
 *
 * **و`selfManageOrders` يقول أيَعرض أزرارَ القبول**: في وضع «المنصّةُ
 * تدير» **تُخفى الجاريةُ عنه ولا يقبل شيئاً** — والسجلُّ يبقى كاملاً.
 */
@Serializable
data class StoresPage(
    val stores: List<Store> = emptyList(),
    @SerialName("self_manage_orders") val selfManageOrders: Boolean = false,
    /** **حدُّ اسم الصنف بالأحرف** (٢٠٢٦-١٠-١٤) — يعدّه المحرّرُ وهو يكتب. */
    @SerialName("item_name_max") val itemNameMax: Int = 30,
)

@Serializable
data class Store(
    val id: String = "",
    val name: String = "",
    val status: String = "",
    @SerialName("category_icon") val categoryIcon: String = "",
    @SerialName("logo_thumb_url") val logoThumbUrl: String? = null,
    /**
     * **أغلقه طارئاً** — وهو المفتاحُ الذي بيده فعلاً.
     */
    @SerialName("emergency_closed") val emergencyClosed: Boolean = false,
    @SerialName("default_prep_minutes") val prepMinutes: Int = 0,
    @SerialName("min_order") val minOrder: Long = 0,
    @SerialName("address_text") val addressText: String = "",
    val lat: Double? = null,
    val lng: Double? = null,
    /**
     * ══════════════════════════════════════════════════════════════════
     * **الحالةُ الفعليّةُ يحسبها الخادم — لا التطبيق** (A3، ٢٠٢٦-٠٩-٢٦)
     * ══════════════════════════════════════════════════════════════════
     *
     * **صار المحرّكُ يردّها الآن** (`merchant_handlers.go`): `open_now`
     * بمنطق `OpenNowSQL` نفسِه الذي يحكم إنشاءَ الطلب، **فلا يفترق ما
     * يراه المتجرُ عمّا يقبله الخادم.** و`next_open` موعدُ الفتح القادم
     * أو `null` إن كان مفتوحاً الآن أو بلا دوام.
     *
     * **وكان التطبيقُ لا يعرف حالتَه الفعليّة** — يرى «موقوف/نشط» و«أغلق
     * طارئاً» ولا يعرف أمفتوحٌ بساعاته الآن. **والكلمةُ تُؤلَّف من هذه
     * الحقول** (`status` + `emergencyClosed` + `openNow` + `nextOpen`)
     * **بلا إعادةِ حسابِ الساعات في الجهاز** — نصٌّ واحدٌ لا نصّان.
     */
    @SerialName("open_now") val openNow: Boolean = false,
    @SerialName("next_open") val nextOpen: String? = null,
    /**
     * **طريقةُ استلامِ المستحقّات** — `cash` نقداً أو `wallet` من المحفظة.
     * **للعرضِ فقط**: المتجرُ لا يغيّرها (الأدمن وحدَه). و`unpaidCashDue`
     * مجموعُ المستحقّ النقديّ غير المسدَّد (صفرٌ للمحفظيّ).
     */
    @SerialName("settlement_method") val settlementMethod: String = "cash",
    @SerialName("unpaid_cash_due") val unpaidCashDue: Long = 0,
    /**
     * **وهل اختار بنفسه أم بقي على الافتراض** (طلبُ المالك ٢٠٢٦-١٠-٠٨) —
     * فلا يُظهر التطبيقُ اختياراً لم يقع، **ويسأله أن يختار.** وافتراضُها
     * `true` لمحرّكٍ أقدمَ لا يردّها — فلا يُسأل عمّا اختاره.
     */
    @SerialName("settlement_confirmed") val settlementConfirmed: Boolean = true,
    @SerialName("accepts_returns") val acceptsReturns: Boolean = false,
    @SerialName("returns_confirmed") val returnsConfirmed: Boolean = true,
    /** **ضبط دوامَه؟** — وافتراضُها `true` لمحرّكٍ أقدمَ لا يردّها. */
    @SerialName("has_hours") val hasHours: Boolean = true,
    /** **سعرُ صرفه** (ليرةٌ للدولار) — وفراغُه: لم يكتبه بعد (٢٠٢٦-١٠-١٢). */
    @SerialName("exchange_rate") val exchangeRate: Double? = null,
    /** **بأيّ عملةٍ يرى زبونُه أسعارَه** «SYP» أو «USD» — والدفعُ بالليرة دائماً. */
    @SerialName("display_currency") val displayCurrency: String = "SYP",
)

@Serializable
data class WarningsPage(val warnings: List<Warning> = emptyList())

@Serializable
data class Warning(
    val id: String = "",
    val reason: String = "",
    val note: String = "",
    @SerialName("created_at") val createdAt: String = "",
)

/**
 * **صفحةُ طلباتٍ بعدّاداتها** — كما يردّها المحرّك.
 *
 * **والعدّاداتُ تُقرأ ولا تُحسب في الشاشة**: عدٌّ في الجهاز يقرأ الصفحةَ
 * الحاضرةَ وحدَها، **فيقول «طلبان» ووراءهما عشرون.**
 */
@Serializable
data class OrdersPage(
    val orders: List<MerchantOrder> = emptyList(),
    val total: Int = 0,
    val page: Int = 1,
    @SerialName("per_page") val perPage: Int = 20,
    @SerialName("status_counts") val statusCounts: Map<String, Int> = emptyMap(),
    /**
     * **أرقامُ الطلبات التي يملك الإبلاغَ عنها** — يقولها المحرّك.
     *
     * **ولا تُشتقّ في الشاشة**: شروطُ القبول ثلاثة (`support/merchant_report.go`)،
     * **وأحدُها محجوبٌ عن التطبيق عمداً** — `redactForMerchant` يمحو
     * `driver_id` قبل أن يخرج الردّ. **فزرٌّ يُعرض على طلبٍ بلا سائقٍ
     * يردّ خطأً لا يفهمه صاحبُ المطعم.**
     *
     * **وتخلو ممّا أُبلغ عنه مرّةً** — فلا يُبلَّغ على الطلب مرّتين.
     */
    val reportable: List<String> = emptyList(),
)

/**
 * **طلبٌ كما يراه المتجر — ولا اسمَ زبونٍ ولا رقمَه.**
 *
 * **الزبونُ لا يعرف المتجرَ والمتجرُ لا يحتاج الزبون** — السائقُ بينهما.
 * **ورقمُ زبونٍ في يد متجرٍ بابُ اتّصالٍ مباشرٍ يوماً ما.**
 */
@Serializable
data class MerchantOrder(
    val id: String = "",
    /** **رقمُه المعروض** — عددٌ لا نصّ، كما يكتبه المحرّك. */
    val number: Long = 0,
    val status: String = "",
    val subtotal: Long = 0,
    @SerialName("delivery_fee") val deliveryFee: Long = 0,
    val total: Long = 0,
    /**
     * ══════════════════════════════════════════════════════════════════
     * **شفافيّةُ المتجر — كم أخذت المنصّة وكم بقي له**
     * ══════════════════════════════════════════════════════════════════
     *
     * (بلاغُ المالك ٢٠٢٦-٠٨-٢٦: «سجلّ الطلبات ما فيه شقد المبلغ المباع
     *  وشقد نسبة العمولة للمنصّة — هيك لازم يكون بشفافية».)
     *
     * **وتصل من المحرّك لصاحب المتجر وحدَه** — ولا تُرسل للزبون.
     *
     * **وطلبٌ لم يُسلَّم بعدُ عمولتُه مقدَّرةٌ بالنسبة** — فتُعرض
     * النسبةُ معها كي لا يُقرأ التقديرُ رقماً نهائيّا.
     */
    @SerialName("platform_commission") val platformCommission: Long = 0,
    @SerialName("merchant_net") val merchantNet: Long = 0,
    @SerialName("commission_percent") val commissionPercent: Int = 0,
    /**
     * **النسبةُ بمنزلتين** (١٫٥) — قرارُ المالك ٢٠٢٦-١٠-١٨.
     *
     * **و`commission_percent` تبقى عدداً صحيحاً مقرَّباً** — النسخُ المنشورةُ
     * تفكّها `Int` فتسقط على `1.5`. **وخادمٌ أقدمُ لا يرسل هذه** ⇒ `null`.
     */
    @SerialName("commission_percent_exact") val commissionPercentExact: Double? = null,
    val items: List<OrderLine> = emptyList(),
    val notes: String = "",
    @SerialName("created_at") val createdAt: String = "",
    /** **لحظةُ «جاهز للاستلام»** — وفارغةٌ ما لم يُعلنها المتجر (`ready_at`). */
    @SerialName("ready_at") val readyAt: String? = null,
)

@Serializable
data class OrderLine(
    val id: String = "",
    val name: String = "",
    val qty: Int = 1,
    @SerialName("unit_price") val unitPrice: Long = 0,
    val note: String = "",
    /** **ما اختاره الزبون** — حجمٌ وإضافات، وبلاها يُصنع الطلبُ ناقصاً. */
    val options: List<LineOption> = emptyList(),
)

@Serializable
data class LineOption(
    val group: String = "",
    val name: String = "",
    @SerialName("price_delta") val priceDelta: Long = 0,
)

@Serializable
data class MenuSection(
    val id: String = "",
    val name: String = "",
    val items: List<MenuItem> = emptyList(),
)

@Serializable
data class MenuItem(
    val id: String = "",
    val name: String = "",
    val description: String = "",
    /**
     * **سعرُ الشراء** — ما يقبضه هو، لا ما يدفعه الزبون.
     *
     * **والمحرّكُ يحجب سعرَ البيع عنه** (`merchant_handlers.go`): «هو يضع
     * سعرَه ويقبض عليه، وما تبيع به المنصّةُ ليس شأنَه».
     */
    val price: Long = 0,
    val available: Boolean = true,
    val approved: Boolean = true,
    @SerialName("review_note") val reviewNote: String = "",
    @SerialName("image_url") val imageUrl: String? = null,
    @SerialName("image_thumb_url") val imageThumbUrl: String? = null,
    /**
     * **قسمُ السوق الذي يظهر فيه للزبون** — لا قسمُ مطبخه.
     *
     * **وهو ما تُرتَّب به الشاشة** (طلبُ المالك ٢٠٢٦-٠٨-٢٣): **الزبونُ
     * يتصفّح بأقسام السوق**، فمن رتّب أصنافَه بترتيبِ مطبخه لم يعرف
     * أين تقع بضاعتُه في السوق.
     */
    @SerialName("platform_section_id") val platformSectionId: String? = null,
    @SerialName("platform_section_name") val platformSectionName: String = "",
    /** **عملةُ الصنف وسعرُه بالدولار** (٢٠٢٦-١٠-١٢) — و`price` بالليرة محسوبٌ منه. */
    @SerialName("price_currency") val priceCurrency: String = "SYP",
    @SerialName("merchant_price_usd") val merchantPriceUsd: Double? = null,
    /** **خياراتُه** — تُقرأ لتُعرض في المحرّر وتُعاد كما هي إن لم تُمسّ. */
    val modifiers: List<ModifierGroup> = emptyList(),
    /** **«أضف مشروباتي»** (٢٠٢٦-١٠-٢٩) — قائمةُ مشروبات المتجر على هذا الصنف. ومحرّكٌ قديمٌ لا يرسله ⇒ نعم. */
    @SerialName("store_drinks") val storeDrinks: Boolean = true,
)

@Serializable
data class SectionsPage(val sections: List<PlatformSectionRef> = emptyList())

@Serializable
data class PlatformSectionRef(
    val id: String = "",
    val name: String = "",
    @SerialName("image_url") val imageUrl: String? = null,
    @SerialName("image_thumb_url") val imageThumbUrl: String? = null,
)

@Serializable
data class DayHours(
    /** ٠ الأحد … ٦ السبت — كما يرقّمها المحرّك (`catalog/hours.go`). */
    @SerialName("day_of_week") val dayOfWeek: Int = 0,
    val closed: Boolean = false,
    @SerialName("open_time") val openTime: String = "",
    @SerialName("close_time") val closeTime: String = "",
    /**
     * **«مفتوح ٢٤ ساعة»** (قرارُ المالك ٢٠٢٦-١٠-١٨) — يُقرأ ويُكتب.
     *
     * **والخادمُ يكتب معها ٠٠:٠٠ ← ٠٠:٠٠** — فخادمٌ أقدمُ يتجاهلها ويحفظ
     * الساعتين، والمعنى نفسُه.
     */
    @SerialName("open_24h") val open24h: Boolean = false,
) {
    /** **أهو ٢٤ ساعة؟** — الرايةُ، أو ٠٠:٠٠ ← ٠٠:٠٠ من خادمٍ لا يرسلها. */
    val isOpen24h: Boolean
        get() = open24h || (openTime.take(5) == "00:00" && closeTime.take(5) == "00:00")

    /**
     * **تشغيلُ «٢٤ ساعة» أو إطفاؤه.**
     *
     * **والإطفاءُ يعيد ٩–٢٣** — ٠٠:٠٠ ← ٠٠:٠٠ بلا رايةٍ تُقرأ ٢٤ ساعة ثانيةً.
     */
    fun with24h(on: Boolean): DayHours =
        if (on) copy(closed = false, open24h = true, openTime = "00:00", closeTime = "00:00")
        else copy(open24h = false, openTime = "09:00", closeTime = "23:00")
}

/**
 * **نسبةٌ للعرض بلا أصفارٍ زائدة** — «1.5» و«2.25» و«10».
 *
 * **والدقيقةُ إن وصلت، وإلّا الصحيحةُ** — خادمٌ أقدمُ يرسل الصحيحةَ وحدَها.
 * **ولا `String.format`**: الحزمةُ المشتركةُ لا تعرف `java.util.Locale`.
 */
fun percentLabel(exact: Double?, rounded: Int): String {
    if (exact == null) return rounded.toString()
    val h = kotlin.math.round(exact * 100).toLong()
    val sign = if (h < 0) "-" else ""
    val a = kotlin.math.abs(h)
    val whole = a / 100
    val frac = a % 100
    return when {
        frac == 0L -> "$sign$whole"
        frac % 10 == 0L -> "$sign$whole.${frac / 10}"
        else -> "$sign$whole.${frac.toString().padStart(2, '0')}"
    }
}

/** **نسبةُ عمولة الطلب للعرض** — «1.5» لا «2». */
val MerchantOrder.commissionLabel: String
    get() = percentLabel(commissionPercentExact, commissionPercent)

@Serializable
data class StoreReport(val summary: ReportSummary = ReportSummary())

/**
 * **مجملُ المدّة** — والحقولُ كما يكتبها المحرّك حرفاً بحرف.
 *
 * **ومبيعاتُه بسعره هو لا بما دفعه الزبون** (قرارُ المالك ٢٠٢٦-٠٨-١٠):
 * **متجرٌ يقرأ مبيعاتٍ فيها هامشُ المنصّة يحسب أرباحاً ليست له.**
 */
@Serializable
data class ReportSummary(
    val orders: Int = 0,
    val delivered: Int = 0,
    val cancelled: Int = 0,
    val sales: Long = 0,
    @SerialName("platform_commission") val commission: Long = 0,
) {
    /** **مستحقُّه** — مبيعاتُه ناقصَ عمولة المنصّة. */
    val due: Long get() = sales - commission
}

@Serializable
data class ReportReason(val code: String = "", val label: String = "")

@Serializable
data class Ack(val ok: Boolean = true)

@Serializable
data class Updated(val updated: Boolean = false)

/** **ما يعني الشاشةَ من الوسيط المرفوع** — معرّفُه وعنوانُه. */
@Serializable
data class MediaRef(
    val id: String = "",
    @SerialName("thumb_url") val thumbUrl: String = "",
)

/** **مشروبٌ في قائمة المتجر** — `price` بالليرة دائماً، و`price_usd` إن كان دولاريّاً. */
@Serializable
data class Drink(
    val name: String = "",
    val price: Long = 0,
    @SerialName("price_usd") val priceUsd: Double? = null,
)

@Serializable
data class DrinksBody(val drinks: List<Drink> = emptyList())

/**
 * **مجموعةُ خياراتٍ ترافق الصنفَ في الحفظ** — «الحجم» و«يُضاف».
 *
 * **وتُرسَل الشجرةُ كاملةً في كلّ حفظ** — **والحذفُ لا بابَ له غير
 * الاستبدال**: من أزال مجموعةً ولم تُرسَل الشجرةُ بقيت في القاعدة.
 */
@Serializable
data class ModifierGroup(
    val id: String = "",
    val name: String = "",
    @SerialName("min_select") val minSelect: Int = 0,
    @SerialName("max_select") val maxSelect: Int = 1,
    val options: List<ModifierOption> = emptyList(),
    /**
     * **طريقةُ السعر** «priced» أو «none» أو «weight» (قرارُ المالك ٢٠٢٦-١٠-٢٨).
     * **وفراغُه لا يُرسَل** فيُبقي الخادمُ طريقةَ المجموعة القائمة.
     */
    @SerialName("price_mode") val priceMode: String? = null,
    /** **الوزنُ الذي يخصّه سعرُ الصنف** — لـ«weight» وحدَها، وفراغُه ١٠٠ غ. */
    @SerialName("base_grams") val baseGrams: Int? = null,
)

@Serializable
data class ModifierOption(
    val id: String = "",
    val name: String = "",
    @SerialName("price_delta") val priceDelta: Long = 0,
    val available: Boolean = true,
    /**
     * **فرقُ الإضافة بالدولار** لصنفٍ دولاريّ (طلبُ المالك ٢٠٢٦-١٠-١٢) — والخادمُ
     * يحسب منه `price_delta` بسعر صرف المتجر. وفراغُه: الإضافةُ بالليرة.
     */
    @SerialName("price_delta_usd") val priceDeltaUsd: Double? = null,
    /** **وزنُ الخيار بالغرام** لمجموعة «حسب الوزن» — وسعرُه يحسبه الخادم. */
    val grams: Int? = null,
    /** **ما يُكتب في الحقل** وهو يُكتب — «0.» لا يضيع قبل الرقم التالي. لا يُرسَل. */
    @kotlinx.serialization.Transient val usdText: String = "",
)

// ── ما يُرسَل ──────────────────────────────────────────────────────────

@Serializable
data class TransitionInput(val to: String, val note: String = "")

@Serializable
data class ReportInput(val reason: String, val note: String = "")

@Serializable
data class AvailabilityInput(val available: Boolean)

@Serializable
data class SectionInput(val name: String)

@Serializable
data class HoursInput(val days: List<DayHours>)

@Serializable
data class ClosedInput(val closed: Boolean)

@Serializable
data class ReasonsPage(val reasons: List<String> = emptyList())

@Serializable
data class ReportsPage(val reports: List<MerchantReport> = emptyList())

/**
 * **بلاغٌ أرسله — وما ردّت به المنصّة.**
 *
 * **ولا اسمَ سائقٍ فيه ولا رقم** — البلاغُ عند المنصة وهي تفصل، **واسمُ
 * من شُكي عليه في يد الشاكي بابُ ثأرٍ لا بابُ عدل.**
 */
@Serializable
data class MerchantReport(
    val id: String = "",
    val number: Long = 0,
    @SerialName("order_number") val orderNumber: Long? = null,
    val reason: String = "",
    val status: String = "",
    /** كلمةُ المنصّة عند الإغلاق — **وبلاغٌ يُغلق بلا كلمةٍ يُقرأ تجاهلاً.** */
    val resolution: String = "",
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class ClosedAck(@SerialName("emergency_closed") val closed: Boolean = false)

/**
 * **ما يقبله `/settings` فعلاً** — قِيس من `merchant_ops_handlers.go`.
 *
 * **ولا `accepts_orders` فيه**: الإغلاقُ بابُه `/emergency` وحدَه،
 * **وحقلٌ يُرسَل ولا يُقرأ يُطرح صامتاً** فيظنّ المرسِلُ أنّه فعل.
 */
@Serializable
data class StoreSettingsInput(
    /** **واسمُه بيده** — (طلبُ المالك ٢٠٢٦-٠٨-٢٣). */
    val name: String? = null,
    @SerialName("default_prep_minutes") val prepMinutes: Int? = null,
    @SerialName("min_order") val minOrder: Long? = null,
    /**
     * **عنوانُه ودبّوسُه** — (طلبُ المالك ٢٠٢٦-٠٨-٢٣).
     *
     * **والنقطةُ تُرسَل كاملةً أو لا تُرسَل**: المحرّكُ يردّ نصفَها خطأً
     * — **ومن أرسل `lat` وحدَه نقل متجرَه إلى خطّ الاستواء.**
     *
     * **وما لم يُرسَل لا يُمحى** — `COALESCE` في المحرّك، **فحفظُ
     * مدّةِ التحضير وحدَها لا يضيّع موضعَ المتجر.**
     */
    @SerialName("address_text") val addressText: String? = null,
    val lat: Double? = null,
    val lng: Double? = null,
    /** **وقبولُ الاسترداد بيده** — (طلبُ المالك ٢٠٢٦-١٠-٠٨). */
    @SerialName("accepts_returns") val acceptsReturns: Boolean? = null,
    /** **عملةُ العرض للزبون** «SYP» أو «USD» (٢٠٢٦-١٠-١٢، الخيار ١). */
    @SerialName("display_currency") val displayCurrency: String? = null,
    /** **شعارُه بيده** (طلبُ المالك ٢٠٢٦-١٠-١٨) — معرّفُ صورةٍ رفعها بـ`uploadLogo`. */
    @SerialName("logo_media_id") val logoMediaId: String? = null,
)

@Serializable
data class MenuItemInput(
    @SerialName("section_id") val sectionId: String? = null,
    val name: String? = null,
    val description: String? = null,
    val price: Long? = null,
    val available: Boolean? = null,
    /** **وقسمُ السوق إلزاميّ** — صنفٌ بلا قسمٍ لا يراه زبون. */
    @SerialName("platform_section_id") val platformSectionId: String? = null,
    @SerialName("image_media_id") val imageMediaId: String? = null,
    val modifiers: List<ModifierGroup>? = null,
    /** **عملةُ الصنف** «SYP» أو «USD» (٢٠٢٦-١٠-١٢) — وبالدولار يُرسَل `price_usd`. */
    @SerialName("price_currency") val priceCurrency: String? = null,
    @SerialName("price_usd") val priceUsd: Double? = null,
    /** **«أضف مشروباتي»** (٢٠٢٦-١٠-٢٩) — و`null` بلا تغيير. */
    @SerialName("store_drinks") val storeDrinks: Boolean? = null,
)

/** **سعرُ صرف المتجر** — `PATCH /merchant/stores/{id}/exchange-rate`. */
@Serializable
data class ExchangeRateInput(val rate: Double, val confirm: Boolean = false)

@Serializable
data class ExchangeRateResult(
    @SerialName("exchange_rate") val exchangeRate: Double = 0.0,
    @SerialName("items_repriced") val itemsRepriced: Long = 0,
)

@Serializable
data class SettlementMethodAck(
    @SerialName("settlement_method") val settlementMethod: String = "",
)
