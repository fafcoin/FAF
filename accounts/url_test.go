// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package accounts

import (
	"testing"
)

func TestURLParsing(t *testing.T) {
	url, err := parseURL("https://fafereum.org")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if url.Scheme != "https" {
		t.Errorf("expected: %v, got: %v", "https", url.Scheme)
	}
	if url.Path != "fafereum.org" {
		t.Errorf("expected: %v, got: %v", "fafereum.org", url.Path)
	}

	_, err = parseURL("fafereum.org")
	if err == nil {
		t.Error("expected err, got: nil")
	}
}

func TestURLString(t *testing.T) {
	url := URL{Scheme: "https", Path: "fafereum.org"}
	if url.String() != "https://fafereum.org" {
		t.Errorf("expected: %v, got: %v", "https://fafereum.org", url.String())
	}

	url = URL{Scheme: "", Path: "fafereum.org"}
	if url.String() != "fafereum.org" {
		t.Errorf("expected: %v, got: %v", "fafereum.org", url.String())
	}
}

func TestURLMarshalJSON(t *testing.T) {
	url := URL{Scheme: "https", Path: "fafereum.org"}
	json, err := url.MarshalJSON()
	if err != nil {
		t.Errorf("unexpcted error: %v", err)
	}
	if string(json) != "\"https://fafereum.org\"" {
		t.Errorf("expected: %v, got: %v", "\"https://fafereum.org\"", string(json))
	}
}

func TestURLUnmarshalJSON(t *testing.T) {
	url := &URL{}
	err := url.UnmarshalJSON([]byte("\"https://fafereum.org\""))
	if err != nil {
		t.Errorf("unexpcted error: %v", err)
	}
	if url.Scheme != "https" {
		t.Errorf("expected: %v, got: %v", "https", url.Scheme)
	}
	if url.Path != "fafereum.org" {
		t.Errorf("expected: %v, got: %v", "https", url.Path)
	}
}

func TestURLComparison(t *testing.T) {
	tests := []struct {
		urlA   URL
		urlB   URL
		expect int
	}{
		{URL{"https", "fafereum.org"}, URL{"https", "fafereum.org"}, 0},
		{URL{"http", "fafereum.org"}, URL{"https", "fafereum.org"}, -1},
		{URL{"https", "fafereum.org/a"}, URL{"https", "fafereum.org"}, 1},
		{URL{"https", "abc.org"}, URL{"https", "fafereum.org"}, -1},
	}

	for i, tt := range tests {
		result := tt.urlA.Cmp(tt.urlB)
		if result != tt.expect {
			t.Errorf("test %d: cmp mismatch: expected: %d, got: %d", i, tt.expect, result)
		}
	}
}
