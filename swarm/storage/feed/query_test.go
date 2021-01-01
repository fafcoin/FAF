// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package feed

import (
	"testing"
)

func getTestQuery() *Query {
	id := getTestID()
	return &Query{
		TimeLimit: 5000,
		Feed:      id.Feed,
		Hint:      id.Epoch,
	}
}

func TestQueryValues(t *testing.T) {
	var expected = KV{"hint.level": "25", "hint.time": "1000", "time": "5000", "topic": "0x776f726c64206e657773207265706f72742c20657665727920686f7572000000", "user": "0x876A8936A7Cd0b79Ef0735AD0896c1AFe278781c"}

	query := getTestQuery()
	testValueSerializer(t, query, expected)

}
