package app.cartero.ui.components

import app.cartero.nowMillis
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlinx.datetime.daysUntil
import kotlinx.datetime.format
import kotlinx.datetime.format.DayOfWeekNames
import kotlinx.datetime.format.MonthNames
import kotlinx.datetime.format.Padding
import kotlinx.datetime.format.char
import kotlinx.datetime.toLocalDateTime
import kotlin.time.Instant

private val shortDate = LocalDate.Format {
    day(padding = Padding.NONE)
    char(' ')
    monthName(MonthNames.ENGLISH_ABBREVIATED)
}

private val fullDate = LocalDate.Format {
    day(padding = Padding.NONE)
    char(' ')
    monthName(MonthNames.ENGLISH_ABBREVIATED)
    char(' ')
    year()
}

private val weekday = LocalDate.Format { dayOfWeek(DayOfWeekNames.ENGLISH_FULL) }

fun Long.localDate(): LocalDate =
    Instant.fromEpochMilliseconds(this).toLocalDateTime(TimeZone.currentSystemDefault()).date

fun today(): LocalDate = nowMillis().localDate()

fun LocalDate.fullDate(): String = format(fullDate)

fun relativeTime(millis: Long, now: Long = nowMillis()): String {
    val minutes = (now - millis) / 60_000
    return when {
        minutes < 1 -> "now"
        minutes < 60 -> "${minutes}m"
        minutes < 24 * 60 -> "${minutes / 60}h"
        minutes < 7 * 24 * 60 -> "${minutes / (24 * 60)}d"
        else -> millis.localDate().format(shortDate)
    }
}

fun dayLabel(day: LocalDate, today: LocalDate): String {
    val days = day.daysUntil(today)
    return when {
        days == 0 -> "Today"
        days == 1 -> "Yesterday"
        days in 2..6 -> day.format(weekday)
        day.year == today.year -> day.format(shortDate)
        else -> day.format(fullDate)
    }
}
