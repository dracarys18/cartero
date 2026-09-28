package app.cartero.ui.feeds

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExperimentalMaterial3ExpressiveApi
import androidx.compose.material3.ExtendedFloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LargeFlexibleTopAppBar
import androidx.compose.material3.ListItem
import androidx.compose.material3.LoadingIndicator
import androidx.compose.material3.MaterialTheme
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
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import app.cartero.data.db.CarteroDatabase
import app.cartero.data.db.FeedWithStats
import app.cartero.resources.Res
import app.cartero.resources.ic_add
import app.cartero.resources.ic_delete
import app.cartero.resources.ic_download
import app.cartero.resources.ic_edit
import app.cartero.resources.ic_more_vert
import app.cartero.resources.ic_rss_feed
import app.cartero.resources.ic_upload
import app.cartero.ui.components.EmptyState
import app.cartero.ui.graphViewModel
import coil3.compose.AsyncImage
import app.cartero.data.feed.Html
import app.cartero.ui.components.relativeTime
import app.cartero.ui.platform.rememberOpmlExport
import app.cartero.ui.platform.rememberOpmlImport
import org.jetbrains.compose.resources.painterResource

@OptIn(ExperimentalMaterial3Api::class, ExperimentalMaterial3ExpressiveApi::class)
@Composable
fun FeedsScreen(pendingUrl: String?, onPendingUrlConsumed: () -> Unit) {
    val viewModel = graphViewModel { FeedsViewModel(this) }
    val feeds by viewModel.feeds.collectAsStateWithLifecycle()
    val addState by viewModel.addState.collectAsStateWithLifecycle()
    val message by viewModel.message.collectAsStateWithLifecycle()

    var addOpen by rememberSaveable { mutableStateOf(false) }
    var addUrl by rememberSaveable { mutableStateOf("") }
    var menuOpen by remember { mutableStateOf(false) }
    var renaming by remember { mutableStateOf<FeedWithStats?>(null) }
    var deleting by remember { mutableStateOf<FeedWithStats?>(null) }
    val snackbar = remember { SnackbarHostState() }

    val importOpml = rememberOpmlImport(viewModel::importOpml)
    val exportOpml = rememberOpmlExport(viewModel::opml, viewModel::exported)

    LaunchedEffect(pendingUrl) {
        if (pendingUrl != null) {
            addUrl = pendingUrl
            addOpen = true
            onPendingUrlConsumed()
        }
    }
    LaunchedEffect(message) {
        message?.let {
            snackbar.showSnackbar(it)
            viewModel.messageShown()
        }
    }
    LaunchedEffect(addState) {
        if (addState == AddFeedState.Added) {
            addOpen = false
            addUrl = ""
            viewModel.resetAdd()
        }
    }

    val scrollBehavior = TopAppBarDefaults.exitUntilCollapsedScrollBehavior()
    Scaffold(
        modifier = Modifier.nestedScroll(scrollBehavior.nestedScrollConnection),
        snackbarHost = { SnackbarHost(snackbar) },
        topBar = {
            LargeFlexibleTopAppBar(
                title = { Text("Feeds") },
                subtitle = { Text(subscriptionCount(feeds?.size ?: 0)) },
                actions = {
                    Box {
                        IconButton(onClick = { menuOpen = true }) {
                            Icon(painterResource(Res.drawable.ic_more_vert), contentDescription = "More")
                        }
                        DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
                            DropdownMenuItem(
                                text = { Text("Import OPML") },
                                leadingIcon = { Icon(painterResource(Res.drawable.ic_download), null) },
                                onClick = {
                                    menuOpen = false
                                    importOpml()
                                },
                            )
                            DropdownMenuItem(
                                text = { Text("Export OPML") },
                                leadingIcon = { Icon(painterResource(Res.drawable.ic_upload), null) },
                                onClick = {
                                    menuOpen = false
                                    exportOpml()
                                },
                            )
                        }
                    }
                },
                scrollBehavior = scrollBehavior,
            )
        },
        floatingActionButton = {
            ExtendedFloatingActionButton(
                onClick = { addOpen = true },
                icon = { Icon(painterResource(Res.drawable.ic_add), contentDescription = null) },
                text = { Text("Add feed") },
            )
        },
    ) { padding ->
        LazyColumn(
            contentPadding = PaddingValues(bottom = 96.dp),
            modifier = Modifier.fillMaxSize().padding(padding),
        ) {
            items(feeds.orEmpty(), key = { it.id }) { feed ->
                FeedRow(feed, onRename = { renaming = feed }, onDelete = { deleting = feed })
            }
            if (feeds?.isEmpty() == true) {
                item(key = "empty") {
                    EmptyState(
                        icon = Res.drawable.ic_rss_feed,
                        title = "No feeds yet",
                        body = "Paste your Cartero feed (…/feed.rss) or any site address. Feeds are discovered automatically.",
                        action = "Add a feed",
                        onAction = { addOpen = true },
                    )
                }
            }
        }
    }

    if (addOpen) {
        AddFeedDialog(
            url = addUrl,
            onUrlChange = {
                addUrl = it
                if (addState is AddFeedState.Failed) viewModel.resetAdd()
            },
            state = addState,
            onAdd = { viewModel.add(addUrl) },
            onDismiss = {
                addOpen = false
                viewModel.resetAdd()
            },
        )
    }
    renaming?.let { feed ->
        RenameDialog(
            feed = feed,
            onRename = { viewModel.rename(feed, it) },
            onDismiss = { renaming = null },
        )
    }
    deleting?.let { feed ->
        AlertDialog(
            onDismissRequest = { deleting = null },
            title = { Text("Unsubscribe?") },
            text = { Text("${feed.title} and its stories will be removed, including saved ones.") },
            confirmButton = {
                TextButton(onClick = {
                    viewModel.delete(feed)
                    deleting = null
                }) { Text("Unsubscribe") }
            },
            dismissButton = { TextButton(onClick = { deleting = null }) { Text("Cancel") } },
        )
    }
}

@Composable
private fun FeedRow(feed: FeedWithStats, onRename: () -> Unit, onDelete: () -> Unit) {
    var menuOpen by remember { mutableStateOf(false) }
    val colors = MaterialTheme.colorScheme
    val icon = remember(feed.siteUrl, feed.url) { feedIcon(feed) }
    val fallback = painterResource(Res.drawable.ic_rss_feed)
    ListItem(
        supportingContent = {
            if (feed.error != null) {
                Text(feed.error, color = colors.error)
            } else {
                Text(feedStatus(feed))
            }
        },
        leadingContent = {
            AsyncImage(
                model = icon,
                contentDescription = null,
                error = fallback,
                fallback = fallback,
                contentScale = ContentScale.Crop,
                modifier = Modifier
                    .size(40.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(colors.surfaceContainerHigh)
                    .padding(if (icon == CARTERO_ICON) 0.dp else 8.dp),
            )
        },
        trailingContent = {
            Box {
                IconButton(onClick = { menuOpen = true }) {
                    Icon(painterResource(Res.drawable.ic_more_vert), contentDescription = "Feed options")
                }
                DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
                    DropdownMenuItem(
                        text = { Text("Rename") },
                        leadingIcon = { Icon(painterResource(Res.drawable.ic_edit), null) },
                        onClick = {
                            menuOpen = false
                            onRename()
                        },
                    )
                    if (feed.url != CarteroDatabase.DEFAULT_FEED_URL) {
                        DropdownMenuItem(
                            text = { Text("Unsubscribe") },
                            leadingIcon = { Icon(painterResource(Res.drawable.ic_delete), null) },
                            onClick = {
                                menuOpen = false
                                onDelete()
                            },
                        )
                    }
                }
            }
        },
    ) {
        Text(feed.title, maxLines = 1, overflow = TextOverflow.Ellipsis)
    }
}

@OptIn(ExperimentalMaterial3ExpressiveApi::class)
@Composable
private fun AddFeedDialog(
    url: String,
    onUrlChange: (String) -> Unit,
    state: AddFeedState,
    onAdd: () -> Unit,
    onDismiss: () -> Unit,
) {
    val busy = state == AddFeedState.Adding
    AlertDialog(
        onDismissRequest = { if (!busy) onDismiss() },
        icon = { Icon(painterResource(Res.drawable.ic_rss_feed), contentDescription = null) },
        title = { Text("Add feed") },
        text = {
            OutlinedTextField(
                value = url,
                onValueChange = onUrlChange,
                placeholder = { Text("example.com/feed.rss") },
                singleLine = true,
                enabled = !busy,
                isError = state is AddFeedState.Failed,
                supportingText = (state as? AddFeedState.Failed)?.let { failed -> { Text(failed.message) } },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri, imeAction = ImeAction.Done),
                modifier = Modifier.fillMaxWidth(),
            )
        },
        confirmButton = {
            if (busy) {
                LoadingIndicator(Modifier.size(40.dp))
            } else {
                TextButton(onClick = onAdd, enabled = url.isNotBlank()) { Text("Add") }
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss, enabled = !busy) { Text("Cancel") }
        },
    )
}

@Composable
private fun RenameDialog(feed: FeedWithStats, onRename: (String) -> Unit, onDismiss: () -> Unit) {
    var title by rememberSaveable { mutableStateOf(feed.title) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Rename feed") },
        text = {
            OutlinedTextField(value = title, onValueChange = { title = it }, singleLine = true, modifier = Modifier.fillMaxWidth())
        },
        confirmButton = {
            TextButton(
                onClick = {
                    onRename(title)
                    onDismiss()
                },
                enabled = title.isNotBlank(),
            ) { Text("Save") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}

private fun feedIcon(feed: FeedWithStats): String? {
    if (feed.url == CarteroDatabase.DEFAULT_FEED_URL) return CARTERO_ICON
    val host = Html.webUrl(feed.siteUrl ?: feed.url)?.host ?: return null
    return "https://$host/favicon.ico"
}

private fun feedStatus(feed: FeedWithStats): String {
    val synced = when {
        feed.syncedAt == 0L -> "not synced yet"
        else -> relativeTime(feed.syncedAt).let { ago ->
            when {
                ago == "now" -> "synced just now"
                ' ' in ago -> "synced $ago"
                else -> "synced $ago ago"
            }
        }
    }
    return if (feed.unread > 0) "${feed.unread} unread · $synced" else synced.replaceFirstChar { it.uppercase() }
}

private fun subscriptionCount(count: Int): String = if (count == 1) "1 subscription" else "$count subscriptions"

private const val CARTERO_ICON = "https://news.karthihegde.dev/assets/icon-192.png"
