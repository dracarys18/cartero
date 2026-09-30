package app.cartero

import android.content.Intent
import android.graphics.Color
import android.os.Bundle
import android.os.Looper
import android.webkit.WebSettings
import androidx.activity.ComponentActivity
import androidx.activity.SystemBarStyle
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.Box
import androidx.compose.runtime.getValue
import androidx.compose.ui.ExperimentalComposeUiApi
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.testTagsAsResourceId
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import app.cartero.notify.AndroidNotifier
import app.cartero.platform.AndroidActions
import app.cartero.ui.CarteroRoot
import app.cartero.ui.navigation.AppRequest
import kotlinx.coroutines.flow.MutableStateFlow

class MainActivity : ComponentActivity() {
    private val requests = MutableStateFlow<AppRequest?>(null)

    @OptIn(ExperimentalComposeUiApi::class)
    override fun onCreate(savedInstanceState: Bundle?) {
        enableEdgeToEdge(
            statusBarStyle = SystemBarStyle.dark(Color.TRANSPARENT),
            navigationBarStyle = SystemBarStyle.dark(Color.TRANSPARENT),
        )
        super.onCreate(savedInstanceState)
        val graph = (application as CarteroApp).graph
        val actions = AndroidActions(this)
        if (savedInstanceState == null) handle(intent)

        setContent {
            val request by requests.collectAsStateWithLifecycle()
            Box(Modifier.semantics { testTagsAsResourceId = true }) {
                CarteroRoot(graph, actions, request, onRequestHandled = { requests.value = null })
            }
        }

        Looper.myQueue().addIdleHandler {
            WebSettings.getDefaultUserAgent(this)
            false
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        handle(intent)
    }

    private fun handle(intent: Intent) {
        val request = when (intent.action) {
            AndroidNotifier.ACTION_OPEN -> intent.getLongExtra(AndroidNotifier.EXTRA_ARTICLE_ID, -1).takeIf { it >= 0 }?.let(AppRequest::OpenArticle)
            Intent.ACTION_SEND -> intent.getStringExtra(Intent.EXTRA_TEXT)?.let(::firstUrl)?.let(AppRequest::AddFeed)
            Intent.ACTION_VIEW -> intent.dataString?.let(AppRequest::LinkDevice)
            else -> null
        }
        if (request != null) requests.value = request
    }

    private fun firstUrl(text: String): String? =
        text.split(Regex("\\s+")).firstOrNull { it.startsWith("http://") || it.startsWith("https://") }
}
