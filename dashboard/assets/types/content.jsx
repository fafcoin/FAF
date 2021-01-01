// @flow

// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

export type Content = {
	general: General,
	home:    Home,
	chain:   Chain,
	txpool:  TxPool,
	network: Network,
	system:  System,
	logs:    Logs,
};

export type ChartEntries = Array<ChartEntry>;

export type ChartEntry = {
	time:  Date,
	value: number,
};

export type General = {
	version: ?string,
	commit:  ?string,
};

export type Home = {
	/* TODO (kurkomisi) */
};

export type Chain = {
	/* TODO (kurkomisi) */
};

export type TxPool = {
	/* TODO (kurkomisi) */
};

export type Network = {
	/* TODO (kurkomisi) */
};

export type System = {
	activeMemory:   ChartEntries,
	virtualMemory:  ChartEntries,
	networkIngress: ChartEntries,
	networkEgress:  ChartEntries,
	processCPU:     ChartEntries,
	systemCPU:      ChartEntries,
	diskRead:       ChartEntries,
	diskWrite:      ChartEntries,
};

export type Record = {
	t:   string,
	lvl: Object,
	msg: string,
	ctx: Array<string>
};

export type Chunk = {
	content: string,
	name:    string,
};

export type Logs = {
	chunks:        Array<Chunk>,
	endTop:        boolean,
	endBottom:     boolean,
	topChanged:    number,
	bottomChanged: number,
};

export type LogsMessage = {
	source: ?LogFile,
	chunk:  Array<Record>,
};

export type LogFile = {
	name: string,
	last: string,
};
