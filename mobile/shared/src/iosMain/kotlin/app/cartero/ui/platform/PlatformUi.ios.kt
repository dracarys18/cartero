package app.cartero.ui.platform

import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.SideEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.rememberUpdatedState
import app.cartero.CarteroIos
import app.cartero.platform.topViewController
import kotlinx.cinterop.BetaInteropApi
import kotlinx.cinterop.ExperimentalForeignApi
import kotlinx.coroutines.launch
import platform.Foundation.NSString
import platform.Foundation.NSTemporaryDirectory
import platform.Foundation.NSURL
import platform.Foundation.NSUTF8StringEncoding
import platform.Foundation.create
import platform.Foundation.writeToURL
import platform.UIKit.UIDocumentPickerDelegateProtocol
import platform.UIKit.UIDocumentPickerViewController
import platform.UniformTypeIdentifiers.UTTypeData
import platform.UniformTypeIdentifiers.UTTypeXML
import platform.UserNotifications.UNAuthorizationOptionAlert
import platform.UserNotifications.UNAuthorizationOptionBadge
import platform.UserNotifications.UNAuthorizationOptionSound
import platform.UserNotifications.UNUserNotificationCenter
import platform.darwin.NSObject
import platform.darwin.dispatch_async
import platform.darwin.dispatch_get_main_queue

@Composable
actual fun rememberNotificationPermission(onResult: (Boolean) -> Unit): () -> Unit {
    val callback by rememberUpdatedState(onResult)
    return remember {
        {
            UNUserNotificationCenter.currentNotificationCenter().requestAuthorizationWithOptions(
                UNAuthorizationOptionAlert or UNAuthorizationOptionSound or UNAuthorizationOptionBadge,
            ) { granted, _ -> dispatch_async(dispatch_get_main_queue()) { callback(granted) } }
        }
    }
}

@OptIn(ExperimentalForeignApi::class, BetaInteropApi::class)
@Composable
actual fun rememberOpmlImport(onOpml: (String) -> Unit): () -> Unit {
    val callback by rememberUpdatedState(onOpml)
    val delegate = remember {
        PickerDelegate(onPick = { urls ->
            val url = urls.firstOrNull() ?: return@PickerDelegate
            NSString.create(contentsOfURL = url, encoding = NSUTF8StringEncoding, error = null)?.toString()?.let(callback)
        })
    }
    return {
        val picker = UIDocumentPickerViewController(forOpeningContentTypes = listOf(UTTypeXML, UTTypeData), asCopy = true)
        picker.delegate = delegate
        topViewController()?.presentViewController(picker, animated = true, completion = null)
    }
}

@OptIn(ExperimentalForeignApi::class, BetaInteropApi::class)
@Composable
actual fun rememberOpmlExport(opml: suspend () -> String, onResult: (Boolean) -> Unit): () -> Unit {
    val scope = rememberCoroutineScope()
    val content by rememberUpdatedState(opml)
    val result by rememberUpdatedState(onResult)
    val delegate = remember { PickerDelegate(onPick = { result(true) }, onCancel = { result(false) }) }
    return {
        scope.launch {
            val file = NSURL.fileURLWithPath(NSTemporaryDirectory() + "cartero.opml")
            @Suppress("CAST_NEVER_SUCCEEDS")
            val written = (content() as NSString).writeToURL(file, atomically = true, encoding = NSUTF8StringEncoding, error = null)
            if (!written) {
                result(false)
                return@launch
            }
            val picker = UIDocumentPickerViewController(forExportingURLs = listOf(file), asCopy = true)
            picker.delegate = delegate
            topViewController()?.presentViewController(picker, animated = true, completion = null)
        }
    }
}

private class PickerDelegate(
    private val onPick: (List<NSURL>) -> Unit,
    private val onCancel: () -> Unit = {},
) : NSObject(), UIDocumentPickerDelegateProtocol {
    override fun documentPicker(controller: UIDocumentPickerViewController, didPickDocumentsAtURLs: List<*>) {
        onPick(didPickDocumentsAtURLs.filterIsInstance<NSURL>())
    }

    override fun documentPickerWasCancelled(controller: UIDocumentPickerViewController) = onCancel()
}

@Composable
actual fun SystemBarsHidden(hidden: Boolean) {
    SideEffect { CarteroIos.setSystemBarsHidden(hidden) }
    DisposableEffect(Unit) {
        onDispose { CarteroIos.setSystemBarsHidden(false) }
    }
}
