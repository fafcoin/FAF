// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

// +build ignore

package main

// This program generates table.go and table_test.go.
// Invoke as
//
//	go run gen.go |gofmt >table.go
//	go run gen.go -test |gofmt >table_test.go

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
)

// identifier converts s to a Go exported identifier.
// It converts "div" to "Div" and "accept-charset" to "AcceptCharset".
func identifier(s string) string {
	b := make([]byte, 0, len(s))
	cap := true
	for _, c := range s {
		if c == '-' {
			cap = true
			continue
		}
		if cap && 'a' <= c && c <= 'z' {
			c -= 'a' - 'A'
		}
		cap = false
		b = append(b, byte(c))
	}
	return string(b)
}

var test = flag.Bool("test", false, "generate table_test.go")

func main() {
	flag.Parse()

	var all []string
	all = append(all, elements...)
	all = append(all, attributes...)
	all = append(all, eventHandlers...)
	all = append(all, extra...)
	sort.Strings(all)

	if *test {
		fmt.Printf("// generated by go run gen.go -test; DO NOT EDIT\n\n")
		fmt.Printf("package atom\n\n")
		fmt.Printf("var testAtomList = []string{\n")
		for _, s := range all {
			fmt.Printf("\t%q,\n", s)
		}
		fmt.Printf("}\n")
		return
	}

	// uniq - lists have dups
	// compute max len too
	maxLen := 0
	w := 0
	for _, s := range all {
		if w == 0 || all[w-1] != s {
			if maxLen < len(s) {
				maxLen = len(s)
			}
			all[w] = s
			w++
		}
	}
	all = all[:w]

	// Find hash that minimizes table size.
	var best *table
	for i := 0; i < 1000000; i++ {
		if best != nil && 1<<(best.k-1) < len(all) {
			break
		}
		h := rand.Uint32()
		for k := uint(0); k <= 16; k++ {
			if best != nil && k >= best.k {
				break
			}
			var t table
			if t.init(h, k, all) {
				best = &t
				break
			}
		}
	}
	if best == nil {
		fmt.Fprintf(os.Stderr, "failed to construct string table\n")
		os.Exit(1)
	}

	// Lay out strings, using overlaps when possible.
	layout := append([]string{}, all...)

	// Remove strings that are substrings of other strings
	for changed := true; changed; {
		changed = false
		for i, s := range layout {
			if s == "" {
				continue
			}
			for j, t := range layout {
				if i != j && t != "" && strings.Contains(s, t) {
					changed = true
					layout[j] = ""
				}
			}
		}
	}

	// Join strings where one suffix matches another prefix.
	for {
		// Find best i, j, k such that layout[i][len-k:] == layout[j][:k],
		// maximizing overlap length k.
		besti := -1
		bestj := -1
		bestk := 0
		for i, s := range layout {
			if s == "" {
				continue
			}
			for j, t := range layout {
				if i == j {
					continue
				}
				for k := bestk + 1; k <= len(s) && k <= len(t); k++ {
					if s[len(s)-k:] == t[:k] {
						besti = i
						bestj = j
						bestk = k
					}
				}
			}
		}
		if bestk > 0 {
			layout[besti] += layout[bestj][bestk:]
			layout[bestj] = ""
			continue
		}
		break
	}

	text := strings.Join(layout, "")

	atom := map[string]uint32{}
	for _, s := range all {
		off := strings.Index(text, s)
		if off < 0 {
			panic("lost string " + s)
		}
		atom[s] = uint32(off<<8 | len(s))
	}

	// Generate the Go code.
	fmt.Printf("// generated by go run gen.go; DO NOT EDIT\n\n")
	fmt.Printf("package atom\n\nconst (\n")
	for _, s := range all {
		fmt.Printf("\t%s Atom = %#x\n", identifier(s), atom[s])
	}
	fmt.Printf(")\n\n")

	fmt.Printf("const hash0 = %#x\n\n", best.h0)
	fmt.Printf("const maxAtomLen = %d\n\n", maxLen)

	fmt.Printf("var table = [1<<%d]Atom{\n", best.k)
	for i, s := range best.tab {
		if s == "" {
			continue
		}
		fmt.Printf("\t%#x: %#x, // %s\n", i, atom[s], s)
	}
	fmt.Printf("}\n")
	datasize := (1 << best.k) * 4

	fmt.Printf("const atomText =\n")
	textsize := len(text)
	for len(text) > 60 {
		fmt.Printf("\t%q +\n", text[:60])
		text = text[60:]
	}
	fmt.Printf("\t%q\n\n", text)

	fmt.Fprintf(os.Stderr, "%d atoms; %d string bytes + %d tables = %d total data\n", len(all), textsize, datasize, textsize+datasize)
}

type byLen []string

func (x byLen) Less(i, j int) bool { return len(x[i]) > len(x[j]) }
func (x byLen) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x byLen) Len() int           { return len(x) }

// fnv computes the FNV hash with an arbitrary starting value h.
func fnv(h uint32, s string) uint32 {
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

// A table represents an attempt at constructing the lookup table.
// The lookup table uses cuckoo hashing, meaning that each string
// can be found in one of two positions.
type table struct {
	h0   uint32
	k    uint
	mask uint32
	tab  []string
}

// hash returns the two hashes for s.
func (t *table) hash(s string) (h1, h2 uint32) {
	h := fnv(t.h0, s)
	h1 = h & t.mask
	h2 = (h >> 16) & t.mask
	return
}

// init initializes the table with the given parameters.
// h0 is the initial hash value,
// k is the number of bits of hash value to use, and
// x is the list of strings to store in the table.
// init returns false if the table cannot be constructed.
func (t *table) init(h0 uint32, k uint, x []string) bool {
	t.h0 = h0
	t.k = k
	t.tab = make([]string, 1<<k)
	t.mask = 1<<k - 1
	for _, s := range x {
		if !t.insert(s) {
			return false
		}
	}
	return true
}

// insert inserts s in the table.
func (t *table) insert(s string) bool {
	h1, h2 := t.hash(s)
	if t.tab[h1] == "" {
		t.tab[h1] = s
		return true
	}
	if t.tab[h2] == "" {
		t.tab[h2] = s
		return true
	}
	if t.push(h1, 0) {
		t.tab[h1] = s
		return true
	}
	if t.push(h2, 0) {
		t.tab[h2] = s
		return true
	}
	return false
}

// push attempts to push aside the entry in slot i.
func (t *table) push(i uint32, depth int) bool {
	if depth > len(t.tab) {
		return false
	}
	s := t.tab[i]
	h1, h2 := t.hash(s)
	j := h1 + h2 - i
	if t.tab[j] != "" && !t.push(j, depth+1) {
		return false
	}
	t.tab[j] = s
	return true
}

// The lists of element names and attribute keys were taken from
// https://html.spec.whatwg.org/multipage/indices.html#index
// as of the "HTML Living Standard - Last Updated 21 February 2015" version.

var elements = []string{
	"a",
	"abbr",
	"address",
	"area",
	"article",
	"aside",
	"audio",
	"b",
	"base",
	"bdi",
	"bdo",
	"blockquote",
	"body",
	"br",
	"button",
	"canvas",
	"caption",
	"cite",
	"code",
	"col",
	"colgroup",
	"command",
	"data",
	"datalist",
	"dd",
	"del",
	"details",
	"dfn",
	"dialog",
	"div",
	"dl",
	"dt",
	"em",
	"embed",
	"fieldset",
	"figcaption",
	"figure",
	"footer",
	"form",
	"h1",
	"h2",
	"h3",
	"h4",
	"h5",
	"h6",
	"head",
	"header",
	"hgroup",
	"hr",
	"html",
	"i",
	"iframe",
	"img",
	"input",
	"ins",
	"kbd",
	"keygen",
	"label",
	"legend",
	"li",
	"link",
	"map",
	"mark",
	"menu",
	"menuitem",
	"meta",
	"meter",
	"nav",
	"noscript",
	"object",
	"ol",
	"optgroup",
	"option",
	"output",
	"p",
	"param",
	"pre",
	"progress",
	"q",
	"rp",
	"rt",
	"ruby",
	"s",
	"samp",
	"script",
	"section",
	"select",
	"small",
	"source",
	"span",
	"strong",
	"style",
	"sub",
	"summary",
	"sup",
	"table",
	"tbody",
	"td",
	"template",
	"textarea",
	"tfoot",
	"th",
	"thead",
	"time",
	"title",
	"tr",
	"track",
	"u",
	"ul",
	"var",
	"video",
	"wbr",
}

// https://html.spec.whatwg.org/multipage/indices.html#attributes-3

var attributes = []string{
	"abbr",
	"accept",
	"accept-charset",
	"accesskey",
	"action",
	"alt",
	"async",
	"autocomplete",
	"autofocus",
	"autoplay",
	"challenge",
	"charset",
	"checked",
	"cite",
	"class",
	"cols",
	"colspan",
	"command",
	"content",
	"contenteditable",
	"contextmenu",
	"controls",
	"coords",
	"crossorigin",
	"data",
	"datetime",
	"default",
	"defer",
	"dir",
	"dirname",
	"disabled",
	"download",
	"draggable",
	"dropzone",
	"enctype",
	"for",
	"form",
	"formaction",
	"formenctype",
	"formmfafod",
	"formnovalidate",
	"formtarget",
	"headers",
	"height",
	"hidden",
	"high",
	"href",
	"hreflang",
	"http-equiv",
	"icon",
	"id",
	"inputmode",
	"ismap",
	"itemid",
	"itemprop",
	"itemref",
	"itemscope",
	"itemtype",
	"keytype",
	"kind",
	"label",
	"lang",
	"list",
	"loop",
	"low",
	"manifest",
	"max",
	"maxlength",
	"media",
	"mediagroup",
	"mfafod",
	"min",
	"minlength",
	"multiple",
	"muted",
	"name",
	"novalidate",
	"open",
	"optimum",
	"pattern",
	"ping",
	"placeholder",
	"poster",
	"preload",
	"radiogroup",
	"readonly",
	"rel",
	"required",
	"reversed",
	"rows",
	"rowspan",
	"sandbox",
	"spellcheck",
	"scope",
	"scoped",
	"seamless",
	"selected",
	"shape",
	"size",
	"sizes",
	"sortable",
	"sorted",
	"span",
	"src",
	"srcdoc",
	"srclang",
	"start",
	"step",
	"style",
	"tabindex",
	"target",
	"title",
	"translate",
	"type",
	"typemustmatch",
	"usemap",
	"value",
	"width",
	"wrap",
}

var eventHandlers = []string{
	"onabort",
	"onautocomplete",
	"onautocompleteerror",
	"onafterprint",
	"onbeforeprint",
	"onbeforeunload",
	"onblur",
	"oncancel",
	"oncanplay",
	"oncanplaythrough",
	"onchange",
	"onclick",
	"onclose",
	"oncontextmenu",
	"oncuechange",
	"ondblclick",
	"ondrag",
	"ondragend",
	"ondragenter",
	"ondragleave",
	"ondragover",
	"ondragstart",
	"ondrop",
	"ondurationchange",
	"onemptied",
	"onended",
	"onerror",
	"onfocus",
	"onhashchange",
	"oninput",
	"oninvalid",
	"onkeydown",
	"onkeypress",
	"onkeyup",
	"onlanguagechange",
	"onload",
	"onloadeddata",
	"onloadedmetadata",
	"onloadstart",
	"onmessage",
	"onmousedown",
	"onmousemove",
	"onmouseout",
	"onmouseover",
	"onmouseup",
	"onmousewheel",
	"onoffline",
	"ononline",
	"onpagehide",
	"onpageshow",
	"onpause",
	"onplay",
	"onplaying",
	"onpopstate",
	"onprogress",
	"onratechange",
	"onreset",
	"onresize",
	"onscroll",
	"onseeked",
	"onseeking",
	"onselect",
	"onshow",
	"onsort",
	"onstalled",
	"onstorage",
	"onsubmit",
	"onsuspend",
	"ontimeupdate",
	"ontoggle",
	"onunload",
	"onvolumechange",
	"onwaiting",
}

// extra are ad-hoc values not covered by any of the lists above.
var extra = []string{
	"align",
	"annotation",
	"annotation-xml",
	"applet",
	"basefont",
	"bgsound",
	"big",
	"blink",
	"center",
	"color",
	"desc",
	"face",
	"font",
	"foreignObject", // HTML is case-insensitive, but SVG-embedded-in-HTML is case-sensitive.
	"foreignobject",
	"frame",
	"frameset",
	"image",
	"isindex",
	"listing",
	"malignmark",
	"marquee",
	"math",
	"mglyph",
	"mi",
	"mn",
	"mo",
	"ms",
	"mtext",
	"nobr",
	"noembed",
	"noframes",
	"plaintext",
	"prompt",
	"public",
	"spacer",
	"strike",
	"svg",
	"system",
	"tt",
	"xmp",
}
