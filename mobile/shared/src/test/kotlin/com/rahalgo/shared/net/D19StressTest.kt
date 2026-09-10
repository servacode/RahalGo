package com.rahalgo.shared.net

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.async
import kotlinx.coroutines.cancel
import kotlinx.coroutines.runBlocking
import java.util.concurrent.atomic.AtomicInteger
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تكرارُ `D19`** — **مرّةٌ واحدةٌ تُصادِف، والمئةُ تحكم**
 * ══════════════════════════════════════════════════════════════════════
 */
class D19StressTest {

    private val hour = 3_600_000L

    private fun eq(expected: Any?, actual: Any?, msg: String = "") =
        org.junit.Assert.assertEquals(msg, expected, actual)

    private fun ok(cond: Boolean, msg: String = "") =
        org.junit.Assert.assertTrue(msg, cond)

    private fun waitFor(ms: Long = 5_000, cond: () -> Boolean): Boolean {
        val end = System.currentTimeMillis() + ms
        while (System.currentTimeMillis() < end) {
            if (cond()) return true
            Thread.sleep(2)
        }
        return cond()
    }

    private fun live(srv: FakeWs, auth: RealtimeAuth) =
        LiveSocket(srv.url, "driver", auth, firstRetryMs = 5L, maxRetryMs = 10L)

    // ══════════════════════════════════════════════════════════════════
    // **تعافي الرمز المنتهي ×١٠٠ · والحدثُ يصل ×١٠٠**
    // ══════════════════════════════════════════════════════════════════
    @Test
    fun expiredRecovery100() {
        var staleLoops = 0
        var missedEvents = 0
        var extraSockets = 0
        repeat(100) {
            val session = FakeSession(access = "OLD", expires = System.currentTimeMillis() - hour)
            val srv = FakeWs { if (it == "NEW") "OPEN" else "401 Unauthorized" }
            val auth = RealtimeAuth(session, refresher = {
                session.save("NEW", "R2", System.currentTimeMillis() + hour)
            })
            val sc = CoroutineScope(SupervisorJob() + Dispatchers.IO)
            val events = AtomicInteger(0)
            live(srv, auth).start(sc) { events.incrementAndGet() }
            if (!waitFor(3_000) { events.get() >= 1 }) missedEvents++
            if (srv.tokensSeen.toList().lastOrNull() != "NEW") staleLoops++
            if (srv.liveSockets.get() > 1) extraSockets++
            sc.cancel(); srv.close()
        }
        eq(0, staleLoops, "**حلقاتٌ بقيت على الرمز القديم**")
        eq(0, missedEvents, "**تعافٍ بلا حدثٍ يصل**")
        eq(0, extraSockets, "**وصلاتٌ زائدة**")
    }

    // ══════════════════════════════════════════════════════════════════
    // **إعادةُ وصلٍ عاديّةٌ ×١٠٠ — ولا تجديدَ واحد**
    // ══════════════════════════════════════════════════════════════════
    @Test
    fun normalReconnect100() {
        var refreshed = 0
        var failedToConnect = 0
        repeat(100) {
            val session = FakeSession(access = "GOOD", expires = System.currentTimeMillis() + hour)
            val attempts = AtomicInteger(0)
            val srv = FakeWs { attempts.incrementAndGet(); if (attempts.get() > 1) "OPEN" else "502 Bad Gateway" }
            val calls = AtomicInteger(0)
            val auth = RealtimeAuth(session, refresher = { calls.incrementAndGet() })
            val sc = CoroutineScope(SupervisorJob() + Dispatchers.IO)
            var connected = false
            live(srv, auth).start(sc, onState = { if (it) connected = true }) {}
            if (!waitFor(3_000) { connected }) failedToConnect++
            if (calls.get() != 0) refreshed++
            sc.cancel(); srv.close()
        }
        eq(0, refreshed, "**تجديدٌ عند انقطاعِ نقلٍ برمزٍ صالح**")
        eq(0, failedToConnect, "**لم يُعَد الوصلُ بعد انقطاعِ نقل**")
    }

    // ══════════════════════════════════════════════════════════════════
    // **سلطةٌ متعذّرةٌ ×٥٠ · ومُبطَلٌ ×٥٠ — لا اعتمادَ يُمحى ولا تجديدَ يُطرَق**
    // ══════════════════════════════════════════════════════════════════
    @Test
    fun unavailable50AndRevoked50() {
        var lostCreds = 0
        var refreshedWrongly = 0
        for ((verdict, times) in listOf("503 Service Unavailable" to 50, "401 Unauthorized" to 50)) {
            repeat(times) {
                val session = FakeSession(access = "GOOD", refresh = "R1", expires = System.currentTimeMillis() + hour)
                val srv = FakeWs { verdict }
                val calls = AtomicInteger(0)
                val auth = RealtimeAuth(session, refresher = { calls.incrementAndGet() })
                val sc = CoroutineScope(SupervisorJob() + Dispatchers.IO)
                live(srv, auth).start(sc) {}
                waitFor(2_000) { srv.tokensSeen.size >= 3 }
                sc.cancel(); srv.close()
                if (session.access != "GOOD" || session.refresh != "R1") lostCreds++
                if (calls.get() != 0) refreshedWrongly++
            }
        }
        eq(0, lostCreds, "**اعتمادٌ مُحي عند تعذّرٍ أو رفض**")
        eq(0, refreshedWrongly, "**طُرق بابُ التجديد ورمزُ الوصول صالح**")
    }

    // ══════════════════════════════════════════════════════════════════
    // **تزامنُ التجديد ×١٠٠ — ولا تدويرَ مرّتين**
    // ══════════════════════════════════════════════════════════════════
    @Test
    fun concurrentRefresh100() = runBlocking<Unit> {
        var doubled = 0
        var staleWinner = 0
        repeat(100) {
            val session = FakeSession(access = "OLD", expires = System.currentTimeMillis() - hour)
            val calls = AtomicInteger(0)
            val gate = kotlinx.coroutines.sync.Mutex()
            val auth = RealtimeAuth(session, refresher = {
                val had = session.refreshToken()
                gate.lock()
                try {
                    if (session.refreshToken() == had) {
                        calls.incrementAndGet()
                        session.save("NEW", "R2", System.currentTimeMillis() + hour)
                    }
                } finally {
                    gate.unlock()
                }
            })
            val out = (1..8).map { this@runBlocking.async(Dispatchers.IO) { auth.tokenForConnect() } }
                .map { it.await() }
            if (calls.get() != 1) doubled++
            if (out.any { it != "NEW" }) staleWinner++
        }
        eq(0, doubled, "**دُوّر رمزُ التجديد أكثرَ من مرّة**")
        eq(0, staleWinner, "**منتظِرٌ خرج برمزٍ قديم**")
    }
}
