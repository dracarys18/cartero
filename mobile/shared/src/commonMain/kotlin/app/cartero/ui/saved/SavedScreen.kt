package app.cartero.ui.saved

import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExperimentalMaterial3ExpressiveApi
import androidx.compose.material3.LargeFlexibleTopAppBar
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.paging.LoadState
import androidx.paging.cachedIn
import androidx.paging.compose.collectAsLazyPagingItems
import app.cartero.AppGraph
import app.cartero.data.db.ArticleRow
import app.cartero.resources.Res
import app.cartero.resources.ic_bookmarks
import app.cartero.ui.components.EmptyState
import app.cartero.ui.components.articleItems
import app.cartero.ui.components.asItems
import app.cartero.ui.graphViewModel
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.launch

class SavedViewModel(private val graph: AppGraph) : ViewModel() {
    val items = graph.articles.saved().map { it.asItems() }.cachedIn(viewModelScope)

    fun toggleSaved(id: Long) {
        viewModelScope.launch { graph.articles.toggleSaved(id) }
    }

    fun toggleRead(row: ArticleRow) {
        viewModelScope.launch { graph.articles.setRead(row.id, !row.isRead) }
    }
}

@OptIn(ExperimentalMaterial3Api::class, ExperimentalMaterial3ExpressiveApi::class)
@Composable
fun SavedScreen(onOpenArticle: (Long) -> Unit) {
    val viewModel = graphViewModel { SavedViewModel(this) }
    val items = viewModel.items.collectAsLazyPagingItems()
    val scrollBehavior = TopAppBarDefaults.exitUntilCollapsedScrollBehavior()

    Scaffold(
        modifier = Modifier.nestedScroll(scrollBehavior.nestedScrollConnection),
        topBar = {
            LargeFlexibleTopAppBar(
                title = { Text("Saved") },
                subtitle = { Text("Available offline") },
                scrollBehavior = scrollBehavior,
            )
        },
    ) { padding ->
        LazyColumn(
            contentPadding = PaddingValues(bottom = 24.dp),
            modifier = Modifier.fillMaxSize().padding(padding),
        ) {
            articleItems(
                items = items,
                onOpen = onOpenArticle,
                onToggleSaved = viewModel::toggleSaved,
                onToggleRead = viewModel::toggleRead,
            )
            if (items.itemCount == 0 && items.loadState.refresh is LoadState.NotLoading) {
                item(key = "empty") {
                    EmptyState(
                        icon = Res.drawable.ic_bookmarks,
                        title = "Nothing saved",
                        body = "Swipe a story right or tap save in the reader. Auto-save rules fill this too.",
                    )
                }
            }
        }
    }
}
