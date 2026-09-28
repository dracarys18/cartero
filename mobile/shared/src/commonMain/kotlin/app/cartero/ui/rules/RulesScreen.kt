package app.cartero.ui.rules

import org.jetbrains.compose.resources.DrawableResource
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExperimentalMaterial3ExpressiveApi
import androidx.compose.material3.ExtendedFloatingActionButton
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LargeFlexibleTopAppBar
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SheetValue
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.material3.rememberBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.compose.LifecycleResumeEffect
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewModelScope
import app.cartero.AppGraph
import app.cartero.data.db.Facet
import app.cartero.data.db.RuleAction
import app.cartero.data.db.RuleEntity
import app.cartero.data.db.RuleField
import app.cartero.notify.label
import app.cartero.platform.LocalPlatformActions
import app.cartero.ui.platform.rememberNotificationPermission
import app.cartero.resources.Res
import app.cartero.resources.ic_add
import app.cartero.resources.ic_arrow_back
import app.cartero.resources.ic_bolt
import app.cartero.resources.ic_delete
import app.cartero.resources.ic_label
import app.cartero.resources.ic_notifications
import app.cartero.resources.ic_public
import app.cartero.resources.ic_search
import app.cartero.ui.components.ConnectedToggleGroup
import app.cartero.ui.components.EmptyState
import app.cartero.ui.graphViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import org.jetbrains.compose.resources.painterResource

class RulesViewModel(private val graph: AppGraph) : ViewModel() {
    val rules: StateFlow<List<RuleEntity>?> = graph.rules.rules.stateIn(viewModelScope, WHILE_SUBSCRIBED, null)
    val topics: StateFlow<List<Facet>> = graph.articles.topics.stateIn(viewModelScope, WHILE_SUBSCRIBED, emptyList())
    val sources: StateFlow<List<Facet>> = graph.articles.sources.stateIn(viewModelScope, WHILE_SUBSCRIBED, emptyList())

    private val notice = MutableStateFlow<String?>(null)
    val message: StateFlow<String?> = notice.asStateFlow()

    val notificationsEnabled: Boolean get() = graph.notifier.enabled

    fun add(action: RuleAction, field: RuleField, value: String) {
        viewModelScope.launch { announceSaved(graph.rules.add(action, field, value)) }
    }

    fun setEnabled(rule: RuleEntity, enabled: Boolean) {
        viewModelScope.launch { announceSaved(graph.rules.setEnabled(rule, enabled)) }
    }

    fun messageShown() {
        notice.value = null
    }

    private fun announceSaved(count: Int) {
        if (count > 0) notice.value = if (count == 1) "Saved 1 existing story" else "Saved $count existing stories"
    }

    fun delete(rule: RuleEntity) {
        viewModelScope.launch { graph.rules.delete(rule) }
    }

    private companion object {
        val WHILE_SUBSCRIBED = SharingStarted.WhileSubscribed(5_000)
    }
}

@OptIn(ExperimentalMaterial3Api::class, ExperimentalMaterial3ExpressiveApi::class)
@Composable
fun RulesScreen(onBack: () -> Unit) {
    val viewModel = graphViewModel { RulesViewModel(this) }
    val rules by viewModel.rules.collectAsStateWithLifecycle()
    val topics by viewModel.topics.collectAsStateWithLifecycle()
    val sources by viewModel.sources.collectAsStateWithLifecycle()
    val message by viewModel.message.collectAsStateWithLifecycle()
    val actions = LocalPlatformActions.current
    val snackbar = remember { SnackbarHostState() }
    LaunchedEffect(message) {
        message?.let {
            snackbar.showSnackbar(it)
            viewModel.messageShown()
        }
    }

    var editorOpen by rememberSaveable { mutableStateOf(false) }
    var notificationsEnabled by remember { mutableStateOf(viewModel.notificationsEnabled) }
    LifecycleResumeEffect(Unit) {
        notificationsEnabled = viewModel.notificationsEnabled
        onPauseOrDispose {}
    }
    val requestPermission = rememberNotificationPermission { granted -> notificationsEnabled = granted }

    val scrollBehavior = TopAppBarDefaults.exitUntilCollapsedScrollBehavior()
    val all = rules.orEmpty()
    val notify = remember(all) { all.filter { it.action == RuleAction.Notify } }
    val save = remember(all) { all.filter { it.action == RuleAction.Save } }

    Scaffold(
        modifier = Modifier.nestedScroll(scrollBehavior.nestedScrollConnection),
        snackbarHost = { SnackbarHost(snackbar) },
        topBar = {
            LargeFlexibleTopAppBar(
                title = { Text("Rules") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(painterResource(Res.drawable.ic_arrow_back), contentDescription = "Back")
                    }
                },
                subtitle = { Text("Get notified or auto-save new stories") },
                scrollBehavior = scrollBehavior,
            )
        },
        floatingActionButton = {
            ExtendedFloatingActionButton(
                onClick = { editorOpen = true },
                icon = { Icon(painterResource(Res.drawable.ic_add), contentDescription = null) },
                text = { Text("New rule") },
            )
        },
    ) { padding ->
        LazyColumn(
            contentPadding = PaddingValues(bottom = 96.dp),
            modifier = Modifier.fillMaxSize().padding(padding),
        ) {
            if (notify.isNotEmpty() && !notificationsEnabled) {
                item(key = "permission") {
                    PermissionCard(onAllow = { actions.openNotificationSettings(null) })
                }
            }
            ruleSection(
                title = "Notify me",
                rules = notify,
                onToggle = viewModel::setEnabled,
                onDelete = viewModel::delete,
                onClick = { actions.openNotificationSettings(it) },
            )
            ruleSection(
                title = "Save for later",
                rules = save,
                onToggle = viewModel::setEnabled,
                onDelete = viewModel::delete,
                onClick = null,
            )
            if (rules?.isEmpty() == true) {
                item(key = "empty") {
                    EmptyState(
                        icon = Res.drawable.ic_bolt,
                        title = "No rules yet",
                        body = "Get a notification or auto-save stories for a topic, a source or a keyword.",
                    )
                }
            }
        }
    }

    if (editorOpen) {
        RuleEditorSheet(
            topics = topics,
            sources = sources,
            onCreate = { action, field, value ->
                viewModel.add(action, field, value)
                if (action == RuleAction.Notify && !notificationsEnabled) requestPermission()
            },
            onDismiss = { editorOpen = false },
        )
    }
}

private fun LazyListScope.ruleSection(
    title: String,
    rules: List<RuleEntity>,
    onToggle: (RuleEntity, Boolean) -> Unit,
    onDelete: (RuleEntity) -> Unit,
    onClick: ((RuleEntity) -> Unit)?,
) {
    if (rules.isEmpty()) return
    item(key = "header:$title") {
        Text(
            title,
            style = MaterialTheme.typography.titleSmall,
            color = MaterialTheme.colorScheme.primary,
            modifier = Modifier.padding(start = 20.dp, end = 20.dp, top = 20.dp, bottom = 4.dp),
        )
    }
    items(rules, key = { "rule:${it.id}" }) { rule ->
        ListItem(
            supportingContent = { Text(rule.field.name) },
            leadingContent = { Icon(painterResource(rule.field.icon()), contentDescription = null) },
            trailingContent = {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    IconButton(onClick = { onDelete(rule) }) {
                        Icon(painterResource(Res.drawable.ic_delete), contentDescription = "Delete rule")
                    }
                    Switch(checked = rule.enabled, onCheckedChange = { onToggle(rule, it) })
                }
            },
            modifier = if (onClick != null) Modifier.clickable { onClick(rule) } else Modifier,
        ) {
            Text(rule.label())
        }
    }
}

@Composable
private fun PermissionCard(onAllow: () -> Unit) {
    Card(
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.tertiaryContainer),
        modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Icon(painterResource(Res.drawable.ic_notifications), contentDescription = null)
            Text("Notifications are off", style = MaterialTheme.typography.bodyLarge, modifier = Modifier.weight(1f))
            FilledTonalButton(onClick = onAllow) { Text("Allow") }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
private fun RuleEditorSheet(
    topics: List<Facet>,
    sources: List<Facet>,
    onCreate: (RuleAction, RuleField, String) -> Unit,
    onDismiss: () -> Unit,
) {
    var action by rememberSaveable { mutableStateOf(RuleAction.Notify) }
    var field by rememberSaveable { mutableStateOf(RuleField.Topic) }
    var value by rememberSaveable { mutableStateOf("") }
    val options = when (field) {
        RuleField.Topic -> topics
        RuleField.Source -> sources
        RuleField.Keyword -> emptyList()
    }
    val query = value.trim()
    val exact = remember(options, query) { options.firstOrNull { it.value.equals(query, ignoreCase = true) } }
    val suggestions = remember(options, query, exact) {
        exact?.let(::listOf) ?: options.filter { it.value.contains(query, ignoreCase = true) }
    }

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = rememberBottomSheetState(
            initialValue = SheetValue.Hidden,
            enabledValues = setOf(SheetValue.Hidden, SheetValue.Expanded),
        ),
    ) {
        Column(
            modifier = Modifier
                .verticalScroll(rememberScrollState())
                .padding(start = 24.dp, end = 24.dp, bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text("New rule", style = MaterialTheme.typography.titleLarge)
            SectionLabel("When a new story matches")
            ConnectedToggleGroup(
                options = RuleField.entries,
                selected = field,
                onSelect = {
                    field = it
                    value = ""
                },
            ) { Text(it.name) }
            OutlinedTextField(
                value = value,
                onValueChange = { value = it },
                label = { Text(field.name) },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            if (suggestions.isNotEmpty()) {
                FlowRow(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalArrangement = Arrangement.spacedBy(4.dp),
                ) {
                    suggestions.forEach { option ->
                        FilterChip(
                            selected = option == exact,
                            onClick = { value = option.value },
                            label = { Text(option.value) },
                        )
                    }
                }
            }
            SectionLabel("Then")
            ConnectedToggleGroup(
                options = RuleAction.entries,
                selected = action,
                onSelect = { action = it },
            ) { Text(if (it == RuleAction.Notify) "Notify me" else "Save it") }
            Button(
                onClick = {
                    onCreate(action, field, exact?.value ?: query)
                    onDismiss()
                },
                enabled = value.isNotBlank(),
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text("Create rule")
            }
        }
    }
}

@Composable
private fun SectionLabel(text: String) {
    Text(text, style = MaterialTheme.typography.labelLarge, color = MaterialTheme.colorScheme.onSurfaceVariant)
}

private fun RuleField.icon(): DrawableResource = when (this) {
    RuleField.Topic -> Res.drawable.ic_label
    RuleField.Source -> Res.drawable.ic_public
    RuleField.Keyword -> Res.drawable.ic_search
}
