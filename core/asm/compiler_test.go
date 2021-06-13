

package asm

import (
	"testing"
)

func TestCompiler(t *testing.T) {
	tests := []struct {
		input, output string
	}{
		{
			input: `
	GAS
	label:
	PUSH @label
`,
			output: "5a5b6300000001",
		},
		{
			input: `
	PUSH @label
	label:
`,
			output: "63000000055b",
		},
		{
			input: `
	PUSH @label
	JUMP
	label:
`,
			output: "6300000006565b",
		},
		{
			input: `
	JUMP @label
	label:
`,
			output: "6300000006565b",
		},
	}
	for _, test := range tests {
		ch := Lex([]byte(test.input), false)
		c := NewCompiler(false)
		c.Feed(ch)
		output, err := c.Compile()
		if len(err) != 0 {
			t.Errorf("compile error: %v\ninput: %s", err, test.input)
			continue
		}
		if output != test.output {
			t.Errorf("incorrect output\ninput: %sgot:  %s\nwant: %s\n", test.input, output, test.output)
		}
	}
}
