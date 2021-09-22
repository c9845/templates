package templates

import "testing"

func TestFuncIndexOf(t *testing.T) {
	haystack := "asdfghjkl"
	goodNeedle := "d"
	badNeedle := "p"

	if idx := FuncIndexOf(goodNeedle, haystack); idx != 2 {
		t.Fatalf("Index of needle in haystack wrong.  Was %v, should be %v.", idx, 2)
		return
	}

	if idx := FuncIndexOf(badNeedle, haystack); idx != -1 {
		t.Fatalf("Index of needle in haystack wrong.  Was %v, should be %v.", idx, -1)
		return
	}

	return
}

func TestFuncDateReformat(t *testing.T) {
	//successful reformat
	old := "2020-01-01"
	format := "01/02/2006"
	new := FuncDateReformat(old, format)
	if new == "" {
		t.Fatal("no new date was returned")
		return
	}

	//input date was bad
	old = "2020-01-32"
	format = "01/02/2006"
	new = FuncDateReformat(old, format)
	if new != old {
		t.Fatal("new date should have matched old date due to input date issue")
		return
	}

	return
}

func TestFuncAddInt(t *testing.T) {
	x := 1
	y := 8
	result := FuncAddInt(x, y)
	if result != x+y {
		t.Fatal("AddInt didn't add correctly")
		return
	}
}
