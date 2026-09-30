package app.cartero.ui.devices

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExperimentalMaterial3ExpressiveApi
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LargeFlexibleTopAppBar
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewModelScope
import app.cartero.AppGraph
import app.cartero.data.devices.LinkedDevice
import app.cartero.platform.LocalPlatformActions
import app.cartero.resources.Res
import app.cartero.resources.ic_arrow_back
import app.cartero.resources.ic_delete
import app.cartero.resources.ic_share
import app.cartero.resources.ic_sync
import app.cartero.ui.components.EmptyState
import app.cartero.ui.components.relativeTime
import app.cartero.ui.graphViewModel
import app.cartero.ui.platform.rememberQrScanner
import app.cartero.ui.settings.section
import io.github.alexzhirkevich.qrose.rememberQrCodePainter
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import org.jetbrains.compose.resources.painterResource

class DevicesViewModel(private val graph: AppGraph) : ViewModel() {
    val devices: StateFlow<List<LinkedDevice>?> = graph.devices.devices.stateIn(viewModelScope, WHILE_SUBSCRIBED, null)
    val inviteCode: StateFlow<String?> = graph.devices.inviteCode.stateIn(viewModelScope, WHILE_SUBSCRIBED, null)
    val syncing: StateFlow<Boolean> = graph.devices.syncing

    private val linkingState = MutableStateFlow(false)
    val linking: StateFlow<Boolean> = linkingState.asStateFlow()

    private val notice = MutableStateFlow<String?>(null)
    val message: StateFlow<String?> = notice.asStateFlow()

    fun showCode() {
        viewModelScope.launch {
            runCatching { graph.devices.createInvite() }.onFailure { notice.value = "Couldn't start device sync" }
        }
    }

    fun hideCode() = graph.devices.cancelInvite()

    fun link(code: String) {
        viewModelScope.launch {
            linkingState.value = true
            notice.value = runCatching { graph.devices.link(code) }.fold(
                onSuccess = { "Linked with ${it.name}" },
                onFailure = { error ->
                    (error as? IllegalArgumentException)?.message
                        ?: "Couldn't reach that device. Keep its QR code on screen and try again."
                },
            )
            linkingState.value = false
        }
    }

    fun unlink(device: LinkedDevice) {
        viewModelScope.launch { graph.devices.unlink(device) }
    }

    fun syncNow() {
        viewModelScope.launch {
            val failed = graph.devices.syncAll()
            notice.value = when (failed) {
                0 -> "Saved stories are up to date"
                1 -> "Couldn't reach 1 device. Open Cartero on it and try again."
                else -> "Couldn't reach $failed devices. Open Cartero on them and try again."
            }
        }
    }

    fun messageShown() {
        notice.value = null
    }

    override fun onCleared() {
        graph.devices.cancelInvite()
    }

    private companion object {
        val WHILE_SUBSCRIBED = SharingStarted.WhileSubscribed(5_000)
    }
}

@OptIn(ExperimentalMaterial3Api::class, ExperimentalMaterial3ExpressiveApi::class)
@Composable
fun DevicesScreen(pendingCode: String?, onPendingCodeConsumed: () -> Unit, onBack: () -> Unit) {
    val viewModel = graphViewModel { DevicesViewModel(this) }
    val devices by viewModel.devices.collectAsStateWithLifecycle()
    val inviteCode by viewModel.inviteCode.collectAsStateWithLifecycle()
    val syncing by viewModel.syncing.collectAsStateWithLifecycle()
    val linking by viewModel.linking.collectAsStateWithLifecycle()
    val message by viewModel.message.collectAsStateWithLifecycle()
    val actions = LocalPlatformActions.current
    val snackbar = remember { SnackbarHostState() }
    LaunchedEffect(message) {
        message?.let {
            snackbar.showSnackbar(it)
            viewModel.messageShown()
        }
    }
    LaunchedEffect(pendingCode) {
        pendingCode?.let {
            viewModel.link(it)
            onPendingCodeConsumed()
        }
    }
    val scan = rememberQrScanner(viewModel::link)
    var code by rememberSaveable { mutableStateOf("") }
    val scrollBehavior = TopAppBarDefaults.exitUntilCollapsedScrollBehavior()
    val linked = devices.orEmpty()

    Scaffold(
        modifier = Modifier.nestedScroll(scrollBehavior.nestedScrollConnection),
        snackbarHost = { SnackbarHost(snackbar) },
        topBar = {
            LargeFlexibleTopAppBar(
                title = { Text("Linked devices") },
                subtitle = { Text("Sync saved stories between your devices") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(painterResource(Res.drawable.ic_arrow_back), contentDescription = "Back")
                    }
                },
                actions = {
                    if (linked.isNotEmpty()) {
                        IconButton(onClick = viewModel::syncNow, enabled = !syncing) {
                            Icon(painterResource(Res.drawable.ic_sync), contentDescription = "Sync now")
                        }
                    }
                },
                scrollBehavior = scrollBehavior,
            )
        },
    ) { padding ->
        LazyColumn(
            contentPadding = PaddingValues(bottom = 32.dp),
            modifier = Modifier.fillMaxSize().padding(padding),
        ) {
            section("This device")
            item(key = "invite") {
                val current = inviteCode
                if (current == null) {
                    ListItem(
                        onClick = viewModel::showCode,
                        supportingContent = { Text("Show a QR code to scan from the device you want to link") },
                    ) {
                        Text("Show QR code")
                    }
                } else {
                    InviteCard(
                        code = current,
                        onShare = { actions.share("Link to Cartero", current) },
                        onHide = viewModel::hideCode,
                    )
                }
            }

            section("Link another device")
            item(key = "scan") {
                Button(
                    onClick = scan,
                    enabled = !linking,
                    modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
                ) {
                    Text("Scan QR code")
                }
            }
            item(key = "link") {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
                ) {
                    OutlinedTextField(
                        value = code,
                        onValueChange = { code = it },
                        label = { Text("Or paste a link") },
                        singleLine = true,
                        modifier = Modifier.weight(1f),
                    )
                    OutlinedButton(
                        onClick = {
                            viewModel.link(code)
                            code = ""
                        },
                        enabled = code.isNotBlank() && !linking,
                    ) {
                        Text("Link")
                    }
                }
            }

            if (devices != null && linked.isEmpty()) {
                item(key = "empty") {
                    EmptyState(
                        icon = Res.drawable.ic_sync,
                        title = "No linked devices",
                        body = "Saved stories sync directly between your devices, with no account. " +
                            "Both devices need Cartero open at the same time to sync.",
                    )
                }
            } else if (linked.isNotEmpty()) {
                section("Linked")
                items(linked, key = { "device:${it.id}" }) { device ->
                    ListItem(
                        supportingContent = {
                            Text(device.syncedAt?.let { "Synced ${relativeTime(it)}" } ?: "Not synced yet")
                        },
                        trailingContent = {
                            IconButton(onClick = { viewModel.unlink(device) }) {
                                Icon(painterResource(Res.drawable.ic_delete), contentDescription = "Unlink ${device.name}")
                            }
                        },
                    ) {
                        Text(device.name)
                    }
                }
                item(key = "hint") {
                    Text(
                        "Both devices need Cartero open at the same time to sync.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(horizontal = 20.dp, vertical = 12.dp),
                    )
                }
            }
        }
    }
}

@Composable
private fun InviteCard(code: String, onShare: () -> Unit, onHide: () -> Unit) {
    Card(
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.secondaryContainer),
        modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(12.dp),
            modifier = Modifier.fillMaxWidth().padding(start = 16.dp, end = 8.dp, top = 16.dp, bottom = 8.dp),
        ) {
            Image(
                painter = rememberQrCodePainter(code),
                contentDescription = "Link QR code",
                modifier = Modifier
                    .size(220.dp)
                    .background(Color.White, RoundedCornerShape(12.dp))
                    .padding(12.dp),
            )
            Text(
                "Scan this with Cartero on your other device. It works once, for 10 minutes.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSecondaryContainer,
            )
            Row(horizontalArrangement = Arrangement.End, modifier = Modifier.fillMaxWidth()) {
                TextButton(onClick = onHide) { Text("Hide") }
                TextButton(onClick = onShare) {
                    Icon(painterResource(Res.drawable.ic_share), contentDescription = null)
                    Text("Share link", modifier = Modifier.padding(start = 8.dp))
                }
            }
        }
    }
}
