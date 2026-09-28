package app.cartero.baselineprofile

import androidx.benchmark.macro.junit4.BaselineProfileRule
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.uiautomator.By
import androidx.test.uiautomator.Direction
import androidx.test.uiautomator.Until
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class BaselineProfileGenerator {
    @get:Rule
    val rule = BaselineProfileRule()

    @Test
    fun generate() = rule.collect(packageName = "app.cartero", includeInStartupProfile = true) {
        pressHome()
        startActivityAndWait()

        val list = device.wait(Until.findObject(By.res("feed_list")), 10_000) ?: return@collect
        list.wait(Until.hasObject(By.clickable(true).hasDescendant(By.textContains(" · "))), 30_000)
        list.setGestureMargin(device.displayWidth / 5)
        repeat(3) { list.fling(Direction.DOWN) }
        device.waitForIdle()
        list.fling(Direction.UP)
        device.waitForIdle()

        list.findObject(By.clickable(true).hasDescendant(By.textContains(" · ")))?.click()
        device.wait(Until.hasObject(By.scrollable(true)), 5_000)
        device.waitForIdle()
        device.findObject(By.scrollable(true))?.fling(Direction.DOWN)
        device.waitForIdle()
        device.pressBack()
    }
}
