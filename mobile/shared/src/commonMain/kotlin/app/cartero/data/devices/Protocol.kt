package app.cartero.data.devices

import app.cartero.data.db.RuleAction
import app.cartero.data.db.RuleField
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

@Serializable
data class SavedStory(
    val guid: String,
    val feedUrl: String,
    val title: String,
    val summary: String,
    val imageUrl: String? = null,
    val author: String? = null,
    val source: String,
    val topic: String? = null,
    val publishedAt: Long,
    val readingMinutes: Int,
)

@Serializable
data class SavedChange(val url: String, val at: Long, val story: SavedStory? = null)

@Serializable
data class RuleChange(
    val action: RuleAction,
    val field: RuleField,
    val value: String,
    val at: Long,
    val enabled: Boolean? = null,
)

@Serializable
sealed interface DeviceMessage {
    @Serializable
    @SerialName("link")
    data class Link(val secret: String, val name: String) : DeviceMessage

    @Serializable
    @SerialName("linked")
    data class Linked(val name: String) : DeviceMessage

    @Serializable
    @SerialName("sync")
    data class Sync(val changes: List<SavedChange>, val rules: List<RuleChange> = emptyList()) : DeviceMessage
}

internal val DeviceJson = Json { ignoreUnknownKeys = true }
