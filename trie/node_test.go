// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package trie

import "testing"

func TestCanUnload(t *testing.T) {
	tests := []struct {
		flag                 nodeFlag
		cachegen, cachelimit uint16
		want                 bool
	}{
		{
			flag: nodeFlag{dirty: true, gen: 0},
			want: false,
		},
		{
			flag:     nodeFlag{dirty: false, gen: 0},
			cachegen: 0, cachelimit: 0,
			want: true,
		},
		{
			flag:     nodeFlag{dirty: false, gen: 65534},
			cachegen: 65535, cachelimit: 1,
			want: true,
		},
		{
			flag:     nodeFlag{dirty: false, gen: 65534},
			cachegen: 0, cachelimit: 1,
			want: true,
		},
		{
			flag:     nodeFlag{dirty: false, gen: 1},
			cachegen: 65535, cachelimit: 1,
			want: true,
		},
	}

	for _, test := range tests {
		if got := test.flag.canUnload(test.cachegen, test.cachelimit); got != test.want {
			t.Errorf("%+v\n   got %t, want %t", test, got, test.want)
		}
	}
}
