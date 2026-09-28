package app.cartero.ui.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExperimentalMaterial3ExpressiveApi
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LargeFlexibleTopAppBar
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Modifier
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import app.cartero.data.settings.Settings
import app.cartero.resources.Res
import app.cartero.resources.ic_arrow_back
import app.cartero.resources.ic_bolt
import app.cartero.resources.ic_favorite
import app.cartero.resources.ic_info
import app.cartero.ui.LocalGraph
import app.cartero.ui.components.ConnectedToggleGroup
import app.cartero.platform.LocalPlatformActions
import kotlinx.coroutines.launch
import org.jetbrains.compose.resources.DrawableResource
import org.jetbrains.compose.resources.painterResource

@OptIn(ExperimentalMaterial3Api::class, ExperimentalMaterial3ExpressiveApi::class)
@Composable
fun SettingsScreen(onBack: () -> Unit, onOpenRules: () -> Unit) {
    val repository = LocalGraph.current.settings
    val settings by repository.settings.collectAsStateWithLifecycle(Settings())
    val scope = rememberCoroutineScope()
    val update: ((Settings) -> Settings) -> Unit = { transform -> scope.launch { repository.update(transform) } }
    val scrollBehavior = TopAppBarDefaults.exitUntilCollapsedScrollBehavior()
    val actions = LocalPlatformActions.current

    Scaffold(
        modifier = Modifier.nestedScroll(scrollBehavior.nestedScrollConnection),
        topBar = {
            LargeFlexibleTopAppBar(
                title = { Text("Settings") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(painterResource(Res.drawable.ic_arrow_back), contentDescription = "Back")
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
            section("About")
            item(key = "about") {
                ListItem(
                    leadingContent = { Icon(painterResource(Res.drawable.ic_info), contentDescription = null) },
                    supportingContent = { Text("Version ${actions.appVersion}") },
                ) {
                    Text("Cartero")
                }
            }
            link(
                icon = Res.drawable.ic_favorite,
                title = "Sponsor on GitHub",
                summary = "Support dracarys18's open source work",
                onClick = { actions.openLink(SPONSORS_URL, settings.inAppBrowser) },
            )

            section("Automation")
            link(
                icon = Res.drawable.ic_bolt,
                title = "Rules",
                summary = "Notify or auto-save stories by topic, source or keyword",
                onClick = onOpenRules,
            )

            section("Reading")
            toggle(
                title = "Open links in app",
                summary = "Use an in-app browser tab instead of your browser",
                checked = settings.inAppBrowser,
                onChange = { update { s -> s.copy(inAppBrowser = it) } },
            )

            section("Sync")
            choice("Check for new stories", SYNC_INTERVALS, settings.syncMinutes, { update { s -> s.copy(syncMinutes = it) } }) {
                if (it < 60) "${it}m" else "${it / 60}h"
            }
            toggle(
                title = "Wi-Fi only",
                summary = "Skip background sync on mobile data",
                checked = settings.wifiOnly,
                onChange = { update { s -> s.copy(wifiOnly = it) } },
            )

            section("Storage")
            choice("Keep stories for", RETENTION_DAYS, settings.retentionDays, { update { s -> s.copy(retentionDays = it) } }) {
                "$it days"
            }
        }
    }
}

private fun LazyListScope.section(title: String) {
    item(key = "section:$title") {
        Text(
            title,
            style = MaterialTheme.typography.titleSmall,
            color = MaterialTheme.colorScheme.primary,
            modifier = Modifier.padding(start = 20.dp, end = 20.dp, top = 24.dp, bottom = 8.dp),
        )
    }
}

private fun LazyListScope.toggle(title: String, summary: String, checked: Boolean, onChange: (Boolean) -> Unit) {
    item(key = "toggle:$title") {
        ListItem(
            supportingContent = { Text(summary) },
            trailingContent = { Switch(checked = checked, onCheckedChange = onChange) },
        ) {
            Text(title)
        }
    }
}

private fun LazyListScope.link(icon: DrawableResource, title: String, summary: String, onClick: () -> Unit) {
    item(key = "link:$title") {
        ListItem(
            onClick = onClick,
            leadingContent = { Icon(painterResource(icon), contentDescription = null) },
            supportingContent = { Text(summary) },
        ) {
            Text(title)
        }
    }
}

private fun <T> LazyListScope.choice(
    title: String,
    options: List<T>,
    selected: T,
    onSelect: (T) -> Unit,
    label: (T) -> String,
) {
    item(key = "choice:$title") {
        Column(
            modifier = Modifier.padding(horizontal = 20.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Text(title, style = MaterialTheme.typography.bodyLarge)
            ConnectedToggleGroup(options, selected, onSelect) { Text(label(it), maxLines = 1) }
        }
    }
}

private const val SPONSORS_URL = "https://github.com/sponsors/dracarys18"
private val SYNC_INTERVALS = listOf(15, 30, 60, 180, 360)
private val RETENTION_DAYS = listOf(7, 30, 90)
