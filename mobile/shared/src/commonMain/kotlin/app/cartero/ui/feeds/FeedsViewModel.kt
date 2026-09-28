package app.cartero.ui.feeds

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import app.cartero.AppGraph
import app.cartero.data.db.FeedWithStats
import app.cartero.data.sync.FeedException
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch

sealed interface AddFeedState {
    data object Idle : AddFeedState
    data object Adding : AddFeedState
    data object Added : AddFeedState
    data class Failed(val message: String) : AddFeedState
}

class FeedsViewModel(private val graph: AppGraph) : ViewModel() {
    val feeds: StateFlow<List<FeedWithStats>?> =
        graph.feeds.feeds.stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), null)

    private val adding = MutableStateFlow<AddFeedState>(AddFeedState.Idle)
    val addState: StateFlow<AddFeedState> = adding.asStateFlow()

    private val notice = MutableStateFlow<String?>(null)
    val message: StateFlow<String?> = notice.asStateFlow()

    fun add(url: String) {
        adding.value = AddFeedState.Adding
        viewModelScope.launch {
            adding.value = try {
                graph.feeds.subscribe(url)
                AddFeedState.Added
            } catch (e: CancellationException) {
                throw e
            } catch (e: FeedException) {
                AddFeedState.Failed(e.message ?: "Couldn't add feed")
            } catch (_: Exception) {
                AddFeedState.Failed("That doesn't look like a feed")
            }
        }
    }

    fun resetAdd() {
        adding.value = AddFeedState.Idle
    }

    fun rename(feed: FeedWithStats, title: String) {
        viewModelScope.launch { graph.feeds.rename(feed.id, title) }
    }

    fun delete(feed: FeedWithStats) {
        viewModelScope.launch { graph.feeds.delete(feed.id) }
    }

    fun importOpml(opml: String) {
        viewModelScope.launch {
            val added = runCatching { graph.feeds.importOpml(opml) }.getOrNull()
            notice.value = when (added) {
                null -> "Couldn't read that file"
                0 -> "No new feeds in that file"
                else -> "Added $added feeds"
            }
        }
    }

    suspend fun opml(): String = graph.feeds.exportOpml()

    fun exported(success: Boolean) {
        notice.value = if (success) "Feeds exported" else "Export failed"
    }

    fun messageShown() {
        notice.value = null
    }
}
