package com.rahalgo.ui.menu

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.setValue

/**
 * **حدُّ اسم الصنف كما يقوله المحرّك** (قرارُ المالك ٢٠٢٦-١٠-١٤: ٣٠ حرفاً).
 *
 * يُكتب من ردّ المتاجر (المتجر) ومن «أنا» (المندوب)، **ويقرؤه المحرّرُ المشترك** —
 * فيتبدّل من اللوحة بلا نسخةٍ جديدة. **والثلاثون حتى يصل الردّ.**
 */
object ItemLimits {
    var nameMax by mutableIntStateOf(30)
}
