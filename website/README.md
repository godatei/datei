# godatei.com

The website for [Datei](https://github.com/godatei/datei), built with [Astro](https://astro.build).

## Project Structure

```
website/
├── public/              # Static assets served as-is
├── src/
│   ├── assets/          # Images processed by Astro
│   ├── content/
│   │   ├── blog/        # Blog posts (md/mdx)
│   │   └── docs/        # Documentation pages (md/mdx)
│   ├── content.config.ts # Content collection schemas
│   ├── layouts/         # Page layouts
│   ├── pages/           # Routes
│   ├── types.ts         # Shared types
│   └── utils/           # Content helpers
├── astro.config.mjs
└── package.json
```

## Commands

All commands are run from the `website/` directory:

| Command        | Action                                     |
| :------------- | :----------------------------------------- |
| `pnpm install` | Install dependencies                       |
| `pnpm dev`     | Start local dev server at `localhost:4321` |
| `pnpm build`   | Build the production site to `./dist/`     |
| `pnpm preview` | Preview the build locally before deploying |
| `pnpm check`   | Type-check the project                     |
| `pnpm lint`    | Check formatting with Prettier             |
| `pnpm format`  | Format all files with Prettier             |
