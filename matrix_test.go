package reedsolomon

import "testing"

func TestMatrixIdentity(t *testing.T) {
	if IdentityMatrix(3).String() != "[[1, 0, 0], [0, 1, 0], [0, 0, 1]]" {
		t.Fatalf("identity(3) is %s", IdentityMatrix(3).String())
	}
}

func TestMatrixBigString(t *testing.T) {
	if IdentityMatrix(2).BigString() != "01 00 \n00 01 \n" {
		t.Fatalf("BigString of identity(2) is %q", IdentityMatrix(2).BigString())
	}
}

func TestMatrixMultiply(t *testing.T) {
	m1 := NewMatrixFromData([][]byte{
		{1, 2},
		{3, 4},
	})
	m2 := NewMatrixFromData([][]byte{
		{5, 6},
		{7, 8},
	})
	actual := m1.Times(m2)
	if actual.String() != "[[11, 22], [19, 42]]" {
		t.Fatalf("m1 * m2 is %s", actual.String())
	}
}

func TestMatrixInverse(t *testing.T) {
	m := NewMatrixFromData([][]byte{
		{56, 23, 98},
		{3, 100, 200},
		{45, 201, 123},
	})
	if m.Invert().String() != "[[175, 133, 33], [130, 13, 245], [112, 35, 126]]" {
		t.Fatalf("inverse is %s", m.Invert().String())
	}
	if !m.Times(m.Invert()).Equal(IdentityMatrix(3)) {
		t.Fatalf("m * inverse(m) is not identity")
	}
}

func TestMatrixInverse2(t *testing.T) {
	m := NewMatrixFromData([][]byte{
		{1, 0, 0, 0, 0},
		{0, 1, 0, 0, 0},
		{0, 0, 0, 1, 0},
		{0, 0, 0, 0, 1},
		{7, 7, 6, 6, 1},
	})
	expected := "[[1, 0, 0, 0, 0]," +
		" [0, 1, 0, 0, 0]," +
		" [123, 123, 1, 122, 122]," +
		" [0, 0, 1, 0, 0]," +
		" [0, 0, 0, 1, 0]]"
	if m.Invert().String() != expected {
		t.Fatalf("inverse is %s", m.Invert().String())
	}
	if !m.Times(m.Invert()).Equal(IdentityMatrix(5)) {
		t.Fatalf("m * inverse(m) is not identity")
	}
}
