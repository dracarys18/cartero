# Setup Guide

## Prerequisites

Make sure you have the following installed:

- **Go** 1.18 or higher - [Install Go](https://golang.org/doc/install)
- **Make** - Usually comes pre-installed on macOS and Linux
- **Git** - [Install Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/yourusername/cartero.git
cd cartero
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Build the Project

```bash
make build
```

The executable will be created in the `bin/` directory.

## Configuration

### Create Your Config File

```bash
cp config.sample.toml config.toml
```

Open `config.toml` and customize:

- **Sources**: Which platforms to fetch content from (HackerNews, Lobsters, etc.)
- **Processors**: How to filter and transform content (score thresholds, keywords, categories)
- **Targets**: Where to post content (Discord channels, forums, etc.)

Check `config.sample.toml` for all available options and examples.

## Running Cartero

### Local Run

```bash
./bin/cartero -config config.toml
```

### Using Docker

```bash
docker-compose up
```

## Database

Cartero uses SQLite (`cartero.db`) to track posted content and prevent duplicates. The database is created automatically on first run.

To reset the database:

```bash
rm cartero.db
./bin/cartero -config config.toml
```

That's it! Head back to the [README](../README.md) for an overview of features.