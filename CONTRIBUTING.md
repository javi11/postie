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


## Updating the Homebrew package

The formula in `Formula/postie.rb` pins the published CLI release archives and
SHA256 digests for macOS and Linux, on ARM64 and AMD64. After publishing a stable
release, regenerate it with Python 3 and the GitHub CLI:

```bash
# Use the tag of the release being packaged.
gh api repos/kipsilabs/postie/releases/tags/vX.Y.Z > /tmp/postie-release.json
python3 scripts/update_homebrew.py < /tmp/postie-release.json
python3 -m unittest discover -s scripts -p 'test_update_homebrew.py'
```

The updater requires all four archives and valid GitHub SHA256 asset digests;
it leaves the formula unchanged if validation fails. Review and commit the
updated formula in a pull request. Homebrew users receive it through `brew update`.
The Homebrew workflow runs updater tests and a real install/test on macOS and Linux.
