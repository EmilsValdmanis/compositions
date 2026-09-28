import { defineConfig } from "vite-plus";

// Vite+ 1.0 uses workspace-root lint and format settings, including from frontend/.
export default defineConfig({
  fmt: {
    ignorePatterns: ["/frontend/src/routeTree.gen.ts"],
    sortTailwindcss: {
      stylesheet: "./frontend/src/styles.css",
      functions: ["cn", "cva"],
    },
  },
  lint: {
    options: { typeAware: true, typeCheck: true },
    overrides: [
      {
        files: ["frontend/**/*.test.{ts,tsx}"],
        plugins: ["vitest"],
        rules: {
          "vitest/no-focused-tests": "error",
          "vitest/valid-expect": "error",
          // Keep this rollout focused on focused tests and assertion validity.
          "vitest/no-conditional-expect": "off",
          "vitest/require-mock-type-parameters": "off",
        },
      },
    ],
  },
});
