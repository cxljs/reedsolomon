package reedsolomon

import (
	"bytes"
	"errors"
	"math/rand"
	"sync"
	"testing"
)

// TestZeroSizeEncode does a zero size encode.
func TestZeroSizeEncode(t *testing.T) {
	codec, err := New(2, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	shards := make([][]byte, 3)
	for i := range shards {
		shards[i] = make([]byte, 0)
	}
	if err := codec.EncodeParity(shards, 0, 0); err != nil {
		t.Fatalf("EncodeParity: %v", err)
	}
}

// TestNewValidation checks that invalid shard counts and a nil coding
// loop are rejected with errors instead of panicking.
func TestNewValidation(t *testing.T) {
	tests := []struct {
		name         string
		data, parity int
		codingLoop   CodingLoop
		wantSentinel error
	}{
		{"zero data shards", 0, 3, &InputOutputByteTableCodingLoop{}, nil},
		{"negative data shards", -1, 3, &InputOutputByteTableCodingLoop{}, nil},
		{"negative parity shards", 2, -1, &InputOutputByteTableCodingLoop{}, nil},
		{"more than 256 shards", 200, 100, &InputOutputByteTableCodingLoop{}, ErrTooManyShards},
		{"nil coding loop", 2, 2, nil, nil},
	}
	for _, tt := range tests {
		codec, err := NewWithCodingLoop(tt.data, tt.parity, tt.codingLoop)
		if err == nil {
			t.Fatalf("%s: expected an error", tt.name)
		}
		if codec != nil {
			t.Fatalf("%s: codec should be nil on error", tt.name)
		}
		if tt.wantSentinel != nil && !errors.Is(err, tt.wantSentinel) {
			t.Fatalf("%s: got error %q, want it to match %q", tt.name, err, tt.wantSentinel)
		}
	}
}

// TestValidConfigurations checks that boundary shard configurations are
// accepted and can encode.
func TestValidConfigurations(t *testing.T) {
	for _, tc := range [][2]int{
		{1, 0},
		{1, 255},
		{256, 0},
		{17, 3},
	} {
		codec, err := New(tc[0], tc[1])
		if err != nil {
			t.Fatalf("New(%d, %d): %v", tc[0], tc[1], err)
		}
		shards := make([][]byte, tc[0]+tc[1])
		for i := range shards {
			shards[i] = make([]byte, 4)
		}
		if err := codec.EncodeParity(shards, 0, 4); err != nil {
			t.Fatalf("New(%d, %d): EncodeParity: %v", tc[0], tc[1], err)
		}
	}
}

// TestInputValidationErrors checks that the encoding and checking
// methods return errors matching the sentinel errors for invalid
// arguments.
func TestInputValidationErrors(t *testing.T) {
	codec, err := New(2, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	shards := make([][]byte, 3)
	for i := range shards {
		shards[i] = make([]byte, 4)
	}

	tests := []struct {
		name         string
		call         func() error
		wantSentinel error
	}{
		{
			"wrong number of shards",
			func() error { return codec.EncodeParity(make([][]byte, 2), 0, 4) },
			ErrWrongShardCount,
		},
		{
			"shards of different sizes",
			func() error {
				bad := [][]byte{make([]byte, 4), make([]byte, 4), make([]byte, 3)}
				return codec.EncodeParity(bad, 0, 4)
			},
			ErrShardSizes,
		},
		{
			"negative offset",
			func() error { return codec.EncodeParity(shards, -1, 4) },
			nil,
		},
		{
			"buffers too small",
			func() error { return codec.EncodeParity(shards, 3, 2) },
			ErrShortBuffer,
		},
		{
			"temp buffer too small",
			func() error {
				_, err := codec.IsParityCorrectWithBuffer(shards, 0, 4, make([]byte, 2))
				return err
			},
			nil,
		},
	}
	for _, tt := range tests {
		err := tt.call()
		if err == nil {
			t.Fatalf("%s: expected an error", tt.name)
		}
		if tt.wantSentinel != nil && !errors.Is(err, tt.wantSentinel) {
			t.Fatalf("%s: got error %q, want it to match %q", tt.name, err, tt.wantSentinel)
		}
	}
}

// TestDecodeMissingValidation checks the argument validation in
// DecodeMissing.
func TestDecodeMissingValidation(t *testing.T) {
	codec, err := New(2, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// shardPresent must have one entry per shard.
	shards := [][]byte{make([]byte, 4), make([]byte, 4), make([]byte, 4)}
	if err := codec.DecodeMissing(shards, make([]bool, 2), 0, 4); err == nil {
		t.Fatal("DecodeMissing with a short shardPresent should fail")
	}

	// Fewer than dataShardCount present shards cannot be decoded.
	shards = [][]byte{make([]byte, 4), make([]byte, 4), make([]byte, 4)}
	err = codec.DecodeMissing(shards, []bool{true, false, false}, 0, 4)
	if !errors.Is(err, ErrTooFewShards) {
		t.Fatalf("DecodeMissing with too few shards: got %v, want %v", err, ErrTooFewShards)
	}
}

// TestConcurrentUse runs decodes with different missing patterns from
// multiple goroutines against one codec, verifying that ReedSolomon is
// safe for concurrent use (the decode matrix cache is mutex-guarded).
// Run with -race.
func TestConcurrentUse(t *testing.T) {
	codec, err := New(5, 5)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	shardSize := 100
	allShards := make([][]byte, 10)
	for i := range allShards {
		allShards[i] = make([]byte, shardSize)
		for b := range allShards[i] {
			allShards[i][b] = byte(i*31 + b)
		}
	}
	if err := codec.EncodeParity(allShards, 0, shardSize); err != nil {
		t.Fatalf("EncodeParity: %v", err)
	}

	var wg sync.WaitGroup
	for goroutine := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 100 {
				shards := make([][]byte, 10)
				for j := range shards {
					shards[j] = bytes.Clone(allShards[j])
				}
				present := []bool{true, true, true, true, true, true, true, true, true, true}
				missing := (goroutine + i) % 10
				clear(shards[missing])
				present[missing] = false
				if err := codec.DecodeMissing(shards, present, 0, shardSize); err != nil {
					t.Errorf("DecodeMissing: %v", err)
					return
				}
				for j := range shards {
					if !bytes.Equal(shards[j], allShards[j]) {
						t.Errorf("shard %d differs", j)
						return
					}
				}
			}
		}()
	}
	wg.Wait()
}

func TestOneEncode(t *testing.T) {
	for _, codingLoop := range AllCodingLoops {
		codec, err := NewWithCodingLoop(5, 5, codingLoop)
		if err != nil {
			t.Fatalf("NewWithCodingLoop: %v", err)
		}
		shards := make([][]byte, 10)
		shards[0] = []byte{0, 1}
		shards[1] = []byte{4, 5}
		shards[2] = []byte{2, 3}
		shards[3] = []byte{6, 7}
		shards[4] = []byte{8, 9}
		shards[5] = make([]byte, 2)
		shards[6] = make([]byte, 2)
		shards[7] = make([]byte, 2)
		shards[8] = make([]byte, 2)
		shards[9] = make([]byte, 2)
		if err := codec.EncodeParity(shards, 0, 2); err != nil {
			t.Fatalf("EncodeParity: %v", err)
		}
		expected := [][]byte{
			{12, 13}, {10, 11}, {14, 15}, {90, 91}, {94, 95},
		}
		for i, want := range expected {
			if !bytes.Equal(shards[5+i], want) {
				t.Fatalf("%T: shards[%d] = %v, want %v", codingLoop, 5+i, shards[5+i], want)
			}
		}

		correct, err := codec.IsParityCorrect(shards, 0, 2)
		if err != nil || !correct {
			t.Fatalf("%T: parity should be correct", codingLoop)
		}
		shards[8][0] += 1
		correct, err = codec.IsParityCorrect(shards, 0, 2)
		if err != nil || correct {
			t.Fatalf("%T: parity should not be correct", codingLoop)
		}
	}
}

// TestSimpleEncodeDecode tries a simple case of encoding and decoding.
func TestSimpleEncodeDecode(t *testing.T) {
	dataShards := [][]byte{
		{0, 1},
		{1, 2},
		{1, 3},
		{2, 4},
		{3, 5},
	}
	runEncodeDecode(t, 5, 5, dataShards)
}

// TestBigEncodeDecode tries encoding and decoding with a lot of shards.
func TestBigEncodeDecode(t *testing.T) {
	random := rand.New(rand.NewSource(0))
	dataCount := 64
	parityCount := 64
	shardSize := 200
	dataShards := make([][]byte, dataCount)
	for i := range dataShards {
		dataShards[i] = make([]byte, shardSize)
		for j := range dataShards[i] {
			dataShards[i][j] = byte(random.Intn(256))
		}
	}
	runEncodeDecode(t, dataCount, parityCount, dataShards)
}

// runEncodeDecode encodes a set of data shards, and then tries decoding
// using all possible subsets of the encoded shards.
//
// Uses 5+5 coding, so there must be 5 input data shards.
func runEncodeDecode(t *testing.T, dataCount, parityCount int, dataShards [][]byte) {
	t.Helper()

	totalCount := dataCount + parityCount
	shardLength := len(dataShards[0])

	// Make the list of data and parity shards.
	if len(dataShards) != dataCount {
		t.Fatalf("expected %d data shards, got %d", dataCount, len(dataShards))
	}
	dataLength := len(dataShards[0])
	allShards := make([][]byte, totalCount)
	for i := range dataCount {
		allShards[i] = bytes.Clone(dataShards[i][:dataLength])
	}
	for i := dataCount; i < totalCount; i++ {
		allShards[i] = make([]byte, dataLength)
	}

	// Encode.
	codec, err := New(dataCount, parityCount)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := codec.EncodeParity(allShards, 0, dataLength); err != nil {
		t.Fatalf("EncodeParity: %v", err)
	}

	// Make a copy to decode with.
	testShards := make([][]byte, totalCount)
	shardPresent := make([]bool, totalCount)
	for i := range totalCount {
		testShards[i] = bytes.Clone(allShards[i][:shardLength])
		shardPresent[i] = true
	}

	// Decode with 0, 1, ..., parityCount shards missing.
	for numberMissing := range parityCount + 1 {
		tryAllSubsetsMissing(t, codec, allShards, testShards, shardPresent, numberMissing)
	}
}

func tryAllSubsetsMissing(t *testing.T, codec *ReedSolomon,
	allShards, testShards [][]byte,
	shardPresent []bool, numberMissing int) {
	t.Helper()

	shardLength := len(allShards[0])
	subsets := allSubsets(numberMissing, 0, 10)
	for _, subset := range subsets {
		// Get rid of the shards specified by this subset.
		for _, missingShard := range subset {
			clear(testShards[missingShard])
			shardPresent[missingShard] = false
		}

		// Reconstruct the missing shards
		if err := codec.DecodeMissing(testShards, shardPresent, 0, shardLength); err != nil {
			t.Fatalf("DecodeMissing: %v", err)
		}

		// Check the results.  After checking, the contents of testShards
		// is ready for the next test, the next time through the loop.
		checkShards(t, allShards, testShards)

		// Put the "present" flags back
		for i := range codec.TotalShardCount() {
			shardPresent[i] = true
		}
	}
}

// TestCodingLoopsProduceSameAnswers checks that all of the coding
// loops produce the same results.
func TestCodingLoopsProduceSameAnswers(t *testing.T) {
	dataCount := 5
	parityCount := 5
	shardSize := 2000
	random := rand.New(rand.NewSource(0))

	// Make a set of input data shards
	dataShards := make([][]byte, dataCount)
	for i := range dataShards {
		dataShards[i] = make([]byte, shardSize)
		for iByte := range dataShards[i] {
			dataShards[i][iByte] = byte(random.Intn(256))
		}
	}

	// Make a reference set of parity shards using an arbitrary coding
	// loop.
	defaultCodec, err := New(dataCount, parityCount)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	expectedParityShards := computeParityShards(t, dataShards, defaultCodec)

	// Check that all coding loops produce the same set of parity
	// shards.
	for _, codingLoop := range AllCodingLoops {
		codec, err := NewWithCodingLoop(dataCount, parityCount, codingLoop)
		if err != nil {
			t.Fatalf("NewWithCodingLoop: %v", err)
		}
		actualParityShards := computeParityShards(t, dataShards, codec)
		for i := range parityCount {
			if !bytes.Equal(expectedParityShards[i], actualParityShards[i]) {
				t.Fatalf("%T: parity shard %d differs", codingLoop, i)
			}
		}
	}
}

// computeParityShards computes parity for the given data shards and
// returns an array of the resulting parity shards.
func computeParityShards(t *testing.T, dataShards [][]byte, codec *ReedSolomon) [][]byte {
	t.Helper()

	shardSize := len(dataShards[0])
	totalShardCount := codec.TotalShardCount()
	dataShardCount := codec.DataShardCount()
	parityShardCount := codec.ParityShardCount()

	parityShards := make([][]byte, parityShardCount)
	for i := range parityShards {
		parityShards[i] = make([]byte, shardSize)
	}

	allShards := make([][]byte, totalShardCount)
	for iShard := range totalShardCount {
		if iShard < dataShardCount {
			allShards[iShard] = dataShards[iShard]
		} else {
			allShards[iShard] = parityShards[iShard-dataShardCount]
		}
	}

	if err := codec.EncodeParity(allShards, 0, shardSize); err != nil {
		t.Fatalf("EncodeParity: %v", err)
	}

	tempBuffer := make([]byte, shardSize)
	allShards[parityShardCount-1][0] += 1
	correct, err := codec.IsParityCorrect(allShards, 0, shardSize)
	if err != nil || correct {
		t.Fatalf("parity should not be correct (no buffer)")
	}
	correct, err = codec.IsParityCorrectWithBuffer(allShards, 0, shardSize, tempBuffer)
	if err != nil || correct {
		t.Fatalf("parity should not be correct (with buffer)")
	}
	allShards[parityShardCount-1][0] -= 1
	correct, err = codec.IsParityCorrect(allShards, 0, shardSize)
	if err != nil || !correct {
		t.Fatalf("parity should be correct (no buffer)")
	}
	correct, err = codec.IsParityCorrectWithBuffer(allShards, 0, shardSize, tempBuffer)
	if err != nil || !correct {
		t.Fatalf("parity should be correct (with buffer)")
	}

	return parityShards
}

func checkShards(t *testing.T, expectedShards, actualShards [][]byte) {
	t.Helper()

	if len(expectedShards) != len(actualShards) {
		t.Fatalf("expected %d shards, got %d", len(expectedShards), len(actualShards))
	}
	for i := range expectedShards {
		if !bytes.Equal(expectedShards[i], actualShards[i]) {
			t.Fatalf("shard %d differs", i)
		}
	}
}

// allSubsets returns a list of arrays with all possible sets of unique
// values where (min <= n < max).
//
// This is NOT EFFICIENT, because it allocates lots of temporary
// arrays, but it's OK for these tests.
//
// To avoid duplicates that are in a different order, each subset is
// generated with elements in increasing order.
//
// Given (n=2, min=1, max=4), returns:
//
//	[1, 2]
//	[1, 3]
//	[1, 4]
//	[2, 3]
//	[2, 4]
//	[3, 4]
func allSubsets(n, min, max int) [][]int {
	var result [][]int
	if n == 0 {
		result = append(result, []int{})
	} else {
		for i := min; i < max-n; i++ {
			prefix := []int{i}
			for _, suffix := range allSubsets(n-1, i+1, max) {
				result = append(result, appendIntArrays(prefix, suffix))
			}
		}
	}
	return result
}

func appendIntArrays(a, b []int) []int {
	result := make([]int, 0, len(a)+len(b))
	result = append(result, a...)
	result = append(result, b...)
	return result
}
