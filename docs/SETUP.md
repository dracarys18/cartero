# Setup

Cartero is a Go binary that needs Postgres (with pgvector) and Redis. Docker Compose runs all three.

## Prerequisites

- **Docker with Compose.** Runs Postgres, Redis and Cartero.
- **A Jev API key.** Jev, from TypeSafe AI, tags each article with one of your topics and rates its quality. Sign in at [console.typesafe.ai](https://console.typesafe.ai/) (early access), create a key and export it:

  ```bash
  export TYPESAFE_API_KEY=your-key
  ```

  Jev is only required when you set interests. Without a key, disable `[platforms.jev]` and drop the `[interests]` section, and Cartero posts everything that passes the other filters.
- **Ollama** with the embedding model, used to drop headlines that repeat a story already posted:

  ```bash
  ollama pull qwen3-embedding:0.6b
  ```

  An OpenAI-compatible `/embeddings` endpoint works too: set `type = "openai"` in `[platforms.embedder]` and add `base_url` and `api_key` to its settings.
- **Credentials for each target**: a Discord bot token, a Bluesky app password or a Telegram bot token.
- **Go 1.26**, only to build from source.

## Configure

```bash
cp config.sample.toml config.toml
```

`config.sample.toml` documents every option. What you need to fill in:

- `[storage] dsn`: the Postgres connection string.
- `[redis] addr`: defaults to `localhost:6379`.
- `[platforms.*]`: the Jev key, the embedder and target credentials.
- `[sources.*]`: where articles come from. Each source lists the `targets` it posts to.
- `[targets.*]`: where articles go.
- `[interests]`: your keywords file and the Jev thresholds.

Cartero replaces `${VAR}` in `config.toml` with the environment variable of that name, so secrets can stay out of the file. The sample does this for `api_key = "${TYPESAFE_API_KEY}"`.

## Keywords file

The keywords file lists the topics you want to read about. Jev picks the closest topic for each article or decides it matches none of them. Articles whose chance of matching none is at or above `off_topic_threshold` are dropped.

It is a JSON array of objects with these fields:

| Field | Required | Meaning |
|---|---|---|
| `keyword` | yes | The topic name. Accepted articles are tagged with it. |
| `context_string` | no | What the topic covers. Without it Jev only sees the name. |
| `not_for` | no | Nearby subjects that should not count. |
| `examples` | no | Headlines that belong to the topic. |

Entries with the same `keyword` are merged, and each `context_string` adds to what the topic covers.

```json
[
  {
    "keyword": "Rust",
    "context_string": "The Rust programming language, its compiler, tooling and crates",
    "not_for": "Rust the video game, or corrosion",
    "examples": ["Announcing Rust 1.95.0", "Cutting our CI time in half with cargo-nextest"]
  },
  {
    "keyword": "Databases",
    "context_string": "Database internals such as storage engines, indexes and query planners"
  },
  {
    "keyword": "Databases",
    "context_string": "Running Postgres, SQLite and other databases in production"
  }
]
```

Point `keywords_file` at it. It takes a local path or a URL, such as the raw URL of a GitHub gist:

```toml
[interests]
keywords_file = "keywords.json"
```

Cartero reads the file at startup, so restart it after editing. Topics can also go inline in `config.toml` as `[[interests.keywords]]` tables with the same fields, and they are combined with the file.

Some things that help Jev sort articles well:

- Keep one subject per keyword. "Rust" and "Go" work better than "Systems languages".
- Write `context_string` the way you would describe the topic to a person, not as a list of search terms.
- Add `not_for` when articles keep landing in the wrong topic.

## Blocklist

`[blocklist] domains_file` drops every article from the listed domains. It takes a local path or a URL to a text file with one domain per line. Lines starting with `#` are ignored.

## Run with Docker Compose

Inside Compose the services reach each other by name, so change these two settings in `config.toml`:

```toml
[storage]
dsn = "postgres://cartero:cartero@postgres:5432/cartero?sslmode=disable"

[redis]
addr = "redis:6379"
```

Then start it:

```bash
docker compose up -d
```

This runs Postgres, Redis and the `ghcr.io/dracarys18/cartero:latest` image. The web feed is at http://localhost:8034. Compose points Cartero at Ollama on the host through `OLLAMA_HOST=http://host.docker.internal:11434`, so keep Ollama running there.

Two optional profiles add services:

- `--profile reader` runs a self-hosted Jina Reader. Cartero falls back to it when its own extraction gets too little text. Set `reader_url = "http://reader:8081"` under `[processors.extract_text.settings]`.
- `--profile tunnel` runs a Cloudflare Tunnel in front of the feed. Set `TUNNEL_TOKEN` first.

```bash
docker compose --profile reader up -d
```

## Run from source

```bash
docker compose up -d postgres redis
make build
./bin/cartero -config config.toml
```

Keep `localhost` in the DSN and Redis address. Run the binary from the repository root, because it loads database migrations from `db/migrations/postgres` and templates from `templates/`.

## Database

The Postgres image in Compose, `pgvector/pgvector:pg17`, includes pgvector. Cartero runs its migrations on startup.

To start over, remove the Compose volumes. This deletes the Postgres, Redis and Cartero data:

```bash
docker compose down -v
```
