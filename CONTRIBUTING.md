# Contributing

## Submitting a Pull Request

A typical workflow is:

1. Create a topic branch.
2. Add tests for your change.
3. Add the new code.
4. Run `make` to ensure it passes all the checks.
5. Add, commit and push your changes.
6. Submit a pull request.

## Testing

No assert library is used in the codebase. Instead, use the standard library `testing` package.

Ensure that your tests have descriptive names and are easily understandable. Make sure that failure messages from tests offer sufficient information for understanding the cause of failure. Utilize tables and Fuzz testing when appropriate.

Take inspiration from the numerous examples of tests available in the codebase. Aim for achieving 100% test coverage whenever feasible.

The project aims to follow Google's [testing guidelines](https://google.github.io/styleguide/go/decisions.html#useful-test-failures).
