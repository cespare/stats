# stats

A command-line utility to provide basic summary statistics over numbers read from files or stdin.

Stats reads from filenames or, if none are given, from stdin.

    $ ./stats
    100
    111
    105
    107
    93
    99
    104
    ^D
    count             7.000
    mean            102.714
    std. dev.         5.470
    quantile 0.5    100.000
    quantile 0.9    107.000
    quantile 0.99   107.000

Note that quantiles are ϵ-approximate for ϵ = 0.01 using the algorithm described in [this
paper](http://www.cs.rutgers.edu/~muthu/bquant.pdf).

## Installation

    go get -u github.com/cespare/stats
