package com.rahalgo.merchant.quickphoto

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.shared.merchant.MenuItemInput
import com.rahalgo.shared.merchant.MerchantApi
import com.rahalgo.ui.Flash
import com.rahalgo.ui.err
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.io.File

/**
 * Fast photo mode state (owner decision 2026-11-06): shoot, save, the next item appears.
 *
 * A saved photo is written to a cache file first, then uploaded in the background with retries;
 * if the upload keeps failing the file stays and the failure is shown, never dropped silently.
 * Leftover files are picked up again the next time the mode opens.
 */
class QuickPhotoState(
    private val scope: CoroutineScope,
    private val api: MerchantApi,
    private val storeId: () -> String,
    private val dir: File,
    /** Called when the screen closes so the menu can reload (new thumbnails). */
    private val onClosed: () -> Unit,
) {
    var open by mutableStateOf(false)
        private set
    var loading by mutableStateOf(false)
        private set
    var queue by mutableStateOf<QuickPhotoQueue?>(null)
        private set

    /** The photo just taken, shown in the preview before saving. */
    var captured by mutableStateOf<ByteArray?>(null)
        private set

    /** Uploads still running. */
    var uploading by mutableStateOf(0)
        private set

    /** Item ids whose upload failed; the file is kept for a retry. */
    var failed by mutableStateOf<Set<String>>(emptySet())
        private set

    private val skipped = mutableSetOf<String>()
    private val inFlight = mutableSetOf<String>()

    private fun file(id: String) = File(dir, "$id.jpg")

    fun show() {
        open = true
        captured = null
        loading = true
        scope.launch {
            // Leftovers from an earlier run that never uploaded.
            runCatching { dir.listFiles()?.filter { it.extension == "jpg" }.orEmpty() }.getOrDefault(emptyList())
                .forEach { f -> upload(f.nameWithoutExtension) }
            runCatching {
                val all = mutableListOf<com.rahalgo.shared.merchant.NoPhotoItem>()
                var after = ""
                do {
                    val page = api.noPhotoItems(storeId(), after)
                    all += page.items
                    after = page.nextAfter
                } while (after.isNotEmpty())
                all
            }.onSuccess { queue = QuickPhotoQueue.of(it, skipped, inFlight + failed) }
                .onFailure { Flash.fail(err(it)); open = false }
            loading = false
        }
    }

    fun close() {
        open = false
        captured = null
        queue = null
        onClosed()
    }

    fun capture(bytes: ByteArray) {
        captured = bytes
    }

    fun retake() {
        captured = null
    }

    fun skip() {
        val q = queue ?: return
        q.current?.let { skipped += it.id }
        captured = null
        queue = q.advance()
    }

    fun saveAndNext() {
        val q = queue ?: return
        val item = q.current ?: return
        val bytes = captured ?: return
        runCatching {
            dir.mkdirs()
            file(item.id).writeBytes(bytes)
        }.onFailure { Flash.fail(err(it)); return }
        captured = null
        queue = q.advance()
        upload(item.id)
    }

    fun retryFailed() {
        failed.toList().forEach { upload(it) }
    }

    private fun upload(id: String) {
        if (!inFlight.add(id)) return
        failed = failed - id
        uploading = inFlight.size
        scope.launch {
            val bytes = runCatching { file(id).readBytes() }.getOrNull()
            var ok = false
            if (bytes != null) {
                for (attempt in 0 until 3) {
                    ok = runCatching {
                        val ref = api.uploadItemImage("item.jpg", bytes)
                        api.updateItem(id, MenuItemInput(imageMediaId = ref.id))
                    }.isSuccess
                    if (ok) break
                    delay(2000L * (attempt + 1))
                }
            }
            if (ok || bytes == null) {
                runCatching { file(id).delete() }
            } else {
                failed = failed + id
            }
            inFlight.remove(id)
            uploading = inFlight.size
        }
    }
}
