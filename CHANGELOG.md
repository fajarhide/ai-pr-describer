# Changelog

All notable changes to `ai-pr-describer` will be documented in this file

## v1.0.0

### Added
- **Go Rewrite**: Replaced PHP logic with a fast, efficient Go implementation.
- **Universal AI Model Support**: Introduced `openai-base-url` for compatibility with any OpenAI-compatible API (ChatGPT, DeepSeek, Groq, local Ollama, etc.).
- **PR Description Auto-Update**: The action now directly updates the Pull Request description (body) with the AI-generated summary.
- **Modern Infrastructure**: Switched to a multi-stage Dockerfile for smaller images and Go modules for dependency management.


