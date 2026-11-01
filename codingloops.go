package reedsolomon

// ByteInputOutputExpCodingLoop nests the loops byte, input, output and
// multiplies using the log/exp tables.
type ByteInputOutputExpCodingLoop struct{ codingLoopBase }

func (ByteInputOutputExpCodingLoop) CodeSomeShards(
	matrixRows [][]byte,
	inputs [][]byte, inputCount int,
	outputs [][]byte, outputCount int,
	offset int, byteCount int) {

	for iByte := offset; iByte < offset+byteCount; iByte++ {
		iInput := 0
		inputShard := inputs[iInput]
		inputByte := inputShard[iByte]
		for iOutput := range outputCount {
			outputShard := outputs[iOutput]
			matrixRow := matrixRows[iOutput]
			outputShard[iByte] = Multiply(matrixRow[iInput], inputByte)
		}

		for iInput = 1; iInput < inputCount; iInput++ {
			inputShard = inputs[iInput]
			inputByte = inputShard[iByte]
			for iOutput := range outputCount {
				outputShard := outputs[iOutput]
				matrixRow := matrixRows[iOutput]
				outputShard[iByte] ^= Multiply(matrixRow[iInput], inputByte)
			}
		}
	}
}

// ByteInputOutputTableCodingLoop nests the loops byte, input, output
// and multiplies using the multiplication table.
type ByteInputOutputTableCodingLoop struct{ codingLoopBase }

func (ByteInputOutputTableCodingLoop) CodeSomeShards(
	matrixRows [][]byte,
	inputs [][]byte, inputCount int,
	outputs [][]byte, outputCount int,
	offset int, byteCount int) {

	table := &multiplicationTable

	for iByte := offset; iByte < offset+byteCount; iByte++ {
		iInput := 0
		inputShard := inputs[iInput]
		inputByte := inputShard[iByte]
		for iOutput := range outputCount {
			outputShard := outputs[iOutput]
			matrixRow := matrixRows[iOutput]
			multTableRow := &table[matrixRow[iInput]]
			outputShard[iByte] = multTableRow[inputByte]
		}
		for iInput = 1; iInput < inputCount; iInput++ {
			inputShard = inputs[iInput]
			inputByte = inputShard[iByte]
			for iOutput := range outputCount {
				outputShard := outputs[iOutput]
				matrixRow := matrixRows[iOutput]
				multTableRow := &table[matrixRow[iInput]]
				outputShard[iByte] ^= multTableRow[inputByte]
			}
		}
	}
}

// ByteOutputInputExpCodingLoop nests the loops byte, output, input and
// multiplies using the log/exp tables.
type ByteOutputInputExpCodingLoop struct{ codingLoopBase }

func (ByteOutputInputExpCodingLoop) CodeSomeShards(
	matrixRows [][]byte,
	inputs [][]byte, inputCount int,
	outputs [][]byte, outputCount int,
	offset int, byteCount int) {

	for iByte := offset; iByte < offset+byteCount; iByte++ {
		for iOutput := range outputCount {
			matrixRow := matrixRows[iOutput]
			var value byte
			for iInput := range inputCount {
				value ^= Multiply(matrixRow[iInput], inputs[iInput][iByte])
			}
			outputs[iOutput][iByte] = value
		}
	}
}

// ByteOutputInputTableCodingLoop nests the loops byte, output, input
// and multiplies using the multiplication table.
type ByteOutputInputTableCodingLoop struct{ codingLoopBase }

func (ByteOutputInputTableCodingLoop) CodeSomeShards(
	matrixRows [][]byte,
	inputs [][]byte, inputCount int,
	outputs [][]byte, outputCount int,
	offset int, byteCount int) {

	table := &multiplicationTable
	for iByte := offset; iByte < offset+byteCount; iByte++ {
		for iOutput := range outputCount {
			matrixRow := matrixRows[iOutput]
			var value byte
			for iInput := range inputCount {
				value ^= table[matrixRow[iInput]][inputs[iInput][iByte]]
			}
			outputs[iOutput][iByte] = value
		}
	}
}

// InputByteOutputExpCodingLoop nests the loops input, byte, output and
// multiplies using the log/exp tables.
type InputByteOutputExpCodingLoop struct{ codingLoopBase }

func (InputByteOutputExpCodingLoop) CodeSomeShards(
	matrixRows [][]byte,
	inputs [][]byte, inputCount int,
	outputs [][]byte, outputCount int,
	offset int, byteCount int) {

	iInput := 0
	inputShard := inputs[iInput]
	for iByte := offset; iByte < offset+byteCount; iByte++ {
		inputByte := inputShard[iByte]
		for iOutput := range outputCount {
			outputShard := outputs[iOutput]
			matrixRow := matrixRows[iOutput]
			outputShard[iByte] = Multiply(matrixRow[iInput], inputByte)
		}
	}

	for iInput = 1; iInput < inputCount; iInput++ {
		inputShard = inputs[iInput]
		for iByte := offset; iByte < offset+byteCount; iByte++ {
			inputByte := inputShard[iByte]
			for iOutput := range outputCount {
				outputShard := outputs[iOutput]
				matrixRow := matrixRows[iOutput]
				outputShard[iByte] ^= Multiply(matrixRow[iInput], inputByte)
			}
		}
	}
}

// InputByteOutputTableCodingLoop nests the loops input, byte, output
// and multiplies using the multiplication table.
type InputByteOutputTableCodingLoop struct{ codingLoopBase }

func (InputByteOutputTableCodingLoop) CodeSomeShards(
	matrixRows [][]byte,
	inputs [][]byte, inputCount int,
	outputs [][]byte, outputCount int,
	offset int, byteCount int) {

	table := &multiplicationTable

	iInput := 0
	inputShard := inputs[iInput]
	for iByte := offset; iByte < offset+byteCount; iByte++ {
		inputByte := inputShard[iByte]
		multTableRow := &table[inputByte]
		for iOutput := range outputCount {
			outputShard := outputs[iOutput]
			matrixRow := matrixRows[iOutput]
			outputShard[iByte] = multTableRow[matrixRow[iInput]]
		}
	}

	for iInput = 1; iInput < inputCount; iInput++ {
		inputShard = inputs[iInput]
		for iByte := offset; iByte < offset+byteCount; iByte++ {
			inputByte := inputShard[iByte]
			multTableRow := &table[inputByte]
			for iOutput := range outputCount {
				outputShard := outputs[iOutput]
				matrixRow := matrixRows[iOutput]
				outputShard[iByte] ^= multTableRow[matrixRow[iInput]]
			}
		}
	}
}

// InputOutputByteExpCodingLoop nests the loops input, output, byte and
// multiplies using the log/exp tables.
type InputOutputByteExpCodingLoop struct{ codingLoopBase }

func (InputOutputByteExpCodingLoop) CodeSomeShards(
	matrixRows [][]byte,
	inputs [][]byte, inputCount int,
	outputs [][]byte, outputCount int,
	offset int, byteCount int) {

	iInput := 0
	inputShard := inputs[iInput]
	for iOutput := range outputCount {
		outputShard := outputs[iOutput]
		matrixRow := matrixRows[iOutput]
		matrixByte := matrixRow[iInput]
		for iByte := offset; iByte < offset+byteCount; iByte++ {
			outputShard[iByte] = Multiply(matrixByte, inputShard[iByte])
		}
	}

	for iInput = 1; iInput < inputCount; iInput++ {
		inputShard = inputs[iInput]
		for iOutput := range outputCount {
			outputShard := outputs[iOutput]
			matrixRow := matrixRows[iOutput]
			matrixByte := matrixRow[iInput]
			for iByte := offset; iByte < offset+byteCount; iByte++ {
				outputShard[iByte] ^= Multiply(matrixByte, inputShard[iByte])
			}
		}
	}
}

// InputOutputByteTableCodingLoop nests the loops input, output, byte
// and multiplies using the multiplication table. This is the default
// coding loop.
type InputOutputByteTableCodingLoop struct{ codingLoopBase }

func (InputOutputByteTableCodingLoop) CodeSomeShards(
	matrixRows [][]byte,
	inputs [][]byte, inputCount int,
	outputs [][]byte, outputCount int,
	offset int, byteCount int) {

	table := &multiplicationTable

	iInput := 0
	inputShard := inputs[iInput]
	for iOutput := range outputCount {
		outputShard := outputs[iOutput]
		matrixRow := matrixRows[iOutput]
		multTableRow := &table[matrixRow[iInput]]
		for iByte := offset; iByte < offset+byteCount; iByte++ {
			outputShard[iByte] = multTableRow[inputShard[iByte]]
		}
	}

	for iInput = 1; iInput < inputCount; iInput++ {
		inputShard = inputs[iInput]
		for iOutput := range outputCount {
			outputShard := outputs[iOutput]
			matrixRow := matrixRows[iOutput]
			multTableRow := &table[matrixRow[iInput]]
			for iByte := offset; iByte < offset+byteCount; iByte++ {
				outputShard[iByte] ^= multTableRow[inputShard[iByte]]
			}
		}
	}
}

// CheckSomeShards, with a non-nil tempBuffer, uses the loop structure
// from OutputInputByteTableCodingLoop. Using the loops from this class
// would require multiple temp buffers.
func (l InputOutputByteTableCodingLoop) CheckSomeShards(
	matrixRows [][]byte,
	inputs [][]byte, inputCount int,
	toCheck [][]byte, checkCount int,
	offset int, byteCount int,
	tempBuffer []byte) bool {

	if tempBuffer == nil {
		return l.codingLoopBase.CheckSomeShards(
			matrixRows, inputs, inputCount, toCheck, checkCount, offset, byteCount, nil)
	}

	table := &multiplicationTable
	for iOutput := range checkCount {
		outputShard := toCheck[iOutput]
		matrixRow := matrixRows[iOutput]
		iInput := 0
		inputShard := inputs[iInput]
		multTableRow := &table[matrixRow[iInput]]
		for iByte := offset; iByte < offset+byteCount; iByte++ {
			tempBuffer[iByte] = multTableRow[inputShard[iByte]]
		}
		for iInput = 1; iInput < inputCount; iInput++ {
			inputShard = inputs[iInput]
			multTableRow = &table[matrixRow[iInput]]
			for iByte := offset; iByte < offset+byteCount; iByte++ {
				tempBuffer[iByte] ^= multTableRow[inputShard[iByte]]
			}
		}
		for iByte := offset; iByte < offset+byteCount; iByte++ {
			if tempBuffer[iByte] != outputShard[iByte] {
				return false
			}
		}
	}

	return true
}

// OutputByteInputExpCodingLoop nests the loops output, byte, input and
// multiplies using the log/exp tables.
type OutputByteInputExpCodingLoop struct{ codingLoopBase }

func (OutputByteInputExpCodingLoop) CodeSomeShards(
	matrixRows [][]byte,
	inputs [][]byte, inputCount int,
	outputs [][]byte, outputCount int,
	offset int, byteCount int) {

	for iOutput := range outputCount {
		outputShard := outputs[iOutput]
		matrixRow := matrixRows[iOutput]
		for iByte := offset; iByte < offset+byteCount; iByte++ {
			var value byte
			for iInput := range inputCount {
				inputShard := inputs[iInput]
				value ^= Multiply(matrixRow[iInput], inputShard[iByte])
			}
			outputShard[iByte] = value
		}
	}
}

// OutputByteInputTableCodingLoop nests the loops output, byte, input
// and multiplies using the multiplication table.
type OutputByteInputTableCodingLoop struct{ codingLoopBase }

func (OutputByteInputTableCodingLoop) CodeSomeShards(
	matrixRows [][]byte,
	inputs [][]byte, inputCount int,
	outputs [][]byte, outputCount int,
	offset int, byteCount int) {

	table := &multiplicationTable
	for iOutput := range outputCount {
		outputShard := outputs[iOutput]
		matrixRow := matrixRows[iOutput]
		for iByte := offset; iByte < offset+byteCount; iByte++ {
			var value byte
			for iInput := range inputCount {
				inputShard := inputs[iInput]
				multTableRow := &table[matrixRow[iInput]]
				value ^= multTableRow[inputShard[iByte]]
			}
			outputShard[iByte] = value
		}
	}
}

// OutputInputByteExpCodingLoop nests the loops output, input, byte and
// multiplies using the log/exp tables.
type OutputInputByteExpCodingLoop struct{ codingLoopBase }

func (OutputInputByteExpCodingLoop) CodeSomeShards(
	matrixRows [][]byte,
	inputs [][]byte, inputCount int,
	outputs [][]byte, outputCount int,
	offset int, byteCount int) {

	for iOutput := range outputCount {
		outputShard := outputs[iOutput]
		matrixRow := matrixRows[iOutput]
		iInput := 0
		inputShard := inputs[iInput]
		matrixByte := matrixRow[iInput]
		for iByte := offset; iByte < offset+byteCount; iByte++ {
			outputShard[iByte] = Multiply(matrixByte, inputShard[iByte])
		}
		for iInput = 1; iInput < inputCount; iInput++ {
			inputShard = inputs[iInput]
			matrixByte = matrixRow[iInput]
			for iByte := offset; iByte < offset+byteCount; iByte++ {
				outputShard[iByte] ^= Multiply(matrixByte, inputShard[iByte])
			}
		}
	}
}

// OutputInputByteTableCodingLoop nests the loops output, input, byte
// and multiplies using the multiplication table.
type OutputInputByteTableCodingLoop struct{ codingLoopBase }

func (OutputInputByteTableCodingLoop) CodeSomeShards(
	matrixRows [][]byte,
	inputs [][]byte, inputCount int,
	outputs [][]byte, outputCount int,
	offset int, byteCount int) {

	table := &multiplicationTable
	for iOutput := range outputCount {
		outputShard := outputs[iOutput]
		matrixRow := matrixRows[iOutput]
		iInput := 0
		inputShard := inputs[iInput]
		multTableRow := &table[matrixRow[iInput]]
		for iByte := offset; iByte < offset+byteCount; iByte++ {
			outputShard[iByte] = multTableRow[inputShard[iByte]]
		}
		for iInput = 1; iInput < inputCount; iInput++ {
			inputShard = inputs[iInput]
			multTableRow = &table[matrixRow[iInput]]
			for iByte := offset; iByte < offset+byteCount; iByte++ {
				outputShard[iByte] ^= multTableRow[inputShard[iByte]]
			}
		}
	}
}

// CheckSomeShards, with a non-nil tempBuffer, uses the loop structure
// of this class, which uses just one temp buffer.
func (l OutputInputByteTableCodingLoop) CheckSomeShards(
	matrixRows [][]byte,
	inputs [][]byte, inputCount int,
	toCheck [][]byte, checkCount int,
	offset int, byteCount int,
	tempBuffer []byte) bool {

	if tempBuffer == nil {
		return l.codingLoopBase.CheckSomeShards(
			matrixRows, inputs, inputCount, toCheck, checkCount, offset, byteCount, nil)
	}

	table := &multiplicationTable
	for iOutput := range checkCount {
		outputShard := toCheck[iOutput]
		matrixRow := matrixRows[iOutput]
		iInput := 0
		inputShard := inputs[iInput]
		multTableRow := &table[matrixRow[iInput]]
		for iByte := offset; iByte < offset+byteCount; iByte++ {
			tempBuffer[iByte] = multTableRow[inputShard[iByte]]
		}
		for iInput = 1; iInput < inputCount; iInput++ {
			inputShard = inputs[iInput]
			multTableRow = &table[matrixRow[iInput]]
			for iByte := offset; iByte < offset+byteCount; iByte++ {
				tempBuffer[iByte] ^= multTableRow[inputShard[iByte]]
			}
		}
		for iByte := offset; iByte < offset+byteCount; iByte++ {
			if tempBuffer[iByte] != outputShard[iByte] {
				return false
			}
		}
	}

	return true
}
