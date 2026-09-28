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
        setContent {
            MaterialTheme {
                ReplayLabScreen(model.controller)
            }
        }
    }
}
