# Spike: A Software Network Load Balancer

Spike runs on commodity Linux servers and is based on Google's network
load balancer, [Maglev][0].

[0]: https://research.google.com/pubs/pub44824.html

# Dependencies

* Go 1.8
* gcc (for the preprocessor)
* [`siphash`](https://github.com/dchest/siphash)
* [`snabb`](https://github.com/snabbco/snabb)
* [`testify`](https://github.com/stretchr/testify)
* [`yaml`](https://github.com/go-yaml/yaml)

# Building

* Make sure that your [go workspace](https://golang.org/doc/code.html)
  is set up properly, and that the spike repository is in
  `$GOPATH/src/github.com/sipb/spike`.
* Clone and build the [snabb repository](https://github.com/snabbco/snabb).
  (This is unlikely to work on non-Linux operating systems.)
* Run `go get github.com/dchest/siphash github.com/stretchr/testify gopkg.in/yaml.v2`.
* Run `make`.

It should now be possible to run the health check demo (`bin/demo`), as
well as the snabb integration demo (`bin/runspike`). `runspike` should
be run as root and requires two environment variables, `SNABB` set to
the path to the `snabb` executable and `SPIKE` set to the path to the
folder, so you might run something like:

    sudo env SNABB=/path/to/snabb SPIKE=/path/to/spike bin/runspike

You can run the tests with `make test`.

# Contributing

Contributing guidelines are [here](CONTRIBUTING.md).

# Copyright

Spike is available under the MIT License. See the `LICENSE` file for
more details.

`maglev` was adapted from
[dgryski/go-maglev](https://github.com/dgryski/go-maglev/), which is
used under the terms of the MIT License, and
[kkdai/maglev](https://github.com/kkdai/maglev), which is used under the
terms of the Apache License version 2.0.
