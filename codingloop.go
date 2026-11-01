package reedsolomon

// CodingLoop is a method of looping over inputs and encoding them.
//
// All of the available coding loop algorithms are in AllCodingLoops.
// The different choices nest the three loops in different orders, and
// either use the log/exponent tables, or use the multiplication table.
//
// The naming of the three loops is (with number of loops in benchmark):
//
//	"byte"   - Index of byte within shard.  (200,000 bytes in each shard)
//	"input"  - Which input shard is being read.  (17 data shards)
//	"output" - Which output shard is being computed.  (3 parity shards)
//
// And the naming for multiplication method is:
//
//	"table"  - Use the multiplication table.
//	"exp"    - Use the logarithm/exponent table.
//
// The benchmarks in reedsolomon_benchmark_test.go compare the
// performance of the different loops, which will depend on the specific
// processor you're running on.
//
// This is the inner loop.  It needs to be fast.  Be careful if you
// change it.
type CodingLoop interface {
	// CodeSomeShards multiplies a subset of rows from a coding matrix
	// by a full set of input shards to produce some output shards.
	//
	// matrixRows: The rows from the matrix to use.
	//
	// inputs: An array of byte arrays, each of which is one input
	// shard. The inputs array may have extra buffers after the ones
	// that are used. They will be ignored. The number of inputs used
	// is determined by the length of each matrix row.
	//
	// inputCount: The number of input byte arrays.
	//
	// outputs: Byte arrays where the computed shards are stored. The
	// outputs array may also have extra, unused, elements at the end.
	// The number of outputs computed, and the number of matrix rows
	// used, is determined by outputCount.
	//
	// outputCount: The number of outputs to compute.
	//
	// offset: The index in the inputs and output of the first byte to
	// process.
	//
	// byteCount: The number of bytes to process.
	CodeSomeShards(matrixRows [][]byte,
		inputs [][]byte, inputCount int,
		outputs [][]byte, outputCount int,
		offset int, byteCount int)

	// CheckSomeShards multiplies a subset of rows from a coding matrix
	// by a full set of input shards to produce some output shards, and
	// checks that the data in those shards matches what's expected.
	//
	// The parameters are the same as CodeSomeShards, except that
	// toCheck/checkCount take the place of outputs/outputCount, and
	// tempBuffer is a place to store temporary results. It may be nil.
	CheckSomeShards(matrixRows [][]byte,
		inputs [][]byte, inputCount int,
		toCheck [][]byte, checkCount int,
		offset int, byteCount int,
		tempBuffer []byte) bool
}

// AllCodingLoops is a list of all of the available coding loop
// algorithms. Callers must not modify the slice or its elements.
var AllCodingLoops = []CodingLoop{
	&ByteInputOutputExpCodingLoop{},
	&ByteInputOutputTableCodingLoop{},
	&ByteOutputInputExpCodingLoop{},
	&ByteOutputInputTableCodingLoop{},
	&InputByteOutputExpCodingLoop{},
	&InputByteOutputTableCodingLoop{},
	&InputOutputByteExpCodingLoop{},
	&InputOutputByteTableCodingLoop{},
	&OutputByteInputExpCodingLoop{},
	&OutputByteInputTableCodingLoop{},
	&OutputInputByteExpCodingLoop{},
	&OutputInputByteTableCodingLoop{},
}

// codingLoopBase provides the default CheckSomeShards implementation
// shared by many of the coding loops. Embed it in a coding loop type to
// inherit the method; a type that overrides CheckSomeShards can call
// the embedded method.
//
// Many of the coding loops do not have custom CheckSomeShards methods.
// The benchmark doesn't measure that method.
type codingLoopBase struct{}

// CheckSomeShards is the default implementation. This is the loop
// structure for ByteOutputInput, which does not require temporary
// buffers for checking.
func (codingLoopBase) CheckSomeShards(
	matrixRows [][]byte,
	inputs [][]byte, inputCount int,
	toCheck [][]byte, checkCount int,
	offset int, byteCount int,
	tempBuffer []byte) bool {

	table := &multiplicationTable
	for iByte := offset; iByte < offset+byteCount; iByte++ {
		for iOutput := range checkCount {
			matrixRow := matrixRows[iOutput]
			var value byte
			for iInput := range inputCount {
				value ^= table[matrixRow[iInput]][inputs[iInput][iByte]]
			}
			if toCheck[iOutput][iByte] != value {
				return false
			}
		}
	}
	return true
}
