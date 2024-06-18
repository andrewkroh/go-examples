# GenAI GitHub issue summarizer

This is a quick experiment with using generative AI to summarize GitHub issues.

# Setup

1. Create a fine-grained github API token that allows reading issues.
2. `export GITHUB_TOKEN=<github token>`
3. Setup local AWS credentials for the Go SDK to access.
4. Ensure that your AWS account has AWS Bedrock model access enabled for
   Anthropic Claude 3.5 Sonnet (or whatever model you specify with `-model`).
