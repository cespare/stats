# stats

A command-line utility to provide basic summary statistics over numbers read
from files or stdin.

`stats` reads from filenames or, if none are given, from stdin.

    $ stats
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

`stats` can also draw you a unicode histogram if you give it the `-hist` flag.
`-buckets N` controls the size of the histogram. This will look
better/worse/broken depending on how your terminal renders box-drawing
characters.

As an example, I have an nginx log file. The last column of the log is the
latency, in seconds. Suppose I want to see what the latency distribution is
like.

```
$ cat nginx.log | awk '{print $NF}' | stats
count           10000.000
min               0.000
max              16.372
mean              0.019
std. dev.         0.222
quantile 0.5      0.002
quantile 0.9      0.030
quantile 0.99     0.290
```

We can see from max and by the difference in the median and the mean that there
are some big outliers. Let's zoom in on the body of the data (by cutting away
the outliers for now) and see what that looks like:

![screenshot](/screenshot.png)

(Using a screenshot to avoid unicode rendering jankiness.)

## Installation

    go get -u github.com/cespare/stats

## Usage

See above examples or use `stats -h`.
