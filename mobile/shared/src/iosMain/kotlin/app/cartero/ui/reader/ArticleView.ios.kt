package app.cartero.ui.reader

import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.viewinterop.UIKitView
import app.cartero.resources.Res
import kotlin.math.roundToInt
import kotlinx.cinterop.BetaInteropApi
import kotlinx.cinterop.ExperimentalForeignApi
import kotlinx.cinterop.ObjCAction
import kotlinx.cinterop.ObjCSignatureOverride
import kotlinx.cinterop.addressOf
import kotlinx.cinterop.readValue
import kotlinx.cinterop.useContents
import kotlinx.cinterop.usePinned
import kotlinx.coroutines.MainScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import platform.CoreGraphics.CGPointMake
import platform.CoreGraphics.CGRectZero
import platform.Foundation.NSData
import platform.Foundation.NSError
import platform.Foundation.NSHTTPURLResponse
import platform.Foundation.NSSelectorFromString
import platform.Foundation.NSURL
import platform.Foundation.create
import platform.UIKit.UIColor
import platform.UIKit.UIGestureRecognizer
import platform.UIKit.UIGestureRecognizerDelegateProtocol
import platform.UIKit.UIScrollView
import platform.UIKit.UIScrollViewContentInsetAdjustmentBehavior
import platform.UIKit.UIScrollViewDelegateProtocol
import platform.UIKit.UITapGestureRecognizer
import platform.WebKit.WKNavigation
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
    topInset: Dp,
    initialScroll: Float,
    onLink: (String) -> Unit,
    onScroll: (progress: Float, delta: Int) -> Unit,
    onTap: () -> Unit,
    onLoaded: () -> Unit,
    modifier: Modifier,
) {
    val link by rememberUpdatedState(onLink)
    val scroll by rememberUpdatedState(onScroll)
    val tap by rememberUpdatedState(onTap)
    val finished by rememberUpdatedState(onLoaded)
    val surface = MaterialTheme.colorScheme.surface
    val background = remember(surface) {
        UIColor.colorWithRed(surface.red.toDouble(), surface.green.toDouble(), surface.blue.toDouble(), 1.0)
    }
    val scrolling = remember { ScrollDelegate { progress, delta -> scroll(progress, delta) } }
    val loaded = remember { LoadedPage() }
    val navigation = remember {
        NavigationDelegate(
            onLink = { link(it) },
            onFinish = { view, finishedNavigation ->
                if (finishedNavigation == loaded.navigation) {
                    if (!scrolling.ready) scrolling.start(view.scrollView, initialScroll)
                    finished()
                }
            },
        )
    }
    val tapping = remember { TapHandler { tap() } }

    UIKitView(
        factory = {
            ReaderWebViews.take().apply {
                setOpaque(false)
                backgroundColor = background
                underPageBackgroundColor = background
                scrollView.backgroundColor = background
                navigationDelegate = navigation
                scrollView.delegate = scrolling
                scrollView.contentInsetAdjustmentBehavior =
                    UIScrollViewContentInsetAdjustmentBehavior.UIScrollViewContentInsetAdjustmentNever
                addGestureRecognizer(
                    UITapGestureRecognizer(target = tapping, action = NSSelectorFromString("tapped")).apply { delegate = tapping },
                )
            }
        },
        update = { view ->
            val padding = topInset.value.roundToInt()
            if (loaded.html != html) {
                loaded.html = html
                loaded.padding = padding
                loaded.textZoom = textZoom
                loaded.navigation = view.loadHTMLString(
                    ArticleHtml.withTextScale(ArticleHtml.withTopInset(html, padding), textZoom),
                    baseURL = NSURL.URLWithString(FONT_ORIGIN),
                )
            } else {
                if (loaded.padding != padding) {
                    loaded.padding = padding
                    view.evaluateJavaScript("document.body.style.paddingTop='${padding}px'", completionHandler = null)
                }
                if (loaded.textZoom != textZoom) {
                    loaded.textZoom = textZoom
                    view.evaluateJavaScript("document.documentElement.style.fontSize='$textZoom%'", completionHandler = null)
                }
            }
        },
        modifier = modifier,
    )
}

internal object ReaderWebViews {
    private val fonts = FontSchemeHandler()
    private var spare: WKWebView? = null

    fun prepare() {
        if (spare == null) spare = create()
    }

    fun take(): WKWebView {
        val view = spare ?: create()
        spare = null
        MainScope().launch {
            delay(SPARE_DELAY_MS)
            prepare()
        }
        return view
    }

    @OptIn(ExperimentalForeignApi::class)
    private fun create(): WKWebView {
        val configuration = WKWebViewConfiguration().apply {
            defaultWebpagePreferences.allowsContentJavaScript = false
            setURLSchemeHandler(fonts, forURLScheme = FONT_SCHEME)
        }
        return WKWebView(frame = CGRectZero.readValue(), configuration = configuration).apply {
            loadHTMLString("", baseURL = NSURL.URLWithString(FONT_ORIGIN))
        }
    }

    private const val SPARE_DELAY_MS = 3_000L
}

private class LoadedPage {
    var html: String? = null
    var padding = 0
    var textZoom = 100
    var navigation: WKNavigation? = null
}

private class NavigationDelegate(
    private val onLink: (String) -> Unit,
    private val onFinish: (WKWebView, WKNavigation?) -> Unit,
) : NSObject(), WKNavigationDelegateProtocol {
    @ObjCSignatureOverride
    override fun webView(webView: WKWebView, didFinishNavigation: WKNavigation?) = onFinish(webView, didFinishNavigation)

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
    private var width = 0.0
    private var settled = 0f
    var ready = false
        private set

    fun start(scrollView: UIScrollView, fraction: Float) {
        settled = fraction
        width = scrollView.bounds.useContents { size.width }
        if (fraction > 0f) scrollView.scrollToFraction(fraction)
        ready = true
    }

    override fun scrollViewDidScroll(scrollView: UIScrollView) {
        if (!ready) return
        val currentWidth = scrollView.bounds.useContents { size.width }
        val byUser = scrollView.tracking || scrollView.dragging || scrollView.decelerating
        if (currentWidth != width && !byUser) {
            width = currentWidth
            scrollView.scrollToFraction(settled)
            return
        }
        width = currentWidth
        val y = scrollView.contentOffset.useContents { y }
        val range = scrollView.contentSize.useContents { height } - scrollView.bounds.useContents { size.height }
        val progress = if (range <= 0) 1f else (y / range).toFloat().coerceIn(0f, 1f)
        if (byUser) settled = progress
        onScroll(progress, if (byUser) (y - last).toInt() else 0)
        last = y
    }
}

@OptIn(ExperimentalForeignApi::class)
private fun UIScrollView.scrollToFraction(fraction: Float) {
    val range = contentSize.useContents { height } - bounds.useContents { size.height }
    if (range > 0) setContentOffset(CGPointMake(0.0, range * fraction), animated = false)
}

@OptIn(BetaInteropApi::class)
private class TapHandler(private val onTap: () -> Unit) : NSObject(), UIGestureRecognizerDelegateProtocol {
    @ObjCAction
    fun tapped() = onTap()

    @ObjCSignatureOverride
    override fun gestureRecognizer(
        gestureRecognizer: UIGestureRecognizer,
        shouldRecognizeSimultaneouslyWithGestureRecognizer: UIGestureRecognizer,
    ): Boolean = true
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
