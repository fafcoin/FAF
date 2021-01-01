// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package feed

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/fafereum/go-fafereum/common/bitutil"
	"github.com/fafereum/go-fafereum/common/hexutil"
	"github.com/fafereum/go-fafereum/swarm/storage"
)

// TopicLength establishes the max length of a topic string
const TopicLength = storage.AddressLength

// Topic represents what a feed is about
type Topic [TopicLength]byte

// ErrTopicTooLong is returned when creating a topic with a name/related content too long
var ErrTopicTooLong = fmt.Errorf("Topic is too long. Max length is %d", TopicLength)

// NewTopic creates a new topic from a provided name and "related content" byte array,
// merging the two togfafer.
// If relatedContent or name are longer than TopicLength, they will be truncated and an error returned
// name can be an empty string
// relatedContent can be nil
func NewTopic(name string, relatedContent []byte) (topic Topic, err error) {
	if relatedContent != nil {
		contentLength := len(relatedContent)
		if contentLength > TopicLength {
			contentLength = TopicLength
			err = ErrTopicTooLong
		}
		copy(topic[:], relatedContent[:contentLength])
	}
	nameBytes := []byte(name)
	nameLength := len(nameBytes)
	if nameLength > TopicLength {
		nameLength = TopicLength
		err = ErrTopicTooLong
	}
	bitutil.XORBytes(topic[:], topic[:], nameBytes[:nameLength])
	return topic, err
}

// Hex will return the topic encoded as an hex string
func (t *Topic) Hex() string {
	return hexutil.Encode(t[:])
}

// FromHex will parse a hex string into this Topic instance
func (t *Topic) FromHex(hex string) error {
	bytes, err := hexutil.Decode(hex)
	if err != nil || len(bytes) != len(t) {
		return NewErrorf(ErrInvalidValue, "Cannot decode topic")
	}
	copy(t[:], bytes)
	return nil
}

// Name will try to extract the topic name out of the Topic
func (t *Topic) Name(relatedContent []byte) string {
	nameBytes := *t
	if relatedContent != nil {
		contentLength := len(relatedContent)
		if contentLength > TopicLength {
			contentLength = TopicLength
		}
		bitutil.XORBytes(nameBytes[:], t[:], relatedContent[:contentLength])
	}
	z := bytes.IndexByte(nameBytes[:], 0)
	if z < 0 {
		z = TopicLength
	}
	return string(nameBytes[:z])

}

// UnmarshalJSON implements the json.Unmarshaller interface
func (t *Topic) UnmarshalJSON(data []byte) error {
	var hex string
	json.Unmarshal(data, &hex)
	return t.FromHex(hex)
}

// MarshalJSON implements the json.Marshaller interface
func (t *Topic) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Hex())
}
