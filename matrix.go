package reedsolomon

import (
	"fmt"
	"strings"
)

// Matrix is a matrix over the 8-bit Galois field.
//
// The data is stored in one flat, row-major allocation instead of one
// allocation per row: constructing a matrix costs a single allocation,
// and the elimination loops walk contiguous memory. Element (r, c) is
// data[r*columns+c].
type Matrix struct {
	rows    int
	columns int
	data    []byte
}

func NewMatrix(initRows, initColumns int) *Matrix {
	return &Matrix{
		rows:    initRows,
		columns: initColumns,
		data:    make([]byte, initRows*initColumns),
	}
}

func NewMatrixFromData(initData [][]byte) *Matrix {
	rows := len(initData)
	columns := len(initData[0])
	m := &Matrix{
		rows:    rows,
		columns: columns,
		data:    make([]byte, rows*columns),
	}
	for r := range rows {
		if len(initData[r]) != columns {
			panic("not all rows have the same number of columns")
		}
		copy(m.data[r*columns:(r+1)*columns], initData[r])
	}
	return m
}

func IdentityMatrix(size int) *Matrix {
	result := NewMatrix(size, size)
	for i := range size {
		result.Set(i, i, 1)
	}
	return result
}

// String returns a human-readable string of the matrix contents.
// Example: [[1, 2], [3, 4]]
func (m *Matrix) String() string {
	var result strings.Builder
	result.WriteByte('[')
	for r := range m.rows {
		if r != 0 {
			result.WriteString(", ")
		}
		result.WriteByte('[')
		for c := range m.columns {
			if c != 0 {
				result.WriteString(", ")
			}
			fmt.Fprintf(&result, "%d", m.data[r*m.columns+c])
		}
		result.WriteByte(']')
	}
	result.WriteByte(']')
	return result.String()
}

// BigString returns a human-readable string of the matrix contents.
//
// Example:
//
//	00 01 02
//	03 04 05
//	06 07 08
//	09 0a 0b
func (m *Matrix) BigString() string {
	var result strings.Builder
	for r := range m.rows {
		for c := range m.columns {
			fmt.Fprintf(&result, "%02x ", m.data[r*m.columns+c])
		}
		result.WriteString("\n")
	}
	return result.String()
}

func (m *Matrix) Rows() int {
	return m.rows
}

func (m *Matrix) Columns() int {
	return m.columns
}

func (m *Matrix) Get(r, c int) byte {
	if r < 0 || m.rows <= r {
		panic(fmt.Sprintf("row index out of range: %d", r))
	}
	if c < 0 || m.columns <= c {
		panic(fmt.Sprintf("column index out of range: %d", c))
	}
	return m.data[r*m.columns+c]
}

func (m *Matrix) Set(r, c int, value byte) {
	if r < 0 || m.rows <= r {
		panic(fmt.Sprintf("row index out of range: %d", r))
	}
	if c < 0 || m.columns <= c {
		panic(fmt.Sprintf("column index out of range: %d", c))
	}
	m.data[r*m.columns+c] = value
}

func (m *Matrix) Equal(other *Matrix) bool {
	if m.rows != other.rows || m.columns != other.columns {
		return false
	}
	for i := range m.data {
		if m.data[i] != other.data[i] {
			return false
		}
	}
	return true
}

// Times multiplies this matrix by another matrix
// It panics if the sizes don't match.
func (m *Matrix) Times(right *Matrix) *Matrix {
	if m.Columns() != right.Rows() {
		panic(fmt.Sprintf("columns on left (%d) is different than rows on right (%d)",
			m.Columns(), right.Rows()))
	}
	result := NewMatrix(m.Rows(), right.Columns())
	for r := range m.Rows() {
		for c := range right.Columns() {
			var value byte
			for i := range m.Columns() {
				value ^= Multiply(m.Get(r, i), right.Get(i, c))
			}
			result.Set(r, c, value)
		}
	}
	return result
}

// Augment returns the concatenation of this matrix and the matrix on
// the right. It panics if the matrices don't have the same number of
// rows.
func (m *Matrix) Augment(right *Matrix) *Matrix {
	if m.rows != right.rows {
		panic("matrices don't have the same number of rows")
	}
	result := NewMatrix(m.rows, m.columns+right.columns)
	for r := range m.rows {
		copy(result.data[r*result.columns:r*result.columns+m.columns], m.data[r*m.columns:(r+1)*m.columns])
		copy(result.data[r*result.columns+m.columns:(r+1)*result.columns], right.data[r*right.columns:(r+1)*right.columns])
	}
	return result
}

// SubMatrix returns a part of this matrix.
func (m *Matrix) SubMatrix(rmin, cmin, rmax, cmax int) *Matrix {
	result := NewMatrix(rmax-rmin, cmax-cmin)
	for r := rmin; r < rmax; r++ {
		copy(result.data[(r-rmin)*result.columns:(r-rmin+1)*result.columns],
			m.data[r*m.columns+cmin:r*m.columns+cmax])
	}
	return result
}

// rowSlice returns row i of the matrix as a slice that aliases the
// matrix data. The caller must not modify it or retain it longer than
// the matrix.
func (m *Matrix) rowSlice(i int) []byte {
	return m.data[i*m.columns : (i+1)*m.columns]
}

// GetRow returns one row of the matrix as a byte array.
func (m *Matrix) GetRow(row int) []byte {
	result := make([]byte, m.columns)
	copy(result, m.rowSlice(row))
	return result
}

// SwapRows exchanges two rows in the matrix.
func (m *Matrix) SwapRows(r1, r2 int) {
	if r1 < 0 || m.rows <= r1 || r2 < 0 || m.rows <= r2 {
		panic("row index out of range")
	}
	r1Row := m.rowSlice(r1)
	r2Row := m.rowSlice(r2)
	for c, v := range r1Row {
		r1Row[c] = r2Row[c]
		r2Row[c] = v
	}
}

// Invert returns the inverse of this matrix.
//
// It panics if the matrix is not square or is singular and doesn't have
// an inverse.
func (m *Matrix) Invert() *Matrix {
	if m.rows != m.columns {
		panic("only square matrices can be inverted")
	}

	// Create a working matrix by augmenting this one with an identity
	// matrix on the right.
	work := m.Augment(IdentityMatrix(m.rows))

	// Do Gaussian elimination to transform the left half into an
	// identity matrix.
	work.gaussianElimination()

	// The right half is now the inverse.
	return work.SubMatrix(0, m.rows, m.columns, m.columns*2)
}

// gaussianElimination does the work of matrix inversion.
// Assumes that this is an r by 2r matrix.
func (m *Matrix) gaussianElimination() {
	// Clear out the part below the main diagonal and scale the main
	// diagonal to be 1.
	for r := range m.rows {
		// If the element on the diagonal is 0, find a row below that
		// has a non-zero and swap them.
		if m.data[r*m.columns+r] == 0 {
			for rowBelow := r + 1; rowBelow < m.rows; rowBelow++ {
				if m.data[rowBelow*m.columns+r] != 0 {
					m.SwapRows(r, rowBelow)
					break
				}
			}
		}

		// If we couldn't find one, the matrix is singular.
		if m.data[r*m.columns+r] == 0 {
			panic("matrix is singular")
		}

		// Scale to 1.
		if m.data[r*m.columns+r] != 1 {
			scale := Divide(1, m.data[r*m.columns+r])
			for c := range m.columns {
				m.data[r*m.columns+c] = Multiply(m.data[r*m.columns+c], scale)
			}
		}

		// Make everything below the 1 be a 0 by subtracting a multiple
		// of it. (Subtraction and addition are both exclusive or in
		// the Galois field.)
		for rowBelow := r + 1; rowBelow < m.rows; rowBelow++ {
			if pivot := m.data[rowBelow*m.columns+r]; pivot != 0 {
				for c := range m.columns {
					m.data[rowBelow*m.columns+c] ^= Multiply(pivot, m.data[r*m.columns+c])
				}
			}
		}
	}

	// Now clear the part above the main diagonal.
	for d := range m.rows {
		for rowAbove := range d {
			if pivot := m.data[rowAbove*m.columns+d]; pivot != 0 {
				for c := range m.columns {
					m.data[rowAbove*m.columns+c] ^= Multiply(pivot, m.data[d*m.columns+c])
				}
			}
		}
	}
}
