# Contributing Guide

Thank you for considering contributing to this project. Your help is greatly appreciated!

## Getting Started

Before you begin, please take a moment to review the [README](README.md) for an overview of the project.
Familiarize yourself with the following steps and guidelines.

## Code Contributions

- Ensure your local Git configuration (`user.name` and `user.email`) matches your GitHub profile.
- [Sign your commits](https://gist.github.com/Beneboe/3183a8a9eb53439dbee07c90b344c77e)
- Use clear, concise commit messages (under 80 characters, or under 120 character lines for body)

## Development Environment

```shell
git clone git@github.com:exokomodo/exoflow
# Local
make setup
# Docker compose (for a dev environment)
docker compose up
# devcontainer: use whatever tool you use here, including Github Codespaces
```

## Branching and Pull Requests

- Create a new branch for your changes.
- Make sure your branch is up-to-date with the `main` branch.
- Open a pull request describing your changes briefly.
- A maintainer or team member will review your changes and provide feedback.

## Communication

- [Create an issue](https://github.com/ExoKomodo/exoflow/issues/new) if you find a bug or have an enhancement suggestion.
- [Ask a question in the Discussions](https://github.com/ExoKomodo/exoflow/discussions) if you need to know something
a bit more specific or discuss a larger topic.
    - If a thread evolves into something actionable, an [issue can be created from a discussion](https://github.com/orgs/community/discussions/2861#discussioncomment-696235)
    - Don't feel the need to only start discussions that may become issues.

## Additional Guidelines

- Keep contributions minimal and focused.
    - We do linear commit history with (generally) single commit pull requests.
    - Sometimes we will accept a multi-commit PR as a linear rebase of commits.
    - [This guide](https://www.bitsnbites.eu/a-tidy-linear-git-history/) explains our reasoning quite completely.
- Avoid guidelines that are overly specific to a particular technology or tool chain.
- Follow the project's existing structure and naming conventions.

## Epilogue

Thank you for your support and happy coding!
