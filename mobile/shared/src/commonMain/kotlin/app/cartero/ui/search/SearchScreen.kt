package app.cartero.ui.search

import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import app.cartero.resources.Res
import app.cartero.resources.ic_arrow_back
import app.cartero.resources.ic_close
import app.cartero.resources.ic_search
import app.cartero.ui.components.EmptyState
import app.cartero.ui.components.articleItems
import app.cartero.ui.graphViewModel
import org.jetbrains.compose.resources.painterResource

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SearchScreen(onOpenArticle: (Long) -> Unit, onBack: () -> Unit) {
    val viewModel = graphViewModel { SearchViewModel(this) }
    val results by viewModel.results.collectAsStateWithLifecycle()
    var text by rememberSaveable { mutableStateOf("") }
    var focused by rememberSaveable { mutableStateOf(false) }
    val focus = remember { FocusRequester() }
    val keyboard = LocalSoftwareKeyboardController.current

    LaunchedEffect(Unit) {
        if (text.isNotBlank()) viewModel.search(text)
        if (!focused) {
            focus.requestFocus()
            focused = true
        }
    }

    fun submit() {
        viewModel.search(text)
        keyboard?.hide()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(painterResource(Res.drawable.ic_arrow_back), contentDescription = "Back")
                    }
                },
                title = {
                    TextField(
                        value = text,
                        onValueChange = {
                            text = it
                            if (it.isBlank()) viewModel.clear()
                        },
                        placeholder = { Text("Search…") },
                        leadingIcon = { Icon(painterResource(Res.drawable.ic_search), contentDescription = null) },
                        trailingIcon = if (text.isNotEmpty()) {
                            {
                                IconButton(
                                    onClick = {
                                        text = ""
                                        viewModel.clear()
                                        focus.requestFocus()
                                    },
                                ) {
                                    Icon(painterResource(Res.drawable.ic_close), contentDescription = "Clear")
                                }
                            }
                        } else {
                            null
                        },
                        singleLine = true,
                        colors = TextFieldDefaults.colors(
                            focusedContainerColor = Color.Transparent,
                            unfocusedContainerColor = Color.Transparent,
                            focusedIndicatorColor = Color.Transparent,
                            unfocusedIndicatorColor = Color.Transparent,
                        ),
                        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
                        keyboardActions = KeyboardActions(onSearch = { submit() }),
                        modifier = Modifier.fillMaxWidth().focusRequester(focus),
                    )
                },
            )
        },
    ) { padding ->
        LazyColumn(
            contentPadding = PaddingValues(bottom = 24.dp),
            modifier = Modifier.fillMaxSize().padding(padding).imePadding(),
        ) {
            when {
                results.query.isEmpty() -> item(key = "hint") {
                    EmptyState(
                        icon = Res.drawable.ic_search,
                        title = "Search your stories",
                        body = "Find stories by title or summary.",
                    )
                }
                results.rows.isEmpty() -> item(key = "empty") {
                    EmptyState(
                        icon = Res.drawable.ic_search,
                        title = "No results found for “${results.query}”",
                        body = "Try different words. Search matches titles and summaries.",
                    )
                }
                else -> {
                    item(key = "head") {
                        Text(
                            text = "results for “${results.query}”",
                            style = MaterialTheme.typography.labelLarge,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.fillMaxWidth()
                                .padding(start = 20.dp, end = 20.dp, top = 16.dp, bottom = 6.dp),
                        )
                    }
                    articleItems(
                        rows = results.rows,
                        onOpen = onOpenArticle,
                        onToggleSaved = viewModel::toggleSaved,
                        onToggleRead = viewModel::toggleRead,
                        showSummary = true,
                    )
                }
            }
        }
    }
}
