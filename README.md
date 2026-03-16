# AI Pull Request Describer 🤖
Automatically generate descriptions for pull requests using any OpenAI-compatible AI model (ChatGPT, DeepSeek, Groq, Ollama, etc.).

When you open or update a pull request labeled with **"ai-describe"**, this action will automatically generate a concise list of changes and update the pull request description (body) or post a comment.

## Demo / Review 🎬

![AI PR Describer Demo](media/ai-pr-describer.gif)

## Features ✨
- **Multi-Model Support**: Works with OpenAI, DeepSeek, Groq, or any OpenAI-compatible API.
- **Auto-Update PR Body**: Automatically updates the PR description for a cleaner workflow.
- **Customizable Prompt**: Generates categorized summaries (Refactor, Bug Fix, etc.).
- **Go-Powered**: Fast, efficient, and lightweight container-based action.

## Requirements 🛠️
* GitHub API token for API access.
* API key from your chosen AI provider (OpenAI, DeepSeek, etc.).

## Usage 🚀

Add the following workflow file to your repository in `.github/workflows/ai-pr-describer.yml`:

```yaml
name: AI Pull Request Describer

on:
  pull_request:
    types: [reopened, labeled]

jobs:
  describe:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: AI Pull Request Describer
        uses: fajarhide/ai-pr-describer@main
        with:
          github-token: ${{ secrets.GH_TOKEN }}
          github-api-base-url: 'https://api.github.com'
          openai-api-key: ${{ secrets.OPENAI_API_KEY }}
          openai-model: ${{ secrets.OPENAI_MODEL }}
          openai-base-url: ${{ secrets.OPENAI_BASE_URL }}
```

## AI Provider Examples 💡

### OpenAI (Default)
```yaml
openai-model: 'gpt-4o'
```

### DeepSeek
```yaml
openai-api-key: ${{ secrets.OPENAI_API_KEY }}
openai-model: 'deepseek-chat'
openai-base-url: 'https://api.deepseek.com'
```

### Groq
```yaml
openai-api-key: ${{ secrets.OPENAI_API_KEY }}
openai-model: 'llama3-8b-8192'
openai-base-url: 'https://api.groq.com/openai/v1'
```

### Ollama (Local/Self-hosted)
```yaml
openai-api-key: 'ollama' # usually not required but cannot be empty
openai-model: 'llama3'
openai-base-url: 'http://your-ollama-host:11434/v1'
```

## Configuration ⚙️

| Input                  | Required | Description                                                                 |
|------------------------|----------|-----------------------------------------------------------------------------|
| `github-token`         | Yes      | The GitHub API token for accessing the repository.                         |
| `openai-api-key`       | Yes      | The API key for your AI provider.                                   |
| `openai-model`         | No       | The AI model to use. Defaults to `gpt-3.5-turbo`.                        |
| `openai-base-url`      | No       | Custom base URL for OpenAI-compatible APIs.                                |
| `github-api-base-url`  | No       | The base URL for the GitHub API. Defaults to `https://api.github.com`.      |

### Changelog

Please see [CHANGELOG](CHANGELOG.md) for more information what has changed recently.
## Contributing 🤝

Please see [CONTRIBUTING](CONTRIBUTING.md) for details.
## Credits

-   [Fajar Hidayat](https://github.com/fajarhide)
-   [All Contributors](../../contributors)
## License 📄

The MIT License (MIT). Please see [License File](LICENSE) for more information.