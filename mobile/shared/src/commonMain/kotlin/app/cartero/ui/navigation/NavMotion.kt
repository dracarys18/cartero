package app.cartero.ui.navigation

import androidx.compose.animation.ContentTransform
import androidx.compose.animation.core.FastOutLinearInEasing
import androidx.compose.animation.core.LinearOutSlowInEasing
import androidx.compose.animation.core.snap
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.material3.ExperimentalMaterial3ExpressiveApi
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.MotionScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.Immutable
import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.dp
import androidx.navigation3.ui.NavDisplay

@Immutable
class NavMotion(private val motion: MotionScheme, private val distance: Int) {
    fun tabSwitch(): ContentTransform =
        fadeIn(tween(TAB_MS)).togetherWith(fadeOut(snap(delayMillis = TAB_MS)))

    fun sharedAxis(forward: Boolean): ContentTransform {
        val direction = if (forward) 1 else -1
        val enter = slideInHorizontally(motion.defaultSpatialSpec()) { direction * distance } +
            fadeIn(tween(FADE_IN_MS, delayMillis = FADE_OUT_MS, easing = LinearOutSlowInEasing))
        val exit = slideOutHorizontally(motion.defaultSpatialSpec()) { -direction * distance } +
            fadeOut(tween(FADE_OUT_MS, easing = FastOutLinearInEasing))
        return enter.togetherWith(exit)
    }

    val pushed: Map<String, Any> =
        NavDisplay.transitionSpec { sharedAxis(forward = true) } +
            NavDisplay.popTransitionSpec { sharedAxis(forward = false) } +
            NavDisplay.predictivePopTransitionSpec { sharedAxis(forward = false) }

    private companion object {
        const val TAB_MS = 150
        const val FADE_OUT_MS = 90
        const val FADE_IN_MS = 210
    }
}

@OptIn(ExperimentalMaterial3ExpressiveApi::class)
@Composable
fun rememberNavMotion(): NavMotion {
    val motion = MaterialTheme.motionScheme
    val distance = with(LocalDensity.current) { 30.dp.roundToPx() }
    return remember(motion, distance) { NavMotion(motion, distance) }
}
