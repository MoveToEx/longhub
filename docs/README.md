# LONG Hub documentation

This directory contains the VitePress site published at https://docs.longhub.top.
Run these commands from `docs/` with Node.js 24 and Corepack:

```sh
corepack enable
yarn install --immutable
yarn docs:dev
```

Build and preview the production site with `yarn docs:build` and
`yarn docs:preview`. The build checks the OpenAPI specification and fails on
broken internal documentation links.

## Editing

- Guides live directly in this directory; navigation is in `.vitepress/config.mts`.
- `spec/openapi.yaml` is the source of truth for API documentation. After editing
  it, run `yarn docs:spec` and commit the generated `spec/openapi.json` alongside
  the YAML. `yarn docs:check` detects stale JSON, unresolved local references, and
  duplicate operation IDs.
- `examples/` contains the Python download example and its tests. The catalog sync
  page embeds the source directly, so there is only one maintained example
  implementation.
- Run `python -m unittest discover -s examples -p 'test_*.py' -v` from this directory
  to test the example. Its download and tests use only the Python standard library.
- `public/CNAME` preserves the custom domain and is copied into the built site.
  The site's base path is `/` because it uses a custom domain.

## GitHub Pages

The repository workflow `.github/workflows/deploy-docs.yml` validates and builds
documentation on pull requests. Pushes affecting documentation on `main`, and
manual runs on `main`, also deploy the build to GitHub Pages. Pull requests and
manual runs from other branches never deploy.

For the initial migration, configure this repository in GitHub:

1. Under **Settings → Pages → Build and deployment**, select **GitHub Actions**.
2. Set the custom domain to **docs.longhub.top** and retain its DNS configuration.
   If the domain is still assigned to the old documentation repository, remove
   that assignment before adding it here.
3. If the `github-pages` environment has branch restrictions, allow `main`.

The build job has read-only repository access. Only the deployment job receives
Pages and OIDC permissions. Dependency installation uses the pinned Yarn version
and an immutable lockfile; example tests and the site build must pass before a
Pages artifact is uploaded.
