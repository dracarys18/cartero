package app.cartero.ui

import androidx.compose.runtime.Composable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewmodel.compose.viewModel
import app.cartero.AppGraph

val LocalGraph = staticCompositionLocalOf<AppGraph> { error("AppGraph not provided") }

@Composable
inline fun <reified VM : ViewModel> graphViewModel(crossinline create: AppGraph.() -> VM): VM {
    val graph = LocalGraph.current
    return viewModel { graph.create() }
}
