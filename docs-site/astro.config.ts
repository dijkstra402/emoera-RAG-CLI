import { defineConfig } from "astro/config";
import icon from "astro-icon";
import tailwindcss from "@tailwindcss/vite";
import nimbus, { defineConfig as defineNimbusConfig } from "@cloudflare/nimbus-docs";
import { tableScroll } from "@cloudflare/nimbus-docs/markdown";

const nimbusConfig = defineNimbusConfig({
  site: "https://dijkstra402.github.io/emoera-RAG-CLI",
  title: "Emoera Agent CLI",
  description: "面向开发者与 Agent 的安全知识库命令行工具。",
  locale: "zh-CN",
  homeLabel: "首页",
  github: "https://github.com/dijkstra402/emoera-RAG-CLI",
  editPattern:
    "https://github.com/dijkstra402/emoera-RAG-CLI/edit/main/docs-site/{path}",
  socialImageAlt: "Emoera Agent CLI documentation",
  sidebar: {
    scope: "section",
    indexDisplay: "overview-leaf",
    overviewLabel: "概览 / Overview",
    items: [
      {
        label: "中文",
        icon: "ph:book-open-text",
        autogenerate: { directory: "zh" },
      },
      {
        label: "English",
        icon: "ph:translate",
        autogenerate: { directory: "en" },
      },
    ],
  },
});

export default defineConfig({
  site: "https://dijkstra402.github.io",
  base: "/emoera-RAG-CLI/",
  output: "static",
  // Tailwind v4 via its Vite plugin (the integration Astro recommends for
  // Tailwind v4 — replaces the PostCSS plugin, which doesn't build under
  // Astro 7's Vite 8 bundler).
  vite: {
    plugins: [tailwindcss()],
  },
  // Hover-prefetch link targets so full-page navigations feel instant without
  // a client-side router.
  prefetch: {
    prefetchAll: true,
    defaultStrategy: "hover",
  },
  integrations: [
    icon(),
    nimbus(nimbusConfig, {
      // Authoring rules are opt-in by design — your repo, your taste. The
      // two below are the load-bearing pair: frontmatter has to validate
      // against the content schema for the page to render properly, and
      // broken internal links are 404s for your readers. Add the others
      // (heading hierarchy, code-block language, style, etc.) when you're
      // ready to enforce them — see `nimbus-docs lint --help`.
      rules: {
        "nimbus/frontmatter-shape": "error",
        "nimbus/internal-link": "error",
      },
      // Wrap wide tables so they scroll instead of overflowing the page
      // (styled by `.nb-table-scroll` in src/styles/prose.css).
      markdown: {
        hastPlugins: [tableScroll()],
      },
    }),
  ],
});
