# Allmend Agent Framework

Allmend is a CLI tool for managing and interacting with AI agents. It allows you to define agents, configure LLM providers, and connect tools (MCP servers) to empower your agents.

## Table of Contents

- [Installation](#installation)
- [Configuration](#configuration)
- [Managing Providers](#managing-providers)
- [Managing Tools](#managing-tools)
- [Managing Agents](#managing-agents)
- [Creating Agents](#creating-agents)

## Installation

### Prerequisites

- Go 1.21 or higher
- Make

### Compilation

To compile the project, run:

```bash
make build
```

This will create an `allmend` binary in the current directory.

To install it to `/usr/local/bin` (or your configured `BINDIR`), run:

```bash
sudo make install
```

## Configuration

Allmend uses a configuration file typically located at `~/.config/allmend/allmend.conf` or `./config/allmend.conf`.

You **must** create a `config` directory and an `allmend.conf` file if they don't exist.

**Example `config/allmend.conf`:**

```yaml
# allmend.conf

# Paths to search for .agt files (agent definitions)
agent_paths:
  - "./agents"
  - "/path/to/my/agents"

# Default model to use if not specified via flag
default_model: "gemini-1.5-flash"

# Optional: Override paths for other config files
# models_file: ./config/modells.yaml
# providers_file: ./config/providers.conf
# tools_file: ./config/tools.conf
```

## Managing Providers

Providers connect Allmend to LLM services like Ollama or Google Gemini. Adding a provider automatically syncs available models to `modells.yaml`.

### Add Ollama Provider

To add a local Ollama instance:

```bash
./allmend provider add ollama my-local-ollama --endpoint http://localhost:11434
```

### Add Google Gemini Provider

To add Google Gemini (requires API key):

```bash
export GEMINI_API_KEY="your-api-key"
./allmend provider add google my-gemini --project-id your-project-id --location us-central1
```

Or pass the key directly:

```bash
./allmend provider add google my-gemini --api-key "your-api-key" --project-id your-project-id
```

### List Providers

```bash
./allmend provider list
```

## Managing Tools

Allmend supports the Model Context Protocol (MCP) to connect tools. You can add HTTP or Stdio-based MCP servers.

### Add a Tool Server (Stdio)

For a local tool running via command line (e.g., a Node.js MCP server):

```bash
./allmend tool serveradd my-tool --type stdio --command "npx" --args "-y,@modelcontextprotocol/server-memory"
```

### Add a Tool Server (HTTP)

For a tool server running over HTTP:

```bash
./allmend tool serveradd my-http-tool http://localhost:8080
```

### List Tools

```bash
./allmend tool list
```

## Using Agents

### List Available Agents

```bash
./allmend agent list
```
This scans the directories configured in `agent_paths`.

### Run an Agent

To run an agent:

```bash
./allmend agent run [AGENT_NAME]
```

**Options:**
- `--model [MODEL_NAME]`: Specify the model to use (overrides default).
- `--chat`: Start an interactive chat session with the agent.
- `--yolo`: Disable human-in-the-loop confirmation for tool calls (proceed without asking).

Example:
```bash
./allmend agent run agent-example --model gemini-1.5-pro --chat
```

## Creating Agents

Agents are defined in `.agt` files. These files use a specific format with sections starting with `%`.

### Agent File Structure

- `%Meta`: Metadata about the agent (Name, Version, Description, etc.).
- `%Manifest`: The system prompt/instructions for the agent. This defines its personality and constraints.
- `%Tools`: Header for tool requirements.
- `%Required`: List of tool names the agent must have access to. The tools must be available via configured MCP servers.
- `%Mission`: The initial prompt or task sent to the agent.
- `%Subagent` (Optional): Defines a sub-agent.

### Example 1: Simple File Assistant

Save this as `file-assistant.agt` in your agents directory.

```text
%Meta
Name: file-assistant
Version: 1.0.0
Description: A simple agent that helps with file operations.
Author: Jane Doe

%Manifest
You are a helpful assistant specialized in file management.
Always ask for confirmation before deleting files.
Be concise in your responses.
Verify file paths before attempting operations.

%Tools
%Required
read_file
write_file
list_directory

%Mission
Please list the files in the current directory and create a summary file named 'summary.txt' containing the list.
```

### Example 2: Code Reviewer

Save this as `code-reviewer.agt`.

```text
%Meta
Name: code-reviewer
Version: 0.1.0
Description: An agent that reviews code for potential bugs and style issues.

%Manifest
You are a senior software engineer conducting a code review.
Focus on:
1. Logic errors.
2. Security vulnerabilities.
3. Code style and readability.
4. Missing tests.

Be constructive and explain *why* something is an issue.
Do not rewrite code unless asked, but provide snippets for suggested fixes.

%Tools
%Required
read_file
grep_search

%Mission
I have a file named 'main.go'. Please read it and provide a detailed code review.
```

### Tips for Writing Agents

1.  **Manifest Clarity**: The `%Manifest` section is your "system prompt". Be explicit about the agent's role, constraints, and tone.
2.  **Tool Availability**: Ensure the tools listed in `%Required` are actually available on your system (added via `tool serveradd`). If a tool is missing, the agent will fail to start.
3.  **Mission Scope**: The `%Mission` sets the initial context. For a chat bot, this might just be "Introduce yourself and wait for user input." For a task agent, it should be the specific goal.
