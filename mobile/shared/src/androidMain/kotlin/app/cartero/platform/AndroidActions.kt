package app.cartero.platform

import android.app.Activity
import android.content.ActivityNotFoundException
import android.content.Intent
import android.provider.Settings
import androidx.browser.customtabs.CustomTabsIntent
import androidx.core.net.toUri
import app.cartero.data.db.RuleEntity
import app.cartero.notify.AndroidNotifier

class AndroidActions(private val activity: Activity) : PlatformActions {
    override val appVersion: String =
        activity.packageManager.getPackageInfo(activity.packageName, 0).versionName.orEmpty()

    override fun openLink(url: String, inApp: Boolean) {
        val uri = url.toUri()
        try {
            if (inApp) {
                CustomTabsIntent.Builder().setShowTitle(true).build().launchUrl(activity, uri)
            } else {
                activity.startActivity(Intent(Intent.ACTION_VIEW, uri))
            }
        } catch (_: ActivityNotFoundException) {
        }
    }

    override fun share(title: String, url: String) {
        val send = Intent(Intent.ACTION_SEND)
            .setType("text/plain")
            .putExtra(Intent.EXTRA_SUBJECT, title)
            .putExtra(Intent.EXTRA_TEXT, url)
        activity.startActivity(Intent.createChooser(send, null))
    }

    override fun openNotificationSettings(rule: RuleEntity?) {
        val intent = if (rule == null) {
            Intent(Settings.ACTION_APP_NOTIFICATION_SETTINGS)
        } else {
            Intent(Settings.ACTION_CHANNEL_NOTIFICATION_SETTINGS)
                .putExtra(Settings.EXTRA_CHANNEL_ID, AndroidNotifier.channelId(rule.id))
        }
        activity.startActivity(intent.putExtra(Settings.EXTRA_APP_PACKAGE, activity.packageName))
    }
}
