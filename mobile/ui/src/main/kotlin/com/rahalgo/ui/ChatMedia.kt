package com.rahalgo.ui

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.media.AudioAttributes
import android.media.MediaPlayer
import android.media.MediaRecorder
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.layout.height
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import com.rahalgo.design.Rahal
import com.rahalgo.shared.model.ChatAttachment
import kotlinx.coroutines.delay

/**
 * ══════════════════════════════════════════════════════════════════════
 * **صورٌ ورسائلُ صوتيّةٌ في حديث الطلب** (قرارُ المالك ٢٠٢٦-١١-١٢)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **عدّةٌ واحدةٌ للزبون والسائق** — زرُّ الصورة (كاميرا أو معرض)، وزرُّ المايك (اضغط مطوّلاً
 * وسجّل، ارفع إصبعك فيُرسَل، اسحب بعيداً فيُلغى، وحدُّه دقيقة)، والفقاعتان.
 *
 * **وإذنُ المايك يُطلب عند أوّل لمسةٍ للزرّ وحدَها** — لا عند فتح التطبيق.
 */

/** **أطولُ رسالةٍ صوتيّة** — يطابق حدَّ المحرّك. */
const val CHAT_VOICE_MAX_MS = 60_000L

/** **أقصرُ تسجيلٍ يُرسَل** — لمسةٌ عابرةٌ ليست رسالة. */
const val CHAT_VOICE_MIN_MS = 700L

/** **«0:12»** — دقائقُ وثوانٍ (الأرقامُ لاتينيّةٌ كسائر المنصّة). */
fun voiceDuration(ms: Long): String {
    val s = (ms.coerceAtLeast(0) + 500) / 1000
    return "" + (s / 60) + ":" + (s % 60).toString().padStart(2, '0')
}

/** **حالُ التسجيل** — يُقرأ في سطر الكتابة ليقول «جارٍ التسجيل». */
class ChatVoiceState {
    var recording by mutableStateOf(false)
        internal set
    var elapsedMs by mutableLongStateOf(0L)
        internal set
    /** **ملاحظةٌ قصيرة** — أُلغي، قصير، رُفض الإذن. وفارغٌ لا شيء. */
    var note by mutableStateOf("")
        internal set
}

@Composable
fun rememberChatVoiceState(): ChatVoiceState = remember { ChatVoiceState() }

/** **سطرُ التسجيل** — يحلّ محلَّ الملاحظة حين يسجّل. */
@Composable
fun ChatVoiceHint(state: ChatVoiceState, modifier: Modifier = Modifier) {
    val text = if (state.recording) {
        stringResource(R.string.chat_voice_recording, voiceDuration(state.elapsedMs))
    } else {
        state.note
    }
    if (text.isEmpty()) return
    Text(
        text = text,
        color = if (state.recording) Rahal.colors.danger else Rahal.colors.inkMuted,
        style = Rahal.type.caption,
        modifier = modifier,
    )
}

/**
 * **زرُّ الصورة** — يسأل: كاميرا أم معرض.
 *
 * **والكاميرا بإذنها إن أعلنه التطبيق** (`rememberCameraShot`) — تطبيقُ السائق يعلنه.
 */
@Composable
fun ChatPhotoButton(enabled: Boolean, onPhoto: (ByteArray) -> Unit) {
    val context = LocalContext.current
    var asking by remember { mutableStateOf(false) }
    val deliver by rememberUpdatedState(onPhoto)
    val camera = rememberCameraShot { deliver(it) }
    val gallery = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { uri ->
        if (uri != null) runCatching { readScaledImage(context, uri) }.getOrNull()?.let { deliver(it) }
    }
    IconButton(onClick = { asking = true }, enabled = enabled) {
        Icon(
            painter = painterResource(R.drawable.ic_photo),
            contentDescription = stringResource(R.string.chat_photo),
            tint = if (enabled) Rahal.colors.brand else Rahal.colors.inkMuted,
        )
    }
    if (asking) {
        RahalDialog(
            onDismissRequest = { asking = false },
            title = { Text(stringResource(R.string.photo_source)) },
            confirmButton = {
                DialogConfirmButton(onClick = {
                    asking = false
                    camera()
                }) { Text(stringResource(R.string.photo_camera)) }
            },
            dismissButton = {
                DialogDismissButton(onClick = {
                    asking = false
                    gallery.launch(PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly))
                }) { Text(stringResource(R.string.photo_gallery)) }
            },
        )
    }
}

/**
 * **زرُّ المايك** — اضغط مطوّلاً وسجّل، ارفع فيُرسَل، اسحب بعيداً فيُلغى، والدقيقةُ ترسله وحدَها.
 */
@Composable
fun ChatMicButton(state: ChatVoiceState, enabled: Boolean, onVoice: (ByteArray, Long) -> Unit) {
    val context = LocalContext.current
    val send by rememberUpdatedState(onVoice)
    val recorder = remember { VoiceRecorder(context.applicationContext) }
    DisposableEffect(recorder) { onDispose { recorder.cancel() } }
    val denied = stringResource(R.string.chat_voice_mic_denied)
    val cancelled = stringResource(R.string.chat_voice_cancelled)
    val tooShort = stringResource(R.string.chat_voice_too_short)
    val failed = stringResource(R.string.chat_voice_failed)
    val hold = stringResource(R.string.chat_voice_hold)
    val ask = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { ok ->
        state.note = if (ok) hold else denied
    }
    val cancelPx = with(LocalDensity.current) { 72.dp.toPx() }

    LaunchedEffect(state.recording) {
        while (state.recording) {
            state.elapsedMs = recorder.elapsedMs()
            delay(200)
        }
    }

    Box(
        Modifier
            .size(Rahal.size.touch)
            .clip(Rahal.shape.pill)
            .background(if (state.recording) Rahal.colors.danger else Rahal.colors.field)
            .semantics { contentDescription = hold }
            .pointerInput(enabled) {
                if (!enabled) return@pointerInput
                awaitEachGesture {
                    val down = awaitFirstDown()
                    // **والإذنُ عند أوّل لمسة** — لا يُسجَّل في هذه اللمسة، والتالية تسجّل.
                    if (ContextCompat.checkSelfPermission(context, Manifest.permission.RECORD_AUDIO) !=
                        PackageManager.PERMISSION_GRANTED
                    ) {
                        ask.launch(Manifest.permission.RECORD_AUDIO)
                        return@awaitEachGesture
                    }
                    if (!recorder.start()) {
                        state.note = failed
                        return@awaitEachGesture
                    }
                    state.note = ""
                    state.elapsedMs = 0
                    state.recording = true
                    var cancel = false
                    try {
                        while (true) {
                            val left = CHAT_VOICE_MAX_MS - recorder.elapsedMs()
                            if (left <= 0) break
                            val ev = withTimeoutOrNull(left) { awaitPointerEvent() } ?: break
                            val ch = ev.changes.firstOrNull { it.id == down.id } ?: break
                            if (!ch.pressed) break
                            if ((ch.position - down.position).getDistance() > cancelPx) {
                                cancel = true
                                break
                            }
                            ch.consume()
                        }
                    } catch (e: kotlinx.coroutines.CancellationException) {
                        cancel = true
                        recorder.cancel()
                        state.recording = false
                        throw e
                    }
                    state.recording = false
                    if (cancel) {
                        recorder.cancel()
                        state.note = cancelled
                        return@awaitEachGesture
                    }
                    val out = recorder.stop()
                    when {
                        out == null -> state.note = failed
                        out.second < CHAT_VOICE_MIN_MS -> state.note = tooShort
                        else -> send(out.first, out.second.coerceAtMost(CHAT_VOICE_MAX_MS))
                    }
                }
            },
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_mic),
            contentDescription = null,
            tint = when {
                state.recording -> Rahal.colors.onBrand
                enabled -> Rahal.colors.brand
                else -> Rahal.colors.inkMuted
            },
        )
    }
}

/**
 * **المسجّل** — AAC في MPEG-4 (`.m4a`) يُشغَّل في كلّ مكان، أحاديٌّ بـ٣٢ ك.ب/ث:
 * دقيقةٌ نحوُ ربع ميغا.
 */
internal class VoiceRecorder(private val context: Context) {
    private var rec: MediaRecorder? = null
    private var file: java.io.File? = null
    private var startedAt = 0L

    fun elapsedMs(): Long = if (rec == null) 0 else android.os.SystemClock.elapsedRealtime() - startedAt

    fun start(): Boolean {
        cancel()
        return runCatching {
            val dir = java.io.File(context.cacheDir, "voice").apply { mkdirs() }
            val f = java.io.File(dir, "voice_" + System.currentTimeMillis() + ".m4a")
            val r = if (Build.VERSION.SDK_INT >= 31) MediaRecorder(context) else @Suppress("DEPRECATION") MediaRecorder()
            r.setAudioSource(MediaRecorder.AudioSource.MIC)
            r.setOutputFormat(MediaRecorder.OutputFormat.MPEG_4)
            r.setAudioEncoder(MediaRecorder.AudioEncoder.AAC)
            r.setAudioChannels(1)
            r.setAudioSamplingRate(44_100)
            r.setAudioEncodingBitRate(32_000)
            r.setMaxDuration((CHAT_VOICE_MAX_MS + 500).toInt())
            r.setOutputFile(f.absolutePath)
            r.prepare()
            r.start()
            rec = r
            file = f
            startedAt = android.os.SystemClock.elapsedRealtime()
            true
        }.getOrElse {
            cancel()
            false
        }
    }

    /** **يوقف ويردّ البايتاتِ والمدّة** — وفارغٌ إن تعثّر. */
    fun stop(): Pair<ByteArray, Long>? {
        val r = rec ?: return null
        val f = file
        val ms = elapsedMs()
        rec = null
        file = null
        val ok = runCatching { r.stop() }.isSuccess
        runCatching { r.release() }
        val bytes = if (ok && f != null) runCatching { f.readBytes() }.getOrNull() else null
        runCatching { f?.delete() }
        return bytes?.takeIf { it.isNotEmpty() }?.let { it to ms }
    }

    fun cancel() {
        val r = rec
        rec = null
        if (r != null) {
            runCatching { r.stop() }
            runCatching { r.release() }
        }
        runCatching { file?.delete() }
        file = null
    }
}

/**
 * **مشغّلٌ واحدٌ للحديث كلِّه** — رسالةٌ تبدأ توقف ما قبلها.
 */
internal object ChatVoicePlayer {
    var playing by mutableStateOf("")
        private set
    var positionMs by mutableLongStateOf(0L)
        private set
    private var player: MediaPlayer? = null

    fun toggle(url: String) {
        if (playing == url) {
            stop()
            return
        }
        stop()
        val p = MediaPlayer()
        player = p
        playing = url
        positionMs = 0
        runCatching {
            p.setAudioAttributes(
                AudioAttributes.Builder()
                    .setContentType(AudioAttributes.CONTENT_TYPE_SPEECH)
                    .setUsage(AudioAttributes.USAGE_MEDIA)
                    .build(),
            )
            p.setDataSource(url)
            p.setOnPreparedListener { it.start() }
            p.setOnCompletionListener { if (player === it) stop() }
            p.setOnErrorListener { mp, _, _ ->
                if (player === mp) stop()
                true
            }
            p.prepareAsync()
        }.onFailure { stop() }
    }

    fun tick() {
        positionMs = runCatching { player?.currentPosition?.toLong() ?: 0L }.getOrDefault(0L)
    }

    fun stop() {
        val p = player
        player = null
        playing = ""
        positionMs = 0
        if (p != null) runCatching { p.release() }
    }
}

/** **المرفقُ داخل الفقاعة** — صورةٌ تُفتح بلمسة، أو صوتٌ بزرّ تشغيلٍ ومدّته. */
@Composable
fun ChatAttachmentView(att: ChatAttachment, mine: Boolean) {
    when (att.kind) {
        "photo" -> ChatPhoto(att)
        "audio" -> ChatVoice(att, mine)
    }
}

@Composable
private fun ChatPhoto(att: ChatAttachment) {
    val core = runCatching { AppCore.get() }.getOrNull()
    val full = core?.media(att.url) ?: att.url
    val small = core?.media(att.thumbUrl.ifEmpty { att.url }) ?: att.thumbUrl
    var open by remember { mutableStateOf(false) }
    val label = stringResource(R.string.chat_photo_open)
    RemoteImage(
        url = small,
        name = "",
        modifier = Modifier
            .width(200.dp)
            .height(150.dp)
            .clip(Rahal.shape.sm)
            .clickable(onClickLabel = label) { open = true },
    )
    if (open) ImageViewer(url = full, onClose = { open = false })
}

@Composable
private fun ChatVoice(att: ChatAttachment, mine: Boolean) {
    val url = runCatching { AppCore.get().media(att.url) }.getOrNull() ?: att.url
    val on = ChatVoicePlayer.playing == url
    LaunchedEffect(on) {
        while (on) {
            ChatVoicePlayer.tick()
            delay(250)
        }
    }
    DisposableEffect(url) { onDispose { if (ChatVoicePlayer.playing == url) ChatVoicePlayer.stop() } }
    val tint = if (mine) Rahal.colors.onBrand else Rahal.colors.brand
    Row(
        Modifier.widthIn(min = 160.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
    ) {
        IconButton(onClick = { ChatVoicePlayer.toggle(url) }) {
            Icon(
                painter = painterResource(if (on) R.drawable.ic_pause else R.drawable.ic_play_arrow),
                contentDescription = stringResource(if (on) R.string.chat_voice_pause else R.string.chat_voice_play),
                tint = tint,
            )
        }
        Column {
            Text(
                text = if (on) voiceDuration(ChatVoicePlayer.positionMs) + " / " + voiceDuration(att.durationMs)
                else voiceDuration(att.durationMs),
                color = if (mine) Rahal.colors.onBrand else Rahal.colors.ink,
                style = Rahal.type.bodyStrong,
            )
        }
    }
}
