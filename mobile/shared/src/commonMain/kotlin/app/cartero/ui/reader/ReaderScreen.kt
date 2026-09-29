package app.cartero.ui.reader

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBars
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExperimentalMaterial3ExpressiveApi
import androidx.compose.material3.FloatingToolbarDefaults
import androidx.compose.material3.HorizontalFloatingToolbar
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.SideEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.produceState
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import app.cartero.data.settings.Settings
import app.cartero.data.sync.needsFullText
import app.cartero.resources.Res
import app.cartero.resources.ic_arrow_back
import app.cartero.resources.ic_bookmark
import app.cartero.resources.ic_bookmark_filled
import app.cartero.resources.ic_close_fullscreen
import app.cartero.resources.ic_format_size
import app.cartero.resources.ic_open_in_full
import app.cartero.resources.ic_open_in_new
import app.cartero.resources.ic_share
import app.cartero.ui.LocalGraph
import app.cartero.platform.LocalPlatformActions
import app.cartero.ui.graphViewModel
import app.cartero.ui.platform.SystemBarsHidden
import kotlin.math.abs
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.jetbrains.compose.resources.painterResource

@OptIn(ExperimentalMaterial3Api::class, ExperimentalMaterial3ExpressiveApi::class)
@Composable
fun ReaderScreen(
    articleId: Long,
    onBack: () -> Unit,
    expanded: Boolean,
    onToggleExpanded: (() -> Unit)?,
    immersive: Boolean,
) {
    val viewModel = graphViewModel { ReaderViewModel(articleId, this) }
    val graph = LocalGraph.current
    val article by viewModel.article.collectAsStateWithLifecycle()
    val loading by viewModel.loadingFullText.collectAsStateWithLifecycle()
    val settings by graph.settings.settings.collectAsStateWithLifecycle(Settings())
    val actions = LocalPlatformActions.current
    val colors = MaterialTheme.colorScheme
    val palette = remember(colors) { colors.readerPalette() }

    var textSheet by rememberSaveable { mutableStateOf(false) }
    var progress by remember { mutableFloatStateOf(0f) }
    var chromeVisible by remember { mutableStateOf(true) }
    SystemBarsHidden(immersive && !chromeVisible)

    val chromeHeight = WindowInsets.statusBars.asPaddingValues().calculateTopPadding() +
        TopAppBarDefaults.TopAppBarExpandedHeight + PROGRESS_HEIGHT
    var contentTop by remember { mutableStateOf(chromeHeight) }
    SideEffect { if (chromeHeight > contentTop) contentTop = chromeHeight }

    val current = article
    val html by produceState<String?>(null, current, palette, loading) {
        value = if (current == null || loading) {
            null
        } else {
            withContext(Dispatchers.Default) { ArticleHtml.build(current, palette, current.needsFullText()) }
        }
    }
    val page = html
    var pageShown by remember(page == null) { mutableStateOf(false) }

    Box(Modifier.fillMaxSize().background(colors.surface)) {
        if (current != null && page != null) {
            ArticleView(
                html = page,
                baseUrl = current.url,
                textZoom = (settings.readerScale * 100).toInt(),
                topInset = contentTop,
                initialScroll = viewModel.scrollFraction,
                onLink = { url ->
                    if (url == FULL_TEXT_ACTION) viewModel.loadFullText() else actions.openLink(url, settings.inAppBrowser)
                },
                onScroll = { fraction, delta ->
                    progress = fraction
                    viewModel.scrollFraction = fraction
                    if (abs(delta) > SCROLL_SLOP) chromeVisible = delta < 0 || fraction < 0.02f
                },
                onTap = { chromeVisible = !chromeVisible },
                onShown = { pageShown = true },
                modifier = Modifier.fillMaxSize(),
            )
        }
        AnimatedVisibility(visible = !pageShown, enter = fadeIn(), exit = fadeOut()) {
            ArticleSkeleton(top = contentTop)
        }

        AnimatedVisibility(
            visible = chromeVisible,
            enter = slideInVertically { -it } + fadeIn(),
            exit = slideOutVertically { -it } + fadeOut(),
            modifier = Modifier.align(Alignment.TopCenter),
        ) {
            Column(Modifier.background(colors.surface)) {
                TopAppBar(
                    title = {},
                    navigationIcon = {
                        IconButton(onClick = onBack) {
                            Icon(painterResource(Res.drawable.ic_arrow_back), contentDescription = "Back")
                        }
                    },
                    actions = {
                        onToggleExpanded?.let { toggle ->
                            IconButton(onClick = toggle) {
                                Icon(
                                    painterResource(if (expanded) Res.drawable.ic_close_fullscreen else Res.drawable.ic_open_in_full),
                                    contentDescription = if (expanded) "Exit full screen" else "Full screen",
                                )
                            }
                        }
                    },
                )
                LinearProgressIndicator(
                    progress = { progress },
                    modifier = Modifier.fillMaxWidth().height(PROGRESS_HEIGHT),
                    gapSize = 0.dp,
                    drawStopIndicator = {},
                )
            }
        }

        AnimatedVisibility(
            visible = chromeVisible && current != null,
            enter = slideInVertically { it } + fadeIn(),
            exit = slideOutVertically { it } + fadeOut(),
            modifier = Modifier
                .align(Alignment.BottomCenter)
                .navigationBarsPadding()
                .padding(bottom = FloatingToolbarDefaults.ScreenOffset),
        ) {
            HorizontalFloatingToolbar(
                expanded = true,
                colors = FloatingToolbarDefaults.vibrantFloatingToolbarColors(),
            ) {
                val saved = current?.savedAt != null
                IconButton(onClick = viewModel::toggleSaved) {
                    Icon(
                        painterResource(if (saved) Res.drawable.ic_bookmark_filled else Res.drawable.ic_bookmark),
                        contentDescription = if (saved) "Unsave" else "Save",
                    )
                }
                IconButton(onClick = { textSheet = true }) {
                    Icon(painterResource(Res.drawable.ic_format_size), contentDescription = "Text settings")
                }
                current?.url?.let { url ->
                    IconButton(onClick = { actions.openLink(url, settings.inAppBrowser) }) {
                        Icon(painterResource(Res.drawable.ic_open_in_new), contentDescription = "Open original")
                    }
                    IconButton(onClick = { actions.share(current.title, url) }) {
                        Icon(painterResource(Res.drawable.ic_share), contentDescription = "Share")
                    }
                }
            }
        }
    }

    if (textSheet) {
        TextSettingsSheet(
            scale = settings.readerScale,
            onScale = viewModel::setScale,
            onDismiss = { textSheet = false },
        )
    }
}

private const val SCROLL_SLOP = 6
private val PROGRESS_HEIGHT = 2.dp
