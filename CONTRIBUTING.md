# Contributing

Thank you for your interest in contributing to Datei!

Datei is open-source software licensed under the [AGPL-3.0 license](https://github.com/godatei/datei/blob/main/LICENSE) and accepts contributions via GitHub pull requests.

## How to run datei for development

To run Datei locally, clone the repository and make sure that all necessary tools defined in [mise.toml](mise.toml) are installed.
We recommend that you use [mise](https://mise.jdx.dev/) to install these (run `mise install` in the current directory) but you don't have to.

You can then start the necessary containers and the application with:

```shell
# Start the database and a mock SMTP server
docker compose up -d
# Start Datei
mise watch serve -r
```

Open your browser and navigate to [`http://localhost:8080`](http://localhost:8080).
You can use Mailpit on [`http://localhost:8025`](http://localhost:8025) to receive E-Mail verification links.

## Backporting bugfixes

If the `main` branch already contains changes that would warrant a major or minor version bump but there is need to create a patch release only,
it is possible to backport commits by pushing to the relevant `v*.*.x` branch.
For example, if a commit should be added to version 0.2.1, it must be pushed to the `v0.2.x` branch.

**Important:** Please keep in mind the following rules for backporting:

1. Do not backport changes that would require an inappropriate version bump. For example, do not add new features to the `v0.2.x` branch, only bugfixes.
2. Only backport changes that are already in `main`. Ideally, use `git cherry-pick`.

## Pre-Releases

Creating pre-releases can be useful if you need to test the current state of the main branch in a demo environment or test artifact building.
To do this, create a `v*.*.*-rc.x` branch (for example, `v0.2.0-rc.x`).
For release-please to create a new release branch in the correct format, change `"prerelease": true` and `"versioning": "prerelease"` in the `release-please-config.json`.
This will also automatically mark your GitHub release as pre-release.
