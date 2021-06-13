

package rlp

import (
	"fmt"
	"io"
)

type MyCoolType struct {
	Name string
	a, b uint
}

// EncodeRLP writes x as RLP list [a, b] that omits the Name field.
func (x *MyCoolType) EncodeRLP(w io.Writer) (err error) {
	return Encode(w, []uint{x.a, x.b})
}

func ExampleEncoder() {
	var t *MyCoolType // t is nil pointer to MyCoolType
	bytes, _ := EncodeToBytes(t)
	fmt.Printf("%v → %X\n", t, bytes)

	t = &MyCoolType{Name: "foobar", a: 5, b: 6}
	bytes, _ = EncodeToBytes(t)
	fmt.Printf("%v → %X\n", t, bytes)

	// Output:
	// <nil> → C0
	// &{foobar 5 6} → C20506
}
