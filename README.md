# reedsolomon

A simple and efficient Reed-Solomon erasure coding implementation.

The `ReedSolomon` type does the encoding and decoding, and is supported
by `Matrix`, which does matrix arithmetic, and the 8-bit Galois field in
`galois.go`.

`ReedSolomon` only implements the Reed-Solomon erasure coding itself.
Splitting a byte stream into shards, padding it to a shard boundary,
and allocating the shard buffers are all the caller's job. The encoding
and decoding methods work in place on caller-provided buffers. This keeps
the hot paths allocation-free and leaves storage layout and memory ownership 
to each caller.

## Roadmap

- [ ] SIMD and CPU cache optimizations for the coding loops: SIMD
  implementations of the Galois-field multiply (AVX2/AVX-512, NEON),
  plus cache-aware loop blocking so tables and shard buffers stay
  resident in L1/L2 cache instead of being re-streamed from memory.
- [ ] Simple and streaming encoder/decoders built on top of `ReedSolomon`,
  taking over the caller-side work that the core deliberately leaves
  out: splitting, padding, and buffer allocation.

## Performance Notes

The performance of the inner loop depends on the specific processor
you're running on.  There are twelve different permutations of the loop
in this library (see `AllCodingLoops`), and the benchmarks will tell
you which one is faster for your particular application:

```
go test -bench . -benchtime 2s
```

The hot paths are allocation-free: `EncodeParity` and
`IsParityCorrect` perform zero heap allocations per call, and
`DecodeMissing` is allocation-free in steady state — it caches the
matrix inversion per pattern of missing shards, so repeated decodes
with the same pattern skip the O(k^3) inversion entirely. The
benchmarks report allocations (`b.ReportAllocs`), so you can verify
this against your own workload.
