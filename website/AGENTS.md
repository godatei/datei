# Agent Instructions

- For imports of Astro components or TypeScript data, always use `~/` and not relative paths.
- Don't do any shortcuts by adding files to `.prettierignore` or similar.
- Always use `pnpm` for package management and execute commands using `pnpm run`, don't use `npm` or `yarn`.
- Blog subheadings should always include a specific reference to the topic rather than using generic titles like "Architecture", "How It Works", or "Implementation". For example, use "How event sourcing works in Datei" instead of "How It Works".
- The docs are a custom content collection rendered by our own layouts (no Starlight). Keep components basic and unstyled for now.
