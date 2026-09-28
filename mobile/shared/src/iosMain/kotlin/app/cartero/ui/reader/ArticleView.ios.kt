package app.cartero.ui.reader

import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.Modifier
import androidx.compose.ui.viewinterop.UIKitView
import app.cartero.resources.Res
import kotlinx.cinterop.BetaInteropApi
import kotlinx.cinterop.ExperimentalForeignApi
import kotlinx.cinterop.ObjCSignatureOverride
import kotlinx.cinterop.addressOf
import kotlinx.cinterop.readValue
import kotlinx.cinterop.useContents
import kotlinx.cinterop.usePinned
import kotlinx.coroutines.MainScope
import kotlinx.coroutines.launch
import platform.CoreGraphics.CGRectZero
import platform.Foundation.NSData
import platform.Foundation.NSError
import platform.Foundation.NSHTTPURLResponse
import platform.Foundation.NSURL
import platform.Foundation.create
import platform.UIKit.UIColor
import platform.UIKit.UIScrollView
import platform.UIKit.UIScrollViewDelegateProtocol
import platform.WebKit.WKNavigationAction
import platform.WebKit.WKNavigationActionPolicy
import platform.WebKit.WKNavigationDelegateProtocol
import platform.WebKit.WKNavigationTypeLinkActivated
import platform.WebKit.WKURLSchemeHandlerProtocol
import platform.WebKit.WKURLSchemeTaskProtocol
import platform.WebKit.WKWebView
import platform.WebKit.WKWebViewConfiguration
import platform.darwin.NSObject

private const val FONT_SCHEME = "cartero-font"

actual val FONT_ORIGIN: String = "$FONT_SCHEME://fonts/"

@OptIn(ExperimentalForeignApi::class)
@Composable
actual fun ArticleView(
    html: String,
    baseUrl: String?,
    textZoom: Int,
    onLink: (String) -> Unit,
    onScroll: (progress: Float, delta: Int) -> Unit,
    modifier: Modifier,
) {
    val link by rememberUpdatedState(onLink)
    val scroll by rememberUpdatedState(onScroll)
    val navigation = remember { NavigationDelegate { link(it) } }
    val scrolling = remember { ScrollDelegate { progress, delta -> scroll(progress, delta) } }
    val fonts = remember { FontSchemeHandler() }
    val loaded = remember { LoadedPage() }

    UIKitView(
        factory = {
            val configuration = WKWebViewConfiguration().apply {
                defaultWebpagePreferences.allowsContentJavaScript = false
                setURLSchemeHandler(fonts, forURLScheme = FONT_SCHEME)
            }
            WKWebView(frame = CGRectZero.readValue(), configuration = configuration).apply {
                setOpaque(false)
                backgroundColor = UIColor.clearColor
                scrollView.backgroundColor = UIColor.clearColor
                navigationDelegate = navigation
                scrollView.delegate = scrolling
            }
        },
        update = { view ->
            view.pageZoom = textZoom / 100.0
            if (loaded.html != html) {
                loaded.html = html
                view.loadHTMLString(html, baseURL = baseUrl?.let { NSURL.URLWithString(it) })
            }
        },
        modifier = modifier,
    )
}

private class LoadedPage {
    var html: String? = null
}

private class NavigationDelegate(private val onLink: (String) -> Unit) : NSObject(), WKNavigationDelegateProtocol {
    override fun webView(
        webView: WKWebView,
        decidePolicyForNavigationAction: WKNavigationAction,
        decisionHandler: (WKNavigationActionPolicy) -> Unit,
    ) {
        if (decidePolicyForNavigationAction.navigationType == WKNavigationTypeLinkActivated) {
            decidePolicyForNavigationAction.request.URL?.absoluteString?.let(onLink)
            decisionHandler(WKNavigationActionPolicy.WKNavigationActionPolicyCancel)
        } else {
            decisionHandler(WKNavigationActionPolicy.WKNavigationActionPolicyAllow)
        }
    }
}

@OptIn(ExperimentalForeignApi::class)
private class ScrollDelegate(private val onScroll: (Float, Int) -> Unit) : NSObject(), UIScrollViewDelegateProtocol {
    private var last = 0.0

    override fun scrollViewDidScroll(scrollView: UIScrollView) {
        val y = scrollView.contentOffset.useContents { y }
        val range = scrollView.contentSize.useContents { height } - scrollView.bounds.useContents { size.height }
        val progress = if (range <= 0) 1f else (y / range).toFloat().coerceIn(0f, 1f)
        onScroll(progress, (y - last).toInt())
        last = y
    }
}

@OptIn(ExperimentalForeignApi::class, BetaInteropApi::class)
private class FontSchemeHandler : NSObject(), WKURLSchemeHandlerProtocol {
    @ObjCSignatureOverride
    override fun webView(webView: WKWebView, startURLSchemeTask: WKURLSchemeTaskProtocol) {
        val url = startURLSchemeTask.request.URL ?: return
        val name = url.lastPathComponent
        if (name == null || name !in READER_FONTS) {
            startURLSchemeTask.didFailWithError(NSError.errorWithDomain(FONT_SCHEME, code = 404, userInfo = null))
            return
        }
        MainScope().launch {
            val bytes = Res.readBytes("font/$name")
            val response = NSHTTPURLResponse(
                uRL = url,
                statusCode = 200,
                HTTPVersion = "HTTP/1.1",
                headerFields = mapOf<Any?, Any?>("Content-Type" to "font/ttf", "Access-Control-Allow-Origin" to "*"),
            )
            startURLSchemeTask.didReceiveResponse(response)
            startURLSchemeTask.didReceiveData(bytes.toNSData())
            startURLSchemeTask.didFinish()
        }
    }

    @ObjCSignatureOverride
    override fun webView(webView: WKWebView, stopURLSchemeTask: WKURLSchemeTaskProtocol) = Unit
}

@OptIn(ExperimentalForeignApi::class, BetaInteropApi::class)
private fun ByteArray.toNSData(): NSData = usePinned { pinned ->
    NSData.create(bytes = pinned.addressOf(0), length = size.toULong())
}
