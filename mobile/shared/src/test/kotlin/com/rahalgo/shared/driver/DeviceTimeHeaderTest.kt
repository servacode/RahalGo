package com.rahalgo.shared.driver

import com.rahalgo.shared.model.TrackPoint
import com.rahalgo.shared.net.ApiClient
import com.rahalgo.shared.net.SessionStore
import java.io.BufferedInputStream
import java.net.ServerSocket
import java.util.Collections
import kotlin.concurrent.thread
import kotlinx.coroutines.runBlocking
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ندائا الموقع يحملان ساعةَ الجهاز** (A6.4، نسخة السائق 0.9.0)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وقتُ النقطة من ساعة الهاتف** — وهاتفٌ ساعتُه متأخّرةٌ لا يُكتب موضعُه. **فترويسةُ
 * `X-Device-Time` تقول للمحرّك ساعتَه لحظةَ الإرسال**، ويصحّح الفرق (`server/driver_clock.go`).
 *
 * **ويُقاس على السلك لا بالنيّة** — مقبسٌ حقيقيٌّ يسجّل الترويسات (نهجُ `D19Harness`).
 */
class DeviceTimeHeaderTest {

    /** **خادمٌ صغيرٌ يسجّل ترويساتِ كلّ نداءٍ ويردّ نجاحاً.** */
    private class Wire {
        private val server = ServerSocket(0)
        val seen: MutableList<Pair<String, Map<String, String>>> =
            Collections.synchronizedList(mutableListOf())
        val url: String get() = "http://127.0.0.1:${server.localPort}"

        init {
            thread(isDaemon = true) {
                while (!server.isClosed) {
                    val s = try {
                        server.accept()
                    } catch (e: Exception) {
                        return@thread
                    }
                    thread(isDaemon = true) {
                        s.use { sock ->
                            val input = BufferedInputStream(sock.getInputStream())
                            val requestLine = readLine(input)
                            val headers = mutableMapOf<String, String>()
                            while (true) {
                                val line = readLine(input)
                                if (line.isEmpty()) break
                                val i = line.indexOf(':')
                                if (i > 0) headers[line.substring(0, i).trim().lowercase()] = line.substring(i + 1).trim()
                            }
                            val length = headers["content-length"]?.toIntOrNull() ?: 0
                            var read = 0
                            while (read < length) {
                                if (input.read() < 0) break
                                read++
                            }
                            seen += requestLine.split(' ').getOrElse(1) { "" } to headers
                            val body = """{"data":{"ok":true}}"""
                            sock.getOutputStream().write(
                                ("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n" +
                                    "Content-Length: ${body.length}\r\nConnection: close\r\n\r\n" + body)
                                    .toByteArray(),
                            )
                            sock.getOutputStream().flush()
                        }
                    }
                }
            }
        }

        private fun readLine(input: BufferedInputStream): String {
            val sb = StringBuilder()
            while (true) {
                val c = input.read()
                if (c < 0 || c == '\n'.code) break
                if (c != '\r'.code) sb.append(c.toChar())
            }
            return sb.toString()
        }

        fun close() = server.close()

        fun headersOf(path: String): Map<String, String> =
            synchronized(seen) { seen.last { it.first == path }.second }
    }

    private val wire = Wire()

    @After
    fun tearDown() = wire.close()

    private val session = object : SessionStore {
        override fun accessToken() = "access"
        override fun refreshToken() = "refresh"
        override fun accessExpiresAt() = 0L
        override fun save(access: String, refresh: String, accessExpiresAt: Long) = Unit
        override fun clear() = Unit
    }

    @Test
    fun `ندائا الموقع يحملان ساعة الجهاز وغيرهما لا`() = runBlocking {
        val driver = DriverApi(ApiClient(wire.url, "driver", session, version = 22))
        val before = System.currentTimeMillis()
        driver.sendLocation(35.95, 39.01, recordedAt = "2026-11-05T10:00:00Z")
        driver.sendBatch(listOf(TrackPoint(lat = 35.95, lng = 39.01, at = "2026-11-05T10:00:00Z")))
        driver.stageReport("order-1", "not_ready")
        val after = System.currentTimeMillis()

        for (path in listOf("/api/v1/driver/location", "/api/v1/driver/location/batch")) {
            val v = wire.headersOf(path)[ApiClient.DEVICE_TIME_HEADER.lowercase()]
            assertNotNull("$path بلا ساعة الجهاز — **فلا يُصحَّح وقتُ هاتفٍ ساعتُه خاطئة**", v)
            val at = java.time.Instant.parse(v).toEpochMilli()
            assertTrue("$path: ساعةُ الجهاز $v ليست لحظةَ الإرسال", at in before..after)
        }
        // **وما عداهما لا يحملها** — لا يُصحَّح بها شيءٌ آخر.
        assertNull(wire.headersOf("/api/v1/driver/orders/order-1/report")[ApiClient.DEVICE_TIME_HEADER.lowercase()])
        assertEquals("X-Device-Time", ApiClient.DEVICE_TIME_HEADER)
    }
}
