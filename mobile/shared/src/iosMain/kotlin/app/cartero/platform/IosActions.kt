package app.cartero.platform

import app.cartero.data.db.RuleEntity
import platform.Foundation.NSBundle
import platform.Foundation.NSURL
import platform.SafariServices.SFSafariViewController
import platform.UIKit.UIActivityViewController
import platform.UIKit.UIApplication
import platform.UIKit.UIApplicationOpenNotificationSettingsURLString
import platform.UIKit.UIApplicationOpenSettingsURLString
import platform.UIKit.UIViewController
import platform.UIKit.UIWindow
import platform.UIKit.UIWindowScene

object IosActions : PlatformActions {
    override val appVersion: String =
        NSBundle.mainBundle.infoDictionary?.get("CFBundleShortVersionString") as? String ?: ""

    override fun openLink(url: String, inApp: Boolean) {
        val target = NSURL.URLWithString(url) ?: return
        if (inApp) {
            topViewController()?.presentViewController(SFSafariViewController(target), animated = true, completion = null)
        } else {
            UIApplication.sharedApplication.openURL(target, options = emptyMap<Any?, Any>(), completionHandler = null)
        }
    }

    override fun share(title: String, url: String) {
        val items = listOfNotNull(NSURL.URLWithString(url) ?: url)
        val sheet = UIActivityViewController(activityItems = items, applicationActivities = null)
        topViewController()?.presentViewController(sheet, animated = true, completion = null)
    }

    override fun openNotificationSettings(rule: RuleEntity?) {
        val target = NSURL.URLWithString(UIApplicationOpenNotificationSettingsURLString)
            ?: NSURL.URLWithString(UIApplicationOpenSettingsURLString)
            ?: return
        UIApplication.sharedApplication.openURL(target, options = emptyMap<Any?, Any>(), completionHandler = null)
    }
}

fun topViewController(): UIViewController? {
    val window = UIApplication.sharedApplication.connectedScenes
        .filterIsInstance<UIWindowScene>()
        .flatMap { scene -> scene.windows.filterIsInstance<UIWindow>() }
        .firstOrNull { it.isKeyWindow() }
    var top = window?.rootViewController
    while (top?.presentedViewController != null) top = top.presentedViewController
    return top
}
