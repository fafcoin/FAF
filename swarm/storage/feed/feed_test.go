// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify
package feed

import (
	"testing"
)

func getTestFeed() *Feed {
	topic, _ := NewTopic("world news report, every hour", nil)
	return &Feed{
		Topic: topic,
		User:  newCharlieSigner().Address(),
	}
}

func TestFeedSerializerDeserializer(t *testing.T) {
	testBinarySerializerRecovery(t, getTestFeed(), "0x776f726c64206e657773207265706f72742c20657665727920686f7572000000876a8936a7cd0b79ef0735ad0896c1afe278781c")
}

func TestFeedSerializerLengthCheck(t *testing.T) {
	testBinarySerializerLengthCheck(t, getTestFeed())
}
