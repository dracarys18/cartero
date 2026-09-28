package app.cartero.data.feed

import com.fleeksoft.ksoup.Ksoup
import io.ktor.client.HttpClient
import io.ktor.client.request.get
import io.ktor.client.request.header
import io.ktor.client.statement.bodyAsText
import io.ktor.http.Headers
import io.ktor.http.HttpHeaders
import io.ktor.http.HttpStatusCode
import io.ktor.http.isSuccess
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.io.IOException

sealed interface FetchResult {
    data class Feed(val url: String, val feed: ParsedFeed, val etag: String?, val lastModified: String?) : FetchResult
    data class Page(val feedUrls: List<String>) : FetchResult
    data object NotModified : FetchResult
}

class FeedFetcher(private val http: HttpClient) {
    suspend fun fetch(url: String, etag: String? = null, lastModified: String? = null): FetchResult {
        val response = http.get(url) {
            header(HttpHeaders.Accept, ACCEPT)
            etag?.let { header(HttpHeaders.IfNoneMatch, it) }
            lastModified?.let { header(HttpHeaders.IfModifiedSince, it) }
        }
        if (response.status == HttpStatusCode.NotModified) return FetchResult.NotModified
        if (!response.status.isSuccess()) throw IOException("HTTP ${response.status.value}")

        val finalUrl = response.call.request.url.toString()
        val body = response.bodyAsText()
        return withContext(Dispatchers.Default) {
            when (sniff(body)) {
                Format.Html -> FetchResult.Page(discover(body, finalUrl))
                Format.Json -> feed(finalUrl, JsonFeedParser.parse(body), response.headers)
                Format.Xml -> feed(finalUrl, XmlFeedParser.parse(body), response.headers)
            }
        }
    }

    private fun feed(url: String, feed: ParsedFeed, headers: Headers) =
        FetchResult.Feed(url, feed, headers[HttpHeaders.ETag], headers[HttpHeaders.LastModified])

    private fun discover(html: String, baseUrl: String): List<String> =
        Ksoup.parse(html, baseUrl)
            .select("link[rel=alternate][href]")
            .filter { link -> FEED_TYPES.any { link.attr("type").contains(it) } }
            .map { it.absUrl("href") }
            .filter { it.isNotEmpty() }
            .distinct()

    private fun sniff(body: String): Format {
        val head = body.take(512)
        val trimmed = head.trimStart('﻿', ' ', '\n', '\r', '\t')
        return when {
            trimmed.startsWith("{") -> Format.Json
            trimmed.startsWith("<!doctype html", ignoreCase = true) || trimmed.startsWith("<html", ignoreCase = true) -> Format.Html
            HTML_HINT.containsMatchIn(head) && !FEED_HINT.containsMatchIn(head) -> Format.Html
            else -> Format.Xml
        }
    }

    private enum class Format { Xml, Json, Html }

    private companion object {
        const val ACCEPT = "application/rss+xml, application/atom+xml, application/feed+json, " +
            "application/xml;q=0.9, text/xml;q=0.9, application/json;q=0.8, text/html;q=0.7, */*;q=0.5"
        val FEED_TYPES = listOf("rss", "atom", "feed+json")
        val HTML_HINT = Regex("<(html|head|body)\\b", RegexOption.IGNORE_CASE)
        val FEED_HINT = Regex("<(rss|feed|rdf:RDF)\\b", RegexOption.IGNORE_CASE)
    }
}
