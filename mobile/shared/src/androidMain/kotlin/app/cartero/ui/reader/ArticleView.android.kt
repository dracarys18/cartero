package app.cartero.ui.reader

import android.content.Context
import android.graphics.Color
import android.webkit.WebResourceRequest
import android.webkit.WebResourceResponse
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.Modifier
import androidx.compose.ui.viewinterop.AndroidView
import app.cartero.resources.Res
import app.cartero.ui.LocalGraph
import kotlinx.coroutines.runBlocking
import coil3.disk.DiskCache
import java.io.FilterInputStream
import java.io.InputStream
import java.net.URLConnection

actual val FONT_ORIGIN: String = "https://cartero.invalid/fonts/"

@Composable
actual fun ArticleView(
    html: String,
    baseUrl: String?,
    textZoom: Int,
    onLink: (String) -> Unit,
    onScroll: (progress: Float, delta: Int) -> Unit,
    modifier: Modifier,
) {
    val imageCache = LocalGraph.current.imageLoader.diskCache
    val link by rememberUpdatedState(onLink)
    val scroll by rememberUpdatedState(onScroll)

    AndroidView(
        factory = { context ->
            ReaderWebView(context).apply {
                setBackgroundColor(Color.TRANSPARENT)
                isVerticalScrollBarEnabled = false
                settings.javaScriptEnabled = false
                settings.allowFileAccess = false
                settings.allowContentAccess = false
                webViewClient = ReaderClient(imageCache) { link(it) }
                setOnScrollChangeListener { _, _, y, _, oldY -> scroll(progress, y - oldY) }
            }
        },
        update = { view ->
            view.settings.textZoom = textZoom
            if (view.tag != html) {
                view.tag = html
                view.loadDataWithBaseURL(baseUrl, html, "text/html", "utf-8", null)
            }
        },
        onRelease = { it.destroy() },
        modifier = modifier,
    )
}

private class ReaderWebView(context: Context) : WebView(context) {
    val progress: Float
        get() {
            val range = computeVerticalScrollRange() - computeVerticalScrollExtent()
            return if (range <= 0) 1f else (scrollY.toFloat() / range).coerceIn(0f, 1f)
        }
}

private class ReaderClient(
    private val imageCache: DiskCache?,
    private val onLink: (String) -> Unit,
) : WebViewClient() {
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
