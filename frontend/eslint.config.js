import js from "@eslint/js";
import ts from "@typescript-eslint/eslint-plugin";
import tsParser from "@typescript-eslint/parser";
import prettier from "eslint-config-prettier";
import svelte from "eslint-plugin-svelte";
import globals from "globals";
import svelteParser from "svelte-eslint-parser";

const unusedVarsRule = [
  "error",
  { varsIgnorePattern: "^_", argsIgnorePattern: "^_" },
];

export default [
  js.configs.recommended,
  ...svelte.configs["flat/recommended"],
  prettier,
  ...svelte.configs["flat/prettier"],
  {
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node,
      },
    },
    rules: {
      "no-unused-vars": unusedVarsRule,
    },
  },
  {
    files: ["**/*.ts"],
    plugins: { "@typescript-eslint": ts },
    languageOptions: {
      parser: tsParser,
    },
    rules: {
      ...ts.configs.recommended.rules,
      "no-undef": "off",
      "no-unused-vars": "off",
      "@typescript-eslint/no-unused-vars": unusedVarsRule,
    },
  },
  {
    files: ["**/*.svelte"],
    plugins: { "@typescript-eslint": ts },
    languageOptions: {
      parser: svelteParser,
      parserOptions: {
        parser: tsParser,
      },
    },
    rules: {
      "no-undef": "off",
      "no-unused-vars": "off",
      "@typescript-eslint/no-unused-vars": unusedVarsRule,
    },
  },
  {
    ignores: [".svelte-kit/", "build/", "dist/", "node_modules/"],
  },
];
