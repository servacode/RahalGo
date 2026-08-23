package com.rahalgo.driver.trip

import android.content.Context
import android.media.AudioAttributes
import android.media.AudioFocusRequest
import android.media.AudioManager
import android.os.Build
import android.os.Bundle
import android.speech.tts.TextToSpeech
import android.speech.tts.UtteranceProgressListener
import android.util.Log
import com.rahalgo.navigation.Speaker
import java.util.Locale

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الناطقُ — وهو وحدَه ما يعرف أندرويد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٤، أمرُ المالك ٢٠٢٦-٠٨-٢٠.)
 *
 * **ولا قرارَ ملاحةٍ هنا** — يأخذ نصّاً ويقوله. **والمخطِّطُ والمنسّقُ
 * قرّرا قبله.**
 *
 * # والصوتُ ميزةٌ لا شرطُ رحلة
 *
 * (أمرُ المالك، البند ١٥.)
 *
 * **فكلُّ إخفاقٍ ينتهي إلى `available = false`** — لا استثناءَ يصعد ولا
 * شاشةَ تسقط: **محرّكٌ غيرُ مثبَّت · عربيّةٌ غيرُ مدعومة · تهيئةٌ
 * تفشل · نطقٌ يردّ خطأً.** **والملاحةُ المرئيّةُ تعمل كاملةً.**
 *
 * # والتركيزُ للجملة وحدَها
 *
 * (أمرُ المالك، البند ١٧: «لا تمسك Audio Focus طوال الرحلة».)
 *
 * **موسيقى السائق تخفت ثانيتين ثمّ تعود** — **ومن أمسكه ساعةً أطفأ
 * راديو صاحبه.**
 */
class AndroidSpeaker(context: Context) : Speaker {

    private val app = context.applicationContext
    private val audio = app.getSystemService(AudioManager::class.java)

    /**
     * **إرشادُ ملاحةٍ لا موسيقى.**
     *
     * (أمرُ المالك، البند ١٦.)
     *
     * **وسمّاعةُ السيّارة تعامله إرشاداً** فتخفض الموسيقى ولا تقطعها،
     * **ونظامُ «عدم الإزعاج» لا يكتمه** كما يكتم الوسائط.
     */
    private val attributes = AudioAttributes.Builder()
        .setUsage(AudioAttributes.USAGE_ASSISTANCE_NAVIGATION_GUIDANCE)
        .setContentType(AudioAttributes.CONTENT_TYPE_SPEECH)
        .build()

    private var tts: TextToSpeech? = null
    private var ready = false
    private var languageOk = false
    private var focus: AudioFocusRequest? = null
    private val callbacks = HashMap<String, (Boolean) -> Unit>()

    /** **ما اختير من لغات** — يُقرأ في التشخيص لا في المنطق. */
    var selectedLocale: String? = null
        private set

    /** **ولماذا تعذّر الصوتُ إن تعذّر.** */
    var failure: String? = null
        private set

    override val available: Boolean get() = ready && languageOk

    /**
     * **يهيّئ المحرّك** — ويردّ حين يُعرف الجواب.
     *
     * **والتهيئةُ غيرُ متزامنة**: أندرويد يردّ في نداءٍ لاحق،
     * **ومن سأل عن الجاهزيّة فوراً وجدها كاذبةً دائما.**
     */
    fun init(onReady: (Boolean) -> Unit = {}) {
        if (tts != null) {
            onReady(available)
            return
        }
        runCatching {
            tts = TextToSpeech(app) { status ->
                ready = status == TextToSpeech.SUCCESS
                if (!ready) failure = "تعذّرت تهيئةُ محرّك النطق"
                if (ready) chooseLanguage()
                if (!available && failure == null) failure = "لا صوتَ عربيّ"
                Log.i(TAG, "الصوت: جاهز=$ready لغة=$selectedLocale سبب=${failure ?: "-"}")
                tts?.setOnUtteranceProgressListener(listener)
                onReady(available)
            }
        }.onFailure {
            failure = "لا محرّكَ نطقٍ مثبَّت: ${it.message}"
            ready = false
            onReady(false)
        }
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **العربيّةُ بترتيبٍ معلن — ولا يُفترض صوتٌ سوريّ**
     * ══════════════════════════════════════════════════════════════════
     *
     *     ar-SY  →  ar  →  أيُّ عربيّةٍ على الجهاز  →  لا صوت
     *
     * (أمرُ المالك، البند ١٤.)
     *
     * **ولا يُنزَّل شيءٌ تلقائيّاً** — أمرُه نصّاً: «لا تنزيلَ Voice
     * packages تلقائيّاً».
     */
    private fun chooseLanguage() {
        val engine = tts ?: return
        val candidates = listOf(
            Locale("ar", "SY"),
            Locale("ar"),
            Locale("ar", "SA"),
            Locale("ar", "EG"),
            Locale("ar", "AE"),
        )
        for (locale in candidates) {
            val verdict = runCatching { engine.isLanguageAvailable(locale) }.getOrDefault(
                TextToSpeech.LANG_NOT_SUPPORTED,
            )
            // **والبياناتُ الناقصةُ ليست دعماً** — `MISSING_DATA` تعني
            // أنّ اللغةَ معروفةٌ وصوتُها غيرُ منزَّل.
            if (verdict >= TextToSpeech.LANG_AVAILABLE) {
                if (runCatching { engine.setLanguage(locale) }.getOrDefault(
                        TextToSpeech.LANG_NOT_SUPPORTED,
                    ) >= TextToSpeech.LANG_AVAILABLE
                ) {
                    languageOk = true
                    selectedLocale = locale.toString()
                    engine.setAudioAttributes(attributes)
                    chooseVoice(engine, locale)
                    return
                }
            }
        }
        languageOk = false
        failure = "العربيّةُ غيرُ مدعومةٍ على هذا الجهاز"
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **وأحسنُ صوتٍ عربيٍّ على الجهاز — لا أوّلُ ما يقع**
     * ══════════════════════════════════════════════════════════════════
     *
     * (شكوى المالك ٢٠٢٦-٠٨-٢٣: «الصوتُ العربيُّ لم يعجبني كثيراً،
     *  أريد صوتاً عربيّاً مفهوماً».)
     *
     * # والعلّةُ أنّنا كنّا نتركه يختار
     *
     * **`setLanguage` تختار صوتَ اللغة الافتراضيَّ** — **وهو في محرّك
     * غوغل أضعفُ ما فيه غالباً**: صوتٌ مضغوطٌ محلّيٌّ خفيفُ الحجم
     * وُضع ليعمل بلا شبكة.
     *
     * **وللمحرّك نفسِه أصواتٌ عربيّةٌ أخرى أعلى درجة** — تُقرأ من
     * `engine.voices` ولا تُختار وحدَها.
     *
     * # وترتيبُ التفضيل
     *
     * **١ · الأعلى درجةً** (`Voice.quality`) — وهو ما يُسمع فرقُه.
     *
     * **٢ · وعند التساوي: المحلّيُّ قبل الشبكيّ** — **وسائقٌ في حيٍّ
     * ضعيفِ التغطية ينتظر جملةً تُجلب من الشبكة فتصله بعد المنعطف**،
     * والتعليمةُ المتأخّرةُ أسوأُ من لا تعليمة.
     *
     * **٣ · وما لم يُنزَّل يُستبعد** (`notInstalled`) — **ولا يُنزَّل
     * شيءٌ تلقائيّاً**: أمرُ المالك نصّاً «لا تنزيلَ Voice packages
     * تلقائيّاً».
     *
     * # ويُقال في السجلّ أيُّها اختير
     *
     * **وصوتٌ يُشكى منه ولا يُعرف اسمُه لا يُبدَّل** — يُقاس أوّلاً.
     */
    private fun chooseVoice(engine: TextToSpeech, locale: Locale) {
        val all = runCatching { engine.voices }.getOrNull().orEmpty()
        val arabic = all.filter {
            it.locale?.language == "ar" &&
                !it.features.orEmpty().contains(TextToSpeech.Engine.KEY_FEATURE_NOT_INSTALLED)
        }
        // **وتُسرَد كلُّها مرّةً** — (قياسُ ٢٠٢٦-٠٨-٢٣: تسعةُ أصواتٍ
        // على جهاز المالك، **والمختارُ منها مصريٌّ مضغوط**.)
        //
        // **ولا يُختار صوتٌ بلا معرفةِ ما تُرك** — والشكوى كانت
        // «أريد صوتَ غوغل ماب»، **وجوابُها في هذه القائمة.**
        for (v in arabic) {
            android.util.Log.i(
                "RahalGo/voice",
                "  متاح: ${v.name} درجة=${v.quality} " +
                    "شبكة=${v.isNetworkConnectionRequired} بلد=${v.locale?.country}",
            )
        }
        if (arabic.isEmpty()) return
        // ══════════════════════════════════════════════════════════════
        // **واللهجةُ قبل كلّ شيء — ثمّ العصبيُّ قبل المضغوط**
        // ══════════════════════════════════════════════════════════════
        //
        // (شكوى المالك ٢٠٢٦-٠٨-٢٣: «الصوتُ العربيُّ لم يعجبني… أريد
        //  صوتَ غوغل ماب».)
        //
        // # وأوّلُ ترتيبٍ كتبتُه وقع على أسوأ تركيبةٍ ممكنة
        //
        // **قِيس على جهاز المالك: تسعةُ أصواتٍ عربيّةٍ كلُّها بدرجة
        // ٤٠٠.** فرتّبتُ بالدرجة أوّلاً **فتساوت كلُّها**، ثمّ رجّحتُ
        // المحلّيَّ على الشبكيّ — **فوقع الاختيارُ على `arz-local`:
        // مصريٍّ ومضغوط.**
        //
        // **والدرجةُ لا تفرّق بينها** — الفرقُ في اللهجة وفي كون الصوت
        // عصبيّاً كاملاً أو نسخةً مضغوطةً منه.
        //
        // # ورموزُ غوغل للّهجات
        //
        //	arc  ←  شاميّ    ← **وهو لهجةُ سائقنا**
        //	arz  ←  مصريّ
        //	ard  ·  are      ←  آخران
        //
        // # والشبكيُّ هو صوتُ غوغل ماب نفسُه
        //
        // **`-network` أصواتُ غوغل العصبيّةُ الكاملة**، و`-local` نسخٌ
        // مضغوطةٌ منها تعمل بلا إنترنت. **وغوغل ماب يستعمل الشبكيَّ
        // حين تتوفّر الشبكة.**
        //
        // **ولا يُترك السائقُ بلا صوتٍ حين تنقطع**: المحلّيُّ من
        // اللهجة نفسِها يليه في الترتيب مباشرةً، **فيرتدّ إليه المحرّكُ
        // وحدَه.**
        val best = arabic
            .sortedWith(
                compareBy<android.speech.tts.Voice> { dialectRank(it.name) }
                    .thenBy { if (it.isNetworkConnectionRequired) 0 else 1 }
                    .thenByDescending { it.quality },
            )
            .first()
        val ok = runCatching { engine.setVoice(best) }.getOrDefault(TextToSpeech.ERROR)
        selectedVoice = "${best.name} درجة=${best.quality} شبكة=${best.isNetworkConnectionRequired}"
        android.util.Log.i(
            "RahalGo/voice",
            "أصواتٌ عربيّة=${arabic.size} · اختير: $selectedVoice · نتيجة=$ok",
        )
        // **وسرعةٌ أهدأُ قليلاً** — **والتعليمةُ تُقال مرّةً وهو يقود**،
        // فمن لم يلحقها لم يعد ليسمعها.
        runCatching { engine.setSpeechRate(0.95f) }
    }

    /**
     * **ترتيبُ اللهجات — الشاميُّ أوّلاً.**
     *
     * **ويُقرأ من اسم الصوت لا من `locale.country`** — **وقِيس أنّ
     * البلدَ فارغٌ في التسعة كلِّها** على جهاز المالك (٢٠٢٦-٠٨-٢٣):
     * `ar-xa` نطاقٌ عامٌّ لا بلدَ فيه، **واللهجةُ في اللاحقة وحدَها.**
     */
    private fun dialectRank(name: String): Int = when {
        name.contains("-arc") -> 0 // شاميّ
        name.contains("-ard") -> 1
        name.contains("-are") -> 2
        name.contains("-arz") -> 3 // مصريّ
        else -> 4
    }

    /** **أيُّ صوتٍ يتكلّم** — يُقرأ في التشخيص لا في المنطق. */
    var selectedVoice: String = ""
        private set

    override fun speak(id: String, text: String, flush: Boolean, done: (Boolean) -> Unit) {
        val engine = tts
        if (!available || engine == null) {
            done(false)
            return
        }
        // **ولا يُنطَق فوق مكالمة** — أمرُ المالك، البند ١٧.
        if (!requestFocus()) {
            done(false)
            return
        }
        callbacks[id] = done
        val params = Bundle().apply {
            putInt(TextToSpeech.Engine.KEY_PARAM_STREAM, AudioManager.STREAM_MUSIC)
        }
        val mode = if (flush) TextToSpeech.QUEUE_FLUSH else TextToSpeech.QUEUE_ADD
        val rc = runCatching { engine.speak(text, mode, params, id) }
            .getOrDefault(TextToSpeech.ERROR)
        if (rc != TextToSpeech.SUCCESS) {
            callbacks.remove(id)
            abandonFocus()
            done(false)
        }
    }

    override fun stop() {
        runCatching { tts?.stop() }
        callbacks.clear()
        abandonFocus()
    }

    /**
     * **يُطفأ ويُحرَّر** — عند انتهاء الملاحة.
     *
     * **ومحرّكُ نطقٍ لا يُطفأ يبقى في الذاكرة** — وهو تسرّبٌ لا يظهر
     * إلّا في جهازٍ يسخن.
     */
    fun shutdown() {
        stop()
        runCatching { tts?.shutdown() }
        tts = null
        ready = false
        languageOk = false
    }

    // ══════════════════════════════════════════════════════════════════
    // **تركيزُ الصوت**
    // ══════════════════════════════════════════════════════════════════

    private fun requestFocus(): Boolean {
        val manager = audio ?: return false
        if (focus != null) return true
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val request = AudioFocusRequest.Builder(
                AudioManager.AUDIOFOCUS_GAIN_TRANSIENT_MAY_DUCK,
            ).setAudioAttributes(attributes).build()
            val granted = runCatching { manager.requestAudioFocus(request) }
                .getOrDefault(AudioManager.AUDIOFOCUS_REQUEST_FAILED)
            if (granted == AudioManager.AUDIOFOCUS_REQUEST_GRANTED) {
                focus = request
                true
            } else {
                false
            }
        } else {
            @Suppress("DEPRECATION")
            val granted = runCatching {
                manager.requestAudioFocus(
                    null,
                    AudioManager.STREAM_MUSIC,
                    AudioManager.AUDIOFOCUS_GAIN_TRANSIENT_MAY_DUCK,
                )
            }.getOrDefault(AudioManager.AUDIOFOCUS_REQUEST_FAILED)
            granted == AudioManager.AUDIOFOCUS_REQUEST_GRANTED
        }
    }

    private fun abandonFocus() {
        val manager = audio ?: return
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            focus?.let { runCatching { manager.abandonAudioFocusRequest(it) } }
            focus = null
        } else {
            @Suppress("DEPRECATION")
            runCatching { manager.abandonAudioFocus(null) }
        }
    }

    private val listener = object : UtteranceProgressListener() {
        override fun onStart(utteranceId: String?) = Unit

        override fun onDone(utteranceId: String?) = finish(utteranceId, true)

        @Deprecated("يُستدعى على النسخ القديمة")
        override fun onError(utteranceId: String?) = finish(utteranceId, false)

        override fun onError(utteranceId: String?, errorCode: Int) = finish(utteranceId, false)

        private fun finish(id: String?, ok: Boolean) {
            // **والتركيزُ يُحرَّر مع آخر جملة** — لا مع كلّ واحدةٍ في
            // طابورٍ متتابع: **الإفلاتُ والطلبُ في الملّي نفسِه يجعل
            // الموسيقى ترتفع وتنخفض نبضاً.**
            callbacks.remove(id)?.invoke(ok)
            if (callbacks.isEmpty()) abandonFocus()
        }
    }

    private companion object {
        const val TAG = "RahalGo/voice"
    }
}
