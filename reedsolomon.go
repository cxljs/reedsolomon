package reedsolomon

import (
	"errors"
	"fmt"
	"sync"
)

var (
	// ErrTooManyShards is returned when more than 256 total shards
	// are requested. The 8-bit Galois field cannot represent more.
	ErrTooManyShards = errors.New("too many shards - max is 256")

	// ErrTooFewShards is returned by DecodeMissing when fewer than
	// dataShardCount shards are present, which is not enough to
	// reconstruct the missing ones.
	ErrTooFewShards = errors.New("not enough shards present")

	// ErrWrongShardCount is returned when the shards argument does
	// not have exactly dataShardCount + parityShardCount entries.
	ErrWrongShardCount = errors.New("wrong number of shards")

	// ErrShardSizes is returned when the shard buffers do not all
	// have the same length.
	ErrShardSizes = errors.New("shards are different sizes")

	// ErrShortBuffer is returned when offset + byteCount does not
	// fit within the shard buffers.
	ErrShortBuffer = errors.New("buffers too small")
)

// ReedSolomon does Reed-Solomon coding over 8-bit values.
//
// A ReedSolomon is safe for concurrent use by multiple goroutines: the
// encoding state is immutable, and the decode matrix cache is guarded
// by a mutex.
type ReedSolomon struct {
	dataShardCount   int
	parityShardCount int
	totalShardCount  int
	matrix           *Matrix
	codingLoop       CodingLoop

	// parityRows are the rows from the matrix for encoding parity. They
	// alias the immutable matrix, so encoding does not copy them.
	parityRows [][]byte

	// decodeCache memoizes, for a given pattern of present shards, the
	// matrix rows that reconstruct the data shards. Repeated decodes
	// with the same missing shards then skip the O(dataShardCount^3)
	// matrix inversion. Guarded by mu; the cached rows are owned by
	// the cache and must not be modified.
	mu          sync.Mutex
	decodeCache map[shardMask][][]byte
}

// maxDecodeCacheEntries bounds the memory held by the decode cache.
// Each entry holds dataShardCount rows of dataShardCount bytes, so the
// worst case (256 data shards) is maxDecodeCacheEntries*64KiB. When the
// cache fills it is discarded and rebuilt; the cost of losing an entry
// is one extra matrix inversion.
const maxDecodeCacheEntries = 128

// shardMask is a bitmap of which shards are present, for up to 256
// shards.
type shardMask [4]uint64

func (m *shardMask) set(i int) {
	m[i/64] |= 1 << (uint(i) % 64)
}

// shardScratch holds the little index slices that DecodeMissing
// assembles per call. They hold only slice headers pointing at the
// caller's shard buffers, so pooling them keeps DecodeMissing
// allocation-free in steady state.
type shardScratch struct {
	subShards  [][]byte
	outputs    [][]byte
	matrixRows [][]byte
}

// shardScratchPool pools shardScratch values. Following the usual pool
// discipline, every reference the scratch held is dropped before it
// goes back in, so the pool never retains the caller's buffers.
var shardScratchPool = sync.Pool{
	New: func() any {
		return &shardScratch{}
	},
}

func getShardScratch(dataCount, parityCount int) *shardScratch {
	s := shardScratchPool.Get().(*shardScratch)
	if cap(s.subShards) < dataCount {
		s.subShards = make([][]byte, dataCount)
	}
	s.subShards = s.subShards[:dataCount]
	if cap(s.outputs) < parityCount {
		s.outputs = make([][]byte, parityCount)
	}
	s.outputs = s.outputs[:parityCount]
	if cap(s.matrixRows) < parityCount {
		s.matrixRows = make([][]byte, parityCount)
	}
	s.matrixRows = s.matrixRows[:parityCount]
	return s
}

func putShardScratch(s *shardScratch) {
	// Clear up to capacity: entries beyond the current length may
	// still hold references from an earlier, larger user of the
	// scratch.
	clear(s.subShards[:cap(s.subShards)])
	clear(s.outputs[:cap(s.outputs)])
	clear(s.matrixRows[:cap(s.matrixRows)])
	shardScratchPool.Put(s)
}

// New creates a ReedSolomon codec with the default coding loop
// (InputOutputByteTableCodingLoop).
//
// dataShardCount must be at least 1, parityShardCount must not be
// negative, and their sum must not exceed 256; otherwise New returns
// an error.
func New(dataShardCount, parityShardCount int) (*ReedSolomon, error) {
	return NewWithCodingLoop(dataShardCount, parityShardCount, &InputOutputByteTableCodingLoop{})
}

// NewWithCodingLoop initializes a new encoder/decoder with a chosen
// coding loop. codingLoop must not be nil.
//
// dataShardCount must be at least 1 and parityShardCount must not be
// negative. We can have at most 256 shards total, as any more would
// lead to duplicate rows in the Vandermonde matrix, which would then
// lead to duplicate rows in the built matrix below. Then any subset of
// the rows containing the duplicate rows would be singular.
func NewWithCodingLoop(dataShardCount, parityShardCount int, codingLoop CodingLoop) (*ReedSolomon, error) {
	if dataShardCount < 1 {
		return nil, fmt.Errorf("dataShardCount must be at least 1, got %d", dataShardCount)
	}
	if parityShardCount < 0 {
		return nil, fmt.Errorf("parityShardCount must not be negative, got %d", parityShardCount)
	}
	if 256 < dataShardCount+parityShardCount {
		return nil, ErrTooManyShards
	}
	if codingLoop == nil {
		return nil, errors.New("codingLoop is nil")
	}

	r := &ReedSolomon{
		dataShardCount:   dataShardCount,
		parityShardCount: parityShardCount,
		codingLoop:       codingLoop,
		totalShardCount:  dataShardCount + parityShardCount,
	}
	r.matrix = buildMatrix(dataShardCount, r.totalShardCount)
	r.parityRows = make([][]byte, parityShardCount)
	for i := range parityShardCount {
		r.parityRows[i] = r.matrix.rowSlice(dataShardCount + i)
	}
	r.decodeCache = make(map[shardMask][][]byte)
	return r, nil
}

func (r *ReedSolomon) DataShardCount() int {
	return r.dataShardCount
}

func (r *ReedSolomon) ParityShardCount() int {
	return r.parityShardCount
}

func (r *ReedSolomon) TotalShardCount() int {
	return r.totalShardCount
}

// EncodeParity encodes parity for a set of data shards.
//
// shards is an array containing data shards followed by parity shards.
// Each shard is a byte array, and they must all be the same size.
// offset is the index of the first byte in each shard to encode, and
// byteCount is the number of bytes to encode in each shard.
func (r *ReedSolomon) EncodeParity(shards [][]byte, offset, byteCount int) error {
	// Check arguments.
	if err := r.checkBuffersAndSizes(shards, offset, byteCount); err != nil {
		return err
	}

	// Do the coding. The coding loops only touch the first
	// parityShardCount entries of the output slice, so the parity
	// section of shards can be passed in place instead of being
	// copied into a new slice.
	r.codingLoop.CodeSomeShards(
		r.parityRows,
		shards, r.dataShardCount,
		shards[r.dataShardCount:], r.parityShardCount,
		offset, byteCount)
	return nil
}

// IsParityCorrect returns true if the parity shards contain the right
// data.
//
// shards is an array containing data shards followed by parity shards.
// Each shard is a byte array, and they must all be the same size.
// firstByte is the index of the first byte in each shard to check, and
// byteCount is the number of bytes to check in each shard.
func (r *ReedSolomon) IsParityCorrect(shards [][]byte, firstByte, byteCount int) (bool, error) {
	return r.IsParityCorrectWithBuffer(shards, firstByte, byteCount, nil)
}

// IsParityCorrectWithBuffer returns true if the parity shards contain
// the right data.
//
// This method may be significantly faster than the one above that does
// not use a temporary buffer, for coding loops that take advantage of
// the buffer.
//
// tempBuffer is a temporary buffer (the same size as each of the
// shards) to use when computing parity. It may be nil, in which case
// this is the same as IsParityCorrect.
func (r *ReedSolomon) IsParityCorrectWithBuffer(shards [][]byte, firstByte, byteCount int, tempBuffer []byte) (bool, error) {
	// Check arguments.
	if err := r.checkBuffersAndSizes(shards, firstByte, byteCount); err != nil {
		return false, err
	}
	if tempBuffer != nil && len(tempBuffer) < firstByte+byteCount {
		return false, errors.New("tempBuffer is not big enough")
	}

	// Do the checking. The coding loops only touch the first
	// parityShardCount entries of the toCheck slice, so the parity
	// section of shards can be checked in place instead of being
	// copied into a new slice.
	return r.codingLoop.CheckSomeShards(
		r.parityRows,
		shards, r.dataShardCount,
		shards[r.dataShardCount:], r.parityShardCount,
		firstByte, byteCount,
		tempBuffer), nil
}

// DecodeMissing fills in the shards that don't have data, given a list
// of shards, some of which contain data.
//
// Quickly does nothing if all of the shards are present.
//
// shardPresent must have exactly totalShardCount entries, one per
// shard. The shards that are not present must still be allocated
// buffers of the same size as the present ones. If any shards are
// missing (based on the flags in shardPresent), the data in those
// shards is recomputed and filled in.
func (r *ReedSolomon) DecodeMissing(shards [][]byte, shardPresent []bool, offset, byteCount int) error {
	// Check arguments.
	if err := r.checkBuffersAndSizes(shards, offset, byteCount); err != nil {
		return err
	}
	if len(shardPresent) != r.totalShardCount {
		return fmt.Errorf("shardPresent has %d entries, want %d", len(shardPresent), r.totalShardCount)
	}

	// Quick check: are all of the shards present?  If so, there's
	// nothing to do.
	numberPresent := 0
	var present shardMask
	for i := range r.totalShardCount {
		if shardPresent[i] {
			numberPresent += 1
			present.set(i)
		}
	}
	if numberPresent == r.totalShardCount {
		// Cool.  All of the shards have data.  We don't need to do
		// anything.
		return nil
	}

	// More complete sanity check
	if numberPresent < r.dataShardCount {
		return ErrTooFewShards
	}

	// Fetch (or build and cache) the matrix rows that reconstruct the
	// data shards from the present ones.
	dataDecodeRows := r.dataDecodeRows(present, shardPresent)

	// Gather the present shards in the same order as the rows of the
	// decode matrix: they are the input to the data pass below.
	scratch := getShardScratch(r.dataShardCount, r.parityShardCount)
	defer putShardScratch(scratch)
	subMatrixRow := 0
	for matrixRow := 0; matrixRow < r.totalShardCount && subMatrixRow < r.dataShardCount; matrixRow++ {
		if shardPresent[matrixRow] {
			scratch.subShards[subMatrixRow] = shards[matrixRow]
			subMatrixRow += 1
		}
	}

	// Re-create any data shards that were missing.
	//
	// The input to the coding is all of the shards we actually have,
	// and the output is the missing data shards. The computation is
	// done using the cached decode matrix rows.
	//
	// outputs and matrixRows are sized parityShardCount, which is
	// enough for both passes: the number of missing data shards can't
	// exceed the number of present parity shards (because at least
	// dataShardCount shards are present in total), and the number of
	// missing parity shards is at most parityShardCount.
	outputCount := 0
	for iShard := range r.dataShardCount {
		if !shardPresent[iShard] {
			scratch.outputs[outputCount] = shards[iShard]
			scratch.matrixRows[outputCount] = dataDecodeRows[iShard]
			outputCount += 1
		}
	}
	r.codingLoop.CodeSomeShards(
		scratch.matrixRows,
		scratch.subShards, r.dataShardCount,
		scratch.outputs, outputCount,
		offset, byteCount)

	// Now that we have all of the data shards intact, we can compute
	// any of the parity that is missing.
	//
	// The input to the coding is ALL of the data shards, including any
	// that we just calculated.  The output is whichever of the parity
	// shards were missing.
	outputCount = 0
	for iShard := r.dataShardCount; iShard < r.totalShardCount; iShard++ {
		if !shardPresent[iShard] {
			scratch.outputs[outputCount] = shards[iShard]
			scratch.matrixRows[outputCount] = r.parityRows[iShard-r.dataShardCount]
			outputCount += 1
		}
	}
	r.codingLoop.CodeSomeShards(
		scratch.matrixRows,
		shards, r.dataShardCount,
		scratch.outputs, outputCount,
		offset, byteCount)
	return nil
}

// dataDecodeRows returns, for the given pattern of present shards, one
// row per data shard: multiplying the present shards (in the order
// gathered by DecodeMissing) by row i reconstructs data shard i. Rows
// for a pattern are computed on first use, then cached; the returned
// slices are owned by the cache and must not be modified.
func (r *ReedSolomon) dataDecodeRows(present shardMask, shardPresent []bool) [][]byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rows, ok := r.decodeCache[present]; ok {
		return rows
	}

	// Build the sub-matrix whose rows generate the present shards from
	// the original data, and invert it to go back to the data shards.
	subMatrix := NewMatrix(r.dataShardCount, r.dataShardCount)
	subMatrixRow := 0
	for matrixRow := 0; matrixRow < r.totalShardCount && subMatrixRow < r.dataShardCount; matrixRow++ {
		if shardPresent[matrixRow] {
			for c := range r.dataShardCount {
				subMatrix.Set(subMatrixRow, c, r.matrix.Get(matrixRow, c))
			}
			subMatrixRow += 1
		}
	}
	dataDecodeMatrix := subMatrix.Invert()

	rows := make([][]byte, r.dataShardCount)
	for i := range rows {
		rows[i] = dataDecodeMatrix.GetRow(i)
	}
	if len(r.decodeCache) >= maxDecodeCacheEntries {
		r.decodeCache = make(map[shardMask][][]byte)
	}
	r.decodeCache[present] = rows
	return rows
}

// checkBuffersAndSizes checks the consistency of arguments passed to
// public methods.
func (r *ReedSolomon) checkBuffersAndSizes(shards [][]byte, offset, byteCount int) error {
	// The number of buffers should be equal to the number of data
	// shards plus the number of parity shards.
	if len(shards) != r.totalShardCount {
		return fmt.Errorf("%w: %d", ErrWrongShardCount, len(shards))
	}

	// All of the shard buffers should be the same length.
	shardLength := len(shards[0])
	for i := 1; i < len(shards); i++ {
		if len(shards[i]) != shardLength {
			return ErrShardSizes
		}
	}

	// The offset and byteCount must be non-negative and fit in the
	// buffers.
	if offset < 0 {
		return fmt.Errorf("offset is negative: %d", offset)
	}
	if byteCount < 0 {
		return fmt.Errorf("byteCount is negative: %d", byteCount)
	}
	if shardLength < offset+byteCount {
		return fmt.Errorf("%w: %d", ErrShortBuffer, byteCount+offset)
	}
	return nil
}

// buildMatrix creates the matrix to use for encoding.
// The top square of the matrix is guaranteed to be an identity matrix,
// which means that the data shards are unchanged after encoding.
func buildMatrix(dataShards, totalShards int) *Matrix {
	// Start with a Vandermonde matrix.  This matrix would work, in
	// theory, but doesn't have the property that the data shards are
	// unchanged after encoding.
	vandermonde := newVandermondeMatrix(totalShards, dataShards)

	// Multiply by the inverse of the top square of the matrix.
	// This will make the top square be the identity matrix, but
	// preserve the property that any square subset of rows is
	// invertible.
	top := vandermonde.SubMatrix(0, 0, dataShards, dataShards)
	return vandermonde.Times(top.Invert())
}

// newVandermondeMatrix creates a Vandermonde matrix, which is
// guaranteed to have the property that any subset of rows that forms a
// square matrix is invertible.
func newVandermondeMatrix(rows, cols int) *Matrix {
	result := NewMatrix(rows, cols)
	for r := range rows {
		for c := range cols {
			result.Set(r, c, Exp(byte(r), c))
		}
	}
	return result
}
