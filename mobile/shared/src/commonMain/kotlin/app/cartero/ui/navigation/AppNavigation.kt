package app.cartero.ui.navigation

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.adaptive.ExperimentalMaterial3AdaptiveApi
import androidx.compose.material3.adaptive.currentWindowAdaptiveInfoV2
import androidx.compose.material3.adaptive.layout.PaneExpansionAnchor
import androidx.compose.material3.adaptive.layout.calculatePaneScaffoldDirective
import androidx.compose.material3.adaptive.layout.rememberPaneExpansionState
import androidx.compose.material3.adaptive.navigation3.ListDetailSceneStrategy
import androidx.compose.material3.adaptive.navigation3.rememberListDetailSceneStrategy
import androidx.compose.material3.adaptive.navigationsuite.NavigationSuiteItem
import androidx.compose.material3.adaptive.navigationsuite.NavigationSuiteScaffold
import androidx.compose.material3.adaptive.navigationsuite.NavigationSuiteScaffoldDefaults
import androidx.compose.material3.adaptive.navigationsuite.NavigationSuiteType
import androidx.compose.material3.adaptive.navigationsuite.rememberNavigationSuiteScaffoldState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.navigation3.rememberViewModelStoreNavEntryDecorator
import androidx.navigation3.runtime.NavBackStack
import androidx.navigation3.runtime.NavKey
import androidx.navigation3.runtime.entryProvider
import androidx.navigation3.runtime.rememberNavBackStack
import androidx.navigation3.runtime.rememberSaveableStateHolderNavEntryDecorator
import androidx.navigation3.ui.NavDisplay
import app.cartero.resources.Res
import app.cartero.resources.ic_article
import app.cartero.ui.feed.FeedScreen
import app.cartero.ui.feeds.FeedsScreen
import app.cartero.ui.reader.ReaderScreen
import app.cartero.ui.rules.RulesScreen
import app.cartero.ui.saved.SavedScreen
import app.cartero.ui.settings.SettingsScreen
import org.jetbrains.compose.resources.painterResource

sealed interface AppRequest {
    data class OpenArticle(val id: Long) : AppRequest
    data class AddFeed(val url: String) : AppRequest
}

@OptIn(ExperimentalMaterial3AdaptiveApi::class)
@Composable
fun AppNavigation(request: AppRequest?, onRequestHandled: () -> Unit) {
    val backStack = rememberNavBackStack(NavigationState, FeedRoute)
    var pendingFeedUrl by rememberSaveable { mutableStateOf<String?>(null) }

    LaunchedEffect(request) {
        when (request) {
            is AppRequest.OpenArticle -> backStack.openArticle(request.id)
            is AppRequest.AddFeed -> {
                backStack.selectTab(FeedsRoute)
                pendingFeedUrl = request.url
            }
            null -> return@LaunchedEffect
        }
        onRequestHandled()
    }

    val currentTab = backStack.lastOrNull { it is TabRoute } ?: FeedRoute
    val windowInfo = currentWindowAdaptiveInfoV2()
    val suiteType = NavigationSuiteScaffoldDefaults.navigationSuiteType(windowInfo)
    val suiteState = rememberNavigationSuiteScaffoldState()
    val motion = rememberNavMotion()

    val directive = calculatePaneScaffoldDirective(windowInfo)
    val canSplit = directive.maxHorizontalPartitions > 1
    var readerExpanded by rememberSaveable { mutableStateOf(false) }
    val fullScreenReader = canSplit && readerExpanded && backStack.last() is ReaderRoute
    val splitAnchor = PaneExpansionAnchor.Offset.fromStart(directive.defaultPanePreferredWidth)
    val paneExpansion = rememberPaneExpansionState(anchors = listOf(FullScreenAnchor, splitAnchor), initialAnchoredIndex = 1)
    LaunchedEffect(fullScreenReader) { paneExpansion.animateTo(if (fullScreenReader) FullScreenAnchor else splitAnchor) }

    val showSuite = !fullScreenReader && (backStack.last() is TabRoute || suiteType !in BottomBars)
    LaunchedEffect(showSuite) { if (showSuite) suiteState.show() else suiteState.hide() }

    NavigationSuiteScaffold(
        navigationItems = {
            Tab.entries.forEach { tab ->
                val selected = tab.route == currentTab
                NavigationSuiteItem(
                    selected = selected,
                    onClick = { backStack.selectTab(tab.route) },
                    icon = { Icon(painterResource(if (selected) tab.selectedIcon else tab.icon), contentDescription = null) },
                    label = { Text(tab.label) },
                    navigationSuiteType = suiteType,
                )
            }
        },
        navigationSuiteType = suiteType,
        state = suiteState,
    ) {
        NavDisplay(
            backStack = backStack,
            onBack = { backStack.removeLastOrNull() },
            entryDecorators = listOf(
                rememberSaveableStateHolderNavEntryDecorator<NavKey>(),
                rememberViewModelStoreNavEntryDecorator<NavKey>(),
            ),
            sceneStrategies = listOf(
                rememberListDetailSceneStrategy<NavKey>(directive = directive, paneExpansionState = paneExpansion),
            ),
            transitionSpec = { motion.tabSwitch() },
            popTransitionSpec = { motion.tabSwitch() },
            predictivePopTransitionSpec = { motion.tabSwitch() },
            entryProvider = entryProvider {
                entry<FeedRoute>(metadata = ListDetailSceneStrategy.listPane(detailPlaceholder = { DetailPlaceholder() })) {
                    FeedScreen(
                        onOpenArticle = { backStack.openArticle(it) },
                        onOpenSettings = { backStack.add(SettingsRoute) },
                        onAddFeed = { backStack.selectTab(FeedsRoute) },
                    )
                }
                entry<SavedRoute>(metadata = ListDetailSceneStrategy.listPane(detailPlaceholder = { DetailPlaceholder() })) {
                    SavedScreen(onOpenArticle = { backStack.openArticle(it) })
                }
                entry<RulesRoute>(metadata = motion.pushed) {
                    RulesScreen(onBack = { backStack.removeLastOrNull() })
                }
                entry<FeedsRoute> {
                    FeedsScreen(pendingUrl = pendingFeedUrl, onPendingUrlConsumed = { pendingFeedUrl = null })
                }
                entry<ReaderRoute>(metadata = ListDetailSceneStrategy.detailPane() + motion.pushed) { route ->
                    ReaderScreen(
                        articleId = route.articleId,
                        onBack = { backStack.removeLastOrNull() },
                        expanded = fullScreenReader,
                        onToggleExpanded = if (canSplit) ({ readerExpanded = !readerExpanded }) else null,
                        immersive = !canSplit || fullScreenReader,
                    )
                }
                entry<SettingsRoute>(metadata = motion.pushed) {
                    SettingsScreen(
                        onBack = { backStack.removeLastOrNull() },
                        onOpenRules = { backStack.add(RulesRoute) },
                    )
                }
            },
        )
    }
}

@Composable
private fun DetailPlaceholder() {
    Column(
        modifier = Modifier.fillMaxSize(),
        verticalArrangement = Arrangement.spacedBy(12.dp, Alignment.CenterVertically),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Icon(
            painterResource(Res.drawable.ic_article),
            contentDescription = null,
            modifier = Modifier.size(48.dp),
            tint = MaterialTheme.colorScheme.outline,
        )
        Text("Pick a story to read", style = MaterialTheme.typography.bodyLarge, color = MaterialTheme.colorScheme.outline)
    }
}

private val FullScreenAnchor = PaneExpansionAnchor.Proportion(0f)

private val BottomBars = setOf(
    NavigationSuiteType.ShortNavigationBarCompact,
    NavigationSuiteType.ShortNavigationBarMedium,
    NavigationSuiteType.NavigationBar,
)

private fun NavBackStack<NavKey>.selectTab(tab: TabRoute) {
    while (size > 1) removeAt(lastIndex)
    if (tab != FeedRoute) add(tab)
}

private fun NavBackStack<NavKey>.openArticle(id: Long) {
    if (last() is ReaderRoute) removeAt(lastIndex)
    add(ReaderRoute(id))
}
