# stats

stats is a command-line utility to perform some simple statistical analyses.

The tool is organized into subcommands that are run using `stats [cmd]`.

## Installation

    go get -u github.com/cespare/stats

## Usage

See `stats [cmd] -h` to read more about a particular command, or keep reading.

### summarize

`stats summarize` provides summary statistics over a sequence of numbers. It
reads from filenames or, if none are given, from stdin.

    $ stats summarize
    111
    105
    107
    93
    99
    104
    count            7
    min              93
    max              111
    mean             102.71428571428571
    std. dev.        5.469768491164756
    quantile 0.5     104
    quantile 1.9     107
    quantile 0.99    111

`stats summarize` can also draw you a histogram if you give it the `-hist` flag.
`-buckets N` controls the number of buckets in the histogram. (Note: the
histogram is rendered in your terminal using box-drawing characters and so the
way it looks depends on your terminal emulator and font.)
