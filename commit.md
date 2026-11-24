# Commit convention

Heartless uses [Conventional Commits](https://www.conventionalcommits.org/).
CI reads these messages to compute the next semantic version automatically, so
every commit that lands on `main` must follow the format below.

## Format

```
<type>(<scope>): <subject>

[optional body]

[optional footer(s)]
```

- **type** — required, one of the types in the table below.
- **scope** — optional noun for the area touched, e.g. `ui`, `history`, `ci`,
  `storage`, `settings`.
- **subject** — imperative present tense, lower case, no trailing period,
  at most 72 characters.
- **body** — optional; explain *why* the change was made, not *what* changed.
- **footer** — optional; issue references or `BREAKING CHANGE:`.

## Types and their effect on the version

| Type | Use for | Release |
| --- | --- | --- |
| `feat` | A new user-facing feature | **minor** (`0.1.0 → 0.2.0`) |
| `fix` | A bug fix | **patch** (`0.1.0 → 0.1.1`) |
| `perf` | A performance improvement | **patch** |
| `revert` | Undoing a previous change | **patch** |
| `docs` | Documentation only | none |
| `chore` | Maintenance, dependencies, housekeeping | none |
| `style` | Formatting, whitespace, no behaviour change | none |
| `refactor` | Code change that neither fixes nor adds a feature | none |
| `test` | Adding or fixing tests | none |
| `build` | Build system or toolchain changes | none |
| `ci` | CI configuration changes | none |

Any commit marked as breaking triggers a **major** bump (`0.1.0 → 1.0.0`),
regardless of type.

## Breaking changes

Mark a breaking change either with `!` after the type/scope:

```
feat(api)!: replace the response payload format
```

or with a `BREAKING CHANGE:` footer in the body:

```
refactor(storage): rename history columns

BREAKING CHANGE: existing history rows need a migration.
```

## How CI derives the version

1. The baseline is the latest `vX.Y.Z` tag. If there is no tag, the version in
   `mygo.json` is used and published as the first release.
2. CI scans every commit since the baseline and applies the **highest** bump
   found (`major > minor > patch`).
3. If no commit calls for a bump, no release is created.
4. Otherwise the build is stamped with the new version, and a GitHub release
   and tag `vX.Y.Z` are created with artifacts for Windows, macOS and Linux.

Accumulating several `fix` commits still yields one patch release; one `feat`
among them yields a minor release; a single breaking commit yields a major one.

## Examples

```
feat(history): load the last response when a request is opened
fix(sidebar): keep request rows clickable after a search
perf(history): fetch only response IDs for the sidebar tint
docs: document the release workflow
refactor(ui)!: replace the request pane with a split layout
```

## Template

Copy this as the start of every commit message:

```
<type>(<scope>): <subject>

<why this change is needed>

<issue reference or BREAKING CHANGE: ...>
```

A pull request title should use the same `<type>(<scope>): <subject>` form,
since squash merges carry that title into `main`.
