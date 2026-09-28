package app.cartero

import kotlin.time.Clock

fun nowMillis(): Long = Clock.System.now().toEpochMilliseconds()
