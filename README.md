# ASSG - Asartalo’s Static Site Generator

[![Coverage Status](https://coveralls.io/repos/github/asartalo/assg/badge.svg)](https://coveralls.io/github/asartalo/assg)

A static site generator written in go custom-built for my needs.

---

*This project is officially hosted at [Codeberg](https://codeberg.org/asartalo/assg) and mirrorred at [GitHub](https://github.com/asartalo/assg).*

## Basic Usage

Examples can be seen under `e2e/fixtures`.

To build the site use the `build` command.

```sh
assg build
```

To run a local server

```sh
assg serve
```

## Configuration

You can configure your site by having a `config.toml` on your root directory. Below is a detailed reference of all available configuration options.

### Core Settings

```toml
# The base URL for your site (required)
base_url = "http://example.com/"

# Site metadata
title = "Test Site"
description = "A test site for ASSG"

# Author name
author = "Jane Doe"

# Content - where your markdown and other content files are located
content_directory = "content"

# Where the generated files will be published
output_directory = "public"

# Draft handling
include_drafts = false
```

### Feed Generation

```toml
# Enable RSS/Atom feed generation
generate_feed = true

# Maximum number of posts in the feed
feed_limit = 10
```

### Site Features

```toml
# Enable XML sitemap generation
sitemap = true

# Define taxonomies for content organization
taxonomies = [{ name = "tags", feed = true }]
```

### Build Hooks

```toml
# Shell commands to run before the build process
prebuild = "sh pre.sh"

# Shell commands to run after the build process
postbuild = "sh post.sh"
```

### Development Server

```toml
[server]
# Port for the development server. Default is 8080.
port = 8080

# List of directories relative to project root to ignore when watching for changes.
watch_ignore = ["sass", "src"]
```

## Development

Testing the local server-related code requires Google chrome for now. Make sure that the chrome binary (e.g. `google-chrome`) is available in your path. Then run:

```sh
go test
```
