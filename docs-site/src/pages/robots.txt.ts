import { config } from "virtual:nimbus/config";
import { withBasePath } from "../lib/base-path";

export const prerender = true;

export function GET() {
  const body = [
    "User-agent: *",
    "Allow: /",
    "",
    `Sitemap: ${new URL(withBasePath("/sitemap-index.xml"), new URL(config.site).origin).href}`,
    "",
  ].join("\n");

  return new Response(body, {
    headers: { "Content-Type": "text/plain; charset=utf-8" },
  });
}
