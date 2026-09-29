package app.cartero.ui.reader

import android.content.Context
import android.graphics.Color
import android.view.GestureDetector
import android.view.MotionEvent
import android.webkit.WebResourceError
import android.webkit.WebResourceRequest
import android.webkit.WebResourceResponse
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.viewinterop.AndroidView
import app.cartero.resources.Res
import app.cartero.ui.LocalGraph
import kotlinx.coroutines.runBlocking
import coil3.disk.DiskCache
import kotlin.math.roundToInt
import java.io.FilterInputStream
import java.io.InputStream
import java.net.URLConnection

actual val FONT_ORIGIN: String = "https://cartero.invalid/fonts/"

@Composable
actual fun ArticleView(
    html: String,
    baseUrl: String?,
    textZoom: Int,
    topInset: Dp,
    initialScroll: Float,
    onLink: (String) -> Unit,
    onScroll: (progress: Float, delta: Int) -> Unit,
    onTap: () -> Unit,
    onShown: () -> Unit,
    modifier: Modifier,
) {
    val imageCache = LocalGraph.current.imageLoader.diskCache
    val link by rememberUpdatedState(onLink)
    val scroll by rememberUpdatedState(onScroll)
    val tap by rememberUpdatedState(onTap)
    val shown by rememberUpdatedState(onShown)
    val page = remember(html, topInset) { ArticleHtml.withTopInset(html, topInset.value.roundToInt()) }

    AndroidView(
        factory = { context ->
            ReaderWebView(context).apply {
                setBackgroundColor(Color.TRANSPARENT)
                isVerticalScrollBarEnabled = false
                settings.javaScriptEnabled = false
                settings.allowFileAccess = false
                settings.allowContentAccess = false
                webViewClient = ReaderClient(
                    imageCache,
                    onLink = { link(it) },
                    onVisible = {
                        restore(initialScroll)
                        shown()
                    },
                )
                setOnScrollChangeListener { _, _, y, _, oldY -> if (ready) scroll(progress, y - oldY) }
                val taps = GestureDetector(context, object : GestureDetector.SimpleOnGestureListener() {
                    override fun onSingleTapConfirmed(e: MotionEvent): Boolean {
                        if (hitTestResult.type != WebView.HitTestResult.SRC_ANCHOR_TYPE) tap()
                        return false
                    }
                })
                setOnTouchListener { _, event ->
                    taps.onTouchEvent(event)
                    false
                }
            }
        },
        update = { view ->
            view.settings.textZoom = textZoom
            if (view.tag != page) {
                view.tag = page
                view.loadDataWithBaseURL(baseUrl, page, "text/html", "utf-8", null)
            }
        },
        onRelease = { it.destroy() },
        modifier = modifier,
    )
}

private class ReaderWebView(context: Context) : WebView(context) {
    private val range: Int get() = computeVerticalScrollRange() - computeVerticalScrollExtent()

    val progress: Float
        get() = if (range <= 0) 1f else (scrollY.toFloat() / range).coerceIn(0f, 1f)

    var ready = false
        private set

    fun restore(fraction: Float) {
        postVisualStateCallback(0, object : VisualStateCallback() {
            override fun onComplete(requestId: Long) {
                if (fraction > 0f) scrollTo(0, (range * fraction).toInt())
                ready = true
            }
        })
    }
}

private class ReaderClient(
    private val imageCache: DiskCache?,
    private val onLink: (String) -> Unit,
    private val onVisible: () -> Unit,
) : WebViewClient() {
    private var visible = false

    override fun onPageCommitVisible(view: WebView, url: String?) = showOnce()

    override fun onReceivedError(view: WebView, request: WebResourceRequest, error: WebResourceError) {
        if (request.isForMainFrame) showOnce()
    }

    private fun showOnce() {
        if (visible) return
        visible = true
        onVisible()
    }

    override fun shouldOverrideUrlLoading(view: WebView, request: WebResourceRequest): Boolean {
        onLink(request.url.toString())
        return true
    }

    override fun shouldInterceptRequest(view: WebView, request: WebResourceRequest): WebResourceResponse? {
        val url = request.url.toString()
        if (url.startsWith(FONT_ORIGIN)) return font(url.removePrefix(FONT_ORIGIN))
        if (request.requestHeaders["Accept"]?.contains("image") != true) return null
        val snapshot = imageCache?.openSnapshot(url) ?: return null
        val stream = object : FilterInputStream(snapshot.data.toFile().inputStream() as InputStream) {
            override fun close() {
                super.close()
                snapshot.close()
            }
        }
        val type = URLConnection.guessContentTypeFromName(url.substringBefore('?')) ?: "image/jpeg"
        return WebResourceResponse(type, null, stream)
    }

    private fun font(name: String): WebResourceResponse? {
        if (name !in READER_FONTS) return null
        val bytes = runBlocking { Res.readBytes("font/$name") }
        return WebResourceResponse("font/ttf", null, bytes.inputStream())
            .apply { responseHeaders = mapOf("Access-Control-Allow-Origin" to "*") }
    }
}
