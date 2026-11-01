// simpledecoder decodes a file using Reed-Solomon 4+2.

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

// The file name given should be the name of the file to decode, say
// "foo.txt".  This program will expect to find "foo.txt.0" through
// "foo.txt.5", with at most two missing.  It will then write
// "foo.txt.decoded".
func main() {
	// Parse the command line
	if len(os.Args) != 2 {
		fmt.Println("Usage: simpledecoder <fileName>")
		return
	}
	originalFileName := os.Args[1]
	if _, err := os.Stat(originalFileName); err != nil {
		fmt.Println("Cannot read input file: " + originalFileName)
		return
	}

	// Read in any of the shards that are present.
	shards := make([][]byte, totalShards)
	shardPresent := make([]bool, totalShards)
	var shardSize int
	var firstShardName string
	shardCount := 0
	for i := range totalShards {
		shardFileName := fmt.Sprintf("%s.%d", originalFileName, i)
		shardFile, err := os.Open(shardFileName)
		if err != nil {
			continue
		}
		stat, err := shardFile.Stat()
		if err != nil {
			shardFile.Close()
			continue
		}
		// The library would reject mismatched shards later, but fail
		// fast here with the file names instead.
		if shardCount > 0 && int(stat.Size()) != shardSize {
			fmt.Printf("Shards have different sizes: %s is %d bytes, %s is %d bytes\n",
				firstShardName, shardSize, shardFileName, stat.Size())
			shardFile.Close()
			return
		}
		shardSize = int(stat.Size())
		firstShardName = shardFileName
		shards[i] = make([]byte, shardSize)
		shardPresent[i] = true
		shardCount += 1
		if _, err := io.ReadFull(shardFile, shards[i]); err != nil {
			fmt.Println("Cannot read shard file: ", shardFileName)
			shardFile.Close()
			return
		}
		shardFile.Close()
		fmt.Println("Read " + shardFileName)
	}

	// We need at least dataShards to be able to reconstruct the file.
	if shardCount < dataShards {
		fmt.Println("Not enough shards present")
		return
	}

	// Make empty buffers for the missing shards.
	for i := range totalShards {
		if !shardPresent[i] {
			shards[i] = make([]byte, shardSize)
		}
	}

	// Use Reed-Solomon to fill in the missing shards
	codec, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		fmt.Println("New: ", err)
		return
	}
	if err := codec.DecodeMissing(shards, shardPresent, 0, shardSize); err != nil {
		fmt.Println("DecodeMissing: ", err)
		return
	}

	// Combine the data shards into one buffer for convenience.
	// (This is not efficient, but it is convenient.)
	allBytes := make([]byte, shardSize*dataShards)
	for i := range dataShards {
		copy(allBytes[shardSize*i:], shards[i])
	}

	// Extract the file length
	fileSize := binary.BigEndian.Uint32(allBytes)

	// Write the decoded file
	decodedFileName := originalFileName + ".decoded"
	if err := os.WriteFile(decodedFileName, allBytes[bytesInInt:bytesInInt+int64(fileSize)], 0666); err != nil {
		fmt.Println("Cannot write output file: ", decodedFileName)
		return
	}
	fmt.Println("Wrote " + decodedFileName)
}
