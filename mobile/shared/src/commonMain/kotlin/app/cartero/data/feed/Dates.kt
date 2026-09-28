package app.cartero.data.feed

import kotlinx.datetime.LocalDate
import kotlinx.datetime.LocalDateTime
import kotlinx.datetime.TimeZone
import kotlinx.datetime.UtcOffset
import kotlinx.datetime.atStartOfDayIn
import kotlinx.datetime.toInstant
import kotlin.time.Instant

object Dates {
    private val rfc822 = Regex(
        """(?:[A-Za-z]+,\s*)?(\d{1,2})\s+([A-Za-z]{3})[A-Za-z]*\s+(\d{2,4})\s+(\d{1,2}):(\d{2})(?::(\d{2}))?\s*([A-Za-z]+|[+-]\d{2}:?\d{2})?""",
    )
    private val months = listOf("jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec")
    private val zones = mapOf(
        "EST" to -5, "EDT" to -4, "CST" to -6, "CDT" to -5,
        "MST" to -7, "MDT" to -6, "PST" to -8, "PDT" to -7,
    )

    fun parse(value: String?): Long? {
        val text = value?.trim()?.takeIf { it.isNotEmpty() } ?: return null
        return if (text.first().isDigit() && text.getOrNull(4) == '-') iso(text) else rfc(text)
    }

    private fun iso(text: String): Long? =
        runCatching { Instant.parse(text).toEpochMilliseconds() }.getOrNull()
            ?: runCatching { LocalDateTime.parse(text).toInstant(TimeZone.UTC).toEpochMilliseconds() }.getOrNull()
            ?: runCatching { LocalDate.parse(text.take(10)).atStartOfDayIn(TimeZone.UTC).toEpochMilliseconds() }.getOrNull()

    private fun rfc(text: String): Long? {
        val match = rfc822.find(text) ?: return null
        val (day, monthName, yearText, hour, minute, second, zone) = match.destructured
        val month = months.indexOf(monthName.lowercase()) + 1
        if (month == 0) return null
        val year = yearText.toInt().let { if (yearText.length == 2) 2000 + it else it }
        return runCatching {
            LocalDateTime(year, month, day.toInt(), hour.toInt(), minute.toInt(), second.ifEmpty { "0" }.toInt())
                .toInstant(offset(zone))
                .toEpochMilliseconds()
        }.getOrNull()
    }

    private fun offset(zone: String): UtcOffset {
        if (zone.isEmpty()) return UtcOffset.ZERO
        if (zone[0] == '+' || zone[0] == '-') {
            val digits = zone.drop(1).replace(":", "")
            val sign = if (zone[0] == '-') -1 else 1
            return UtcOffset(hours = sign * digits.take(2).toInt(), minutes = sign * digits.drop(2).toInt())
        }
        return UtcOffset(hours = zones[zone.uppercase()] ?: 0)
    }
}
