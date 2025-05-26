# Contributing to NewsBalancer

Thank you for considering contributing to NewsBalancer! Please take a moment to review these guidelines.

## Getting Started

1.  **Fork the repository** on GitHub.
2.  **Clone your fork** locally: `git clone <your-fork-url>`
3.  **Set up the development environment:** Follow the instructions in the `README.md` under "Development -> Environment Setup".
4.  **Create a new branch** for your feature or bug fix: `git checkout -b <branch-name>` (e.g., `feature/add-new-feed` or `fix/api-cache-bug`).

## Making Changes

1.  Implement your changes.
2.  Add tests for new functionality or bug fixes.
3.  Ensure all tests pass by running the full test suite:
    ```bash
    # Windows
    ./run_all_tests.cmd

    # Linux/macOS
    ./run_all_tests.sh
    ```
    Refer to `docs/testing.md` for more details on testing.
4.  Format your Go code using standard Go tools (e.g., `go fmt ./...`).
5.  Update documentation (`README.md`, files in `docs/`) if your changes affect usage, architecture, or setup.

## Submitting a Pull Request

1.  **Commit your changes** with a clear and concise commit message.
2.  **Push your branch** to your fork: `git push origin <branch-name>`
3.  **Open a Pull Request (PR)** from your fork's branch to the main repository's `main` branch (or the appropriate target branch).
4.  Provide a clear description of the changes in the PR.
5.  Ensure the PR passes any automated checks (CI/CD).
6.  Be prepared to discuss your changes and make further adjustments based on feedback.

Thank you for your contribution!
