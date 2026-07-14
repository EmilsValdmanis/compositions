import { defineConfig } from "vite-plus";
import { devtools } from "@tanstack/devtools-vite";
import { tanstackStart } from "@tanstack/react-start/plugin/vite";
import viteReact, { reactCompilerPreset } from "@vitejs/plugin-react";
import babelPlugin from "@rolldown/plugin-babel";
import tailwindcss from "@tailwindcss/vite";
import { nitro } from "nitro/vite";
import { sentryTanstackStart } from "@sentry/tanstackstart-react/vite";
import { paraglideVitePlugin } from "@inlang/paraglide-js";

const config = defineConfig({
  fmt: {
    ignorePatterns: ["/src/routeTree.gen.ts"],
  },
  lint: { options: { typeAware: true, typeCheck: true } },
  resolve: { tsconfigPaths: true },
  plugins: [
    paraglideVitePlugin({
      project: "./project.inlang",
      outdir: "./src/paraglide",
      outputStructure: "message-modules",
      emitTsDeclarations: true,
      cookieName: "PARAGLIDE_LOCALE",
      strategy: ["url", "cookie", "preferredLanguage", "baseLocale"],
      urlPatterns: [
        {
          pattern: "/",
          localized: [
            ["en", "/en"],
            ["lv", "/lv"],
          ],
        },
        {
          pattern: "/:path(.*)?",
          localized: [
            ["en", "/en/:path(.*)?"],
            ["lv", "/lv/:path(.*)?"],
          ],
        },
      ],
    }),
    devtools(),
    nitro(),
    tailwindcss(),
    tanstackStart(),
    babelPlugin({
      presets: [reactCompilerPreset()],
    }),
    viteReact(),
    sentryTanstackStart({
      org: "emils-valdmanis",
      project: "frontend",
      sourcemaps: {
        disable: !process.env.SENTRY_AUTH_TOKEN,
      },
      telemetry: false,
      authToken: process.env.SENTRY_AUTH_TOKEN,
    }),
  ],
});

export default config;
