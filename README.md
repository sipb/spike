# Spike: A Software Network Load Balancer
Spike runs on commodity Linux servers and is based on Google's network load balancer, Maglev, which can be found [here](https://research.google.com/pubs/pub44824.html).

# Install

```
go get github.com/dchest/siphash
```
# License
spike is available under the MIT license. See `LICENSE` file for more details.

`maglev` was adapted from https://github.com/dgryski/go-maglev/ released under the MIT license.

`maglev` contains modifications inspired by https://github.com/kkdai/maglev released under Apache 2.0.