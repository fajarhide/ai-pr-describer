# Changelog

All notable changes to `ai-pr-describer` will be documented in this file

## v1.0.0

### Added
- **Go Rewrite**: Efficient Go implementation.
- **Universal AI Model Support**: Introduced `openai-base-url` for compatibility with any OpenAI-compatible API (ChatGPT, DeepSeek, Groq, local Ollama, etc.).
- **PR Description Auto-Update**: The action now directly updates the Pull Request description (body) with the AI-generated summary.
- **Modern Infrastructure**: Switched to a multi-stage Dockerfile for smaller images and Go modules for dependency management.
- **Improved Prompting**: Redesigned the AI prompt for more structured, information-dense summaries.

## v1.1.0

### Added
- **High-Performance Execution**: Transitioned from a Docker-based action to a **Composite Action** using pre-built binaries. Initial startup time is significantly reduced from ~2 minutes to under 15 seconds.
- **Configurable Limits**: 
    - `max-tokens`: Control the length of the generated PR summary (default: 2000).
    - `max-context-tokens`: Adjust the total context window size (default: 256,000, optimized for large models like GLM).
- **Automated Releases**: Added a GitHub Actions workflow to automatically build and release binaries for various platforms (Linux, macOS) on tag push.
- **Enhanced Debugging**: Integrated `::debug::` logs to help troubleshoot and monitor execution times for different stages.

### Improved
- **AI Summary Quality**: Updated the core prompt format to include specialized sections like 🚀 Summary, 🔑 Key Changes, and ⚠️ Breaking Changes.
- **Documentation**: Comprehensive update to `README.md` with modern examples and performance tips.


