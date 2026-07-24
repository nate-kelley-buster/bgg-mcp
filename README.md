<p align="center">
  <img src="images/bgg-mcp-logo.png" width="200" alt="BGG MCP Logo">
</p>
<h1 align="center">BGG MCP: BoardGameGeek MCP Server</h1>

<p align="center">
  <a href="https://archestra.ai/mcp-catalog/kkjdaniel__bgg-mcp"><img src="https://archestra.ai/mcp-catalog/api/badge/quality/kkjdaniel/bgg-mcp" alt="trust score badge"></a>
  <a href="https://github.com/modelcontextprotocol/registry"><img src="https://img.shields.io/badge/MCP_Registry-BGG_MCP-green" alt="MCP Registry"></a>
  <br>
  <a href="https://go.dev/"><img src="https://img.shields.io/github/go-mod/go-version/kkjdaniel/bgg-mcp" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/kkjdaniel/bgg-mcp" alt="License"></a>
  <a href="https://modelcontextprotocol.io"><img src="https://img.shields.io/badge/MCP-Protocol-blue" alt="MCP Protocol"></a>
</p>

BGG MCP provides access to the BoardGameGeek API through the [Model Context Protocol](https://www.anthropic.com/news/model-context-protocol), enabling retrieval and filtering of board game data, user collections, and profiles. The server is implemented in Go, using the [GoGeek](https://github.com/kkjdaniel/gogeek) library, which helps ensure robust API interactions.

Price data is provided by [BoardGamePrices.co.uk](https://boardgameprices.co.uk), offering real-time pricing from multiple retailers.

Game recommendations are powered by [Recommend.Games](https://recommend.games/), which provides algorithmic similarity recommendations based on BoardGameGeek data.

<a href="https://boardgamegeek.com/">
  <img src="images/powered-bgg.webp" width="160" alt="Powered by BGG">
</a>

## Demo

<div align="center">
  
  [![Rules Tool Demo Video](https://img.youtube.com/vi/cNX4WwVbFko/maxresdefault.jpg)](https://youtu.be/cNX4WwVbFko)
  
  **[▶️ Watch the Rules Tool Demo Video](https://youtu.be/cNX4WwVbFko)**
  
</div>

## Tools

### Core Tools

| Tool                 | Description                                                                 |
| -------------------- | --------------------------------------------------------------------------- |
| `bgg-search`         | Search for board games with type filtering (base games, expansions, or all) |
| `bgg-details`        | Get detailed information about a specific board game                        |
| `bgg-collection`     | Query and filter a user's game collection with extensive filtering options  |
| `bgg-hot`            | Get the current BGG hotness list                                            |
| `bgg-user`           | Get user profile information                                                |
| `bgg-price`          | Get current prices from multiple retailers using BGG IDs                    |
| `bgg-trade-finder`   | Find trading opportunities between two BGG users                            |
| `bgg-recommender`    | Get game recommendations based on similarity to a specific game             |
| `bgg-thread-details` | Get the full content of a specific BGG forum thread including all posts     |
| `bgg-plays`          | Get a user's BGG play history                                               |
| `bgg-game-family`    | Look up board game families/series, or list all games in a family           |
| `bgg-versions`       | Get edition/printing/version data for a game (publishers, languages, dimensions, product codes) |
| `bgg-geeklist`       | Read a BGG GeekList by id (legacy XML API1 — read-only, BGG has no GeekList write API) |

### 🧪 Experimental Tools

| Tool        | Description                                                                                |
| ----------- | ------------------------------------------------------------------------------------------ |
| `bgg-rules` | Answer rules questions by searching BGG forums for relevant discussions and clarifications |
| `bgc-nlp-search` | Natural-language game search via semantic embeddings. **Requires a separate BGCollector backend** (not included in this repo) — will not work without one running. |
| `bgc-graph-recommend` | Graph-based recommendations via a Neo4j knowledge graph. **Requires a separate BGCollector backend** (not included in this repo) — will not work without one running. |

### ✍️ Write Tools

BoardGameGeek has no official write API — these tools use undocumented endpoints BGG's own website calls internally, authenticated via a real login (see [Write Authentication](#write-authentication) below). `bgg-update-collection-status` (the `own` flag) and `bgg-log-play` are confirmed working against a real reference implementation; everything else is a best-effort implementation inferred from BGG's own documented field names and has not yet been verified live — check each tool's description for its confidence level before relying on it.

| Tool                          | Description                                                                 |
| ------------------------------ | ---------------------------------------------------------------------------- |
| `bgg-update-collection-status` | Set collection status flags (own, want, want to play, want to buy, for trade, preordered, previously owned, wishlist + priority) |
| `bgg-remove-from-collection`   | Remove a game from your collection (unverified — see tool description)       |
| `bgg-rate-game`                | Set your rating for a game                                                   |
| `bgg-set-collection-notes`     | Set the private notes/comment field on a collection item                     |
| `bgg-log-play`                 | Log a play to your BGG play history — **confirmed working**                  |
| `bgg-update-play`              | Edit an existing play by play id (unverified)                                |
| `bgg-delete-play`              | Delete a play by play id (unverified)                                        |
| `bgg-batch-update-collection`  | Apply a collection status change to many games in one call, with per-item success/failure reporting |

## Resources

BGG MCP exposes resources that AI assistants can access directly for contextual information:

| Resource            | URI                   | Description                                                      |
| ------------------- | --------------------- | ---------------------------------------------------------------- |
| `BGG Hotness`       | `bgg://hotness`       | Current BGG hotness list, always available                       |
| `My BGG Collection` | `bgg://my-collection` | Your personal BGG collection (requires `BGG_USERNAME` to be set) |

## Prompts

BGG MCP includes pre-configured prompts for common workflows:

| Prompt                   | Description                                                                          |
| ------------------------ | ------------------------------------------------------------------------------------ |
| `Trade Sales Post`       | Generate a formatted sales post for your BGG 'for trade' collection with discounted market prices |
| `Game Recommendations`   | Get personalized game recommendations based on your BGG collection and preferences   |

## Example Prompts

Here are some example prompts you can use to interact with the BGG MCP tools:

### 🔍 Search

```
"Search for Wingspan on BGG"
"How many expansions does Grand Austria Hotel have?"
"Search for Wingspan expansions only"
```

### 📊 Game Details

```
"Get details for Azul"
"Show me information about game ID 224517"
"What's the BGG rating for Gloomhaven?"
```

### 📚 Collection

```
"Show me ZeeGarcia's game collection"
"Show games rated 9+ in kkjdaniel's collection"
"List unplayed games in rahdo's collection"
"Find games for 6 players in kkjdaniel's collection"
"Show me all the games rated 3 and below in my collection"
"What games in my collection does rahdo want?"
"What games does kkjdaniel have that I want?"
```

### 🔥 Hotness

```
"Show me the current BGG hotness list"
"What's trending on BGG?"
```

### 👤 User Profile

```
"Show me details about BGG user rahdo"
"When did user ZeeGarcia join BGG?"
"How many buddies do I have on bgg?"
```

### 💰 Prices

```
"Get the best price for Wingspan in GBP"
"Show me the best UK price for Ark Nova"
"Compare prices for: Wingspan & Ark Nova"
```

### 🎯 Recommendations

```
"Recommend games similar to Wingspan"
"What games are like Azul but with at least 1000 ratings?"
"Find 5 games similar to Troyes"
```

### 📖 Rules (Experimental)

```
"[Your rules question about any board game] - use bgg-rules"
"How does [game mechanic] work in [game name]? use bgg-rules"
"Can I [specific action] in [game name]? use bgg-rules"
"What happens when [situation] in [game name]? use bgg-rules"
```

Note: Include "use bgg-rules" in your question to ensure the AI searches BGG forums for answers.

### ✍️ Collection & Play Writes

```
"Mark Wingspan as owned on my BGG collection"
"Add Ark Nova to my wishlist with priority 2"
"Rate Gloomhaven a 9"
"Remove Small World from my collection"
"Log a play of Azul from tonight, 4 players, I won"
"Mark all of these as owned: Wingspan, Azul, Gloomhaven, Ark Nova"
```

Requires `BGG_PASSWORD` to be set (see [Write Authentication](#write-authentication)). See the [Write Tools](#️-write-tools) table for which of these are confirmed working vs. best-effort.

## Installation

> **Authentication Required**: Most BGG MCP tools require authentication to access BoardGameGeek's API. See the [Configuration section](#configuration) below for setup instructions.

### A) Docker (Recommended)

BGG MCP is published to [Docker Hub](https://hub.docker.com/r/kdaniel/bgg-mcp) and listed on the [MCP Registry](https://github.com/modelcontextprotocol/registry). Add the following to your `claude_desktop_config.json` (Claude Desktop) or `settings.json` (VS Code / Cursor):

```json
"bgg": {
    "command": "docker",
    "args": ["run", "-i", "--rm",
        "-e", "BGG_API_KEY",
        "-e", "BGG_USERNAME",
        "kdaniel/bgg-mcp"
    ],
    "env": {
        "BGG_API_KEY": "your_api_key_here",
        "BGG_USERNAME": "your_bgg_username"
    }
}
```

> See [Configuration](#configuration) below for details on obtaining a BGG API key and setting up your username.

For more details on connecting MCP servers to your client, see the [official MCP guide](https://modelcontextprotocol.io/docs/develop/connect-local-servers).

### B) Manual Setup

#### 1. Install Go

You will need to have Go installed on your system to build binary. This can be easily [downloaded and setup here](https://go.dev/doc/install), or you can use the package manager that you prefer such as Brew.

#### 2. Build

The project includes a Makefile to simplify building and managing the binary.

```bash
# Build the application (output goes to build/bgg-mcp)
make build

# Clean build artifacts
make clean

# Both clean and build
make all
```

Or you can simply build it directly with Go...

```bash
go build -o build/bgg-mcp
```

#### 3. Add MCP Config

In the `settings.json` (VS Code / Cursor) or `claude_desktop_config.json` add the following to your list of servers, pointing it to the binary you created earlier, once you load up your AI tool you should see the tools provided by the server connected:

```json
"bgg": {
    "command": "path/to/build/bgg-mcp",
    "args": ["-mode", "stdio"]
}
```

More details for configuring Claude can be [found here](https://modelcontextprotocol.io/quickstart/user).

## Configuration

### Authentication

BGG MCP v2.0+ uses the GoGeek v2.0 library which requires authentication for reliable access to BoardGameGeek's API.

You can configure authentication using `BGG_API_KEY` (recommended), `BGG_COOKIE`, or — if you don't have an API key yet (BGG's application approval can take several days) — just `BGG_USERNAME`/`BGG_PASSWORD`. If neither an API key nor a cookie is set but both of those are, the server logs in with them once and reuses that session's cookie for read requests too, the same login the [write tools](#️-write-tools) already need — so you don't have to wait on an API key to get started. Precedence: `BGG_API_KEY` > `BGG_COOKIE` > `BGG_USERNAME`/`BGG_PASSWORD` > unauthenticated.

#### Authentication Setup

##### Option 1: API Key (Recommended)

Get an API key from [BoardGameGeek's API application form](https://boardgamegeek.com/applications) and add it to your configuration:

```json
"bgg": {
    "env": {
        "BGG_API_KEY": "your_api_key_here"
    }
}
```

##### Option 2: Cookie Authentication

Alternatively, you can use cookie-based authentication:

```json
"bgg": {
    "env": {
        "BGG_COOKIE": "bggusername=user; bggpassword=pass; SessionID=xyz"
    }
}
```

**Note**: If both are provided, `BGG_API_KEY` will be used by default.

### Username Configuration

You can optionally set the `BGG_USERNAME` environment variable to enable "me" and "my" references in queries without needing to explicitly state your username:

```json
"bgg": {
    "env": {
        "BGG_USERNAME": "your_bgg_username",
        "BGG_API_KEY": "your_api_key_here"
    }
}
```

This enables:

- **Collection queries**: "Show my collection" instead of specifying your username
- **User queries**: "Show my BGG profile"
- **AI assistance**: The AI can automatically use your username for comparisons and analysis

**Note**: When you use self-references (me, my, I) without setting BGG_USERNAME, you'll get a clear error message.

### Write Authentication

The [write tools](#️-write-tools) need a real logged-in BGG session, not just an API key — BGG's write endpoints are undocumented and check for an actual browser-style login. Set both `BGG_USERNAME` and `BGG_PASSWORD`:

```json
"bgg": {
    "env": {
        "BGG_API_KEY": "your_api_key_here",
        "BGG_USERNAME": "your_bgg_username",
        "BGG_PASSWORD": "your_bgg_password"
    }
}
```

- Login for the *write* tools is lazy — nothing touches the network until you actually call one, so leaving `BGG_PASSWORD` unset is fine if you only want read access. (If you set `BGG_USERNAME`/`BGG_PASSWORD` with no API key/cookie, the *read* client does log in once at startup to borrow that session for reads too — see [Authentication](#authentication) above.)
- `BGG_PASSWORD` is **env-only**. Unlike `BGG_API_KEY`/`BGG_COOKIE`/`BGG_USERNAME`, it is deliberately not accepted as an HTTP-mode query parameter, since a password in a URL risks exposure via proxy or access logs.
- These endpoints are unofficial and reverse-engineered — see each write tool's description for whether its exact payload has been confirmed against a live response or is still best-effort.
