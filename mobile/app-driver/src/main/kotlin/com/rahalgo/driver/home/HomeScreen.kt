package com.rahalgo.driver.home

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import com.rahalgo.ui.StatBox
import com.rahalgo.ui.StatRow
import com.rahalgo.ui.Bar
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Switch
import com.rahalgo.shared.model.IncentivesPayload
import com.rahalgo.design.Rahal
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import com.rahalgo.driver.trip.MapPackageWorker
import androidx.annotation.StringRes
import com.rahalgo.driver.trip.OfflineMap
import com.rahalgo.map.MapStyleRepository
import androidx.compose.ui.platform.LocalContext
import androidx.compose.runtime.LaunchedEffect
import kotlinx.coroutines.withContext
import kotlinx.coroutines.Dispatchers
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import com.rahalgo.driver.R
import com.rahalgo.ui.grouped
import com.rahalgo.ui.money
import com.rahalgo.driver.location.Readiness
import com.rahalgo.shared.model.DriverMe
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.RahalLoader
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لوحة السائق — الوردية أوّلا**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (خطوة البناء الرابعة، قرار المالك ٢٠٢٦-٠٨-١١.)
 *
 * # لماذا الوردية في أعلى الشاشة وأكبر شيء فيها
 *
 * **من ليس على وردية لا يصله طلب** — وهذا كلّ شيء بالنسبة له. **وسائق
 * ينتظر طلبا لا يأتي لأنّ ورديته مغلقة** هو أسوأ ما يقع في يومه، ولا شيء
 * في الشاشة يقول له لماذا.
 *
 * **فتقول الشاشة حاله بلونها كلّها** لا بسطر صغير: أخضر يعمل، رمادي
 * واقف.
 *
 * # وأرقام اليوم لا أرقام الشهر
 *
 * **السائق يسأل: كم سلّمت اليوم وكم كسبت؟** — والمجاميع الكبيرة تخصّ
 * المكتب. **ومن عرض الشهر أضاع الرقم الذي يُسأل عنه في الشارع.**
 *
 * # ولا منطق هنا
 *
 * (`GROUND-RULES.md` §7.2 البند ٥.) **الشاشة تعرض وترسل** —
 * والنداءات في `HomeViewModel`.
 */
@Composable
fun HomeScreen(state: HomeState, actions: HomeActions) {
    if (state.me == null) {
        Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            if (state.error.isEmpty()) {
                RahalLoader()
            } else {
                // **وأوّل نداء قد يسقط والشاشة فارغة** — فلا يبقى صاحبها
                // أمام بياض بلا سبب ولا زرّ.
                com.rahalgo.ui.ErrorState(title = state.error, onRetry = actions.refresh)
            }
        }
        return
    }

    val me = state.me
    val context = LocalContext.current
    // **ويُفحص عند كلّ فتح** — قد يكون نزّلها ثمّ مسح بيانات التطبيق.
    // ══════════════════════════════════════════════════════════════════
    // **ومنطقتُه تُشتقّ من موضعه قبل أن يُسأل عن حزمتها**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-٢٣: «كي لا نضطرَّ لاحقاً لتعديل البرنامج
    //  والكود».)
    //
    // **والموضعُ من `LastPoint`** — يكتبه `Here` وخدمةُ الورديّة.
    // **ولا يُطلب إذنٌ لأجله**: من لم يُعرف موضعُه بقي على الارتداد،
    // **وشاشةٌ تسأل إذنَ موقعٍ لتعرف أيَّ خريطةٍ تعرض تُقرأ تطفّلا.**
    //
    // **ويُعاد مع كلّ تبدّلِ موضع** — **والسائقُ ينتقل بين مدنٍ**،
    // ومن نزّل حزمةَ مدينته الأولى لا يبقى عليها إلى الأبد.
    val here = com.rahalgo.ui.LastPoint.value
    LaunchedEffect(here?.lat, here?.lng) {
        // ══════════════════════════════════════════════════════════════
        // **والفهرسُ يُطلب هنا — لا في `MapCanvas` وحدَها**
        // ══════════════════════════════════════════════════════════════
        //
        // (قِيس على جهاز المالك ٢٠٢٦-٠٨-٢٣.)
        //
        // **كان `ensureManifest` يُنادى من رسم الخريطة فقط** — **وهذه
        // الشاشةُ لا ترسم خريطةً وتعرض بطاقةَ تنزيلها.** فتسأل فهرساً
        // لم يُطلب بعد.
        //
        // **فبقي فهرسُ الجهاز على الرقّة وحدَها** بعد أن أُضيفت دمشقُ
        // إلى الخادم، **ولا سبيلَ لسائقٍ هناك أن يعرف أنّ لمدينته
        // حزمة.**
        MapStyleRepository.ensureManifest()
        if (here != null) OfflineMap.pickRegion(here.lat, here.lng)
        // **و`check` تسرد مجلّدات الحزم وتقرأ أوصافَها** — قرصٌ لا
        // ذاكرة. **وفي الخيط الرئيسيّ تُجمّد الشاشة** (بلاغُ المالك
        // ٢٠٢٦-٠٩-٠٢: «لا يستجيب»).
        withContext(Dispatchers.IO) { OfflineMap.check(context) }

        // ══════════════════════════════════════════════════════════════
        // **وتُنزَّل من نفسها على الواي فاي — بلا أن يُسأل**
        // ══════════════════════════════════════════════════════════════
        //
        // **(قرارُ المالك ٢٠٢٦-٠٩-٠١:** «بدنا نلاقي حلّاً لحجم التنزيل،
        // أو نعمل التنزيلَ بالخلفيّة بدون ما يحسّ الشخصُ أصلاً · **يمكن
        // ما ينزّلها ويمكن يتجاهلها ويمكن يؤجّلها**».)
        //
        // **وهو محقٌّ**: قرارٌ بأربعمئةِ ميغا يُؤجَّل، **والمؤجَّلُ يُنسى**
        // — ثمّ ينقطع الإنترنتُ في الطريق فلا خريطة.
        //
        // **والعاملُ مبنيٌّ منذ زمنٍ ولم يُنادَ إلّا بالزرّ**: `AUTOMATIC`
        // تشترط شبكةً غيرَ محسوبة (`UNMETERED`) ومساحةً كافية،
        // **وتستأنف إن انقطعت، ولا تعيد من الصفر.**
        //
        // **و`KEEP` لا `REPLACE`** — فلو نُودي عند كلّ فتحةِ شاشةٍ لم
        // يُعَد ما يعمل، **ولا تُهدر ميغاباتٌ نُزّلت.**
        //
        // **ولا تُسحب من حزمته**: الواي فاي شرطٌ في `AUTOMATIC` —
        // **ومن نزّل أربعمئةَ ميغا على بيانات جوّاله لا ينسى ذلك.**
        //
        // **والزرُّ يبقى** لمن أراد الآن على أيّ شبكة (`USER`).
        if (!OfflineMap.ready && !OfflineMap.downloading) {
            MapPackageWorker.enqueue(context, trigger = MapPackageWorker.Companion.Trigger.AUTOMATIC)
        }
    }
    Column(
        Modifier
            .fillMaxSize()
            // **وتحت شريط الحالة لا خلفه** — الشاشة تُرسم من حافة إلى
            // حافة. **وشريط النظام السفليّ يحسبه `Scaffold` مرّة**،
            // فلو أُضيف هنا `safeDrawing` لحُسب مرّتين وارتفع المحتوى.
            .verticalScroll(rememberScrollState())
            .padding(horizontal = Rahal.space.lg),
    ) {
        // **فراغٌ واحدٌ فوق العنوان لا اثنان** (الفحصُ البصريّ ٢٠٢٦-١١-١١): الشريطُ العلويُّ يحسب شريطَ
        // الحالة، **وحشوتُه هنا مع الفراغ كانا ٤٨ نقطةً فارغةً** — والآن كشاشات الزبون والمتجر.
        Spacer(Modifier.height(Rahal.space.lg))
        // **عنوانُ الشاشة بالدور الواحد** كباقي الشاشات (إعادةُ التصميم ٢٠٢٦-١١-١١).
        com.rahalgo.ui.ScreenTitle(title = stringResource(R.string.login_welcome, me.fullName))

        // ══════════════════════════════════════════════════════════════
        // **الموقع مطفأ يُقال فوق كلّ شيء**
        // ══════════════════════════════════════════════════════════════
        //
        // **وهو أوّل ما يُسأل عنه المكتب**: «لماذا لا تصلني طلبات؟»
        // — لأنّ المحرّك لا يعرف أين هو، **فلا يحسب مسافة ولا يعرض
        // عليه أقرب طلب.** والسائق لا يرى من ذلك شيئا.
        // **والحسابُ الموقوفُ يُقال أوّلاً** (٢٠٢٦-١٠-٠٢).
        if (me.suspended) {
            // **والإيقافُ وحدَه بالأحمر** — ليس نقصاً يُصلَح بزرّ.
            BannerCard(
                icon = R.drawable.ic_block,
                text = stringResource(R.string.suspended_banner),
                tone = com.rahalgo.ui.StatusTone.Danger,
            )
            Spacer(Modifier.height(Rahal.space.md))
        }

        // **وشروطُ العمل الجديدة قبل مفتاح الورديّة** (قرارُ المالك ٢٠٢٦-١١-٠٥) —
        // شريطٌ يفتح اللوحة، **والمفتاحُ نفسُه يفتحها إن ضُغط قبل الموافقة.**
        if (me.termsRequired && !me.suspended) {
            BannerCard(
                icon = R.drawable.ic_terms,
                // **والخادمُ يمنع فتحَ الدوام وحدَه** (`driverTermsBlock` عند `on=true`، ولا يمنع الإغلاقَ ولا ما بيده):
                // فمن دوامُه مفتوحٌ يُقال له إنّ الموافقةَ لدوامه الجاي، لا «لتقدر تفتح» وهو مفتوح (٢٠٢٦-١١-١١).
                text = stringResource(if (me.onShift) R.string.terms_banner_on_shift else R.string.terms_banner),
                tone = com.rahalgo.ui.StatusTone.Info,
                onClick = actions.openTerms,
            )
            Spacer(Modifier.height(Rahal.space.md))
        }
        state.shiftSummary?.let { ShiftSummaryDialog(it, onClose = actions.closeSummary) }
        state.terms?.let {
            TermsSheet(
                terms = it,
                busy = state.termsBusy,
                error = state.termsError,
                onAccept = actions.acceptTerms,
                onClose = actions.closeTerms,
            )
        }

        // ══════════════════════════════════════════════════════════════
        // **ومفتاحُ الورديّة أوّلُ ما يُرى — دائماً**
        // ══════════════════════════════════════════════════════════════
        //
        // (تدقيقُ التصميم ٢٠٢٦-١٠-٢٧: ستُّ بطاقاتٍ برتقاليّةٍ فوق بعض كانت تدفعه
        //  إلى أسفل الشاشة.) **وما ينقصه بعده في بطاقةٍ واحدة.**
        ShiftCard(me = me, busy = state.busy, onToggle = actions.toggleShift)

        // ══════════════════════════════════════════════════════════════
        // **«قبل ما تشتغل: ناقصك…» — بطاقةٌ واحدةٌ لا ستّ**
        // ══════════════════════════════════════════════════════════════
        //
        // **الموقعُ، وإشارتُه، والموقعُ طولَ الوقت، والتشغيلُ التلقائيّ، والإشعارات** —
        // كلٌّ سطرٌ بزرّه. **وما يُبقي الموقعَ حيّاً والتطبيقُ مغلق** (٢٠٢٦-١٠-١٣):
        // «أثناء الاستخدام» يُوقف الموقعَ متى أُغلق التطبيق، **وجوّالاتُ شاومي
        // وأخواتُها تقتل الخدمة.** **والإشعارُ شرطُ الإعلان عن التوفّر** (`DRF-04`):
        // كشفُ الطلبات دفعٌ من `FCM`، **ومرفوضُه تُسقط رسالتُه صامتة.**
        var autoStartDone by remember { mutableStateOf(com.rahalgo.driver.location.AutoStart.done(context)) }
        val fixes = buildList {
            if (!state.locationOn) {
                add(
                    Fix(
                        title = stringResource(R.string.loc_permission_title),
                        text = stringResource(R.string.loc_permission_text),
                        button = stringResource(R.string.loc_permission_button),
                        onClick = actions.enableLocation,
                    ),
                )
            } else if (me.onShift && com.rahalgo.driver.location.GpsSignal.lost) {
                add(
                    Fix(
                        title = stringResource(R.string.gps_lost_title),
                        text = stringResource(R.string.gps_lost_text),
                    ),
                )
            }
            if (state.locationOn && !com.rahalgo.driver.location.LocationPermission.backgroundGranted(context)) {
                add(
                    Fix(
                        title = stringResource(R.string.keep_bg_title),
                        text = stringResource(R.string.keep_bg_text),
                        button = stringResource(R.string.keep_bg_button),
                        onClick = actions.enableLocation,
                    ),
                )
            }
            if (com.rahalgo.driver.location.AutoStart.applies() && !autoStartDone) {
                add(
                    Fix(
                        title = stringResource(R.string.keep_autostart_title),
                        text = stringResource(R.string.keep_autostart_text),
                        button = stringResource(R.string.keep_autostart_button),
                        onClick = {
                            com.rahalgo.driver.location.AutoStart.open(context)
                            autoStartDone = true
                        },
                    ),
                )
            }
            if (state.notifyMissing) {
                add(
                    Fix(
                        title = stringResource(R.string.notify_needed_title),
                        text = stringResource(R.string.notify_needed_text),
                        button = stringResource(R.string.notify_needed_button),
                        onClick = actions.enableNotifications,
                        link = stringResource(R.string.notify_needed_settings),
                        onLink = actions.openAppSettings,
                    ),
                )
            }
        }
        if (fixes.isNotEmpty()) {
            Spacer(Modifier.height(Rahal.space.md))
            FixesCard(fixes, online = me.onShift)
        }
        Spacer(Modifier.height(Rahal.space.md))

        MapPrepStatus(
            // **وطلبتها الورديّةُ أو طلبٌ مقبول** (٢٠٢٦-١٠-٣٠) — تُرى ولو لم يبدأ العاملُ بعد.
            downloading = (OfflineMap.downloading || MapPackageWorker.tripWanted) && !OfflineMap.ready,
            ready = OfflineMap.ready,
            percent = OfflineMap.status.progress,
            downloadedBytes = OfflineMap.status.downloadedBytes,
            totalBytes = OfflineMap.status.totalBytes,
            failed = OfflineMap.failed,
            failure = OfflineMap.failure,
            onRetry = { MapPackageWorker.enqueue(context) },
        )

        // ══════════════════════════════════════════════════════════════
        // **وتنبيهُ البطّاريّة تحت مفتاح الورديّة مباشرةً**
        // ══════════════════════════════════════════════════════════════
        //
        // **(قِيس ٢٠٢٦-٠٩-٠٢.)** ويُعرض بعد رفع الورديّة وحدَها —
        // **وهي اللحظةُ التي يفهم فيها لماذا يُسأل.**
        //
        // **ولا يُغلق البابَ عليه**: بطاقةٌ يقرؤها ويقرّر، **لا نافذةٌ
        // تحجب الشاشةَ حتّى يستجيب.** فمن رفض اليومَ يعمل، ومن قبل
        // بقي موضعُه حيّاً.
        // **ولا بطاقةَ «اعمل في الخلفية» في الرئيسيّة** (قرارُ المالك ٢٠٢٦-٠٩-٢٨):
        // إعفاءُ البطّاريّة صار خطوةً في تهيئة أوّل دخول (`SetupScreen`)، لا بطاقةً
        // دائمةً بـ«لاحقاً» على لوحة العمل. **والرئيسيّةُ للعمل لا للتهيئة.**

        if (state.error.isNotEmpty()) {
            Spacer(Modifier.height(Rahal.space.md))
            Text(state.error, color = Rahal.colors.danger, style = Rahal.type.bodyStrong, modifier = Modifier.fillMaxWidth())
        }

        com.rahalgo.ui.SectionHeader(title = stringResource(R.string.act_today))
        // **أرباحُ اليوم رقمٌ كبيرٌ بعملته** — كانت مربّعاً ثالثاً بلا «ل.س» جنب عدّين،
        // **فيضيع الفرقُ بين عددٍ ومبلغ.** والعدّان تحتها بخطٍّ أصغر.
        com.rahalgo.ui.RahalCard(padding = Rahal.space.lg) {
            Text(
                text = stringResource(R.string.home_earned),
                color = Rahal.colors.inkMuted,
                style = Rahal.type.label,
            )
            Spacer(Modifier.height(Rahal.space.xxs))
            Text(
                text = com.rahalgo.driver.Bidi.isolate(money(me.todayEarned)),
                color = Rahal.colors.success,
                style = Rahal.type.display,
            )
            if (me.todayCompensated > 0) {
                Text(
                    text = stringResource(R.string.home_compensated, money(me.todayCompensated)),
                    color = Rahal.colors.inkMuted,
                    style = Rahal.type.caption,
                )
            }
            Spacer(Modifier.height(Rahal.space.md))
            // **ثلاثةُ أرقامٍ مستقلّةٍ في ثلاثة مربّعاتٍ متساوية** — تُقرأ بنظرةٍ وهو واقف.
            StatRow {
                com.rahalgo.ui.StatTile(
                    label = stringResource(R.string.home_stat_delivered),
                    value = grouped(me.todayDelivered.toLong()),
                    compact = true,
                    modifier = Modifier.weight(1f),
                    valueColor = Rahal.colors.success,
                    icon = R.drawable.ic_check_circle,
                )
                com.rahalgo.ui.StatTile(
                    label = stringResource(R.string.home_stat_failed),
                    value = grouped(me.todayFailed.toLong()),
                    compact = true,
                    modifier = Modifier.weight(1f),
                    valueColor = if (me.todayFailed > 0) Rahal.colors.danger else Rahal.colors.ink,
                    icon = R.drawable.ic_warning,
                )
                // **ومسافةُ طلباته اليوم** (قرارُ المالك ٢٠٢٦-١١-٠٥) — تبدأ بصفرٍ وتزيد مع كلّ طلب،
                // من القيادة الفعليّة كما يحسبها المحرّك. **وصفرُها شرطة** — لا «٠ م».
                com.rahalgo.ui.StatTile(
                    label = stringResource(R.string.home_stat_drive),
                    value = if (me.todayDriveM > 0) com.rahalgo.ui.dist(me.todayDriveM.toDouble()) else "—",
                    compact = true,
                    modifier = Modifier.weight(1f),
                    icon = R.drawable.ic_moto,
                )
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **ولا سطرَ مالٍ هنا — رصيدُه في الشريط فوقه**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «المالُ نحذفه، ما يلزم — موجودٌ فوق
        //  أساساً».)
        //
        // **ورقمٌ يُعاد في شاشةٍ واحدةٍ مرّتين يُقرأ رقمين** — فيسأل
        // صاحبُه أيُّهما الصحيح.

        // ══════════════════════════════════════════════════════════════
        // **وصندوقُه شريطٌ لا سطر**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «نحطّ شريطَ صندوقي، شقد في ذمّة
        //  السائق».)
        //
        // **والسقفُ ليس زينة**: من بلغه لا يُعرض عليه طلبٌ نقديّ، **فيقف
        // عملُه ولا يعرف لماذا.** **وشريطٌ يُرى قبل أن يُقرأ رقم.**
        if (me.cashLimit > 0) {
            Spacer(Modifier.height(Rahal.space.md))
            com.rahalgo.ui.RahalCard(padding = Rahal.space.lg) { CashBar(held = me.cashHeld, limit = me.cashLimit) }
        }

        // **وأفضلُ أوقات الشغل** (قرارُ المالك ٢٠٢٦-١١-١١) — بطاقةٌ تُخفى إن لم تصل بياناتٌ كافية.
        state.busyHours?.takeIf { it.top.isNotEmpty() }?.let {
            Spacer(Modifier.height(Rahal.space.md))
            BusyHoursCard(it)
        }

        // ══════════════════════════════════════════════════════════════
        // **وهدفُه بعده — وجائزتُه معه**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «بعدها نحطّ شريطَ الهدف وقيمةَ
        //  المكافأة عند تحقيق الهدف».)
        //
        // **وحافزٌ لا يُرى لا يحفّز** — ومن لا يعرف أنّه على بُعد ثلاثةِ
        // طلباتٍ من مكافأةٍ لا يسعى إليها.
        val st = state.goal?.standing
        if (st != null && st.target > 0) {
            Spacer(Modifier.height(Rahal.space.md))
            com.rahalgo.ui.RahalCard(padding = Rahal.space.lg) {
                GoalBar(
                    done = st.done,
                    target = st.target,
                    reached = st.reached,
                    reward = state.goal?.targetReward ?: 0,
                )
            }
        }

        // **والخروجُ انتقل إلى أسفل القائمة الجانبيّة** — (قرارُ المالك
        // ٢٠٢٦-٠٨-١٣). **وموضعان لفعلٍ واحدٍ يجعلان أحدَهما يُنسى**،
        // ولوحةُ العمل ليست موضعَ فعلٍ يُنهي الجلسة.

        // ══════════════════════════════════════════════════════════════
        // **لا أدواتِ تطويرٍ في واجهة السائق** (قرارُ المالك ٢٠٢٦-٠٩-٢٨)
        // ══════════════════════════════════════════════════════════════
        //
        // **حُذف مدخلُ «أدوات التطوير · QA»** (الرحلةُ التجريبيّة ومختبرُ
        // الملاحة) من الواجهة العاديّة — **فلا يبلغه سائقٌ حقيقيّ.** ومحرّكُ
        // الإعادة وأنشطتُه تبقى في `src/debug` (تُطلَق بـ`adb` عند الحاجة
        // الهندسيّة)، **ولا مدخلَ لها في التطبيق، والإصدارُ لا يحملها أصلاً.**

        Spacer(Modifier.height(Rahal.space.xl))
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حالةُ تحضير الخريطة — سطرٌ صغيرٌ غيرُ حاجب، لا بطاقةَ قرار** (B، ٢٠٢٦-٠٩-٢٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك بعد شاهد الجهاز: حزمةُ الرقّة ~١٫٩م لا ٥٠٢م، **فالتحضيرُ
 *  تلقائيٌّ في الخلفيّة على الواي فاي** — لا بطاقةَ تنزيلٍ ولا زرَّ قرار.)
 *
 * # وما زال السقوطُ يُقال
 *
 * **إغلاقُ `TD-MAP-SILENT-RESOURCE-FAIL` باقٍ**: عند الفشلِ سطرٌ صغيرٌ
 * يقول ما جرى ويتيح إعادةً يدويّةً (`RahalTextButton`، ليست حلقةً آليّة)
 * — **غيرُ حاجبٍ للعمل، لا نافذةَ قرار.** ونبضةٌ خفيفةٌ أثناء التحضير.
 * **ولا شيءَ حين تكون جاهزةً** (النجاحُ صامت).
 */
@Composable
private fun MapPrepStatus(
    downloading: Boolean,
    ready: Boolean,
    percent: Int,
    downloadedBytes: Long,
    totalBytes: Long,
    failed: Boolean,
    failure: com.rahalgo.map.data.MapFailure?,
    onRetry: () -> Unit,
) {
    // ══════════════════════════════════════════════════════════════════
    // **تقدّمٌ يُرى، و«جاهزة ✓» ثوانيَ ثمّ تغيب** (ملاحظةُ المالك ٢٠٢٦-١١-١١)
    // ══════════════════════════════════════════════════════════════════
    //
    // **«تُحضَّر خريطةُ مدينتك…» كانت تبقى بلا رقم**، فلا يُعرف أتتحرّك أم عالقة. **والنجاحُ يُقال مرّةً**
    // إن رأى التحضيرَ في هذه الشاشة، ثمّ يُخفى — فمن فتحها والخريطةُ جاهزةٌ من أسبوع لا يرى شيئاً.
    var sawPreparing by remember { mutableStateOf(false) }
    var showReady by remember { mutableStateOf(false) }
    if (downloading) sawPreparing = true
    LaunchedEffect(ready, sawPreparing) {
        if (ready && sawPreparing) {
            showReady = true
            kotlinx.coroutines.delay(READY_SHOWN_MS)
            showReady = false
            sawPreparing = false
        }
    }
    when {
        showReady -> {
            Text(
                text = stringResource(R.string.offline_map_ready),
                color = Rahal.colors.success,
                style = Rahal.type.caption,
            )
            Spacer(Modifier.height(Rahal.space.md))
        }
        failed -> {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(
                    text = stringResource(failureMessage(failure)),
                    color = Rahal.colors.accent,
                    style = Rahal.type.caption,
                    modifier = Modifier.weight(1f),
                )
                RahalTextButton(onClick = onRetry) {
                    Text(stringResource(R.string.offline_map_retry))
                }
            }
            Spacer(Modifier.height(Rahal.space.md))
        }
        downloading -> {
            // **المئويّةُ والميغاباتُ حين يُعرف الحجم** — وقبل ذلك (في الطابور) النصُّ القديم بلا رقمٍ كاذب.
            val sized = totalBytes > 0
            Text(
                text = if (sized) {
                    stringResource(
                        R.string.offline_map_preparing_pct,
                        percent.toString(),
                        megabytes(downloadedBytes),
                        megabytes(totalBytes),
                    )
                } else {
                    stringResource(R.string.offline_map_preparing)
                },
                color = Rahal.colors.inkMuted,
                style = Rahal.type.caption,
            )
            if (sized) {
                Spacer(Modifier.height(Rahal.space.xs))
                Bar(ratio = percent / 100f, color = Rahal.colors.brand)
            }
            Spacer(Modifier.height(Rahal.space.md))
        }
    }
}

/** **كم تبقى «جاهزة ✓» ظاهرة** قبل أن تغيب. */
private const val READY_SHOWN_MS = 4_000L

/** **الميغاباتُ بمنزلةٍ عشريّةٍ واحدة** — بأرقامٍ غربيّة. */
private fun megabytes(bytes: Long): String =
    String.format(java.util.Locale.US, "%.1f", bytes / 1_048_576.0)

/**
 * ══════════════════════════════════════════════════════════════════════
 * **من السبب الداخليّ إلى ما يفعله السائق**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (البند ٣ من قرار ٢٠٢٦-٠٨-٢٢: «لا تعرض تفاصيل تقنية للمستخدم».)
 *
 * **وعشرةُ أسبابٍ تُختصر إلى ثلاث عبارات** — لأنّ ما يفعله السائقُ
 * ثلاثةٌ لا عشرة: **يصلح الشبكة، أو يفرّغ مساحة، أو يعيد.**
 *
 * **و`CHECKSUM_MISMATCH` مثلاً لا يُقال للسائق** — لا يملك له حيلة،
 * **وقولُه يخيفه ولا يفيده.** والسببُ الدقيقُ في السجلّ لمن يصلح.
 *
 * **والدائمُ يُعطى «حاول مرّة أخرى» أيضاً** — فقد يكون المنبعُ أُصلح،
 * **والإعادةُ اليدويّةُ ليست حلقةَ إعادةٍ آليّة** (البند ٥).
 */
@StringRes
internal fun failureMessage(failure: com.rahalgo.map.data.MapFailure?): Int = when (failure) {
    com.rahalgo.map.data.MapFailure.NETWORK,
    com.rahalgo.map.data.MapFailure.TIMEOUT,
    -> R.string.offline_map_failed_network

    com.rahalgo.map.data.MapFailure.NO_SPACE -> R.string.offline_map_failed_space

    else -> R.string.offline_map_failed_other
}

/**
 * **بطاقة الوردية.**
 *
 * **والزرّ يقول ما سيحدث لا ما هو حادث**: «ابدأ الوردية» حين تكون مغلقة.
 * **ومن كتب حاله على الزرّ** جعل نصفهم يضغط ليصل إلى ما هو فيه أصلا.
 */
@Composable
private fun ShiftCard(me: DriverMe, busy: Boolean, onToggle: (Boolean) -> Unit) {
    val on = me.onShift
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.lg)
            // **وأخضرُ حين يعمل** (ملاحظةُ المالك ٢٠٢٦-١١-١١): كان أزرقَ بمفتاحٍ أخضرَ عليه فلا يُقرأ.
            .background(if (on) Rahal.colors.success else Rahal.colors.field)
            .padding(horizontal = Rahal.space.lg, vertical = Rahal.space.xl),
    ) {
        // ══════════════════════════════════════════════════════════════
        // **مفتاحٌ واحدٌ — حالٌ وفعلٌ معا**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «نخلّيه زرّاً ذكيّاً بالعمل وخارج
        //  العمل، وتحته يُكتب: الآن تستقبل الطلبات · لا يمكنك استقبال
        //  الطلبات أنت خارج العمل».)
        //
        // **وكانت البطاقةُ تقول شيئين متضادّين**: أعلاها الحالُ «وردية
        // مفتوحة» وأسفلها الفعلُ «أنهِ الوردية» — **ومن قرأ بسرعةٍ وهو
        // يقود لا يعرف أيُّهما حالُه.**
        //
        // **والمفتاحُ يقول الحالَ بموضعه ويقبل الفعلَ بلمسته** — ولا
        // كلمتين متضادّتين في بطاقةٍ واحدة.
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            // **نقطةُ حالٍ قبل الكلمة** — خضراءُ تعمل ورماديّةٌ واقفة، تُرى بطرف العين.
            Row(Modifier.weight(1f), verticalAlignment = Alignment.CenterVertically) {
                Box(
                    Modifier
                        .size(Rahal.space.md)
                        .clip(androidx.compose.foundation.shape.CircleShape)
                        .background(if (on) Rahal.colors.onBrand else Rahal.colors.inkMuted),
                )
                Spacer(Modifier.size(Rahal.space.sm))
                Text(
                    text = stringResource(if (on) R.string.shift_on else R.string.shift_off),
                    style = Rahal.type.headline,
                    color = if (on) Rahal.colors.onBrand else Rahal.colors.ink,
                )
            }
            if (busy) {
                // **ويُرى على البطاقة** — كان بلون العلامة فوق بطاقةٍ بلون العلامة.
                CircularProgressIndicator(
                    Modifier.size(Rahal.icon.md),
                    strokeWidth = Rahal.stroke.focus,
                    color = if (on) Rahal.colors.onBrand else Rahal.colors.brand,
                )
            } else {
                // **مفتاحٌ بعلامة ✓ في زرّه حين يعمل** — الزرُّ الأبيضُ وحدَه على اليسار كان يُقرأ «مطفأ».
                Switch(
                    checked = on,
                    onCheckedChange = onToggle,
                    thumbContent = if (on) {
                        {
                            Icon(
                                painter = painterResource(com.rahalgo.ui.R.drawable.ic_check),
                                contentDescription = null,
                                tint = Rahal.colors.onBrand,
                                modifier = Modifier.size(SwitchDefaults.IconSize),
                            )
                        }
                    } else {
                        null
                    },
                    colors = SwitchDefaults.colors(
                        checkedThumbColor = Rahal.colors.success,
                        checkedTrackColor = Rahal.colors.onBrand,
                        checkedBorderColor = Rahal.colors.onBrand,
                        uncheckedThumbColor = Rahal.colors.canvas,
                        uncheckedTrackColor = Rahal.colors.inkMuted,
                    ),
                )
            }
        }
        Spacer(Modifier.height(Rahal.space.xs))
        // **وما يفعله المفتاحُ يُقال بجانبه** (ملاحظةُ المالك ٢٠٢٦-١١-١١) — «شغّال — اضغط لتسكّر الدوام».
        Text(
            text = stringResource(if (on) R.string.shift_switch_on else R.string.shift_switch_off),
            color = if (on) Rahal.colors.onBrand else Rahal.colors.ink,
            style = Rahal.type.bodyStrong,
        )
        Spacer(Modifier.height(Rahal.space.sm))
        // **وما يترتّب عليه تحته** — لا يُترك ليُستنتج من كلمتين.
        Text(
            text = stringResource(
                if (on) R.string.shift_on_hint else R.string.shift_off_hint,
            ),
            color = if (on) Rahal.colors.onBrand.copy(alpha = 0.85f) else Rahal.colors.inkMuted,
            style = Rahal.type.body,
        )
        if (me.activeOrders > 0) {
            Spacer(Modifier.height(Rahal.space.md))
            Text(
                text = stringResource(
                    R.string.shift_active_orders,
                    grouped(me.activeOrders.toLong()),
                    grouped(me.maxActiveOrders),
                ),
                color = if (on) Rahal.colors.onBrand else Rahal.colors.inkMuted,
                style = Rahal.type.bodyStrong,
            )
        }
    }
}

/**
 * **شريطُ تنبيهٍ بأيقونته** — الإيقافُ والشروطُ بقطعةٍ واحدة (إعادةُ التصميم ٢٠٢٦-١١-١١).
 * **وما يُضغط عليه سهمٌ في طرفه** — فيُعرف أنّه يُفتح.
 */
@Composable
private fun BannerCard(
    icon: Int,
    text: String,
    tone: com.rahalgo.ui.StatusTone,
    onClick: (() -> Unit)? = null,
) {
    val ink = if (tone == com.rahalgo.ui.StatusTone.Danger) Rahal.colors.danger else Rahal.colors.brand
    com.rahalgo.ui.RahalCard(
        variant = com.rahalgo.ui.CardVariant.Tinted,
        tone = tone,
        onClick = onClick,
        padding = Rahal.space.lg,
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Icon(
                painter = painterResource(icon),
                contentDescription = null,
                tint = ink,
                modifier = Modifier.size(Rahal.icon.md),
            )
            Spacer(Modifier.size(Rahal.space.md))
            Text(text = text, color = ink, style = Rahal.type.bodyStrong, modifier = Modifier.weight(1f))
            if (onClick != null) {
                Icon(
                    painter = painterResource(R.drawable.ic_chevron_end),
                    contentDescription = null,
                    tint = ink,
                    modifier = Modifier.size(Rahal.icon.sm),
                )
            }
        }
    }
}



/** ما تعرضه اللوحة — **ولا تملكه هي.** */
data class HomeState(
    val me: DriverMe? = null,
    /**
     * **هدفُه ومكافأتُه** — (قرارُ المالك ٢٠٢٦-٠٨-١٣: «نحطّ شريطَ الهدف
     *  وقيمةَ المكافأة عند تحقيق الهدف»).
     *
     * **وفشلُ جلبه لا يُسقط اللوحة** — يُطوى الشريطُ ويبقى ما سواه:
     * **مفتاحُ العمل أهمُّ ما فيها، ولا يُحجب لأجل رقمٍ لم يصل.**
     */
    val goal: IncentivesPayload? = null,
    /**
     * **حالُ الجاهزيّة كما قرأها النظام** (`DRF-01`، ٢٠٢٦-٠٩-١٥).
     *
     * **وكانت رايةً منقولةً واحدة** (`locationOn`) — **فلا تُعرف منها
     * درجةٌ من درجة**: **ومن مُنع من الإعلان عن توفّره لأجل إشعارٍ
     * لا يُمنَع من متابعة رحلةٍ في يده.**
     *
     * **فالحالُ تُحمل كما هي، والدرجاتُ تُشتقّ منها** — **ولا رايتان
     * تفترقان.**
     */
    val readiness: Readiness.State = Readiness.State(),
    val busy: Boolean = false,
    val error: String = "",
    /**
     * **ورقةُ عدّاد الدرّاجة المفتوحة** (قرارُ المالك ٢٠٢٦-١١-٠٥) — وفارغةٌ لا ورقة.
     * تُرسم في `MainActivity` فوق كلّ تبويب (الخروجُ يطلبها من أيّ مكان).
     */
    val odometer: OdometerAsk? = null,
    /** **لوحةُ شروط العمل مفتوحةٌ بنصّها** — وفارغةٌ مغلقة (قرارُ المالك ٢٠٢٦-١١-٠٥). */
    val terms: com.rahalgo.shared.model.DriverTerms? = null,
    val termsBusy: Boolean = false,
    val termsError: String = "",
    /** **ملخّصُ نهاية الدوام** — يُفتح عند سكّره، وفارغٌ لا نافذة (٢٠٢٦-١١-١١). */
    val shiftSummary: ShiftSummary? = null,
    /** **أفضلُ أوقات الشغل** — وفارغٌ أو بلا ساعاتٍ تُخفى البطاقة. */
    val busyHours: com.rahalgo.shared.model.BusyHours? = null,
) {
    /** **أيُنتَج موقعٌ فعلاً؟** — **إذناً وخدمةً معاً.** */
    val locationOn: Boolean get() = Readiness.canWork(readiness)

    /**
     * **أيُعلَن متاحاً لطلبٍ جديد؟**
     *
     * **ومن أُعلن متاحاً ولا يبلغه النداءُ يُحسَب رافضاً وهو لا يعلم.**
     */
    val canGoOnline: Boolean
        get() = Readiness.allows(readiness, Readiness.Level.CAN_GO_ONLINE)

    /** **أينقص إذنُ الإشعار؟** — **تقرؤها البطاقةُ وحدَها.** */
    val notifyMissing: Boolean
        get() = readiness.blockers.contains(
            Readiness.Blocker.NOTIFICATION_PERMISSION_REQUIRED,
        )
}

/** **نقصٌ يمنع الطلبات** — عنوانٌ وسببٌ وزرٌّ يصلحه (وبلا زرٍّ إن لم يكن له حيلة). */
private data class Fix(
    val title: String,
    val text: String,
    val button: String = "",
    val onClick: () -> Unit = {},
    val link: String = "",
    val onLink: () -> Unit = {},
)

/**
 * **بطاقةُ «قبل ما تشتغل»** — كلُّ نقصٍ سطرٌ بأيقونةٍ وزرّه، وفوقها كم ينقصه.
 *
 * **والعنوانُ غامقٌ لا برتقاليّ** — البرتقاليُّ على أرضٍ فاتحةٍ تباينُه ٢ إلى ١،
 * **والأيقونةُ وحدَها بلون التنبيه.**
 */
@Composable
private fun FixesCard(fixes: List<Fix>, online: Boolean) {
    com.rahalgo.ui.RahalCard(
        variant = com.rahalgo.ui.CardVariant.Tinted,
        tone = com.rahalgo.ui.StatusTone.Warning,
        padding = Rahal.space.lg,
    ) {
        Text(
            // **وهو يعمل الآن؟ «حتى تستمر بالعمل»** لا «قبل أن تبدأ» (المالك ٢٠٢٦-١١-١٠).
            text = stringResource(if (online) R.string.home_fixes_title_online else R.string.home_fixes_title, fixes.size),
            fontWeight = FontWeight.Bold,
            color = Rahal.colors.ink,
            style = Rahal.type.sectionTitle,
        )
        fixes.forEachIndexed { i, f ->
            if (i > 0) {
                Spacer(Modifier.height(Rahal.space.md))
                com.rahalgo.ui.RahalDivider()
            }
            Spacer(Modifier.height(Rahal.space.md))
            Row(verticalAlignment = Alignment.Top) {
                Icon(
                    painter = painterResource(R.drawable.ic_warning),
                    contentDescription = null,
                    tint = Rahal.colors.accent,
                    modifier = Modifier.padding(top = Rahal.space.xxs).size(Rahal.icon.sm),
                )
                Spacer(Modifier.size(Rahal.space.md))
                Column(Modifier.weight(1f)) {
                    Text(f.title, color = Rahal.colors.ink, style = Rahal.type.cardTitle)
                    Spacer(Modifier.height(Rahal.space.xxs))
                    Text(f.text, color = Rahal.colors.inkMuted, style = Rahal.type.body)
                    if (f.button.isNotEmpty()) {
                        Spacer(Modifier.height(Rahal.space.sm))
                        RahalButton(onClick = f.onClick, compact = true) { Text(f.button) }
                    }
                    if (f.link.isNotEmpty()) {
                        RahalTextButton(onClick = f.onLink) {
                            Text(f.link, color = Rahal.colors.inkMuted)
                        }
                    }
                }
            }
        }
    }
}

data class HomeActions(
    val toggleShift: (Boolean) -> Unit,
    val enableLocation: () -> Unit,
    /** **يطلب إذنَ الإشعار** — **شرطُ الإعلان عن التوفّر.** */
    val enableNotifications: () -> Unit = {},
    /** **يفتح صفحةَ التطبيق في النظام** — **لمن رُفض نهائيّاً.** */
    val openAppSettings: () -> Unit = {},
    val refresh: () -> Unit,
    val logout: () -> Unit,
    /** **شروطُ العمل** — فتحُ اللوحة والموافقةُ وإغلاقُها (قرارُ المالك ٢٠٢٦-١١-٠٥). */
    val openTerms: () -> Unit = {},
    val acceptTerms: () -> Unit = {},
    val closeTerms: () -> Unit = {},
    val closeSummary: () -> Unit = {},
)

/**
 * **شريطُ الصندوق** — ما في ذمّته وكم بقي قبل سقفه.
 *
 * **ويُقال ما يبقى لا ما مضى**: «يبقى ٣٫١٠٠» جوابُ سؤاله، **و«١٬٩٠٠ من
 * ٥٬٠٠٠» كسرٌ يُحسب في الرأس.**
 */
@Composable
private fun CashBar(held: Long, limit: Long) {
    val left = limit - held
    val ratio = if (limit > 0) held.toFloat() / limit else 0f
    // **جملةٌ واحدةٌ بمبلغٍ معزول** — «بـ» تبقى لاصقةً بالمبلغ ولا تنفصل في الطرف الآخر.
    val owedColor = if (ratio >= 1f) Rahal.colors.danger else Rahal.colors.accent
    Text(
        text = androidx.compose.ui.text.buildAnnotatedString {
            append(stringResource(R.string.home_cash_bar))
            append(" ")
            pushStyle(
                androidx.compose.ui.text.SpanStyle(fontWeight = FontWeight.Bold, color = owedColor),
            )
            append(com.rahalgo.driver.Bidi.isolate(money(held)))
            pop()
        },
        style = Rahal.type.sectionTitle,
    )
    Spacer(Modifier.height(Rahal.space.sm))
    Bar(ratio = ratio, color = com.rahalgo.driver.menu.cashBarColor(ratio))
    Spacer(Modifier.height(Rahal.space.sm))
    // **«بتقدر تحمل لحد X» — السقفُ كما هو** لا ما بقي منه (ملاحظةُ المالك ٢٠٢٦-١١-١١: «يبقى ٩٬٨٩٨» لم يُفهم).
    Text(
        text = if (left > 0) {
            stringResource(R.string.home_cash_left, money(limit))
        } else {
            stringResource(R.string.home_cash_over)
        },
        color = if (left > 0) Rahal.colors.inkMuted else Rahal.colors.danger,
        style = Rahal.type.caption,
    )
    // **ومن معه نقدٌ يسلّمه من هنا** — ورقةٌ بالمبلغ وعنوان المكتب ورقمه، بلا حركة مال.
    if (held > 0) {
        Spacer(Modifier.height(Rahal.space.md))
        HandInButton(held = held)
    }
}

/**
 * **شريطُ الهدف** — ما أنجزه من شهره، وما ينتظره إن بلغه.
 *
 * **والجائزةُ تُقال قبل أن تُنال** — وشاشةٌ تقول «٨ من ٥٠» ولا تقول ماذا
 * بعدها **تطلب جهداً بلا وعد.** **وصفرٌ يُخفيها**: لا يُعرَض وعدٌ بلا
 * مبلغ.
 */
@Composable
private fun GoalBar(done: Int, target: Int, reached: Boolean, reward: Long) {
    Row(
        Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = stringResource(R.string.act_month_target),
            style = Rahal.type.sectionTitle,
        )
        Text(
            text = grouped(done.toLong()) + " / " + grouped(target.toLong()),
            fontWeight = FontWeight.Bold,
            color = if (reached) Rahal.colors.success else Rahal.colors.brand,
        )
    }
    Spacer(Modifier.height(Rahal.space.sm))
    Bar(
        ratio = done.toFloat() / target,
        color = if (reached) Rahal.colors.success else Rahal.colors.brand,
    )
    Spacer(Modifier.height(Rahal.space.sm))
    Row(
        Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        if (!reached) {
            Text(
                text = targetLeftText(target - done),
                color = Rahal.colors.inkMuted,
                style = Rahal.type.caption,
            )
        }
        if (reward > 0) {
            Text(
                text = stringResource(
                    if (reached) R.string.home_target_got else R.string.home_target_reward,
                    money(reward),
                ),
                color = if (reached) Rahal.colors.success else Rahal.colors.accent,
                fontWeight = FontWeight.Medium,
                style = Rahal.type.caption,
            )
        }
    }
}

/**
 * **«يبقى …» بعدده الصحيح** — طلب واحد · طلبان · ٣–١٠ طلبات · ١١–٩٩ طلباً · ١٠٠ طلب
 * (مراجعةُ المالك ٢٠٢٦-١٠-٣٠، كـ`goalPendingText` عند المندوب).
 */
@Composable
private fun targetLeftText(n: Int): String {
    val r = n % 100
    val num = grouped(n.toLong())
    return when {
        n == 1 -> stringResource(R.string.home_target_left_one)
        n == 2 -> stringResource(R.string.home_target_left_two)
        r in 3..10 -> stringResource(R.string.home_target_left_few, num)
        r in 11..99 -> stringResource(R.string.home_target_left_many, num)
        else -> stringResource(R.string.home_target_left_other, num)
    }
}
