package reedsolomon

import (
	"bytes"
	"slices"
	"testing"
)

func TestAssociativity(t *testing.T) {
	for i := range 256 {
		a := byte(i)
		for j := range 256 {
			b := byte(j)
			for k := range 256 {
				c := byte(k)
				if Add(a, Add(b, c)) != Add(Add(a, b), c) {
					t.Fatalf("addition not associative at (%d, %d, %d)", i, j, k)
				}
				if Multiply(a, Multiply(b, c)) != Multiply(Multiply(a, b), c) {
					t.Fatalf("multiplication not associative at (%d, %d, %d)", i, j, k)
				}
			}
		}
	}
}

func TestIdentity(t *testing.T) {
	for i := range 256 {
		a := byte(i)
		if Add(a, 0) != a {
			t.Fatalf("Add(%d, 0) != %d", i, i)
		}
		if Multiply(a, 1) != a {
			t.Fatalf("Multiply(%d, 1) != %d", i, i)
		}
	}
}

func TestInverse(t *testing.T) {
	for i := range 256 {
		a := byte(i)
		if b := Subtract(0, a); Add(a, b) != 0 {
			t.Fatalf("additive inverse of %d is not %d", i, b)
		}
		if a != 0 {
			if b := Divide(1, a); Multiply(a, b) != 1 {
				t.Fatalf("multiplicative inverse of %d is not %d", i, b)
			}
		}
	}
}

func TestCommutativity(t *testing.T) {
	for i := range 256 {
		for j := range 256 {
			a := byte(i)
			b := byte(j)
			if Add(a, b) != Add(b, a) {
				t.Fatalf("addition not commutative at (%d, %d)", i, j)
			}
			if Multiply(a, b) != Multiply(b, a) {
				t.Fatalf("multiplication not commutative at (%d, %d)", i, j)
			}
		}
	}
}

func TestDistributivity(t *testing.T) {
	for i := range 256 {
		a := byte(i)
		for j := range 256 {
			b := byte(j)
			for k := range 256 {
				c := byte(k)
				if Multiply(a, Add(b, c)) != Add(Multiply(a, b), Multiply(a, c)) {
					t.Fatalf("multiplication not distributive at (%d, %d, %d)", i, j, k)
				}
			}
		}
	}
}

func TestExp(t *testing.T) {
	for i := range 256 {
		a := byte(i)
		power := byte(1)
		for j := range 256 {
			if Exp(a, j) != power {
				t.Fatalf("Exp(%d, %d) != %d", i, j, power)
			}
			power = Multiply(power, a)
		}
	}
}

func TestGenerateLogTable(t *testing.T) {
	generated, err := GenerateLogTable(GeneratingPolynomial)
	if err != nil {
		t.Fatalf("GenerateLogTable failed: %v", err)
	}
	if !slices.Equal(logTable[:], generated) {
		t.Fatalf("generated log table does not match logTable")
	}

	generatedExp := GenerateExpTable(generated)
	if !bytes.Equal(expTable, generatedExp) {
		t.Fatalf("generated exp table does not match expTable")
	}

	polynomials := []int{
		29, 43, 45, 77, 95, 99, 101, 105, 113,
		135, 141, 169, 195, 207, 231, 245,
	}
	actual := AllPossiblePolynomials()
	if len(polynomials) != len(actual) {
		t.Fatalf("expected %v, got %v", polynomials, actual)
	}
	for i := range polynomials {
		if polynomials[i] != actual[i] {
			t.Fatalf("expected %v, got %v", polynomials, actual)
		}
	}
}

func TestMultiplicationTable(t *testing.T) {
	table := &multiplicationTable
	for a := range 256 {
		for b := range 256 {
			if Multiply(byte(a), byte(b)) != table[a][b] {
				t.Fatalf("Multiply(%d, %d) != table[%d][%d]", a, b, a, b)
			}
		}
	}
}

func TestMatrixCalc(t *testing.T) {
	if Multiply(3, 4) != 12 {
		t.Fatalf("Multiply(3, 4) != 12")
	}
	if Multiply(7, 7) != 21 {
		t.Fatalf("Multiply(7, 7) != 21")
	}
	if Multiply(23, 45) != 41 {
		t.Fatalf("Multiply(23, 45) != 41")
	}

	if Exp(2, 2) != 4 {
		t.Fatalf("Exp(2, 2) != 4")
	}
	if Exp(5, 20) != 235 {
		t.Fatalf("Exp(5, 20) != 235")
	}
	if Exp(13, 7) != 43 {
		t.Fatalf("Exp(13, 7) != 43")
	}
}
