package com.rahalgo.customer.shamcash

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import com.rahalgo.customer.R
import com.rahalgo.customer.cart.Cart
import com.rahalgo.design.Rahal
import com.rahalgo.shared.customer.Bill
import com.rahalgo.shared.customer.ShamCashApi
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Note
import com.rahalgo.ui.PageHeader
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.Refresh
import com.rahalgo.ui.apiError
import com.rahalgo.ui.money
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/** Server error codes with their own wording. */
private fun shamErrors(): Map<String, Int> = mapOf(
    "shamcash_unavailable" to R.string.sham_unavailable,
    "shamcash_amount_out_of_range" to R.string.sham_out_of_range,
)

internal fun shamError(context: android.content.Context, e: Throwable): String =
    apiError(context, e, shamErrors())

private const val POLL_MS = 3_000L

/** Amount input, then the payment page in the real browser, then the status screen. */
@Composable
fun ShamTopupScreen(onBack: () -> Unit) {
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()
    val cfg = ShamNav.config
    var amount by rememberSaveable { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }
    val value = amount.trim().toLongOrNull() ?: 0L
    val min = cfg?.min ?: 0L
    val max = cfg?.max ?: 0L
    val inRange = value > 0 && value >= min && (max <= 0 || value <= max)

    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(Rahal.space.lg),
    ) {
        PageHeader(stringResource(R.string.sham_topup_title), onBack)
        Spacer(Modifier.height(Rahal.space.md))
        if (error.isNotEmpty()) Note(error, Rahal.colors.danger)
        OutlinedTextField(
            value = amount,
            onValueChange = { amount = it.filter(Char::isDigit).take(12) },
            label = { Text(stringResource(R.string.sham_amount)) },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth(),
        )
        if (cfg != null && (min > 0 || max > 0)) {
            Spacer(Modifier.height(Rahal.space.xs))
            Text(
                stringResource(R.string.sham_range, money(min), money(max)),
                color = if (value > 0 && !inRange) Rahal.colors.danger else Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        Spacer(Modifier.height(Rahal.space.lg))
        RahalButton(
            onClick = {
                if (busy) return@RahalButton
                busy = true
                error = ""
                scope.launch {
                    runCatching { ShamCashApi(AppCore.get().api).topup(value) }
                        .onSuccess { ShamNav.show(it.bill.id, it.paymentUrl.ifBlank { it.bill.paymentUrl }) }
                        .onFailure { error = shamError(ctx, it) }
                    busy = false
                }
            },
            enabled = !busy && inRange,
            modifier = Modifier.fillMaxWidth(),
        ) {
            if (busy) CircularProgressIndicator(Modifier.height(18.dp), strokeWidth = 2.dp)
            else Text(stringResource(R.string.sham_continue))
        }
    }
}


/**
 * Polls the bill every 3 s while it is pending, and at once when the app resumes
 * (the user comes back from the browser / the Sham Cash app).
 */
@Composable
fun ShamStatusScreen(
    billId: String,
    onWallet: () -> Unit,
    onOrders: () -> Unit,
    onCart: () -> Unit,
    onClose: () -> Unit,
) {
    val ctx = LocalContext.current
    var bill by remember(billId) { mutableStateOf<Bill?>(null) }
    var checkFailed by remember(billId) { mutableStateOf(false) }
    var browserFailed by remember(billId) { mutableStateOf(false) }
    var resumes by remember(billId) { mutableIntStateOf(0) }
    var fatal by remember(billId) { mutableStateOf("") }
    // Where the user can still go back to the payment page from.
    var payUrl by remember(billId) { mutableStateOf("") }

    // First time: open the payment page in the real browser.
    LaunchedEffect(billId) {
        ShamNav.takePendingUrl()?.let {
            payUrl = it
            if (!openPaymentPage(ctx, it)) browserFailed = true
        }
    }

    // Back from the browser: check now.
    val owner = androidx.compose.ui.platform.LocalLifecycleOwner.current
    DisposableEffect(owner, billId) {
        val obs = LifecycleEventObserver { _, e ->
            if (e == Lifecycle.Event.ON_RESUME) resumes++
        }
        owner.lifecycle.addObserver(obs)
        onDispose { owner.lifecycle.removeObserver(obs) }
    }

    // Poll: restarts at every resume; stops once the bill is final.
    LaunchedEffect(billId, resumes) {
        val api = ShamCashApi(AppCore.get().api)
        while (true) {
            runCatching { api.bill(billId) }
                .onSuccess {
                    bill = it
                    checkFailed = false
                    if (payUrl.isEmpty()) payUrl = it.paymentUrl
                }
                .onFailure {
                    checkFailed = true
                    if (bill == null) fatal = shamError(ctx, it)
                }
            if (bill?.isFinal == true) break
            delay(POLL_MS)
        }
    }

    // A paid order empties the cart; a paid bill refreshes the balance.
    val b = bill
    LaunchedEffect(b?.status, b?.orderId) {
        if (b != null && b.paid) {
            if (b.purpose == "order" && !b.orderId.isNullOrBlank()) Cart.clear()
            Refresh.bump()
        }
    }

    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(Rahal.space.lg),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(Rahal.space.md),
    ) {
        PageHeader(stringResource(R.string.sham_status_title), onClose)
        Spacer(Modifier.height(Rahal.space.lg))

        if (browserFailed && b?.pending != false) Note(stringResource(R.string.sham_no_browser), Rahal.colors.danger)

        when {
            b == null && fatal.isNotEmpty() -> {
                Note(fatal, Rahal.colors.danger)
                RahalOutlineButton(onClick = onClose, modifier = Modifier.fillMaxWidth()) {
                    Text(stringResource(R.string.sham_close))
                }
            }

            b == null -> com.rahalgo.ui.RahalLoader()

            b.pending -> {
                com.rahalgo.ui.RahalLoader()
                Title(stringResource(R.string.sham_waiting_title))
                Body(stringResource(R.string.sham_waiting_body))
                if (b.amount > 0) Body(money(b.amount))
                if (checkFailed) Body(stringResource(R.string.sham_checking_error))
                RahalButton(
                    onClick = {
                        val url = payUrl.ifBlank { b.paymentUrl }
                        browserFailed = !openPaymentPage(ctx, url)
                    },
                    enabled = payUrl.isNotBlank() || b.paymentUrl.isNotBlank(),
                    modifier = Modifier.fillMaxWidth(),
                ) { Text(stringResource(R.string.sham_reopen)) }
            }

            b.paid && b.purpose == "order" && b.orderError.isNotEmpty() -> {
                // The money is in the wallet, but the order was not placed.
                Note(stringResource(R.string.sham_order_failed), Rahal.colors.danger)
                RahalButton(onClick = onCart, modifier = Modifier.fillMaxWidth()) {
                    Text(stringResource(R.string.sham_to_cart))
                }
                RahalOutlineButton(onClick = onWallet, modifier = Modifier.fillMaxWidth()) {
                    Text(stringResource(R.string.sham_to_wallet))
                }
            }

            b.paid && b.purpose == "order" -> {
                val n = b.orderNumber
                Note(
                    if (n != null) stringResource(R.string.sham_paid_order_number, n.toString())
                    else stringResource(R.string.sham_paid_order),
                    Rahal.colors.success,
                )
                RahalButton(onClick = onOrders, modifier = Modifier.fillMaxWidth()) {
                    Text(stringResource(R.string.sham_to_order))
                }
            }

            b.paid -> {
                Note(stringResource(R.string.sham_paid_topup), Rahal.colors.success)
                RahalButton(onClick = onWallet, modifier = Modifier.fillMaxWidth()) {
                    Text(stringResource(R.string.sham_to_wallet))
                }
            }

            else -> {
                // expired, failed or refunded.
                val text = when (b.status) {
                    "expired" -> R.string.sham_expired
                    "refunded" -> R.string.sham_refunded
                    else -> R.string.sham_failed
                }
                Note(stringResource(text), Rahal.colors.danger)
                if (b.purpose == "order") {
                    RahalButton(onClick = onCart, modifier = Modifier.fillMaxWidth()) {
                        Text(stringResource(R.string.sham_try_again))
                    }
                    RahalOutlineButton(onClick = onCart, modifier = Modifier.fillMaxWidth()) {
                        Text(stringResource(R.string.sham_pay_cash))
                    }
                } else {
                    RahalButton(
                        onClick = { ShamNav.openTopup() },
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text(stringResource(R.string.sham_try_again)) }
                    RahalOutlineButton(onClick = onWallet, modifier = Modifier.fillMaxWidth()) {
                        Text(stringResource(R.string.sham_to_wallet))
                    }
                }
            }
        }
    }
}

@Composable
private fun Title(text: String) = Text(
    text,
    style = MaterialTheme.typography.titleLarge,
    textAlign = TextAlign.Center,
    modifier = Modifier.fillMaxWidth(),
)

@Composable
private fun Body(text: String) = Text(
    text,
    color = Rahal.colors.inkMuted,
    textAlign = TextAlign.Center,
    modifier = Modifier.fillMaxWidth(),
)
