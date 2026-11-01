// simpleencoder encodes one file using Reed-Solomon 4+2.

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/cxljs/reedsolomon"
)

const (
	dataShards   = 4
	parityShards = 2
	totalShards  = 6

	bytesInInt = 4
)

// Encode one file with Reed-Solomon 4+2. The argument is the file to
// encode, say "foo.txt"; the output is six files next to it,
// "foo.txt.0" through "foo.txt.5", where shards 0-3 hold the data and
// shards 4-5 hold the parity.
//
// The file is encoded through a byte stream:
//
//	file → [4-byte big-endian file size][file bytes][zero padding]
//	     → cut into 4 equal data shards → +2 parity shards → 6 files
//
// Reed-Solomon requires all shards to be the same length, so the
// stream is padded with zeros up to a multiple of 4 before the cut.
// The size prefix exists because the reassembled stream is longer
// than the file: a file may itself end in zeros, so the padding is
// indistinguishable from content, and the recorded size is the only
// way for simpledecoder to strip the prefix and padding exactly.
func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: simpleencoder <fileName>")
		return
	}
	inputFileName := os.Args[1]
	inputFile, err := os.Open(inputFileName)
	if err != nil {
		fmt.Println("Cannot read input file: ", inputFileName)
		return
	}
	defer inputFile.Close()

	// Get the size of the input file.
	info, err := inputFile.Stat()
	if err != nil || info.Size() > 0xFFFFFFFF {
		fmt.Println("File is too large: ", inputFileName)
		return
	}
	fileSize := info.Size()

	// Figure out how big each shard will be.  The total size stored
	// will be the file size (4 bytes) plus the file.
	storedSize := fileSize + bytesInInt
	shardSize := (storedSize + dataShards - 1) / dataShards

	// Create a buffer holding the file size, followed by the contents
	// of the file.
	bufferSize := shardSize * dataShards
	allBytes := make([]byte, bufferSize)
	binary.BigEndian.PutUint32(allBytes, uint32(fileSize))
	if _, err := io.ReadFull(inputFile, allBytes[bytesInInt:bytesInInt+fileSize]); err != nil {
		fmt.Println("not enough bytes read")
		return
	}

	// Make the buffers to hold the shards.
	shards := make([][]byte, totalShards)
	for i := range shards {
		shards[i] = make([]byte, shardSize)
	}

	// Fill in the data shards
	for i := range dataShards {
		start := int64(i) * shardSize
		copy(shards[i], allBytes[start:start+shardSize])
	}

	// Use Reed-Solomon to calculate the parity.
	codec, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		fmt.Println("New: ", err)
		return
	}
	if err := codec.EncodeParity(shards, 0, int(shardSize)); err != nil {
		fmt.Println("EncodeParity: ", err)
		return
	}

	// Write out the resulting files.
	for i := range totalShards {
		outputFileName := fmt.Sprintf("%s.%d", inputFileName, i)
		if err := os.WriteFile(outputFileName, shards[i], 0666); err != nil {
			fmt.Println("Cannot write output file: ", outputFileName)
			return
		}
		fmt.Println("wrote " + outputFileName)
	}
}
