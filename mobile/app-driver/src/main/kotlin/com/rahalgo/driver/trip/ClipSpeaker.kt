package com.rahalgo.driver.trip

import android.content.Context
import android.media.AudioAttributes
import android.media.AudioFocusRequest
import android.media.AudioManager
import android.media.MediaPlayer
import android.os.Build
import android.util.Log
import com.rahalgo.navigation.Speaker

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الناطقُ بمقاطعَ مسجَّلة — صوتُ `rahalgo2` لا صوتُ الجهاز**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٢٤: «نعتمد الصوت».)
 *
 * # ولماذا تُرك النطقُ الآليّ
 *
 * **صوتُ الجهاز يتبدّل من هاتفٍ إلى هاتف**: محرّكٌ عربيٌّ هنا وآخرُ
 * هناك ولا محرّكَ ثالث. **فسائقان يسمعان تعليمتين مختلفتين** — ولا
 * نملك من ذلك شيئاً. **والمسجَّلُ واحدٌ عند الجميع.**
 *
 * # والاحتياطُ يبقى ويُعدّ
 *
 * **إن غاب مقطعٌ نُطق آليّاً** — **وصمتٌ عند منعطفٍ أسوأُ من صوتين
 * مختلفين.** و[fallbacks] يعدّها، **فإن كانت صفراً في رحلةٍ كاملةٍ
 * حُذف المحرّكُ الآليُّ كلُّه** ونقص حجمُ التطبيق.
 *
 * # وحذارِ من تقليم الموارد
 *
 * **المقاطعُ تُنادى باسمها في وقت التشغيل** (`getIdentifier`) **فلا
 * يراها المُقلِّم مستعمَلةً فيحذفها** — والتطبيقُ يُبنى ناجحاً ويخرس
 * في الشارع. **ولذلك `res/raw/keep.xml`.**
 */
class ClipSpeaker(
    context: Context,
    /** **الاحتياطُ الآليّ** — وفارغٌ يعني «اصمت إن غاب المقطع». */
    private val fallback: Speaker? = null,
) : Speaker {

    private val app = context.applicationContext
    private val audio = app.getSystemService(Context.AUDIO_SERVICE) as? AudioManager

    private val attributes = AudioAttributes.Builder()
        .setUsage(AudioAttributes.USAGE_ASSISTANCE_NAVIGATION_GUIDANCE)
        .setContentType(AudioAttributes.CONTENT_TYPE_SPEECH)
        .build()

    private var focus: AudioFocusRequest? = null
    private var player: MediaPlayer? = null

    /** **كم مرّةً لجأنا إلى النطق الآليّ** — يُقرأ في تقرير الرحلة. */
    var fallbacks = 0
        private set

    /** **وكم مقطعاً شُغّل** — كذلك. */
    var played = 0
        private set

    /** **وآخرُ ما تعذّر** — للتشخيص لا للمنطق. */
    var lastMiss: String? = null
        private set

    /**
     * **متاحٌ دائماً** — المقاطعُ داخل التطبيق، **ولا تنتظر تنزيلاً ولا
     * محرّكاً ولا لغةً مثبَّتة.**
     */
    override val available: Boolean get() = true

    override fun speak(
        id: String,
        text: String,
        clip: String?,
        flush: Boolean,
        done: (Boolean) -> Unit,
    ) = speak(id, text, clip, null, flush, done)

    /**
     * ══════════════════════════════════════════════════════════════════
     * **ويقول لاحقةً بعد الجملة إن وُجدت**
     * ══════════════════════════════════════════════════════════════════
     *
     * «انعطف يميناً» ← «ثمّ انعطف يساراً مباشرةً».
     *
     * **وجملتان تامّتان تتلوان إحداهما الأخرى** — لا جملةٌ مركَّبةٌ من
     * قِطَع. **والوقفةُ بينهما طبيعيّةٌ يفعلها المتكلّم**، وهي عينُ ما
     * تفعله خرائطُ غوغل عند المناورتين المتقاربتين.
     *
     * **وبؤرةُ الصوت لا تُترك بينهما** — **ومن تركها استعادت الموسيقى
     * صوتَها ثمّ خُفضت ثانيةً**، فيُسمع نشازٌ في نصف ثانية.
     */
    fun speak(
        id: String,
        text: String,
        clip: String?,
        thenClip: String?,
        flush: Boolean,
        done: (Boolean) -> Unit,
    ) {
        val res = clip?.let { resourceFor(it) } ?: 0
        if (res == 0) {
            lastMiss = clip
            fallbacks++
            Log.w("RahalGo/voice", "لا مقطعَ باسم «$clip» — نطقٌ آليّ: $text")
            val alt = fallback
            if (alt == null || !alt.available) {
                done(false)
                return
            }
            alt.speak(id, text, null, flush, done)
            return
        }
        // **ولا يُنطَق فوق مكالمة** — أمرُ المالك، البند ١٧.
        if (!requestFocus()) {
            done(false)
            return
        }
        // **والقطعُ قرارُ المنسّق لا قرارُنا** — فإن قال `flush` قطعنا،
        // **وإلّا صمتنا** حتّى ينتهي ما يُقال. والمنسّقُ لا يسلّم اثنتين
        // معاً أصلاً.
        val busy = player
        if (busy != null) {
            if (!flush) {
                done(false)
                return
            }
            release()
        }
        val mp = runCatching {
            MediaPlayer.create(app, res)?.apply { setAudioAttributes(attributes) }
        }.getOrNull()
        if (mp == null) {
            lastMiss = clip
            fallbacks++
            abandonFocus()
            Log.w("RahalGo/voice", "تعذّر تشغيلُ «$clip»")
            done(false)
            return
        }
        player = mp
        // **ويُحرَّر في الحالين** — **ومشغّلٌ لا يُحرَّر يبتلع مقبضاً
        // لكلّ تعليمة**، وبعد رحلةٍ طويلةٍ لا يبقى مقبض.
        mp.setOnCompletionListener {
            release()
            played++
            // **ويُسجَّل ما نُطق لا ما تعذّر وحدَه** — **ورحلةٌ صامتةٌ
            // بلا سطرٍ واحدٍ لا يُعرف أصمتت لأنّ الصوتَ مكتومٌ أم لأنّ
            // التعليماتِ لم تُولَّد** (وقع ٢٠٢٦-٠٨-٢٤ فضاع وقت).
            Log.i("RahalGo/voice", "نُطق «$clip» ($played) — $text")
            val next = thenClip?.let { resourceFor(it) } ?: 0
            if (next == 0) {
                abandonFocus()
                done(true)
                return@setOnCompletionListener
            }
            // **واللاحقةُ بلا `flush`** — الأولى انتهت أصلاً، **والبؤرةُ
            // ممسوكةٌ فلا تُطلب ثانية.**
            playInto(next, thenClip) {
                abandonFocus()
                done(true)
            }
        }
        mp.setOnErrorListener { _, what, extra ->
            release()
            abandonFocus()
            Log.w("RahalGo/voice", "عطبٌ في «$clip»: $what/$extra")
            done(false)
            true
        }
        runCatching { mp.start() }.onFailure {
            release()
            abandonFocus()
            done(false)
        }
    }

    /** **يشغّل مورداً ويُنهي** — لا بؤرةَ يطلب ولا يترك. */
    private fun playInto(res: Int, name: String?, finish: () -> Unit) {
        val mp = runCatching {
            MediaPlayer.create(app, res)?.apply { setAudioAttributes(attributes) }
        }.getOrNull()
        if (mp == null) {
            finish()
            return
        }
        player = mp
        mp.setOnCompletionListener {
            release()
            played++
            Log.i("RahalGo/voice", "نُطق «$name» ($played)")
            finish()
        }
        mp.setOnErrorListener { _, _, _ ->
            release()
            finish()
            true
        }
        runCatching { mp.start() }.onFailure {
            release()
            finish()
        }
    }

    override fun stop() {
        release()
        fallback?.stop()
        abandonFocus()
    }

    private fun release() {
        val mp = player ?: return
        player = null
        runCatching { mp.stop() }
        runCatching { mp.release() }
    }

    /**
     * **اسمٌ نصّيٌّ إلى مورد.**
     *
     * **ويُذاكَر** — `getIdentifier` تفتّش جدولَ الموارد كلَّه، **وتُنادى
     * عند كلّ تعليمة**، والسائقُ يسمع مئاتٍ في نوبته.
     */
    private fun resourceFor(clip: String): Int = cache.getOrPut(clip) {
        runCatching { app.resources.getIdentifier(clip, "raw", app.packageName) }.getOrDefault(0)
    }

    private val cache = HashMap<String, Int>()

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
}
