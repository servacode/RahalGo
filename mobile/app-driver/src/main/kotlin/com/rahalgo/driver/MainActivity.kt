package com.rahalgo.driver

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier

/**
 * **الخطوة الأولى — لا شيء إلا أن يفتح.**
 *
 * (خطة البناء خطوة بخطوة، قرار المالك ٢٠٢٦-٠٨-١١.)
 *
 * **هدفها واحد: أن نتأكد أن سلسلة البناء والتثبيت تعمل كلها** — من
 * الشيفرة إلى الجهاز. **لا ثيم ولا خط ولا اتجاه** — تلك الخطوة الثانية.
 *
 * **ولماذا يُبدأ بشيء تافه**: لو وقع خلل في البناء أو التثبيت أردناه
 * معزولا الآن، **لا مخبّأ تحت شاشة دخول لم تعمل لعشرة أسباب محتملة.**
 */
class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent { FirstScreen() }
    }
}

@Composable
private fun FirstScreen() {
    MaterialTheme {
        Surface(modifier = Modifier.fillMaxSize()) {
            Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Text("رحّال غو", style = MaterialTheme.typography.headlineLarge)
            }
        }
    }
}
