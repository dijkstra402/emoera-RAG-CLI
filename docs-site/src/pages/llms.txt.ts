export function GET() {
  const body = `# Emoera Agent CLI

> Official command-line client for the Emoera RAG knowledge platform. Designed for humans, scripts and AI agents.

## Canonical resources

- Documentation: https://dijkstra402.github.io/emoera-RAG-CLI/
- Source: https://github.com/dijkstra402/emoera-RAG-CLI
- Releases: https://github.com/dijkstra402/emoera-RAG-CLI/releases
- Security policy: https://github.com/dijkstra402/emoera-RAG-CLI/blob/main/SECURITY.md

## Documentation

- Installation: https://dijkstra402.github.io/emoera-RAG-CLI/install
- Quick start: https://dijkstra402.github.io/emoera-RAG-CLI/quickstart
- Authentication: https://dijkstra402.github.io/emoera-RAG-CLI/authentication
- Command reference: https://dijkstra402.github.io/emoera-RAG-CLI/commands
- Agent automation: https://dijkstra402.github.io/emoera-RAG-CLI/automation
- Troubleshooting: https://dijkstra402.github.io/emoera-RAG-CLI/troubleshooting
- Release security: https://dijkstra402.github.io/emoera-RAG-CLI/release-security

## Minimal workflow

1. Create an Agent API Token in Personal Center > Agent API.
2. Run: emoera config init
3. Run: emoera auth set-token
4. Verify: emoera whoami
5. Ask: emoera ask "Summarize the deployment guide"

For automation, prefer --json or --jsonl, pass a stable --request-id and inject EMOERA_API_TOKEN through a secret manager.
`;
  return new Response(body, {
    headers: { 'Content-Type': 'text/plain; charset=utf-8' }
  });
}
