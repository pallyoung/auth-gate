# Contributing to Auth Gate

Thank you for your interest in contributing! This guide will help you get started.

## Development Setup

### Prerequisites

- Go 1.22+
- Node.js 20+
- npm

### Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/auth-gate.git
   cd auth-gate
   ```

3. Install dependencies:
   ```bash
   # Backend
   cd packages/server
   go mod download

   # Frontend
   cd ../web
   npm install
   ```

4. Run in development mode:
   ```bash
   make dev
   ```

## Development Workflow

### Branch Naming

- `feature/` - New features
- `fix/` - Bug fixes
- `refactor/` - Code refactoring
- `docs/` - Documentation updates

Example: `feature/add-websocket-auth`

### Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Formatting, missing semicolons, etc.
- `refactor`: Code refactoring
- `test`: Adding tests
- `chore`: Maintenance tasks

Examples:
```
feat(auth): add API key authentication
fix(proxy): handle WebSocket upgrade correctly
docs(readme): update installation instructions
```

### Testing

Always write tests for new features:

```bash
# Backend tests
cd packages/server
go test ./...

# Frontend tests
cd packages/web
npm test

# Run all tests
make test
```

### Code Style

#### Go

- Follow standard Go conventions
- Run `golangci-lint` before committing
- Use meaningful variable names
- Add comments for complex logic

#### TypeScript/React

- Use TypeScript strict mode
- Follow the existing component patterns
- Use functional components with hooks
- Keep components small and focused

### Pull Request Process

1. Create a feature branch from `main`
2. Make your changes
3. Add/update tests
4. Update documentation if needed
5. Ensure all tests pass
6. Submit a pull request

### PR Guidelines

- Keep PRs focused on a single feature/fix
- Write clear PR descriptions
- Reference related issues
- Include screenshots for UI changes
- Update CHANGELOG if applicable

## Project Structure

```
auth-gate/
├── packages/
│   ├── server/          # Go backend
│   │   ├── cmd/         # Main entry points
│   │   ├── internal/    # Internal packages
│   │   └── configs/     # Configuration files
│   └── web/             # React frontend
│       ├── src/
│       │   ├── components/
│       │   ├── pages/
│       │   └── lib/
│       └── public/
├── scripts/             # Build and deployment scripts
├── docs/                # Documentation
└── e2e/                 # End-to-end tests
```

## Reporting Issues

- Use the issue templates provided
- Include reproduction steps
- Share relevant logs
- Specify your environment

## Code of Conduct

- Be respectful
- Be constructive
- Focus on the code, not the person
- Help others learn

## Questions?

- Open a discussion on GitHub
- Check existing issues and docs first

Thank you for contributing!
