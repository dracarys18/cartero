# Cartero

Cartero (Spanish for "postman") is a content aggregation and distribution bot written in Go.

## What it does

Cartero fetches content from multiple sources like HackerNews and Lobsters, processes it through customizable filters and transformations, and posts it to Discord channels or forums.

## Features

- Fetch from multiple sources simultaneously
- Filter content by score, keywords, or categories
- Deduplicate posts automatically
- Post to Discord text channels or forum threads
- Configure everything via TOML files
- SQLite-based tracking to prevent duplicate posts

## Usage

```bash
make build
./bin/cartero -config config.toml
```

## Configuration

Copy `config.sample.toml` to `config.toml` and configure:

- Sources: Which platforms to fetch from
- Processors: How to filter and transform content
- Targets: Where to post the content

See `config.sample.toml` for examples.