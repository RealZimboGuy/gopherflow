# Contributing to GopherFlow

Thank you for your interest in contributing to GopherFlow! This document provides guidelines and instructions for contributing to this project.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for everyone. Please be considerate in your interactions with other contributors.

## Getting Started

### Prerequisites

- Go 1.24+ (or Docker if you prefer containers)
- Basic understanding of workflow engines and state machines

### Project Setup

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```
   git clone https://github.com/YOUR-USERNAME/gopherflow.git
   cd gopherflow
   ```
3. Add the original repository as an upstream remote:
   ```
   git remote add upstream https://github.com/RealZimboGuy/gopherflow.git
   ```
4. Keep your fork in sync with the upstream repository:
   ```
   git fetch upstream
   git checkout main
   git merge upstream/main
   ```

## Development Workflow

### Building the Project

The project uses [just](https://github.com/casey/just) for task automation. You can build the project using:

```
just build
```

This will build a Docker image with the current version specified in the justfile.

### Running Tests

The project includes integration tests for multiple database backends:

- MySQL
- PostgreSQL
- SQLite

To run tests for a specific database, navigate to the appropriate test directory and run:

```
go test ./...
```

### Code Style

Please follow these guidelines for your code contributions:

1. Follow standard Go code style and conventions
2. Use meaningful variable and function names
3. Write comments for complex logic
4. Add appropriate error handling
5. Use idiomatic Go patterns

## Making Contributions

### Types of Contributions

We welcome the following types of contributions:

1. Bug fixes
2. Feature enhancements
3. Documentation improvements
4. Performance optimizations
5. Test coverage improvements

### Pull Request Process

1. Create a new branch for your feature or bugfix:
   ```
   git checkout -b feature/your-feature-name
   ```
   or
   ```
   git checkout -b fix/your-bugfix-name
   ```

2. Make your changes and commit them with clear, descriptive commit messages that explain the purpose of the change

3. Push your branch to your fork:
   ```
   git push origin your-branch-name
   ```

4. Submit a pull request to the main repository's `main` branch

5. Your pull request will be reviewed by maintainers who may request changes or provide feedback

6. Once your pull request is approved, it will be merged into the main codebase

### Pull Request Requirements

Before submitting a pull request, please ensure:

1. Your code follows the project's code style guidelines
2. All tests pass successfully
3. You've added tests for new functionality
4. Documentation has been updated if necessary
5. Your changes are properly described in the pull request

## Adding Workflows

When adding new workflows to GopherFlow:

1. Implement the `core.Workflow` interface
2. Define clear state transitions
3. Ensure each state function is idempotent
4. Add appropriate error handling
5. Include tests for your workflow

## Reporting Issues

If you find a bug or have a feature request, please create an issue on GitHub with:

1. A clear, descriptive title
2. A detailed description of the issue or feature
3. Steps to reproduce (for bugs)
4. Expected vs. actual behavior (for bugs)
5. Any relevant code snippets or screenshots

## License

By contributing to GopherFlow, you agree that your contributions will be licensed under the project's [Apache License 2.0](LICENSE).

## Questions?

If you have questions about contributing that aren't addressed here, please feel free to open an issue with your question.

Thank you for contributing to GopherFlow!
