package com.rahalgo.shared.net

import java.io.InputStream
import java.io.OutputStream
import java.net.ServerSocket
import java.net.Socket
import java.security.MessageDigest
import java.util.Base64
import java.util.Collections
import java.util.concurrent.atomic.AtomicInteger
import kotlin.concurrent.thread

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مِسنَدُ `D19` — مقبسٌ حقيقيٌّ ومصافحةٌ حقيقيّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ولا مِسنَدَ وهميٌّ لعميل `Ktor`**: **العطبُ في ما يُعرَض على
 * السلك** — **أيُّ رمزٍ حملته المصافحة** — **ومِسنَدٌ يعترض النداء
 * يقيس نيّةَ الشيفرة لا فعلَها.**
 *
 * **فيُفتح `ServerSocket` حقيقيٌّ**: يقرأ سطرَ الطلب، **ويسجّل الرمزَ
 * الذي وصله**، ثمّ يردّ ما يُطلب منه — **٤٠١ أو ٤٠٣ أو ٥٠٣، أو مصافحةً
 * كاملةً بـ`101`** مع `Sec-WebSocket-Accept` محسوبةً كما تقول
 * `RFC 6455`.
 */
class FakeWs(
    /** **يقرّر ردَّ كلّ محاولةٍ من الرمز الذي حملته.** */
    private val decide: (token: String) -> String,
) {
    private val server = ServerSocket(0)

    /** **كلُّ رمزٍ عُرض على السلك، بترتيبه.** */
    val tokensSeen: MutableList<String> = Collections.synchronizedList(mutableListOf<String>())

    /** **الوصلاتُ الحيّةُ الآن** — **`D19-T9`: واحدةٌ لا اثنتان.** */
    val liveSockets = AtomicInteger(0)

    val url: String get() = "ws://127.0.0.1:${server.localPort}"

    private val accepted = mutableListOf<Socket>()

    init {
        thread(isDaemon = true) {
            while (!server.isClosed) {
                val s = try {
                    server.accept()
                } catch (e: Exception) {
                    return@thread
                }
                synchronized(accepted) { accepted += s }
                thread(isDaemon = true) { serve(s) }
            }
        }
    }

    private fun serve(s: Socket) {
        try {
            val input = s.getInputStream()
            val out = s.getOutputStream()
            val (line, headers) = readHead(input)
            val token = Regex("token=([^ &]*)").find(line)?.groupValues?.get(1).orEmpty()
            tokensSeen += token
            when (val verdict = decide(token)) {
                "OPEN" -> upgrade(s, out, headers)
                else -> {
                    out.write("HTTP/1.1 $verdict\r\nContent-Length: 0\r\nConnection: close\r\n\r\n".toByteArray())
                    out.flush()
                    s.close()
                }
            }
        } catch (e: Exception) {
            runCatching { s.close() }
        }
    }

    private fun upgrade(s: Socket, out: OutputStream, headers: Map<String, String>) {
        val key = headers["sec-websocket-key"].orEmpty()
        val accept = Base64.getEncoder().encodeToString(
            MessageDigest.getInstance("SHA-1")
                .digest((key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11").toByteArray()),
        )
        out.write(
            ("HTTP/1.1 101 Switching Protocols\r\n" +
                "Upgrade: websocket\r\nConnection: Upgrade\r\n" +
                "Sec-WebSocket-Accept: $accept\r\n\r\n").toByteArray(),
        )
        out.flush()
        liveSockets.incrementAndGet()
        // **وإطارٌ نصّيٌّ واحد** — **`D19-T10`: حدثٌ حقيقيٌّ بعد
        // التعافي، لا مجرّدُ مصافحةٍ نجحت.**
        val payload = """{"type":"order"}""".toByteArray()
        out.write(byteArrayOf(0x81.toByte(), payload.size.toByte()))
        out.write(payload)
        out.flush()
        // **وتبقى الوصلةُ مفتوحةً حتّى يُغلقها صاحبُها.**
        runCatching { while (s.getInputStream().read() != -1) Unit }
        liveSockets.decrementAndGet()
    }

    private fun readHead(input: InputStream): Pair<String, Map<String, String>> {
        val sb = StringBuilder()
        var last4 = 0
        while (true) {
            val c = input.read()
            if (c == -1) break
            sb.append(c.toChar())
            last4 = ((last4 shl 8) or c) and 0xFFFFFFFF.toInt()
            if (last4 == 0x0D0A0D0A) break
        }
        val lines = sb.toString().split("\r\n")
        val headers = lines.drop(1).mapNotNull {
            val i = it.indexOf(':')
            if (i <= 0) null else it.substring(0, i).trim().lowercase() to it.substring(i + 1).trim()
        }.toMap()
        return lines.firstOrNull().orEmpty() to headers
    }

    fun close() {
        runCatching { server.close() }
        synchronized(accepted) { accepted.forEach { runCatching { it.close() } } }
    }
}

/**
 * **مخزنُ جلسةٍ في الذاكرة** — **يحكي عقدَ `SessionStore` كاملاً.**
 */
class FakeSession(
    var access: String = "OLD",
    var refresh: String = "R1",
    var expires: Long = 0L,
) : SessionStore {
    val saves = AtomicInteger(0)
    override fun accessToken(): String = access
    override fun refreshToken(): String = refresh
    override fun accessExpiresAt(): Long = expires
    override fun save(access: String, refresh: String, accessExpiresAt: Long) {
        this.access = access
        this.refresh = refresh
        this.expires = accessExpiresAt
        saves.incrementAndGet()
    }
    override fun clear() {
        access = ""; refresh = ""; expires = 0L
    }
}
