// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package fuse

import (
	"sync"
	"time"

	"github.com/fafereum/go-fafereum/swarm/api"
)

const (
	Swarmfs_Version = "0.1"
	mountTimeout    = time.Second * 5
	unmountTimeout  = time.Second * 10
	maxFuseMounts   = 5
)

var (
	swarmfs     *SwarmFS // Swarm file system singleton
	swarmfsLock sync.Once

	inode     uint64 = 1 // global inode
	inodeLock sync.RWMutex
)

type SwarmFS struct {
	swarmApi     *api.API
	activeMounts map[string]*MountInfo
	swarmFsLock  *sync.RWMutex
}

func NewSwarmFS(api *api.API) *SwarmFS {
	swarmfsLock.Do(func() {
		swarmfs = &SwarmFS{
			swarmApi:     api,
			swarmFsLock:  &sync.RWMutex{},
			activeMounts: map[string]*MountInfo{},
		}
	})
	return swarmfs

}

// Inode numbers need to be unique, they are used for caching inside fuse
func NewInode() uint64 {
	inodeLock.Lock()
	defer inodeLock.Unlock()
	inode += 1
	return inode
}
