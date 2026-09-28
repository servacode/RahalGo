package com.rahalgo.driver.replaylab

import android.app.Application
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.material3.MaterialTheme
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.ViewModelProvider

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نشاطُ مختبر الإعادة** — بناءُ التطوير وحدَه (`src/debug`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **غيرُ موجودٍ في بناء الإصدار البتّة**: الملفُّ في مجموعةِ مصادرِ التطوير
 * (`src/debug`)، والتسجيلُ في بيان التطوير وحدَه — **فلا يُصرَّف ولا يُعلَن
 * في الإنتاج.** والمتحكّمُ في `ViewModel` فتنجو الرحلةُ من دوران الشاشة
 * والخلفيّة/المقدّمة؛ وإعادةُ الفتحِ تعيد البناءَ إلى مشهدٍ حتميّ.
 */
class ReplayLabActivity : ComponentActivity() {

    internal class Model(app: Application) : AndroidViewModel(app) {
        val controller = ReplayLabController(app)
        override fun onCleared() {
            controller.dispose()
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val model = ViewModelProvider(this)[Model::class.java]
        val c = model.controller

        // **تشغيلٌ آليٌّ لاختبار QA** (اختياريّ): يُطلَق بمُدخَلاتِ النيّة
        //   --ez rlab_autostart true --es rlab_scenario normal|reroute
        // فيمكن قياسُ السلوكِ عبر logcat (شاشةُ الخريطة تُصعّب قراءةَ الواجهة).
        val scId = intent?.getStringExtra("rlab_scenario")
        if (scId != null) c.scenarios().firstOrNull { it.id == scId }?.let { c.selectScenario(it) }
        intent?.getIntExtra("rlab_speed", 0)?.takeIf { it > 0 }?.let { c.changeSpeed(it) }
        val autostart = intent?.getBooleanExtra("rlab_autostart", false) == true

        setContent {
            MaterialTheme {
                ReplayLabScreen(c)
            }
        }
        if (savedInstanceState == null && autostart) c.start()
    }
}
