package reedsolomon

import "errors"

// FieldSize is the number of elements in the field: 256.
const FieldSize = 256

// GeneratingPolynomial is the polynomial used to generate the logarithm
// table.
//
// There are a number of polynomials that work to generate a Galois field
// of 256 elements. The choice is arbitrary, and we just use the first
// one. The possibilities are: 29, 43, 45, 77, 95, 99, 101, 105, 113,
// 135, 141, 169, 195, 207, 231, and 245.
const GeneratingPolynomial = 29

// logTable maps members of the Galois field to their integer
// logarithms. The entry for 0 is -1 because there is no log of 0.
// This variable must not be modified: expTable and multiplicationTable
// are derived from it when the package is initialized.
var logTable = [FieldSize]int16{
	-1, 0, 1, 25, 2, 50, 26, 198,
	3, 223, 51, 238, 27, 104, 199, 75,
	4, 100, 224, 14, 52, 141, 239, 129,
	28, 193, 105, 248, 200, 8, 76, 113,
	5, 138, 101, 47, 225, 36, 15, 33,
	53, 147, 142, 218, 240, 18, 130, 69,
	29, 181, 194, 125, 106, 39, 249, 185,
	201, 154, 9, 120, 77, 228, 114, 166,
	6, 191, 139, 98, 102, 221, 48, 253,
	226, 152, 37, 179, 16, 145, 34, 136,
	54, 208, 148, 206, 143, 150, 219, 189,
	241, 210, 19, 92, 131, 56, 70, 64,
	30, 66, 182, 163, 195, 72, 126, 110,
	107, 58, 40, 84, 250, 133, 186, 61,
	202, 94, 155, 159, 10, 21, 121, 43,
	78, 212, 229, 172, 115, 243, 167, 87,
	7, 112, 192, 247, 140, 128, 99, 13,
	103, 74, 222, 237, 49, 197, 254, 24,
	227, 165, 153, 119, 38, 184, 180, 124,
	17, 68, 146, 217, 35, 32, 137, 46,
	55, 63, 209, 91, 149, 188, 207, 205,
	144, 135, 151, 178, 220, 252, 190, 97,
	242, 86, 211, 171, 20, 42, 93, 158,
	132, 60, 57, 83, 71, 109, 65, 162,
	31, 45, 67, 216, 183, 123, 164, 118,
	196, 23, 73, 236, 127, 12, 111, 246,
	108, 161, 59, 82, 41, 157, 85, 170,
	251, 96, 134, 177, 187, 204, 62, 90,
	203, 89, 95, 176, 156, 169, 160, 81,
	11, 245, 22, 235, 122, 117, 44, 215,
	79, 174, 213, 233, 230, 231, 173, 232,
	116, 214, 244, 234, 168, 80, 88, 175,
}

// expTable is the inverse of the logarithm table. It maps integer
// logarithms to members of the field. The table is repeated a second
// time so that Multiply does not have to bound the sum of two
// logarithms: the highest index used is 254 + 254 = 508 = len-2.
var expTable = GenerateExpTable(logTable[:])

// multiplicationTable is a multiplication table for the Galois field.
// The result of multiplying a and b is multiplicationTable[a][b].
//
// Using this table is an alternative to calling Multiply, which uses
// log/exp table lookups. It is what the table-based coding loops use in
// their inner loops.
var multiplicationTable = GenerateMultiplicationTable()

// Add adds two elements of the field. If you're in an inner loop, you
// should inline this function: it's just XOR.
func Add(a, b byte) byte {
	return a ^ b
}

// Subtract is the inverse of addition. If you're in an inner loop, you
// should inline this function: it's just XOR.
func Subtract(a, b byte) byte {
	return a ^ b
}

// Multiply multiplies two elements of the field.
func Multiply(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return expTable[int(logTable[a])+int(logTable[b])]
}

// Divide is the inverse of multiplication. It panics if b is 0.
func Divide(a, b byte) byte {
	if a == 0 {
		return 0
	}
	if b == 0 {
		panic("reedsolomon: argument 'divisor' is 0")
	}
	logResult := int(logTable[a]) - int(logTable[b])
	if logResult < 0 {
		logResult += 255
	}
	return expTable[logResult]
}

// Exp computes a**n, n must not be negative.
func Exp(a byte, n int) byte {
	if n == 0 {
		return 1
	} else if a == 0 {
		return 0
	}
	logResult := int(logTable[a]) * n
	for logResult >= 255 {
		logResult -= 255
	}
	return expTable[logResult]
}

// GenerateLogTable generates a logarithm table for the given generating
// polynomial. It returns an error if the polynomial is not a generator
// of the field.
func GenerateLogTable(polynomial int) ([]int16, error) {
	result := make([]int16, FieldSize)
	for i := range result {
		result[i] = -1 // -1 means "not set"
	}
	b := 1
	for log := range FieldSize - 1 {
		if result[b] != -1 {
			return nil, errors.New("BUG: duplicate logarithm (bad polynomial?)")
		}
		result[b] = int16(log)
		b <<= 1
		if FieldSize <= b {
			b = (b - FieldSize) ^ polynomial
		}
	}
	return result, nil
}

// GenerateExpTable generates the inverse log table for the given
// logarithm table. The result is FIELD_SIZE*2-2 entries: the table is
// repeated twice so that lookups of the sum of two logarithms never go
// out of bounds.
//
// logTable must be a complete log table, such as the one returned by
// GenerateLogTable; an incomplete one causes an index panic.
func GenerateExpTable(logTable []int16) []byte {
	result := make([]byte, FieldSize*2-2)
	for i := 1; i < FieldSize; i++ {
		log := logTable[i]
		result[log] = byte(i)
		result[log+FieldSize-1] = byte(i)
	}
	return result
}

// GenerateMultiplicationTable generates a multiplication table as an
// array of arrays. To get the result of multiplying a and b:
//
//	table[a][b]
func GenerateMultiplicationTable() [FieldSize][FieldSize]byte {
	var result [FieldSize][FieldSize]byte
	for a := range FieldSize {
		for b := range FieldSize {
			result[a][b] = Multiply(byte(a), byte(b))
		}
	}
	return result
}

// AllPossiblePolynomials returns a list of all polynomials that can be
// used to generate the field.
//
// This is never used in the code; it's just here for completeness.
func AllPossiblePolynomials() []int {
	var result []int
	for i := range FieldSize {
		if _, err := GenerateLogTable(i); err == nil {
			result = append(result, i)
		}
	}
	return result
}
