# semver-sugar

[![GitHub release](https://img.shields.io/github/v/release/mikolajmikolajczyk/semver-sugar)](https://github.com/mikolajmikolajczyk/semver-sugar/releases)
[![GitHub issues](https://img.shields.io/github/issues/mikolajmikolajczyk/semver-sugar)](https://github.com/mikolajmikolajczyk/semver-sugar/issues)
[![GitHub license](https://img.shields.io/github/license/mikolajmikolajczyk/semver-sugar)](https://github.com/mikolajmikolajczyk/semver-sugar/blob/main/LICENSE)

`semver-sugar` is a GitHub Action designed to simplify the process of tagging and releasing applications using [Semantic Versioning (SemVer)](https://semver.org/). This action supports multiple release strategies and offers flexibility in handling custom versioning scenarios.

## Features

- **Automated Versioning:** Automatically determine the next version based on SemVer labels in pull requests.
- **Multiple Release Strategies:** Supports creating GitHub releases or just tags.
- **Customizable Tag Formats:** Use custom tag formats to match your project's requirements.
- **Multiple Release Lines:** Handle multiple branches and release lines with ease.
- **Supports GitHub Enterprise:** Configurable API and upload URLs for GitHub Enterprise environments.

## Inputs

| Name                | Description                               | Required | Default             |
|---------------------|-------------------------------------------|----------|---------------------|
| `release_branch`    | Branch to use for release                 | true     | `master`            |
| `release_strategy`  | Release strategy (`release` or `tag` or `none`)     | true     | `release`           |
| `tag_format`        | Format used to create tags                | true     | `v%major%.%minor%.%patch%` |
| `tag`               | Tag to use                                | false    |                     |
| `github_api_url`    | URL to GitHub Enterprise API              | false    |                     |
| `github_uploads_url`| URL to GitHub Enterprise uploads          | false    |                     |
| `custom_release_sha`| SHA to use for custom release             | false    |                     |
| `version_range`     | Version range to use for latest tag       | true     | `>0.0.0`            |

## Outputs

| Name      | Description                          |
|-----------|--------------------------------------|
| `tag`     | Tag created by this action           |
| `increment` | Increment type performed if any     |

## Usage

To use this action in your GitHub workflows, include the following steps:

```yaml
name: Release Workflow

on:
  push:
    branches:
      - master
  pull_request:
    types:
      - closed

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v3

      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: 20

      - name: Run semver-sugar
        uses: mikolajmikolajczyk/semver-sugar@v1
        with:
          release_branch: 'master'
          release_strategy: 'release'
          tag_format: 'v%major%.%minor%.%patch%'
          github_token: ${{ secrets.GITHUB_TOKEN }}
```

## Configuration

### Release Strategies

This action supports the following release strategies:

- **`release`**: Creates a GitHub release with the new version tag.
- **`tag`**: Creates a lightweight tag without a GitHub release.

### Custom Release SHA

If you want to create a release or tag for a specific commit, you can provide a custom SHA using the `custom_release_sha` input.

### Version Range

The `version_range` input allows you to specify a range to use when searching for the latest tag. This is useful for managing multiple release lines.

## Based on semver-release-action

This action is based on [K-Phoen/semver-release-action](https://github.com/K-Phoen/semver-release-action). It builds upon and extends the original functionality, providing additional features and customization options to better suit various workflows and environments.

## Contributing

Contributions are welcome! If you'd like to improve this action, feel free to fork the repository and submit a pull request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

This action was created by [Mikołaj Mikołajczyk](https://github.com/mikolajmikolajczyk) and inspired by the work of [Kevin Dunglas](https://github.com/K-Phoen) in the [semver-release-action](https://github.com/K-Phoen/semver-release-action) project.
