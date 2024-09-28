# 🌊 AgentFlow

## 🚀 Introduction

AgentFlow is an innovative AI-powered workflow automation tool that combines the power of language models with customizable action flows. It allows you to create, manage, and execute complex workflows with ease, leveraging the capabilities of AI to process and generate content.

## 🌟 Features

- 🧠 AI-powered workflow execution using OpenAI models
- 🔍 Built-in file indexing and searching capabilities
- 🛠 Customizable actions and flows
- 🔄 Dynamic variable handling
- 🐳 Docker support for easy deployment

## 📋 Prerequisites

Before you begin, ensure you have the following installed:

- Go 1.16 or later
- Make (for using the Makefile)
- Docker (optional, for containerized deployment)

## 🛠 Installation

1. Clone the repository:

   ```
   git clone https://github.com/yourusername/agentflow.git
   cd agentflow
   ```

2. Install dependencies:

   ```
   go mod tidy
   ```

3. Build the project:
   ```
   make build
   ```

## ⚙️ Configuration

1. Create a `.env` file in the project root with the following content:

   ```
   OR_KEY=your_openrouter_api_key
   OR_MODEL_HIGH=gpt-3.5-turbo
   OR_MODEL_LOW=gpt-3.5-turbo
   OPENAI_API_KEY=your_openai_api_key
   ```

2. Replace `your_openrouter_api_key` and `your_openai_api_key` with your actual API keys.

## 🚀 Usage

### Indexing Files

Before running flows, you need to index your files:

```
make index
```

This will index all text files in the current directory and subdirectories, excluding the `.git` and `agentflow.bleve` directories.

### Running a Flow

To start a flow:

```
./agentflow start "Your input here"
```

### Searching Indexed Files

To search indexed files and start a flow with the results:

```
./agentflow search "Your search query" flow_name
```

## 🌊 Flows and Actions

### Flows

Flows are defined in JSON or YAML files in the `flows` directory. Here's an example structure:

```yaml
name: example_flow
model: high
action: [action1, action2]
input:
  type: object
  properties:
    query:
      type: string
output:
  type: object
  properties:
    result:
      type: string
system-prompt: "You are a helpful assistant."
prompt: "Process this query: {USER}"
flow:
  - validate: "input.query.length > 0"
    next: "next_flow"
```

### Actions

Actions are defined in JSON or YAML files in the `actions` directory. They contain JavaScript code that processes the output of a flow.

## 🐳 Docker

To build and run AgentFlow in a Docker container:

```
make docker-build
make docker-run
```

## 📚 Documentation

For more detailed documentation on creating flows and actions, please refer to the `docs` directory.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgements

- OpenAI, Anthropic for their powerful language models
- OpenRouter for the easy access to all models
- The Go community for the excellent libraries used in this project

---

🌟 Happy flowing with AgentFlow! If you have any questions or run into issues, please open an issue on the GitHub repository.
