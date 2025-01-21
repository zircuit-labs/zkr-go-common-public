# Jitter

This is a copy of <https://github.com/Rican7/retry/tree/master/jitter> with minor changes and updates. While the jitter implementations were wonderful, the retry implementation was something we wanted to do differently. Given the low popularity of the original, a copy seemed wiser than an import in this case.

As indicated in the blog post which inspired the original author <https://www.awsarchitectureblog.com/2015/03/backoff.html>, Full Jitter tends to produce the best results.

## Types of Jitter

### None

No jitter at all. The output equals the input.

### Full (recommended)

Provides an output randomly, uniformly distributed in the range [0, n), where n is the given duration.

### Equal

Provides an output randomly, uniformly distributed in the range [n/2, n), where n is the given duration.
