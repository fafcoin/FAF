// @flow

// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

import React from 'react';
import {render} from 'react-dom';

import MuiThemeProvider from 'material-ui/styles/MuiThemeProvider';
import createMuiTheme from 'material-ui/styles/createMuiTheme';

import Dashboard from './components/Dashboard';

const theme: Object = createMuiTheme({
	palette: {
		type: 'dark',
	},
});
const dashboard = document.getElementById('dashboard');
if (dashboard) {
	// Renders the whole dashboard.
	render(
		<MuiThemeProvider theme={theme}>
			<Dashboard />
		</MuiThemeProvider>,
		dashboard,
	);
}
