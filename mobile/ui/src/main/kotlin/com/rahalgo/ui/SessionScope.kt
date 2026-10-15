package com.rahalgo.ui

import android.app.Application
import androidx.lifecycle.HasDefaultViewModelProviderFactory
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.ViewModelStore
import androidx.lifecycle.ViewModelStoreOwner
import androidx.lifecycle.viewmodel.CreationExtras
import androidx.lifecycle.viewmodel.MutableCreationExtras

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مالكُ نماذج الجلسة — مخزنٌ يُفرَّغ مع الخروج**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (بلاغُ المالك ٢٠٢٦-١٠-١٦ — انظر `AuthViewModel.dropSessionScope`.)
 *
 * **و`viewModel()` تحته يبني كما يبني على النشاط** — بمصنعِ أندرويد الافتراضيّ
 * وبالتطبيق في الإضافات، **فكلُّ `AndroidViewModel(app)` يُبنى بلا سطرٍ جديدٍ في
 * شاشته.** والفرقُ وحدَه: **عمرُه عمرُ الجلسة.**
 */
class SessionScope(private val app: Application) :
    ViewModelStoreOwner,
    HasDefaultViewModelProviderFactory {

    override val viewModelStore: ViewModelStore = ViewModelStore()

    override val defaultViewModelProviderFactory: ViewModelProvider.Factory =
        ViewModelProvider.AndroidViewModelFactory.getInstance(app)

    override val defaultViewModelCreationExtras: CreationExtras
        get() = MutableCreationExtras().apply {
            set(ViewModelProvider.AndroidViewModelFactory.APPLICATION_KEY, app)
        }
}
