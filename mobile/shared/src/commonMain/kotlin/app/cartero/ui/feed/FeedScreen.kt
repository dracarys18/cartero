package app.cartero.ui.feed

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExperimentalMaterial3ExpressiveApi
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LargeFlexibleTopAppBar
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.material3.pulltorefresh.PullToRefreshDefaults
import androidx.compose.material3.pulltorefresh.rememberPullToRefreshState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.paging.LoadState
import androidx.paging.compose.collectAsLazyPagingItems
import app.cartero.data.FeedFilter
import app.cartero.resources.Res
import app.cartero.resources.ic_check
import app.cartero.resources.ic_done_all
import app.cartero.resources.ic_expand_more
import app.cartero.resources.ic_newspaper
import app.cartero.resources.ic_rss_feed
import app.cartero.resources.ic_search
import app.cartero.resources.ic_settings
import app.cartero.ui.components.EmptyState
import app.cartero.ui.components.MultiSelectSheet
import app.cartero.ui.components.articleItems
import app.cartero.ui.graphViewModel
import org.jetbrains.compose.resources.painterResource

@OptIn(ExperimentalMaterial3Api::class, ExperimentalMaterial3ExpressiveApi::class)
@Composable
fun FeedScreen(onOpenArticle: (Long) -> Unit, onOpenSettings: () -> Unit, onAddFeed: () -> Unit) {
    val viewModel = graphViewModel { FeedViewModel(this) }
    val items = viewModel.items.collectAsLazyPagingItems()
    val filter by viewModel.filter.collectAsStateWithLifecycle()
    val topics by viewModel.topics.collectAsStateWithLifecycle()
    val sources by viewModel.sources.collectAsStateWithLifecycle()
    val unread by viewModel.unreadCount.collectAsStateWithLifecycle()
    val feedCount by viewModel.feedCount.collectAsStateWithLifecycle()
    val syncing by viewModel.syncing.collectAsStateWithLifecycle()

    var picker by rememberSaveable { mutableStateOf<Picker?>(null) }
    val scrollBehavior = TopAppBarDefaults.exitUntilCollapsedScrollBehavior()
    val listState = rememberLazyListState()
    val pullState = rememberPullToRefreshState()

    Scaffold(
        modifier = Modifier.nestedScroll(scrollBehavior.nestedScrollConnection),
        topBar = {
            LargeFlexibleTopAppBar(
                title = { Text("Cartero") },
                subtitle = { Text(if (unread == 0) "All caught up" else "$unread unread") },
                actions = {
                    IconButton(onClick = viewModel::markAllRead) {
                        Icon(painterResource(Res.drawable.ic_done_all), contentDescription = "Mark all read")
                    }
                    IconButton(onClick = onOpenSettings) {
                        Icon(painterResource(Res.drawable.ic_settings), contentDescription = "Settings")
                    }
                },
                scrollBehavior = scrollBehavior,
            )
        },
    ) { padding ->
        PullToRefreshBox(
            isRefreshing = syncing,
            onRefresh = viewModel::refresh,
            state = pullState,
            modifier = Modifier.padding(padding),
            indicator = {
                PullToRefreshDefaults.LoadingIndicator(
                    state = pullState,
                    isRefreshing = syncing,
                    modifier = Modifier.align(Alignment.TopCenter),
                )
            },
        ) {
            LazyColumn(
                state = listState,
                contentPadding = PaddingValues(bottom = 24.dp),
                modifier = Modifier.fillMaxSize().testTag("feed_list"),
            ) {
                item(key = "filters", contentType = "filters") {
                    FilterRow(
                        filter = filter,
                        onToggleUnread = viewModel::toggleUnreadOnly,
                        onOpen = { picker = it },
                    )
                }
                articleItems(
                    items = items,
                    onOpen = onOpenArticle,
                    onToggleSaved = viewModel::toggleSaved,
                    onToggleRead = viewModel::toggleRead,
                )
                if (items.itemCount == 0 && items.loadState.refresh is LoadState.NotLoading) {
                    item(key = "empty", contentType = "empty") {
                        when {
                            feedCount == 0 -> EmptyState(
                                icon = Res.drawable.ic_rss_feed,
                                title = "No feeds yet",
                                body = "Add the Cartero feed or any RSS, Atom or JSON feed to get started.",
                                action = "Add a feed",
                                onAction = onAddFeed,
                            )
                            !filter.isEmpty -> EmptyState(
                                icon = Res.drawable.ic_search,
                                title = "Nothing matches",
                                body = "No stories for these filters.",
                                action = "Clear filters",
                                onAction = viewModel::clearFilters,
                            )
                            filter.unreadOnly -> EmptyState(
                                icon = Res.drawable.ic_done_all,
                                title = "All caught up",
                                body = "New stories show up here as feeds refresh.",
                            )
                            feedCount != null -> EmptyState(
                                icon = Res.drawable.ic_newspaper,
                                title = "No stories yet",
                                body = "Pull down to refresh.",
                            )
                        }
                    }
                }
            }
        }
    }

    when (picker) {
        Picker.Topics -> MultiSelectSheet(
            title = "Topics",
            options = topics,
            selected = filter.topics,
            onApply = viewModel::setTopics,
            onDismiss = { picker = null },
        )
        Picker.Sources -> MultiSelectSheet(
            title = "Sources",
            options = sources,
            selected = filter.sources,
            onApply = viewModel::setSources,
            onDismiss = { picker = null },
        )
        null -> Unit
    }
}

private enum class Picker { Topics, Sources }

@Composable
private fun FilterRow(
    filter: FeedFilter,
    onToggleUnread: () -> Unit,
    onOpen: (Picker) -> Unit,
) {
    LazyRow(
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        item(key = "unread") {
            FilterChip(
                selected = filter.unreadOnly,
                onClick = onToggleUnread,
                label = { Text("Unread") },
                leadingIcon = if (filter.unreadOnly) {
                    { Icon(painterResource(Res.drawable.ic_check), null, Modifier.size(FilterChipDefaults.IconSize)) }
                } else {
                    null
                },
            )
        }
        item(key = "topics") {
            DropdownChip(label = "Topics", selected = filter.topics, onClick = { onOpen(Picker.Topics) })
        }
        item(key = "sources") {
            DropdownChip(label = "Sources", selected = filter.sources, onClick = { onOpen(Picker.Sources) })
        }
    }
}

@Composable
private fun DropdownChip(label: String, selected: Set<String>, onClick: () -> Unit) {
    val text = when (selected.size) {
        0 -> label
        1 -> selected.first()
        else -> "${selected.size} ${label.lowercase()}"
    }
    FilterChip(
        selected = selected.isNotEmpty(),
        onClick = onClick,
        label = { Text(text) },
        trailingIcon = {
            Icon(painterResource(Res.drawable.ic_expand_more), null, Modifier.size(FilterChipDefaults.IconSize))
        },
    )
}
