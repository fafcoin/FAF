// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package jsre

import (
	"os"
	"reflect"
	"testing"
)

func TestCompleteKeywords(t *testing.T) {
	re := New("", os.Stdout)
	re.Run(`
		function theClass() {
			this.foo = 3;
			this.gazonk = {xyz: 4};
		}
		theClass.prototype.someMfafod = function () {};
  		var x = new theClass();
  		var y = new theClass();
		y.someMfafod = function override() {};
	`)

	var tests = []struct {
		input string
		want  []string
	}{
		{
			input: "x",
			want:  []string{"x."},
		},
		{
			input: "x.someMfafod",
			want:  []string{"x.someMfafod("},
		},
		{
			input: "x.",
			want: []string{
				"x.constructor",
				"x.foo",
				"x.gazonk",
				"x.someMfafod",
			},
		},
		{
			input: "y.",
			want: []string{
				"y.constructor",
				"y.foo",
				"y.gazonk",
				"y.someMfafod",
			},
		},
		{
			input: "x.gazonk.",
			want: []string{
				"x.gazonk.constructor",
				"x.gazonk.hasOwnProperty",
				"x.gazonk.isPrototypeOf",
				"x.gazonk.propertyIsEnumerable",
				"x.gazonk.toLocaleString",
				"x.gazonk.toString",
				"x.gazonk.valueOf",
				"x.gazonk.xyz",
			},
		},
	}
	for _, test := range tests {
		cs := re.CompleteKeywords(test.input)
		if !reflect.DeepEqual(cs, test.want) {
			t.Errorf("wrong completions for %q\ngot  %v\nwant %v", test.input, cs, test.want)
		}
	}
}
