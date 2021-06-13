

package p2p

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
)

func TestExpHeap(t *testing.T) {
	var h expHeap

	var (
		basetime = mclock.AbsTime(10)
		exptimeA = basetime.Add(2 * time.Second)
		exptimeB = basetime.Add(3 * time.Second)
		exptimeC = basetime.Add(4 * time.Second)
	)
	h.add("b", exptimeB)
	h.add("a", exptimeA)
	h.add("c", exptimeC)

	if h.nextExpiry() != exptimeA {
		t.Fatal("wrong nextExpiry")
	}
	if !h.contains("a") || !h.contains("b") || !h.contains("c") {
		t.Fatal("heap doesn't contain all live items")
	}

	h.expire(exptimeA.Add(1), nil)
	if h.nextExpiry() != exptimeB {
		t.Fatal("wrong nextExpiry")
	}
	if h.contains("a") {
		t.Fatal("heap contains a even though it has already expired")
	}
	if !h.contains("b") || !h.contains("c") {
		t.Fatal("heap doesn't contain all live items")
	}
}
