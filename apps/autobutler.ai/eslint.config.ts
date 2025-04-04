import svrbcConfig from "@svrbc/eslint-config";
import { withNuxt } from "./.nuxt/eslint.config.mjs";

export default withNuxt(...svrbcConfig, {
  rules: {
    "n/no-unsupported-features/node-builtins": [
      "error",
      {
        version: ">=20.0.0",
        ignores: ["crypto"],
      },
    ],
  },
});
