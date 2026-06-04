import { defineConfig } from "vite-plus";
import { devtools } from "@tanstack/devtools-vite";
import { tanstackStart } from "@tanstack/react-start/plugin/vite";
import viteReact, { reactCompilerPreset } from "@vitejs/plugin-react";
import babelPlugin from "@rolldown/plugin-babel";
import tailwindcss from "@tailwindcss/vite";
import { nitro } from "nitro/vite";
import { sentryTanstackStart } from "@sentry/tanstackstart-react/vite";

const config = defineConfig({
  fmt: {
    ignorePatterns: ["/src/routeTree.gen.ts"],
  },
  lint: { options: { typeAware: true, typeCheck: true } },
  resolve: { tsconfigPaths: true },
  plugins: [
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
        disable: process.env.NODE_ENV !== "production",
      },
      telemetry: false,
      authToken: process.env.SENTRY_AUTH_TOKEN,
    }),
  ],
});

export default config;
