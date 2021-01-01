// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

// Package deps contains the console JavaScript dependencies Go embedded.
package deps

//go:generate go-bindata -nometadata -pkg deps -o bindata.go bignumber.js
//go:generate gofmt -w -s bindata.go
