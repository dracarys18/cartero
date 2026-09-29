package app.cartero.data.reader

import app.cartero.data.feed.Html
import com.fleeksoft.ksoup.Ksoup
import io.ktor.client.HttpClient
import io.ktor.client.plugins.timeout
import io.ktor.client.request.get
import io.ktor.client.request.header
import io.ktor.client.statement.bodyAsText
import io.ktor.http.ContentType
import io.ktor.http.HttpHeaders
import io.ktor.http.contentType
import io.ktor.http.isSuccess
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

class ArticleExtractor(private val http: HttpClient) {
    suspend fun extract(url: String): Extracted? {
        val target = Html.webUrl(url) ?: return null
        val page = try {
            val response = http.get(target) {
                header(HttpHeaders.Accept, "text/html,application/xhtml+xml")
                timeout { requestTimeoutMillis = TIMEOUT_MS }
            }
            val type = response.contentType()
            if (!response.status.isSuccess() || (type != null && !type.match(ContentType.Text.Html) && type.contentSubtype != "xhtml+xml")) return null
            response.call.request.url.toString() to response.bodyAsText()
        } catch (e: CancellationException) {
            throw e
        } catch (_: Exception) {
            return null
        }
        return withContext(Dispatchers.Default) { Readability.parse(Ksoup.parse(page.second, page.first)) }
    }
}

private const val TIMEOUT_MS = 15_000L
