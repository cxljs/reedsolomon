// Benchmark of Reed-Solomon encoding.
//
//	go test -bench . -benchtime 2s
//
// Counts the number of bytes of input data that can be processed per
// second.

package reedsolomon

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

const (
	benchmarkDataCount   = 17
	benchmarkParityCount = 3
	benchmarkTotalCount  = benchmarkDataCount + benchmarkParityCount
	benchmarkBufferSize  = 200 * 1000
	benchmarkCacheSize   = 10 * 1024 * 1024
	benchmarkBufferSets  = 2*benchmarkCacheSize/benchmarkDataCount/benchmarkBufferSize + 1
)

func makeBenchmarkBufferSets() [][][]byte {
	random := rand.New(rand.NewSource(0))
	sets := make([][][]byte, benchmarkBufferSets)
	for i := range sets {
		set := make([][]byte, benchmarkTotalCount)
		for j := range set {
			set[j] = make([]byte, benchmarkBufferSize)
			for k := range set[j] {
				set[j][k] = byte(random.Intn(256))
			}
		}
		sets[i] = set
	}
	return sets
}

func codingLoopName(codingLoop CodingLoop) string {
	name := fmt.Sprintf("%T", codingLoop)
	// %T prints the type with the package's short name, which does
	// not change when the module path changes.
	name = strings.TrimPrefix(name, "*reedsolomon.")
	return strings.TrimSuffix(name, "CodingLoop")
}

func BenchmarkEncodeParity(b *testing.B) {
	sets := makeBenchmarkBufferSets()
	for _, codingLoop := range AllCodingLoops {
		codec, err := NewWithCodingLoop(benchmarkDataCount, benchmarkParityCount, codingLoop)
		if err != nil {
			b.Fatalf("NewWithCodingLoop: %v", err)
		}
		b.Run(codingLoopName(codingLoop), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(benchmarkBufferSize * benchmarkDataCount)
			nextBuffer := 0
			for b.Loop() {
				if err := codec.EncodeParity(sets[nextBuffer], 0, benchmarkBufferSize); err != nil {
					b.Fatalf("EncodeParity: %v", err)
				}
				nextBuffer = (nextBuffer + 1) % len(sets)
			}
		})
	}
}

func BenchmarkIsParityCorrect(b *testing.B) {
	sets := makeBenchmarkBufferSets()

	// The buffer sets hold random data in the parity shards, so encode
	// the parity once before checking. Every coding loop produces the
	// same parity (see TestCodingLoopsProduceSameAnswers), so the
	// checks below are valid for all of the codecs.
	encodeCodec, err := New(benchmarkDataCount, benchmarkParityCount)
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	for _, set := range sets {
		if err := encodeCodec.EncodeParity(set, 0, benchmarkBufferSize); err != nil {
			b.Fatalf("EncodeParity: %v", err)
		}
	}

	tempBuffer := make([]byte, benchmarkBufferSize)
	for _, codingLoop := range AllCodingLoops {
		codec, err := NewWithCodingLoop(benchmarkDataCount, benchmarkParityCount, codingLoop)
		if err != nil {
			b.Fatalf("NewWithCodingLoop: %v", err)
		}
		b.Run(codingLoopName(codingLoop), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(benchmarkBufferSize * benchmarkDataCount)
			nextBuffer := 0
			for b.Loop() {
				correct, err := codec.IsParityCorrectWithBuffer(
					sets[nextBuffer], 0, benchmarkBufferSize, tempBuffer)
				if err != nil {
					b.Fatalf("IsParityCorrectWithBuffer: %v", err)
				}
				// if the parity is not correct, it will throw off the
				// benchmarking because it may return early.
				if !correct {
					b.Fatal("parity not correct")
				}
				nextBuffer = (nextBuffer + 1) % len(sets)
			}
		})
	}
}

// BenchmarkDecodeMissing measures reconstructing one missing data shard
// per iteration. The missing shard rotates so the decode matrix cache
// is exercised across different patterns.
func BenchmarkDecodeMissing(b *testing.B) {
	sets := makeBenchmarkBufferSets()
	codec, err := New(benchmarkDataCount, benchmarkParityCount)
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	for _, set := range sets {
		if err := codec.EncodeParity(set, 0, benchmarkBufferSize); err != nil {
			b.Fatalf("EncodeParity: %v", err)
		}
	}
	shardPresent := make([]bool, benchmarkTotalCount)
	for i := range shardPresent {
		shardPresent[i] = true
	}
	b.Run("OneDataShard", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(benchmarkBufferSize * benchmarkDataCount)
		nextBuffer := 0
		for b.Loop() {
			missing := nextBuffer % benchmarkDataCount
			set := sets[nextBuffer]
			clear(set[missing])
			shardPresent[missing] = false
			if err := codec.DecodeMissing(set, shardPresent, 0, benchmarkBufferSize); err != nil {
				b.Fatalf("DecodeMissing: %v", err)
			}
			shardPresent[missing] = true
			nextBuffer = (nextBuffer + 1) % len(sets)
		}
	})
}
