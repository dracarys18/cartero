<div align="center">
  <h1>Cartero</h1>
  <img src="docs/assets/postman.png" alt="Cartero Postman" width="200">
</div>

Cartero (Spanish for "postman") is a content aggregation and distribution bot written in Go.

## The Problem
I am someone who likes read tech news on HackerNews, Lobsters or 100's of RSS feed that I have curated over the years. But a lof of the time they have too much content and some of them are not any of my interests. So I built Cartero a personal curator of mine which gets the content from different sources filters it based on my interests, quality of content and pushes it into one single feed of any kind

> **Getting started?** Head over to the [Setup Guide](docs/SETUP.md) for step-by-step instructions.

## Sources

Cartero can pull content from multiple platforms. Here are the available sources:

| Source | Description | Key Settings |
|--------|-----------|--------------|
| **HackerNews** | Top stories and best posts from HackerNews | `story_type`: topstories, bestories, newstories |
| **Lobsters** | Tech-focused community news aggregator | `sort_by`: hot, recent; filter by categories |
| **RSS** | Any RSS/Atom feed URL | `feed_url`: Your feed URL |
| **LessWrong** | Rationality and AI alignment discussions | General feed configuration |

## Targets

Targets define where your content gets posted. You can send content to multiple destinations:

| Target | Type | Description |
|--------|----------|-------------|
| **Discord** | text or forum | Posts content to Discord channels or forum threads |
| **Feed** | RSS/Atom | Exposes your aggregated content as a web feed |

