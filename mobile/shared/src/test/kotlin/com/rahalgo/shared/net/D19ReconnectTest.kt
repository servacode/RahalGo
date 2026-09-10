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
 * **`D19` — إعادةُ الوصل برمزٍ منتهٍ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والمقيسُ ما عُرض على السلك** — لا ما نوت الشيفرةُ أن تعرضه.
 */
class D19ReconnectTest {

    private val hour = 3_600_000L

    // **وترتيبُ JUnit يضع الرسالةَ أوّلاً** — فيُلَفّ ليُقرأ الفحص.
    private fun eq(expected: Any?, actual: Any?, msg: String = "") =
        org.junit.Assert.assertEquals(msg, expected, actual)

    private fun ok(cond: Boolean, msg: String = "") =
        org.junit.Assert.assertTrue(msg, cond)

    private fun scope() = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    /** **ينتظر حتّى يصدُق الشرطُ أو تنفد المهلة** — ولا ينام جزافاً. */
    private fun waitFor(ms: Long = 5_000, cond: () -> Boolean): Boolean {
        val end = System.currentTimeMillis() + ms
        while (System.currentTimeMillis() < end) {
            if (cond()) return true
            Thread.sleep(5)
        }
        return cond()
    }

    private fun liveSocket(
        srv: FakeWs,
        auth: RealtimeAuth,
    ) = LiveSocket(srv.url, "driver", auth, firstRetryMs = 10L, maxRetryMs = 20L)

    // ══════════════════════════════════════════════════════════════════
    // **`D19-T2` · رمزٌ منتهٍ وعائلةٌ سليمة ⇒ تجديدٌ واحدٌ ثمّ وصل**
    // ══════════════════════════════════════════════════════════════════
    @Test
    fun t2_expiredAccessRefreshesOnceAndConnects() {
        val session = FakeSession(access = "OLD", expires = System.currentTimeMillis() - hour)
        val refreshes = AtomicInteger(0)
        val srv = FakeWs { if (it == "NEW") "OPEN" else "401 Unauthorized" }
        val auth = RealtimeAuth(session, refresher = {
            refreshes.incrementAndGet()
            session.save("NEW", "R2", System.currentTimeMillis() + hour)
        })
        val sc = scope()
        var connected = false
        val events = AtomicInteger(0)
        liveSocket(srv, auth).start(sc, onState = { if (it) connected = true }) { events.incrementAndGet() }

        ok(waitFor { connected }, "**لم تُفتح وصلةٌ بعد انتهاء الرمز** — وذاك `D19`")
        ok(waitFor { events.get() >= 1 }, "**وصلةٌ فُتحت ولا حدثَ يصل** — `D19-T10`")

        sc.cancel()
        srv.close()
        eq(1, refreshes.get(), "**التجديدُ مرّةً واحدةً لا أكثر**")
        eq("NEW", session.access)
    }

    // ══════════════════════════════════════════════════════════════════
    // **`D19-T3` · والمصافحةُ تحمل الجديدَ لا الملتقَطَ القديم**
    // ══════════════════════════════════════════════════════════════════
    @Test
    fun t3_handshakeCarriesNewTokenNotStaleCapture() {
        val session = FakeSession(access = "OLD", expires = System.currentTimeMillis() - hour)
        val srv = FakeWs { if (it == "NEW") "OPEN" else "401 Unauthorized" }
        val auth = RealtimeAuth(session, refresher = {
            session.save("NEW", "R2", System.currentTimeMillis() + hour)
        })
        val sc = scope()
        var connected = false
        liveSocket(srv, auth).start(sc, onState = { if (it) connected = true }) {}
        ok(waitFor { connected })
        sc.cancel(); srv.close()

        val seen = srv.tokensSeen.toList()
        ok(seen.isNotEmpty(), "**لم يصل الخادمَ رمزٌ أصلاً**")
        ok(
            seen.contains("NEW"),
            "**الرمزُ الجديدُ لم يُعرَض على السلك** — عُرض: $seen",
        )
        // **وأوّلُ مصافحةٍ لا آخرُها** — **والفرقُ جوهريّ**:
        // **حلقةٌ تلتقط القديمَ تشفى في الجولة التالية** (تُجدّد ثمّ
        // تقرأ الجديد) — **فالنظرُ إلى الآخِر يُخضِر العطبَ.**
        //
        // **والرمزُ منتهٍ قبل أوّل محاولة** — **فالصوابُ أن تحمل
        // المصافحةُ الأولى الجديدَ، ولا تكون ثانيةٌ أصلاً.**
        eq(
            listOf("NEW"), seen,
            "**عُرض القديمُ على السلك قبل الجديد** — عُرض: $seen",
        )
    }

    // ══════════════════════════════════════════════════════════════════
    // **`D19-T1` · وانقطاعُ نقلٍ برمزٍ صالحٍ لا يُجدَّد له**
    // ══════════════════════════════════════════════════════════════════
    @Test
    fun t1_ordinaryTransportDropDoesNotRefresh() {
        val session = FakeSession(access = "GOOD", expires = System.currentTimeMillis() + hour)
        val refreshes = AtomicInteger(0)
        val attempts = AtomicInteger(0)
        // **يُغلق أوّلَ محاولتين إغلاقَ نقلٍ ثمّ يفتح** — انقطاعٌ لا رفضُ سلطة.
        val srv = FakeWs { attempts.incrementAndGet(); if (attempts.get() > 2) "OPEN" else "502 Bad Gateway" }
        val auth = RealtimeAuth(session, refresher = { refreshes.incrementAndGet() })
        val sc = scope()
        var connected = false
        liveSocket(srv, auth).start(sc, onState = { if (it) connected = true }) {}
        ok(waitFor { connected }, "**لم يُعَد الوصلُ بعد انقطاع نقل**")
        sc.cancel(); srv.close()

        eq(
            0, refreshes.get(),
            "**جُدّد الرمزُ عند انقطاعِ نقلٍ والرمزُ صالح** — " +
                "**وكلُّ تدويرٍ فرصةُ سباق**",
        )
    }

    // ══════════════════════════════════════════════════════════════════
    // **`D19-T4`/`T5` · مُبطَلٌ ومحظورٌ ⇒ لا دورانَ على بابِ التجديد**
    // ══════════════════════════════════════════════════════════════════
    //
    // **ورمزُهما لم ينتهِ** — **فلا سببَ للتجديد أصلاً**: تُردّ
    // المصافحةُ وتتضاعف المهلة.
    @Test
    fun t4t5_revokedAndBlockedDoNotHammerRefresh() {
        for (verdict in listOf("401 Unauthorized", "403 Forbidden")) {
            val session = FakeSession(access = "GOOD", expires = System.currentTimeMillis() + hour)
            val refreshes = AtomicInteger(0)
            val srv = FakeWs { verdict }
            val auth = RealtimeAuth(session, refresher = { refreshes.incrementAndGet() })
            val sc = scope()
            liveSocket(srv, auth).start(sc) {}
            ok(waitFor { srv.tokensSeen.size >= 5 }, "**لم تُعَد المحاولةُ أصلاً** ($verdict)")
            sc.cancel(); srv.close()
            eq(
                0, refreshes.get(),
                "**طُرق بابُ التجديد لحسابٍ مرفوضٍ رمزُه صالح** ($verdict)",
            )
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **`D19-T6` · وسلطةٌ متعذّرةٌ (٥٠٣) لا تمحو اعتماداً**
    // ══════════════════════════════════════════════════════════════════
    @Test
    fun t6_authorityUnavailableKeepsCredentials() {
        val session = FakeSession(access = "GOOD", refresh = "R1", expires = System.currentTimeMillis() + hour)
        val srv = FakeWs { "503 Service Unavailable" }
        val auth = RealtimeAuth(session, refresher = {})
        val sc = scope()
        liveSocket(srv, auth).start(sc) {}
        ok(waitFor { srv.tokensSeen.size >= 5 })
        sc.cancel(); srv.close()

        eq("GOOD", session.access, "**٥٠٣ محت رمزَ الوصول** — و`R16` تقول تعذّرٌ لا إبطال")
        eq("R1", session.refresh, "**٥٠٣ محت عائلةَ التجديد**")
    }

    // ══════════════════════════════════════════════════════════════════
    // **`D19-T9` · والتعافي يترك وصلةً واحدة**
    // ══════════════════════════════════════════════════════════════════
    @Test
    fun t9_recoveryLeavesOneSocket() {
        val session = FakeSession(access = "OLD", expires = System.currentTimeMillis() - hour)
        val srv = FakeWs { if (it == "NEW") "OPEN" else "401 Unauthorized" }
        val auth = RealtimeAuth(session, refresher = {
            session.save("NEW", "R2", System.currentTimeMillis() + hour)
        })
        val sc = scope()
        val live = liveSocket(srv, auth)
        var connected = false
        live.start(sc, onState = { if (it) connected = true }) {}
        // **والبدءُ مرّتين لا يفتح وصلتين** — عقدُ `start`.
        live.start(sc) {}
        ok(waitFor { connected })
        Thread.sleep(200)
        val n = srv.liveSockets.get()
        sc.cancel(); srv.close()
        eq(1, n, "**وصلتان حيّتان بعد التعافي**")
    }

    // ══════════════════════════════════════════════════════════════════
    // **`D19-T11` · ولا فعلَ من صاحب الجهاز**
    // ══════════════════════════════════════════════════════════════════
    //
    // **ولا يُنادى شيءٌ بين البدء والتعافي** — **لا خروجٌ ولا دخولٌ
    // ولا سحبٌ للتحديث**: تُقاس الحلقةُ وحدَها.
    @Test
    fun t11_noManualActionNeeded() {
        val session = FakeSession(access = "OLD", expires = System.currentTimeMillis() - hour)
        val srv = FakeWs { if (it == "NEW") "OPEN" else "401 Unauthorized" }
        val auth = RealtimeAuth(session, refresher = {
            session.save("NEW", "R2", System.currentTimeMillis() + hour)
        })
        val sc = scope()
        val events = AtomicInteger(0)
        liveSocket(srv, auth).start(sc) { events.incrementAndGet() }
        val ok = waitFor { events.get() >= 1 }
        sc.cancel(); srv.close()
        ok(ok, "**لزِم فعلٌ من صاحب الجهاز ليعود البثّ**")
    }

    // ══════════════════════════════════════════════════════════════════
    // **`D19-T8` · ومحاولاتٌ متزامنةٌ لا تُدوّر التجديدَ مرّتين**
    // ══════════════════════════════════════════════════════════════════
    //
    // **والقفلُ في `ApiClient.refresh` نفسِه** — **وهذا يقيس أنّ
    // `RealtimeAuth` لا تلتفّ عليه**: من وجد الرمزَ قد جُدّد لا يجدّد.
    @Test
    fun t8_concurrentTriggersRefreshOnce() = runBlocking<Unit> {
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
        val jobs = (1..16).map {
            this@runBlocking.async(Dispatchers.IO) { auth.tokenForConnect() }
        }
        val tokens = jobs.map { it.await() }
        eq(1, calls.get(), "**دُوّر رمزُ التجديد أكثرَ من مرّة**")
        ok(tokens.all { it == "NEW" }, "**منتظِرٌ خرج برمزٍ قديم**: $tokens")
    }
}
