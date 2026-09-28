package app.cartero.ui.feed

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.paging.PagingData
import androidx.paging.cachedIn
import app.cartero.AppGraph
import app.cartero.data.FeedFilter
import app.cartero.data.db.ArticleRow
import app.cartero.data.db.Facet
import app.cartero.ui.components.ListItem
import app.cartero.ui.components.withDayHeaders
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.flatMapLatest
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

class FeedViewModel(private val graph: AppGraph) : ViewModel() {
    private val selection = MutableStateFlow(FeedFilter())

    val filter: StateFlow<FeedFilter> = combine(
        selection,
        graph.settings.settings.map { it.unreadOnly }.distinctUntilChanged(),
    ) { selected, unreadOnly -> selected.copy(unreadOnly = unreadOnly) }
        .stateIn(viewModelScope, SharingStarted.Eagerly, FeedFilter())

    @OptIn(ExperimentalCoroutinesApi::class)
    val items: Flow<PagingData<ListItem>> = filter
        .flatMapLatest { graph.articles.feed(it) }
        .map { it.withDayHeaders() }
        .cachedIn(viewModelScope)

    val topics: StateFlow<List<Facet>> = graph.articles.topics.stateIn(viewModelScope, WHILE_SUBSCRIBED, emptyList())
    val sources: StateFlow<List<Facet>> = graph.articles.sources.stateIn(viewModelScope, WHILE_SUBSCRIBED, emptyList())
    val unreadCount: StateFlow<Int> = graph.articles.unreadCount.stateIn(viewModelScope, WHILE_SUBSCRIBED, 0)
    val feedCount: StateFlow<Int?> = graph.feeds.count.stateIn(viewModelScope, WHILE_SUBSCRIBED, null)
    val syncing: StateFlow<Boolean> = graph.sync.syncing

    init {
        graph.scope.launch { graph.sync.syncIfStale() }
    }

    fun refresh() {
        graph.scope.launch { graph.sync.syncAll() }
    }

    fun setTopics(topics: Set<String>) = selection.update { it.copy(topics = topics) }

    fun setSources(sources: Set<String>) = selection.update { it.copy(sources = sources) }

    fun clearFilters() = selection.update { FeedFilter() }

    fun toggleUnreadOnly() {
        viewModelScope.launch { graph.settings.update { it.copy(unreadOnly = !it.unreadOnly) } }
    }

    fun toggleSaved(id: Long) {
        viewModelScope.launch { graph.articles.toggleSaved(id) }
    }

    fun toggleRead(row: ArticleRow) {
        viewModelScope.launch { graph.articles.setRead(row.id, !row.isRead) }
    }

    fun markAllRead() {
        viewModelScope.launch { graph.articles.markAllRead(filter.value) }
    }

    private companion object {
        val WHILE_SUBSCRIBED = SharingStarted.WhileSubscribed(5_000)
    }
}
