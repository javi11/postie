# Coverage Setup Instructions

This document explains how to set up code coverage reporting for the Postie project.

## Codecov Integration

To enable automatic coverage reporting with Codecov:

1. **Sign up for Codecov**: Go to [codecov.io](https://codecov.io) and sign up with your GitHub account.

2. **Add your repository**: Navigate to your Codecov dashboard and add the `javi11/postie` repository.

3. **Get your Codecov token**: Once the repository is added, Codecov will provide you with a repository token.

4. **Add the token to GitHub Secrets**:

   - Go to your GitHub repository settings
   - Navigate to Secrets and Variables → Actions
   - Add a new secret with the name `CODECOV_TOKEN` and paste your Codecov token as the value

5. **Update the README badge**: Replace `YOUR_CODECOV_TOKEN` in the README.md file with your actual Codecov token (this is safe to expose in the badge URL).

## Alternative: GitHub Actions Coverage Badge

If you prefer not to use Codecov, you can use a simpler GitHub Actions-based coverage badge:

1. Replace the codecov badge in README.md with:

```markdown
[![Coverage](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/YOUR_USERNAME/GIST_ID/raw/coverage.json)](https://github.com/javi11/postie/actions/workflows/coverage.yml)
```

2. You'll need to create a workflow that uploads coverage data to a GitHub Gist.

## Running Coverage Locally

To generate coverage reports locally:

```bash
# Generate coverage profile
make coverage

# View coverage as HTML
make coverage-html
open coverage.html

# View coverage summary
make coverage-func
```

## Coverage Goals

- Maintain at least 80% code coverage
- All new features should include comprehensive tests
- Critical paths should have near 100% coverage

## Excluded Files

The following files/patterns are typically excluded from coverage:

- Generated files (mocks, protobuf)
- Main functions
- Test files themselves
- Vendor dependencies
