# Changes

Make any nontrivial changes on a feature branch, and create a pull
request on GitHub.  Pull requests should be reviewed by the maintainer
before being merged.  Ensure that your changes do not break the build or
tests.

ikdc <ikdc@mit.edu> is the current maintainer.

# Style

Ensure that none of `gofmt`, `govet`, `golint` complains about your code.
For non-Go code, make sure it is well-formatted according to your best
judgment.  Please spell-check your comments.  In addition, try to wrap
lines of code to 80 characters.


You can use [pre-commit](http://pre-commit.com/) to verify that your code
passes `gofmt`, `golint`, and `go vet`'s muster and a few other checks.
To install `pre-commit`, follow the instructions there or run
`pip install pre-commit` or `brew install pre-commit`; then add them to
your local git config with `pre-commit install`.
