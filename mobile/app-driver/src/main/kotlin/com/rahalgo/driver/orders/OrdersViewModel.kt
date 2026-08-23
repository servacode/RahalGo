package com.rahalgo.driver.orders

import android.app.Application
import com.rahalgo.ui.apiError
import android.os.SystemClock
import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.driver.R
import com.rahalgo.driver.data.Backend
import com.rahalgo.ui.Refresh
import com.rahalgo.ui.LastPoint
import com.rahalgo.driver.location.LocationPermission
import com.rahalgo.driver.trip.ChatState
import com.rahalgo.driver.trip.Stop
import com.rahalgo.driver.trip.TripState
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.OrderRoute
import com.rahalgo.driver.trip.TripStep
import org.maplibre.android.geometry.LatLng
import com.rahalgo.shared.net.ApiClient
import io.ktor.client.plugins.HttpRequestTimeoutException
import java.io.IOException
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حال الطلبات**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وثلاثة نداءات في واحد**: المعروض · ما في يده · وحال ورديّته. **وهي
 * تُقرأ معا** لأنّ قائمة عروض فارغة معناها مختلف تماما حسب الوردية:
 * **راكد، أو مغلق على نفسه.**
 */
class OrdersViewModel(app: Application) : AndroidViewModel(app) {

    private companion object {
        // ══════════════════════════════════════════════════════════════
        // **حدّ «وصلت» — خمسةَ عشرَ مترا**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرار المالك ٢٠٢٦-٠٨-١٢: «ثمانون كثيرٌ جدّا… خمسةَ عشرَ
        //  ممتازة، المسافات في الرقّة قريبة، مدينةٌ صغيرة».)
        //
        // **وثمانون شارعٌ كامل** — يقف على إشارةٍ قربَ المتجر فيُعدّ
        // واصلا، **ووقتُ الوصول يدخل في حساب التأخّر.**
        //
        // # وخمسةَ عشرَ هو حدُّ الجهاز نفسِه
        //
        // **دقّةُ الـGPS في هاتفٍ بمدينةٍ خمسةَ عشرَ متراً في أحسن
        // حالاتها** — فهذا أضيقُ ما يمكن أن يُطلب **وهو لا يزال يعمل.**
        // **وما دونه حظّ**: يقف السائقُ ينتظر إعلاناً لا يأتي، **وميزةٌ
        // لا تعمل أسوأُ من ميزةٍ لا توجد.**
        //
        // **ومدينةٌ صغيرةٌ تحتمله**: بيوتُ الرقّة متلاصقة، **وخمسةٌ
        // وعشرون فيها قد تكون بيتَ الجار.**
        //
        // ══════════════════════════════════════════════════════════════
        // **ومصدرُه واحدٌ منذ ٢٠٢٦-٠٨-٢١ — البند ١٤**
        // ══════════════════════════════════════════════════════════════
        //
        // **ولا يجوز أن تقول الشاشةُ «قريب» عند خمسةَ عشرَ
        // ويقول الصوتُ «وصلت» عند مئة.** فالرقمُ هنا وهناك واحد.
        //
        // **والفرقُ في الشهادة لا في المسافة**: هذا **اقتراحٌ لا فعل**
        // (نصُّ الشاشة)، **ودعوى الصوت تحتاج قراءةً مقبولةً
        // وسماحاً مسقوفاً للدقّة** — انظر `BusinessArrival`.
        const val ARRIVAL_M = com.rahalgo.navigation.BusinessArrival.RADIUS_M.toFloat()

        /** **كم بين نظرةٍ وأخرى** — والوقوفُ يُقاس بالثواني لا بالنبضات. */
        const val ARRIVAL_TICK_MS = 5_000L

        /** **كم يقف حتّى يُعدّ واصلا** — (قرار المالك: بين ٣٠ و٦٠ ثانية). */
        const val ARRIVAL_HOLD_MS = 30_000L
    }


    var state by mutableStateOf(OrdersState())
        private set

    // ══════════════════════════════════════════════════════════════════
    // **وما يقرؤه `init` يُعلَن قبله**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وخصائصُ `by mutableStateOf` تُهيَّأ بترتيب كتابتها**: و`init`
    // ينادي `refresh()` **فيقرأ خاصّيّةً لم تُهيَّأ بعد** — ويسقط
    // التطبيق عند الإقلاع بـ`NullPointerException`، **لا عند الاستعمال
    // فيُعرف سببُه.** (وقع ٢٠٢٦-٠٨-١٢ حين صار `load()` يقرأ `openId`.)
    //
    // **والمترجم لا يمسكها**: النوعُ غيرُ فارغٍ في التوقيع، **والفراغ
    // يقع في الزمن لا في النوع.**

    /**
     * **الطلب المفتوح** — وفارغ يعني القائمة معروضة.
     *
     * **ويُحدَّث من القائمة نفسها** بعد كلّ خطوة — فلا نسختان لطلب
     * واحد تفترقان.
     */
    var openId by mutableStateOf<String?>(null)
        private set

    /** **كم رسالةً تنتظره في حديث طلبه** — وصفرٌ يعني لا شارة. */
    var chatUnread by mutableStateOf(0)
        private set

    // ══════════════════════════════════════════════════════════════════
    // **مسارُ الطرف الحاليّ** — بالشوارع لا بالهواء
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرار المالك ٢٠٢٦-٠٨-١٢.)
    //
    // **وفارغٌ يعني «لا مسار»**: محرّكُ المسارات قد ينام، **والخريطة
    // ترسم خطَّها المستقيم كما كانت** ولا تقف.
    var route by mutableStateOf<OrderRoute?>(null)

    // ══════════════════════════════════════════════════════════════════
    // **اختيارُ المسار — إغلاقُ واجهة ٧، ٢٠٢٦-٠٨-٢١**
    // ══════════════════════════════════════════════════════════════════
    //
    // **والحالُ هنا لا في الشاشة** (البند ١٢): «تبقى في ViewModel، لا في
    // local Composable state الذي يضيع بالتدوير». **والسائقُ يدير جهازَه
    // في الحامل وهو يقود.**

    /** **الخياراتُ المجلوبة** — أو `null` إن لم تُطلب أو سقط الجلب. */
    var routeChoices by mutableStateOf<com.rahalgo.navigation.RouteChoices?>(null)
        private set

    /** **المعايَنُ الآن** — بصريٌّ محضٌ لا ملاحة (البند ١٣). */
    var choicePreview by mutableStateOf<String?>(null)
        private set

    /** **وقد بطل المسارُ المعايَن** — يُعرض إشعارٌ قصير (البند ١٩). */
    var choiceStale by mutableStateOf(false)
        private set

    /**
     * **مسارٌ اعتُمد وينتظر التسليم** — البند ١٧.
     *
     * **ولا يُسلَّم من هنا**: `NavigationSession` تعيش في الشاشة.
     * **فيُوضع هنا بعد الفحص**، والشاشةُ تسلّمه ثمّ تُخليه.
     *
     * **والترتيبُ محفوظ** (البند ١٨): **فحصٌ ثمّ تسليمٌ ثمّ تعكس
     * الواجهة** — ولا تفاؤلَ يسبق الفحص.
     */
    var committedRoute by mutableStateOf<com.rahalgo.navigation.NavRoute?>(null)
        private set

    /** **متى جُلبت** — به يُقاس العمر. */
    private var choicesAtMs = 0L

    /** **وجهةُ الجلبة** — فلا تُخلط بجلبةٍ لوجهةٍ أخرى (البند ٢٢). */
    private var choicesTarget: com.rahalgo.navigation.RouteTarget? = null

    // ══════════════════════════════════════════════════════════════════
    // **حارسُ النشر — إغلاقُ صحّة الواجهة، ٢٠٢٦-٠٨-٢١**
    // ══════════════════════════════════════════════════════════════════
    //
    // **أمرُ المالك نصّاً**: «أريد حماية صريحة، لا الاعتماد فقط على
    // validation عند Confirm».
    //
    // **والسباقُ حقيقيّ**: طلبٌ إلى المتجر ينطلق، ثمّ يُستلم الطلب،
    // **ثمّ يصل ردُّ الأوّل متأخّراً** — فيعود خطٌّ إلى متجرٍ فارقه
    // السائق. **والفحصُ عند الاعتماد لا يمنع رسمَ الخطّ.**

    /** **تسلسلُ الطلبات** — الأحدثُ وحدَه يُنشَر (البند ٢). */
    private var altSeq = 0L

    /**
     * **مهمّةُ الجلب الحاليّة** — تُلغى عند إطلاق جديدة.
     *
     * **والإلغاءُ وحدَه ليس حمايةً** (البند ٤): **لا يصل الخادمَ**،
     * **وقد يكون الردُّ في الطريق.** فالتسلسلُ يبقى حارساً فوقه.
     */
    private var altJob: kotlinx.coroutines.Job? = null

    /** **سببُ آخرِ تركيبٍ للمسار** — البنود ٦ و٧ و٩. */
    var lastInstallReason by mutableStateOf(
        com.rahalgo.navigation.RouteInstallReason.INITIAL,
    )
        private set

    /** **آخرُ رفضٍ لردٍّ** — للتشخيص لا للعرض (البند ٣). */
    var lastGateReject by mutableStateOf(
        com.rahalgo.navigation.RouteChoiceGate.Reject.NONE,
    )
        private set

    private val backend = Backend.of(getApplication())

    init {
        refresh()
        // ══════════════════════════════════════════════════════════════
        // **والقائمة تتحدّث بنفسها**
        // ══════════════════════════════════════════════════════════════
        //
        // (البند الثالث في قائمة المالك ٢٠٢٦-٠٨-١٢: «القائمة تتحدّث
        //  لحالها» — وكان يلزم أن يخرج من التبويب ويعود.)
        //
        // **والإشارة بلا حمولة**: يقول المحرّك «تغيّر شيء»، **وما يراه
        // هذا السائق تقرّره نقطة الطابور** بحسب ورديّته ونمط التوزيع.
        watchArrival()
        backend.live.start(
            scope = viewModelScope,
            onState = { up -> Log.i("RahalGo/live", if (up) "الوصلة قامت" else "الوصلة انقطعت") },
            onEvent = {
                refresh()
                // **وما وصل الوصلةَ يُبَثّ للجميع** — (شكوى المالك
                // ٢٠٢٦-٠٨-١٣: «لا يحدث تحديثٌ لحظيٌّ تلقائيٌّ لكثيرٍ من
                // الأمور بالتطبيق»).
                //
                // **والوصلةُ واحدةٌ ويبدؤها هذا المحرّك** — فكان
                // يُنعش الطلباتِ وحدَها، **والرصيدُ والتقييمُ والورديّةُ
                // تبقى قديمةً حتّى يخرج من التبويب ويعود.**
                Refresh.bump()
                // **والحديثُ المفتوح يُعاد قراءته** — الإشارةُ تقول
                // «تغيّر شيء»، **ومن أعاد القائمة وحدَها** ترك صاحبَه
                // ينظر إلى حديثٍ لا يتحرّك وقد وصلته رسالة.
                if (chat != null) reloadChat()
            },
        )
    }

    // ══════════════════════════════════════════════════════════════════
    // **والوصولُ يُعلن نفسَه — لا يُضغط**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرار المالك ٢٠٢٦-٠٨-١٢: «يكون ذكيّاً بمجرّد أن الموقع أصبح عند
    //  المتجر… في حال السائق توقّف بين الـ٣٠ والـ٦٠ ثانية».)
    //
    // **و«وصلت» حقيقةٌ يعرفها الجهازُ أدقَّ من ذاكرة سائقٍ يقود** —
    // وكلُّ ضغطةٍ تُرفع عنه ربحٌ خالص.
    //
    // # ولماذا وقوفٌ لا لمسةُ دائرة
    //
    // **الموضعُ ينطّ**: يمرّ بجانب المتجر وهو ذاهبٌ إلى غيره فيُسجَّل
    // «وصل» ولم يصل. **ووقتُ الوصول يدخل في حساب التأخّر والشكاوى** —
    // فلا يُكتب بقراءةٍ عابرة.
    //
    // **فثلاثون ثانيةً داخل الدائرة** — من وقف عندها وقف حقّا.
    //
    // # ولماذا هذه الخطوة وحدَها
    //
    // **«وصلت» لا مالَ فيها ولا إثبات** — أسوأُ ما يقع أن تتقدّم دقيقة.
    // **و«استلمت» و«سلّمت» فيهما مالٌ وذمّة** — فتبقيان بيده وحدَه.
    //
    // **والزرُّ باقٍ**: سوقٌ مسقوفٌ يُضعف الموضع، **ومن انتظر آليّةً لا
    // تأتي** وقف عند الباب لا يعرف ماذا يفعل.
    private var nearSince: Long = 0L

    private fun watchArrival() {
        viewModelScope.launch {
            while (true) {
                delay(ARRIVAL_TICK_MS)
                val order = state.mine.firstOrNull { it.id == openId }
                    ?: state.mine.firstOrNull()
                val to = when (order?.status) {
                    "assigned" -> "at_pickup"
                    "on_the_way" -> "at_dropoff"
                    else -> null
                }
                if (order == null || to == null || !near(LastPoint.value, order)) {
                    nearSince = 0L
                    continue
                }
                val now = SystemClock.elapsedRealtime()
                if (nearSince == 0L) {
                    nearSince = now
                } else if (now - nearSince >= ARRIVAL_HOLD_MS) {
                    nearSince = 0L
                    Log.i("RahalGo/وصول", "وقوفٌ عند الوجهة — تُعلَن " + to)
                    step(to)
                }
            }
        }
    }

    override fun onCleared() {
        // **ولا تبقى وصلة بلا شاشة** — تستنزف البطارية وتوقظ الجهاز.
        backend.live.stop()
        super.onCleared()
    }

    fun refresh() {
        viewModelScope.launch { load() }
    }

    /**
     * **القراءة نفسها — وتُنتظَر.**
     *
     * **ومن قبِل طلبا يُنقل إلى رحلته فورا**، ولو نُقل قبل أن تصل
     * القائمة **لرأى شاشة رحلة فارغة لحظة** ثمّ امتلأت أمامه.
     */
    private suspend fun load() {
        // **وشارةُ الحديث تُقرأ مع كلّ تحديث** — والوصلةُ الحيّة تنادي
        // التحديث، **فما يصل يُرى في ثانيته لا في فتحةٍ تالية.**
        loadChatBadge()
        state = try {
            state.copy(
                offers = backend.driver.queue(),
                mine = backend.driver.orders(),
                // **والحال كاملا لا الوردية وحدها** — شاشة «لماذا
                // لا تصلني طلبات» تُبنى منه.
                me = backend.driver.me(),
                locationOn = LocationPermission.granted(getApplication()),
                loading = false,
                error = "",
            )
        } catch (e: Exception) {
            state.copy(loading = false, error = describe(e))
        }
        // ══════════════════════════════════════════════════════════════
        // **والمسارُ بعد الطلبات لا قبلها**
        // ══════════════════════════════════════════════════════════════
        //
        // (شكوى المالك ٢٠٢٦-٠٨-٢٣، مرّتين: «الطريقُ خطٌّ مستقيمٌ بلا
        //  أيّ دلالة، هذا أكبرُ غلط».)
        //
        // **كان النداءُ أوّلَ السطر** — و`loadRoute` تسأل عن الطلب
        // الحاليّ: `openId ?: state.mine.firstOrNull()`. **وعند أوّل
        // فتحةٍ `state.mine` فارغةٌ لأنّ الطلبات لم تصل بعد** — فيردّ
        // فراغاً، **فتخرج `loadRoute` بلا نداءٍ واحدٍ وتكتب `route =
        // null`.**
        //
        // **والخريطةُ ترسم احتياطيَّها**: `route.ifEmpty { سائق · متجر
        // · زبون }` — **ثلاثُ نقاطٍ في خطٍّ مستقيمٍ من فوق البيوت**،
        // وهو بعينه ما شكا منه المالكُ في ٢٠٢٦-٠٨-١٢ وظنناه أُغلق.
        //
        // **ثمّ تأتي دورةُ تحديثٍ تاليةٌ فيُرسم الطريقُ الحقيقيّ** —
        // وذاك «أخذ بعضَ الوقت ليصبح خطَّ طريقٍ حقيقيّ» الذي وصفه.
        //
        // **والفرقُ ليس تجميلاً**: قِيس على الطلب ١٠٣٠ — المستقيمُ
        // ١٫٨ كم والطريقُ ٢٫٩ كم. **ستّون بالمئة زيادة**، وسائقٌ يقود
        // على خطٍّ لا يوجد.
        loadRoute()
    }

    // ══════════════════════════════════════════════════════════════════
    // **والرحلة تبدأ بنفسها — لا بضغطة**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرار المالك ٢٠٢٦-٠٨-١٢: «برنامج ذكيّ، ما في داعي السائق يظلّ
    //  يضغط على الشاشة وألف كبسة — مجرّد ما وافق على طلب تبدأ الرحلة
    //  بشكل تلقائيّ».)
    //
    // **والقبول نفسه هو البدء**: من ضغط «موافق» لا ينتظر شيئا آخر —
    // **وضغطةٌ ثانية تقول ما قالته الأولى** خطوةٌ زائدة في يد رجل يقود.
    //
    // **ورايةٌ لا حالٌ محفوظ**: تُرفع مرّة وتُنزَل حين يُنقل، **ولو
    // بقيت مرفوعة** لأعادته إلى الرحلة كلّما فتح الطلبات.
    var startTrip by mutableStateOf(false)
        private set

    fun tripOpened() {
        startTrip = false
        // **ورحلةٌ جديدةٌ أولويّةٌ جديدة** — من أخذ طلباً بدأ طريقاً
        // آخر، **وما رفضه على الطريق الأوّل لا يُحسب عليه في الثاني.**
        offerSpent = false
    }

    /**
     * **يأخذ الطلب — ثمّ يعيد قراءة القائمتين.**
     *
     * **ولا يُنقل من قائمة إلى قائمة في الشاشة تفاؤلا**: القبول قد يُرفض
     * (**سبقه غيره** أو بلغ حدّه)، **ومن نقله قبل الجواب** أرى صاحبه
     * طلبا في يده وهو ليس له.
     */
    fun accept(orderId: String) {
        if (state.acceptingId != null) return
        state = state.copy(acceptingId = orderId, error = "", actionError = "")
        viewModelScope.launch {
            try {
                backend.driver.accept(orderId)
            } catch (e: Exception) {
                state = state.copy(acceptingId = null, actionError = describe(e))
                // **والقائمة تُعاد قراءتها حتّى بعد الرفض**: «سبقك غيره»
                // يعني أنّ البطاقة لم تعد موجودة، **ومن أبقاها** جعله
                // يضغطها ثانية.
                refresh()
                return@launch
            }
            state = state.copy(acceptingId = null)
            // **والقائمة أوّلا ثمّ النقل** — لا شاشةَ رحلةٍ فارغة.
            load()
            openId = orderId
            startTrip = true
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **الطلب المفتوح — في النموذج لا في الشاشة**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وفارغ يعني القائمة معروضة.** ولماذا هنا: بعد كلّ خطوة تُعاد
    // قراءة القائمتين، **والطلب المفتوح يُحدَّث من القائمة نفسها** —
    // فلا نسختان لطلب واحد تفترقان.

    var detail by mutableStateOf(DetailState(order = com.rahalgo.shared.model.DriverOrder()))
        private set

    fun open(orderId: String) {
        openId = orderId
        detail = DetailState(order = state.mine.firstOrNull { it.id == orderId } ?: return)
    }

    fun close() {
        openId = null
    }

    // ══════════════════════════════════════════════════════════════════
    // **الطلبُ الذي يعمل عليه — موضعٌ واحدٌ يقرّره**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وشاشةُ الرحلة تعرض `mine.first()` إن لم يُفتح شيء** (`trip()`)،
    // **وكانت الأفعالُ تشترط `openId`** — فتُقرأ البطاقةُ ويُضغط زرُّها
    // **ولا يقع شيء ولا تُقال كلمة.**
    //
    // **ويقع بعد كلّ إقلاق للتطبيق**: `openId` رايةٌ في الذاكرة تُمحى مع
    // العملية، **والطلبُ باقٍ في يده** — فيفتح تطبيقَه ويضغط «وصلت
    // المتجر» فلا يستجيب. (أمسكه المالك ٢٠٢٦-٠٨-١٢.)
    //
    // **وأسوأُ ما فيه أنّه صامت**: لا خطأَ ولا دوّارة — **وزرٌّ لا يفعل
    // ولا يقول** يُقرأ تطبيقاً معطوباً.
    private fun currentId(): String? = openId ?: state.mine.firstOrNull()?.id

    /**
     * ══════════════════════════════════════════════════════════════════
     * **بابُ إعادة الحساب — يُبنى هنا لأنّ هنا يُعرف الطلب**
     * ══════════════════════════════════════════════════════════════════
     *
     * (المرحلة ٣ب، ٢٠٢٦-٠٨-٢٠.)
     *
     * **ووحدةُ الملاحة لا تعرف طلباً ولا خادما** — تعرف دالّةً واحدة.
     *
     * **ورقمُ الطلب يُقرأ لحظةَ النداء لا لحظةَ البناء**: الطلبُ
     * يتبدّل والباب واحد.
     */
    val routeSource: com.rahalgo.navigation.RouteSource by lazy {
        com.rahalgo.driver.trip.BackendRouteSource(backend, viewModelScope) { currentId() }
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **والصوتُ يعيش هنا لا في التركيب**
     * ══════════════════════════════════════════════════════════════════
     *
     * (المرحلة ٤، أمرُ المالك ٢٠٢٦-٠٨-٢٠، البند ٢١.)
     *
     * **`NavigationSession` تُبنى بـ`remember` فتموت بدوران الجهاز** —
     * ويُصفَّر معها عبورُ العتبات، **فتُعاد تعليماتٌ قيلت.**
     *
     * **وهذان في `ViewModel` فيتجاوزان ذلك**: سجِلُّ ما قيل باقٍ،
     * **ومحرّكُ النطق لا يُبنى مرّتين فيتسرّب.**
     */
    val speaker: com.rahalgo.driver.trip.AndroidSpeaker by lazy {
        com.rahalgo.driver.trip.AndroidSpeaker(getApplication()).also { it.init() }
    }

    val voice: com.rahalgo.driver.trip.VoiceOrchestrator by lazy {
        com.rahalgo.driver.trip.VoiceOrchestrator(speaker)
    }

    /** **الكتم** — حالٌ مستقلّةٌ عن الملاحة. */
    var voiceMuted by mutableStateOf(false)
        private set

    fun toggleVoice() {
        voiceMuted = !voiceMuted
        voice.muted = voiceMuted
    }

    /** يحرّك الطلب خطوة — **ثمّ يعيد قراءة كلّ شيء.** */
    fun step(to: String) {
        val id = currentId() ?: return
        if (detail.busy) return
        detail = detail.copy(busy = true, error = "")
        viewModelScope.launch {
            try {
                backend.driver.transition(id, to)
            } catch (e: Exception) {
                detail = detail.copy(busy = false, error = describe(e))
                return@launch
            }
            // ══════════════════════════════════════════════════════════
            // **ومن استلم انطلق — لا يُسأل عنه**
            // ══════════════════════════════════════════════════════════
            //
            // (قرار المالك ٢٠٢٦-٠٨-١٢: «ما في داعي يضغط انطلقت للزبون —
            //  لأنّه شو بدّو يعمل، ياخد حمّام بعد ما استلم الطلب؟».)
            //
            // **وضغطةٌ لا اختيارَ فيها ليست خطوة**: ما بين «استلمت»
            // و«انطلقت» لا يفعل السائقُ شيئا إلّا أن يركب، **وزرٌّ
            // جوابُه معروفٌ سلفا** يُسأل عبثا.
            //
            // **والحالُ في المحرّك يبقى كما هو**: خطوتان في الخطّ
            // الزمنيّ لا واحدة — **ووقتُ الانطلاق يُقرأ في الشكوى**،
            // إنّما تقعان بضغطةٍ واحدة.
            if (to == "picked_up") {
                runCatching { backend.driver.transition(id, "on_the_way") }
            }
            // **والتسليم يُغلق الطلب** — فيُعاد إلى القائمة لا إلى شاشة
            // طلب لم يعد له وجود فيها.
            if (to == "delivered") {
                openId = null
            }
            detail = detail.copy(busy = false)
            reload()
        }
    }

    /** يسأل المحرّك أيّ الأسباب تصلح في هذا الحال. */
    fun askFail() {
        val order = state.mine.firstOrNull { it.id == openId }
            ?: state.mine.firstOrNull()
            ?: detail.order
        viewModelScope.launch {
            val reasons = runCatching { backend.driver.failReasons(order.status) }
                .getOrDefault(emptyList())
            // **وقائمةٌ فارغةٌ ليست «لا شيء»** — هي «لا سببَ يُختار هنا»،
            // **وبابُها الطارئ** لا نافذةٌ فارغةٌ تُغلق فورا.
            if (reasons.isEmpty()) {
                emergencyOpen = true
            } else {
                detail = detail.copy(order = order, failReasons = reasons)
            }
        }
    }

    fun dismissFail() {
        detail = detail.copy(failReasons = null)
    }

    fun fail(reason: String) {
        val id = currentId() ?: return
        detail = detail.copy(failReasons = null, busy = true, error = "")
        viewModelScope.launch {
            try {
                backend.driver.transition(id, "failed", reason = reason)
                openId = null
            } catch (e: Exception) {
                detail = detail.copy(busy = false, error = describe(e))
                return@launch
            }
            detail = detail.copy(busy = false)
            reload()
        }
    }

    fun release() {
        val id = currentId() ?: return
        detail = detail.copy(busy = true, error = "")
        viewModelScope.launch {
            try {
                backend.driver.release(id)
                openId = null
            } catch (e: Exception) {
                detail = detail.copy(busy = false, error = describe(e))
                return@launch
            }
            detail = detail.copy(busy = false)
            reload()
        }
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **حال الرحلة — مشتقّ لا محفوظ**
     * ══════════════════════════════════════════════════════════════════
     *
     * **والخطوة تُقرأ من حال الطلب في المحرّك** لا من راية في الجهاز:
     * **من حفظها عنده** رأى «استلمت» وقد أُلغي الطلب من المكتب.
     *
     * **والطلب المعروض هو المفتوح** — وإن لم يُفتح شيء فأوّل ما في يده:
     * **من له طلب واحد لا يُطلب منه أن يختاره.**
     */
    fun trip(driver: LastPoint.Point?): TripState {
        val order = state.mine.firstOrNull { it.id == openId } ?: state.mine.firstOrNull()
            ?: return TripState()
        val line = route?.points.orEmpty().mapNotNull {
            if (it.size >= 2) LatLng(it[0], it[1]) else null
        }
        return TripState(
            order = order,
            // **والمسار يُفضَّل على الخطّ المستقيم** — وفارغٌ يعني أنّ
            // المحرّك لم يردّ، **فيُرسم المستقيمُ ولا تبقى الخريطةُ
            // بلا خطّ.**
            routeLine = line,
            routeM = route?.takeIf { it.available }?.distanceM ?: -1.0,
            routeSec = route?.takeIf { it.available }?.durationS ?: -1.0,
            // **وبياناتُ الملاحة تُحوَّل هنا مرّةً** — المرحلة ٣أ.
            navRoute = com.rahalgo.driver.trip.NavRouteMapper.toNavRoute(route),
            // **وهويّتُه معه** — فلا يُقاد مسارٌ لا يُعرف بماذا يُقارَن.
            navRouteId = route?.routeId?.takeIf { it.isNotEmpty() },
            // **وخياراتُ المسار** — إغلاقُ واجهة ٧.
            routeChoices = routeChoices,
            choicePreview = choicePreview,
            choiceStale = choiceStale,
            committedRoute = committedRoute,
            installReason = lastInstallReason,
            step = TripStep.of(order.status),
            // ══════════════════════════════════════════════════════════
            // **وما بقي يتبدّل بتبدّل الوجهة**
            // ══════════════════════════════════════════════════════════
            //
            // **قبل الاستلام المسافةُ إلى المتجر، وبعده طولُ المشوار
            // إلى الزبون.**
            //
            // **والأولى كانت تغيب دائماً** (شكوى المالك ٢٠٢٦-٠٨-١٥:
            // «المسافةُ إلى المتجر لا تظهر أبداً، فقط المسافةُ إلى
            // الزبون»).
            //
            // **وسببُها أنّ المحرّك يقيسها من موضعٍ مخزَّنٍ في القاعدة**
            // (`to_pickup_m`) **يشترط ألّا يزيد عمرُه على ربع ساعة** —
            // ومن لم تُرسل خدمتُه موضعَه بعدُ **يُردّ له `-1`.**
            //
            // **والثانيةُ لا تحتاج موضعَه أصلاً** — من المتجر إلى الباب،
            // نقطتان ثابتتان. **فتظهر دائماً وتغيب الأولى دائما.**
            //
            // **والجهازُ يعرف موضعَه الآن** — أدقَّ ممّا في القاعدة
            // وأحدث. **فيُحسب عليه**، ولا يبقى السائقُ بلا رقمٍ وهو
            // يقود إلى المتجر.
            //
            // ══════════════════════════════════════════════════════════
            // **والثانيةُ كانت تكذب — وهي أخطرُ من الغياب**
            // ══════════════════════════════════════════════════════════
            //
            // (شكوى المالك ٢٠٢٦-٠٨-١٥: «هناك خطأٌ فادحٌ بالمسافات».)
            //
            // **`leg_m` طولُ المشوار من المتجر إلى الباب** — **نقطتان
            // ثابتتان لا تتبدّلان بحركة السائق.** فكانت الشاشةُ تقول
            // «٢٫٩ كم» وهو ينطلق، **وتقول «٢٫٩ كم» وهو أمام الباب.**
            //
            // **ورقمٌ لا ينقص لا يُقرأ مسافةً باقية** — يُقرأ عطباً في
            // التطبيق، **أو يُصدَّق فيُخطئ في وقتِ وصولٍ يقوله للزبون.**
            //
            // **وما بقي يُقاس من حيث هو الآن** — وهذا ما يعرفه الجهازُ
            // وحدَه.
            remainingM = when (order.status) {
                "assigned", "at_pickup" ->
                    away(driver, order.navLat, order.navLng)
                        .takeIf { it >= 0 } ?: order.toPickupM

                else -> away(driver, order.lat, order.lng)
                    .takeIf { it >= 0 } ?: order.legM
            },
            // **والسرعة من المحرّك لا من الشيفرة** — تُضبط للمدينة كلّها.
            avgSpeedKmh = state.me?.avgSpeedKmh ?: 0,
            requirePhoto = state.me?.requirePhoto ?: false,
            agreeOpen = agreeOpen,
            emergencyOpen = emergencyOpen,
            stops = state.mine.map { Stop(it.id, it.number) },
            // **وأوّل عرضٍ معروضٍ عليه وهو في رحلة** — وما رُفض لا يعود.
            // ══════════════════════════════════════════════════════════
            // **وعرضٌ واحدٌ في الرحلة لا خمسة**
            // ══════════════════════════════════════════════════════════
            //
            // (قرارُ المالك ٢٠٢٦-٠٨-١٥: «في خمسةُ طلباتٍ على نفس طريق
            //  السائق — مسموحٌ يظهر له طلبٌ واحدٌ فقط يأخذه أو يتركه،
            //  ما في داعٍ تظلّ تظهر له كلَّ الطلبات التي على طريقه».)
            //
            // **وكان يُعرض التالي كلّما رفض** — خمسةُ طلباتٍ على طريقه
            // تعني خمسَ لافتاتٍ متتاليةً وهو يقود. **ورفضٌ يُجيب عنه
            // سؤالٌ آخرُ ليس رفضا، هو مساومة.**
            //
            // **وأولويّتُه طلبٌ واحد**: يُعرض، فإن أخذه انتهى الأمر،
            // **وإن تركه لم يُسأل ثانيةً حتّى تنتهي هذه الرحلة.**
            //
            // **والباقي في الطابور لغيره** — لا يضيع شيء.
            onRouteOffer = if (offerSpent) {
                null
            } else {
                state.offers.firstOrNull { it.id !in dismissedOffers }
            },
            // ══════════════════════════════════════════════════════════
            // **يعرف بنفسه أنّك وصلت**
            // ══════════════════════════════════════════════════════════
            //
            // (البند الخامس في قائمة المالك ٢٠٢٦-٠٨-١٢.)
            //
            // **ولا يُحرّك الطلب بنفسه**: قربٌ ليس وصولا — قد يمرّ من
            // الشارع، **وطلبٌ يمشي خطوةً بلا أن يضغطها صاحبه** يُفقده
            // الثقة بالتطبيق كلِّه. **فيُقترح ويُضغط.**
            nearDestination = near(driver, order),
            failReasons = detail.failReasons,
            driver = driver?.let { LatLng(it.lat, it.lng) },
            // **ونقطة المتجر قد تغيب** — متجرٌ قديمٌ بلا دبّوس:
            // **فتُرسم الرحلة بنقطتين** بدل أن تسقط الشاشة.
            pickup = order.navLat?.let { la -> order.navLng?.let { ln -> LatLng(la, ln) } },
            dropoff = LatLng(order.lat, order.lng),
            busy = detail.busy,
            // ══════════════════════════════════════════════════════════
            // **وخطأُ قبولِ العرض يصل الرحلةَ أيضاً**
            // ══════════════════════════════════════════════════════════
            //
            // (كشفته دورةٌ حقيقيّةٌ على المحاكي ٢٠٢٦-٠٨-١٣.)
            //
            // **ضُغط «موافق» في لافتة «طلبٌ على طريقك» فرُدَّ**
            // بـ`too_many_active_orders` — **ولم يظهر شيء**: اللافتةُ
            // تبقى، والزرُّ يُضغط ولا يقع فعلٌ ولا كلمة.
            //
            // **والسببُ أنّ القبولَ يكتب خطأه في حال القائمة**
            // (`state.error`) **والرحلةُ تقرأ حال التفصيل وحدَه** —
            // فيضيع بين حالين.
            //
            // **وزرٌّ يُضغط فلا يقع شيءٌ ولا يُقال لماذا يُقرأ عطباً في
            // التطبيق** — ثمّ يُعاد الضغطُ ويُعاد.
            error = detail.error.ifEmpty { state.actionError },
        )
    }

    /**
     * **أهو على بُعد خطواتٍ من وجهته؟**
     *
     * **وثمانون مترا لا عشرة**: دقّة القمر في المدينة بين خمسة وعشرين
     * مترا، **ومن ضيّق الحدّ** لم يقترح شيئا على من يقف عند الباب.
     *
     * **والمسافة من دالّة النظام** — لا حساب مثلّثات بأيدينا: **خطأ في
     * سطر منه يزيح الوصول مئات الأمتار.**
     */
    /**
     * **كم بينه وبين وجهته — من موضعه الآن.**
     *
     * **وهو خطٌّ مستقيمٌ لا طريق** — أقصرُ من الواقع، **ولا يُدّعى غيرَ
     * ذلك**: الرقمُ الصحيحُ يأتي من محرّك المسارات حين يُضبط.
     *
     * **لكنّه ينقص وهو يقود** — **وهو ما يجعله مسافةً باقية.**
     *
     * **وسالبٌ يعني «لا يُعرف»** — لا صفر: **صفرٌ يقول «أنت هناك»،
     * والجهلُ ليس قربا.**
     */
    private fun away(driver: LastPoint.Point?, lat: Double?, lng: Double?): Double {
        if (driver == null || lat == null || lng == null) return -1.0
        val out = FloatArray(1)
        android.location.Location.distanceBetween(driver.lat, driver.lng, lat, lng, out)
        return out[0].toDouble()
    }

    private fun near(driver: LastPoint.Point?, order: DriverOrder): Boolean {
        if (driver == null) return false
        val lat: Double
        val lng: Double
        when (order.status) {
            "assigned" -> {
                lat = order.navLat ?: return false
                lng = order.navLng ?: return false
            }

            "on_the_way" -> {
                lat = order.lat
                lng = order.lng
            }

            else -> return false
        }
        val out = FloatArray(1)
        android.location.Location.distanceBetween(driver.lat, driver.lng, lat, lng, out)
        return out[0] <= ARRIVAL_M
    }

    /**
     * **يرسل صورة التسليم** — ثمّ يحرّك الطلب إلى «سُلّم».
     *
     * **والخطوة بعدها لا قبلها**: المحرّك يرفض «سُلّم» بلا إثبات حين
     * يُفعَّل الإعداد، **ومن حرّك أوّلا** ردّه بخطأ وهو يحمل الصورة.
     */
    fun sendProof(jpeg: ByteArray, point: LastPoint.Point?) {
        val id = currentId() ?: return
        if (detail.busy) return
        detail = detail.copy(busy = true, error = "")
        viewModelScope.launch {
            try {
                backend.driver.sendProof(id, jpeg, point?.lat, point?.lng)
                backend.driver.transition(id, "delivered")
                openId = null
            } catch (e: Exception) {
                detail = detail.copy(busy = false, error = describe(e))
                return@launch
            }
            detail = detail.copy(busy = false)
            reload()
        }
    }

    /** **يعيد الطلب إلى الطابور** — قبل أن يستلم البضاعة. */
    fun releaseCurrent() {
        openId = currentId()
        release()
    }

    // ══════════════════════════════════════════════════════════════════
    // **حديث الطلب**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وفارغ يعني أنّه غير مفتوح** — لا حديث بلا طلب.
    var chat by mutableStateOf<ChatState?>(null)
        private set

    // ══════════════════════════════════════════════════════════════════
    // **شارةُ الحديث — كم ينتظرك فيه**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرار المالك ٢٠٢٦-٠٨-١٢: «ولازم يطلع إشعار وعدّاد في حال وصلت
    //  رسالة».)
    //
    // **ورسالةٌ تصل ولا شيء يقولها** تُقرأ بعد ساعة — والزبون ينتظر
    // جوابا عن «الباب الثاني أم الأوّل؟»

    /**
     * **يقرأ مسارَ الطلب الذي في يده** — وفشلُه صامت.
     *
     * **والمحرّك يخزّنه عشرَ دقائق بمفتاح نقطتين مقرّبتين**، فنداءٌ مع
     * كلّ تحديثٍ لا يثقل عليه: **من مشى خطوةً قرأ الجوابَ المخزّن.**
     */
    private suspend fun loadRoute() {
        val id = currentId()
        if (id == null) {
            route = null
            return
        }
        // **ويُطلب معرّفُ المسار** — المرحلة ٨ب، البند ٣: **ضمنَ
        // النداء القائم لا بنداءٍ ثانٍ.**
        runCatching { backend.driver.route(id, correlation = true) }
            .onSuccess {
                route = if (it.available) it else null
                // ══════════════════════════════════════════════════════
                // **وجوابُ المسار يُقال في السجلّ**
                // ══════════════════════════════════════════════════════
                //
                // (٢٠٢٦-٠٨-٢٣: شكا المالكُ «الطريقُ ما زال مستقيماً».)
                //
                // **وفشلُ هذا النداء صامتٌ بالتصميم** — الشاشةُ ترسم
                // مستقيمَها ولا تسقط. **وصمتٌ صحيحٌ في يد السائق
                // عمًى في يد من يُصلح**: لا يُعرف أردّ المحرّكُ
                // `available:false` أم ردّ مساراً برأسين.
                android.util.Log.i(
                    "RahalGo/route",
                    "جواب: متاح=${it.available} رؤوس=${it.points.size} " +
                        "مسافة=${it.distanceM} زمن=${it.durationS}",
                )
            }
            .onFailure { android.util.Log.w("RahalGo/route", "تعذّر نداءُ المسار", it) }
    }

    // ══════════════════════════════════════════════════════════════════
    // **جلبُ البدائل — ولا تنتظره الملاحة**
    // ══════════════════════════════════════════════════════════════════
    //
    // (البنود ٢ و٣ و٤.)
    //
    // **أمرُ المالك نصّاً**: «Primary navigation must not wait for
    // alternative computation».
    //
    // **فـ`loadRoute` تبقى كما هي** — تجلب الموصى به وتُسلّمه، **والملاحةُ
    // تعمل.** ثمّ تُطلب البدائلُ في نداءٍ مستقلٍّ عند أحداثٍ بعينها.
    //
    // **ولا تُطلب في حلقةِ الهرتز** ولا أثناء إعادة الحساب (البند ٤).

    /**
     * **يطلب البدائل** — عند حدثٍ لا في حلقة.
     *
     * **ولا يُنشَر ردٌّ إلّا إن كان سياقُه هو الحاضر** (البند ٢):
     * التسلسلُ، والطلبُ، والوجهةُ، والجيلُ، **وتوافقُ الموصى به.**
     *
     * @param reason **لماذا نطلب** — واختيارُ السائق لا يطلب (البند ٦).
     * @param navRouteFingerprint **بصمةُ ما تقوده الملاحةُ الآن.**
     */
    fun loadAlternatives(
        generation: Long,
        target: com.rahalgo.navigation.RouteTarget,
        originLat: Double,
        originLng: Double,
        reason: com.rahalgo.navigation.RouteInstallReason =
            com.rahalgo.navigation.RouteInstallReason.INITIAL,
        navRouteFingerprint: Long = 0L,
        currentGeometry: List<com.rahalgo.navigation.GeoPoint> = emptyList(),
    ) {
        // **واختيارُ السائق لا يُطلق جلباً** — البند ٦.
        if (!reason.fetchesAlternatives) return

        val id = currentId() ?: return

        val request = com.rahalgo.navigation.RouteChoiceGate.RequestContext(
            seq = ++altSeq,
            orderId = id,
            target = target,
            generation = generation,
            navRouteFingerprint = navRouteFingerprint,
        )

        // **وتُلغى السابقةُ** — ولا يُعتمد عليها وحدَها.
        altJob?.cancel()
        altJob = viewModelScope.launch {
            val fetched = try {
                backend.driver.route(id, originLat, originLng, alternatives = true)
            } catch (e: kotlinx.coroutines.CancellationException) {
                /**
                 * **والإلغاءُ يُمرَّر لا يُبتلع** — البند ٥.
                 *
                 * **`runCatching` تلتقط `Throwable`** ومنه
                 * `CancellationException`. **فمهمّةٌ أُلغيت تبدو
                 * ساقطةً**، والكوروتينُ لا يعلم أنّه أُلغي.
                 */
                throw e
            } catch (e: Exception) {
                // **وسقوطُ الجلب صامت** — البند ٣٠: لا شاشةَ خطأ.
                null
            }

            if (fetched == null || !fetched.available) return@launch

            val built = com.rahalgo.driver.trip.RouteChoicesMapper.toChoices(
                route = fetched,
                setId = id + ":" + generation + ":" + target.name + ":" + request.seq,
                generation = generation,
                originLat = originLat,
                originLng = originLng,
            ) ?: return@launch

            /**
             * **والبوّابةُ قبل النشر** — البندان ٢ و٣.
             *
             * **ولا يظهر ولو للحظة**: الفحصُ قبل أن تُسنَد
             * `routeChoices`، **لا بعدها.**
             */
            val now = com.rahalgo.navigation.RouteChoiceGate.RequestContext(
                seq = altSeq,
                orderId = currentId() ?: "",
                target = choicesTargetNow(),
                generation = navGeneration,
                navRouteFingerprint = navRouteFingerprint,
            )
            val verdict = com.rahalgo.navigation.RouteChoiceGate.verdict(
                request = request,
                latestSeq = altSeq,
                now = now,
                recommendedFingerprint =
                    com.rahalgo.navigation.RouteFingerprint.of(built.recommended.route),
                alternativeCount = built.alternatives.size,
                recommendedGeometry = built.recommended.route.geometry,
                currentGeometry = currentGeometry.ifEmpty { null },
            )
            lastGateReject = verdict
            if (!com.rahalgo.navigation.RouteChoiceGate.accepted(verdict)) {
                // **ويُسجّل لماذا طُرح** — البند ٣.
                //
                // **فردٌّ يُطرح صامتاً لا يُفهم في الميدان** — ولا فرقَ
                // في الشاشة بين «لا بدائل» و«وصلت متأخّرة».
                Log.i(
                    "RahalGo/بدائل",
                    "طُرح ردٌّ: " + verdict.name +
                        " · تسلسل=" + request.seq + "/" + altSeq +
                        " · جيل=" + request.generation + "/" + now.generation +
                        " · وجهة=" + request.target.name + "/" + now.target.name,
                )
                return@launch
            }

            routeChoices = built
            choicesAtMs = System.currentTimeMillis()
            choicesTarget = target
            choicePreview = null
            choiceStale = false
        }
    }

    /**
     * **جيلُ الملاحة الحاضر** — تكتبه الشاشةُ عند كلّ تبدّل.
     *
     * **ولا يُقرأ من `NavigationSession`** — هي في التركيب، **ونموذجُ
     * العرض لا يراها.**
     */
    var navGeneration: Long = 0L
        private set

    private var navTarget: com.rahalgo.navigation.RouteTarget? = null

    private fun choicesTargetNow(): com.rahalgo.navigation.RouteTarget =
        navTarget ?: choicesTarget ?: com.rahalgo.navigation.RouteTarget.PICKUP

    /**
     * **تُخبر الشاشةُ نموذجَ العرض بما تركّب ولماذا** — البند ٧.
     *
     * **والسببُ يُمسك هنا حيث يُعرف من ناداه** — **ولا يُعدَّل
     * `NavEngine`.**
     */
    fun onRouteInstalled(
        generation: Long,
        target: com.rahalgo.navigation.RouteTarget,
        reason: com.rahalgo.navigation.RouteInstallReason,
    ) {
        navGeneration = generation
        navTarget = target
        lastInstallReason = reason
    }

    /** **عمرُ الجلبة** — تقرؤه الشاشةُ لتبني السياق. */
    fun choicesAgeMs(): Long =
        if (choicesAtMs == 0L) Long.MAX_VALUE else System.currentTimeMillis() - choicesAtMs

    /**
     * **معاينةُ بديل** — البند ١١.
     *
     * **ولا `setRoute`** — تغييرٌ بصريٌّ محض.
     */
    fun previewRoute(routeId: String?) {
        choiceStale = false
        val c = routeChoices ?: return
        // **والموصى به يُلغي المعاينة** — فهو الفعّالُ أصلاً (البند ١٦).
        choicePreview = routeId?.takeIf { it != c.recommended.routeId }
    }

    fun cancelPreview() {
        choicePreview = null
        choiceStale = false
    }

    /**
     * **اعتمادُ المعايَن** — البندان ١٧ و١٨.
     *
     * **يُفحص هنا ثمّ يُوضع في `committedRoute`** — والشاشةُ تسلّمه.
     * **ولا تعكس الواجهةُ شيئاً قبل الفحص.**
     *
     * @return **صحيحٌ إن قُبل** — وإلّا فالمسارُ الحاليُّ لم يُمسّ.
     */
    fun confirmRoute(
        generation: Long,
        target: com.rahalgo.navigation.RouteTarget,
        originLat: Double,
        originLng: Double,
        progressM: Double,
        healthy: Boolean,
    ): Boolean {
        val c = routeChoices ?: return false
        val previewId = choicePreview ?: return false

        val ctx = com.rahalgo.navigation.RouteChoiceMachine.Context(
            generation = generation,
            target = target,
            originLat = originLat,
            originLng = originLng,
            progressM = progressM,
            ageMs = choicesAgeMs(),
            healthy = healthy,
        )
        val ui = com.rahalgo.navigation.RouteChoiceUi.Previewing(c, previewId)

        return when (val out = com.rahalgo.navigation.RouteChoiceMachine.commit(ui, ctx)) {
            is com.rahalgo.navigation.RouteChoiceMachine.Commit.Accept -> {
                committedRoute = out.option.route
                true
            }
            is com.rahalgo.navigation.RouteChoiceMachine.Commit.Reject -> {
                /**
                 * **والمرفوضُ يُزال ولا يُمسّ المسارُ الحاليّ**
                 * (البند ١٩): «ألغِ Preview · أزل Alternative
                 * المنتهية · لا تغيّر Current navigation».
                 */
                choicePreview = null
                choiceStale = true
                routeChoices = c.copy(
                    alternatives = c.alternatives.filter { it.routeId != previewId },
                )
                false
            }
        }
    }

    /**
     * **بعد أن تسلّمه الشاشةُ** — فلا يُسلَّم مرّتين.
     *
     * **ويُعلَن السببُ `USER_SELECTION`** (البند ٦) — **فالجيلُ الذي
     * سبّبه الاختيارُ لا يُطلق جلباً جديداً**، ولا تعود اللوحةُ فورَ
     * ما اختار السائق.
     */
    /**
     * ══════════════════════════════════════════════════════════════
     * **يسأل الخادمَ: أنا على المسار أم على غيره؟** — المرحلة ٨ب
     * ══════════════════════════════════════════════════════════════
     *
     * **ولا يُنادى في حلقة** (البند ٢٠ من التحليل): تناديه الشاشةُ
     * حين يُخرج المُحلِّلُ طلباً — **وذلك عند اشتباهٍ مستمرٍّ لا مع
     * كلّ قراءة.**
     *
     * **وسقوطُه صامت**: لا شبكةَ يعني `INSUFFICIENT_DATA`
     * (البند ١٩ من التنفيذ) — **ولا إعادةَ محاولةٍ ولا حلقة.**
     */
    fun askCorrelation(
        routeId: String,
        fixes: List<com.rahalgo.shared.model.CorrelationFix>,
        done: (com.rahalgo.navigation.RoadCorrelation) -> Unit,
    ) {
        val id = currentId() ?: return done(
            com.rahalgo.navigation.RoadCorrelation.INSUFFICIENT_DATA,
        )
        viewModelScope.launch {
            val verdict = try {
                backend.driver.roadCorrelation(id, routeId, fixes)
            } catch (e: kotlinx.coroutines.CancellationException) {
                throw e
            } catch (e: Exception) {
                null
            }
            done(
                when (verdict?.status) {
                    "ON_PLANNED_ROUTE" -> com.rahalgo.navigation.RoadCorrelation.ON_PLANNED_ROUTE
                    "PARALLEL_ROUTE" -> com.rahalgo.navigation.RoadCorrelation.PARALLEL_ROUTE
                    "AMBIGUOUS" -> com.rahalgo.navigation.RoadCorrelation.AMBIGUOUS
                    // **وحالةٌ لا نعرفها تُقرأ نقصاً لا حكماً.**
                    else -> com.rahalgo.navigation.RoadCorrelation.INSUFFICIENT_DATA
                },
            )
        }
    }

    fun onRouteCommitted() {
        committedRoute = null
        lastInstallReason = com.rahalgo.navigation.RouteInstallReason.USER_SELECTION
        clearChoices()
    }

    /**
     * **مسحٌ صريح** — البندان ٢٢ و٢٣.
     *
     * **عند تبدّل الوجهة، أو نجاح إعادة الحساب، أو نهاية الرحلة.**
     * **والمجموعةُ القديمةُ انتهت بنيويّاً** لأنّ الجيلَ زاد.
     */
    fun clearChoices() {
        routeChoices = null
        choicePreview = null
        choiceStale = false
        choicesAtMs = 0L
        choicesTarget = null
    }

    /** **يقرأ ما ينتظره في طلبه الحاليّ** — وفشلُه صامت. */
    private suspend fun loadChatBadge() {
        val id = currentId() ?: return
        runCatching { backend.chat.threads().threads }
            .onSuccess { rows -> chatUnread = rows.firstOrNull { it.orderId == id }?.unread ?: 0 }
    }

    /** **يعيد قراءة الحديث المفتوح** — بلا وميضِ تحميلٍ ولا فقدِ ما كُتب. */
    private fun reloadChat() {
        val id = currentId() ?: return
        viewModelScope.launch {
            runCatching { backend.chat.thread(id) }.onSuccess { th ->
                chat = ChatState(
                    messages = th.messages,
                    peerName = th.peerName,
                    open = th.open,
                )
                // **وما قُرئ لا يبقى في الشارة** — فتحُ الحديث يوسمه.
                chatUnread = 0
            }
        }
    }

    fun openChat() {
        val id = currentId() ?: return
        chat = ChatState(busy = true)
        viewModelScope.launch { loadChat(id) }
    }

    fun closeChat() {
        chat = null
    }

    fun sendMessage(body: String) {
        val id = currentId() ?: return
        if (body.isBlank()) return
        viewModelScope.launch {
            try {
                backend.chat.send(id, body)
            } catch (e: Exception) {
                Log.w("RahalGo/chat", "تعذّر إرسال الرسالة", e)
            }
            loadChat(id)
        }
    }

    private suspend fun loadChat(id: String) {
        chatUnread = 0
        chat = try {
            val thread = backend.chat.thread(id)
            ChatState(
                messages = thread.messages,
                peerName = thread.peerName,
                open = thread.open,
            )
        } catch (e: Exception) {
            Log.w("RahalGo/chat", "تعذّرت قراءة الحديث", e)
            chat?.copy(busy = false) ?: ChatState(busy = false)
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **«لدي مشكلة» — وليس لكلّ حالٍ سببٌ يُختار**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرار المالك ٢٠٢٦-٠٨-١٢.)
    //
    // **والمحرّك يقبل أسبابَ تعذّرٍ عند بابين فقط**: عند المتجر وعند
    // باب الزبون — **وهو صواب**: «الزبون غائب» لا تُقال وأنت في الطريق.
    //
    // **وما بينهما ليس بلا مشاكل**: عطلٌ في الدرّاجة، أو إيقافٌ في
    // الطريق. **فيُفتح بابُ الطارئ** — تُنبَّه العمليات ويُقرأ موضعُه.
    var emergencyOpen by mutableStateOf(false)
        private set

    fun dismissEmergency() {
        emergencyOpen = false
    }

    /** **يبلّغ العمليات** — ثمّ يُغلق النافذة ويعيد القراءة. */
    fun emergency(point: LastPoint.Point?, note: String = "") {
        val id = currentId() ?: return
        emergencyOpen = false
        viewModelScope.launch {
            runCatching { backend.driver.emergency(id, point?.lat, point?.lng, note) }
                .onFailure { Log.w("RahalGo/طارئ", "تعذّر البلاغ", it) }
            load()
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **ومشكلةُ السائق فعلٌ يختلف بحسب موضعه من الرحلة**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرار المالك ٢٠٢٦-٠٨-١٢.)
    //
    //	قبل الاستلام  ←  يعود الطلبُ للطابور، ويأخذه سائقٌ ثانٍ في دقيقة
    //	بعد الاستلام  ←  البضاعةُ في يده، **فتُنبَّه العملياتُ ولا تُترك**
    //
    // **والسببُ يُكتب في الحالين**: «تعطّلت درّاجتي» تُقرأ في المكتب
    // ويُعرف بها لماذا تأخّر الطلب، **وإعادةٌ بلا سبب تُقرأ تردّدا.**
    fun reportProblem(reason: String, point: LastPoint.Point?) {
        val order = state.mine.firstOrNull { it.id == openId }
            ?: state.mine.firstOrNull()
            ?: return
        detail = detail.copy(failReasons = null)
        if (order.status == "assigned" || order.status == "at_pickup") {
            releaseWith(reason)
        } else {
            emergency(point, reason)
        }
    }

    private fun releaseWith(note: String) {
        val id = currentId() ?: return
        detail = detail.copy(busy = true, error = "")
        viewModelScope.launch {
            try {
                backend.driver.release(id, note)
                openId = null
            } catch (e: Exception) {
                detail = detail.copy(busy = false, error = describe(e))
                return@launch
            }
            detail = detail.copy(busy = false)
            reload()
        }
    }

    var agreeOpen by mutableStateOf(false)
        private set

    fun askAgree() {
        agreeOpen = true
    }

    fun dismissAgree() {
        agreeOpen = false
    }

    /** **يوثّق ما اتُّفق عليه** — ثمّ يعيد قراءة الطلب بسعره الجديد. */
    fun agree(goods: Long, fee: Long) {
        val id = currentId() ?: return
        agreeOpen = false
        detail = detail.copy(busy = true, error = "")
        viewModelScope.launch {
            try {
                backend.driver.agree(id, goods, fee)
            } catch (e: Exception) {
                detail = detail.copy(busy = false, error = describe(e))
                return@launch
            }
            detail = detail.copy(busy = false)
            reload()
        }
    }

    /**
     * **عروضٌ تركها وهو في رحلة** — فلا تعود لافتتُها.
     *
     * **وفي الذاكرة لا على القرص**: العرض يفوت خلال دقائق، **وقائمةٌ
     * تعيش بعد إقلاعٍ جديدٍ تخفي عرضاً جديدا.**
     */
    private val dismissedOffers = mutableSetOf<String>()

    /**
     * **أاستُعملت أولويّتُه في هذه الرحلة؟**
     *
     * **وتُستهلك بالرفض كما تُستهلك بالقبول** — (قرارُ المالك
     * ٢٠٢٦-٠٨-١٥): **عرضٌ واحدٌ لا خمسة.**
     *
     * **وتُصفَّر حين تبدأ رحلةٌ جديدة** — وهو ما يفعله `openTrip`.
     */
    private var offerSpent by mutableStateOf(false)

    fun dismissOffer() {
        state.offers.firstOrNull { it.id !in dismissedOffers }?.let { dismissedOffers += it.id }
        // **ورفضةٌ واحدةٌ تُنهي عرضَ الرحلة** — لا تنقله إلى التالي.
        offerSpent = true
        // **ولمسةٌ للحالة** — لتُعاد قراءة الشاشة.
        state = state.copy()
    }

    /** يعيد القراءة **ويحدّث الطلب المفتوح من القائمة نفسها.** */
    private fun reload() {
        viewModelScope.launch {
            try {
                val mine = backend.driver.orders()
                state = state.copy(
                    offers = backend.driver.queue(),
                    mine = mine,
                    loading = false,
                    error = "",
                )
                val id = openId
                if (id != null) {
                    val found = mine.firstOrNull { it.id == id }
                    // **وطلب اختفى من القائمة يُغلق** — أُلغي من المكتب
                    // أو فُكّ إسناده، **ومن أبقى شاشته** جعله يضغط خطوة
                    // على طلب لم يعد له.
                    if (found == null) openId = null else detail = detail.copy(order = found)
                }
            } catch (e: Exception) {
                state = state.copy(error = describe(e))
            }
        }
    }

    /**
     * **رفض المحرّك ليس عطبا** — والشاشة تقول سببه كما قاله.
     *
     * (رموز `driver_handlers.go`: `order_taken` · `not_on_shift` ·
     * `cash_limit_reached` · `too_many_active_orders`.)
     */
    /** **الرمزُ بعربيّة** — من الخريطة المركزيّة (`data/ApiErrors.kt`). */
    private fun describe(e: Exception): String = apiError(getApplication(), e)
}
