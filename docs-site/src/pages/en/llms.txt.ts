export const prerender = true;

export function GET() {
  return new Response(`Emoera Agent CLI documentation\n\n- Overview: /en/\n- Install: /en/install\n- Quickstart: /en/quickstart\n- Authentication: /en/authentication\n- Commands: /en/commands\n- Agent automation: /en/automation\n- Troubleshooting: /en/troubleshooting\n- Release security: /en/release-security\n- Repository: https://github.com/dijkstra402/emoera-RAG-CLI\n`, {
    headers: { 'Content-Type': 'text/plain; charset=utf-8' }
  });
}
