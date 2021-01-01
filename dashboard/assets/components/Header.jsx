// @flow

// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

import React, {Component} from 'react';

import withStyles from 'material-ui/styles/withStyles';
import AppBar from 'material-ui/AppBar';
import Toolbar from 'material-ui/Toolbar';
import IconButton from 'material-ui/IconButton';
import Icon from 'material-ui/Icon';
import MenuIcon from 'material-ui-icons/Menu';
import Typography from 'material-ui/Typography';

// styles contains the constant styles of the component.
const styles = {
	header: {
		height: '8%',
	},
	toolbar: {
		height: '100%',
	},
};

// themeStyles returns the styles generated from the theme for the component.
const themeStyles = (theme: Object) => ({
	header: {
		backgroundColor: theme.palette.grey[900],
		color:           theme.palette.getContrastText(theme.palette.grey[900]),
		zIndex:          theme.zIndex.appBar,
	},
	toolbar: {
		paddingLeft:  theme.spacing.unit,
		paddingRight: theme.spacing.unit,
	},
	title: {
		paddingLeft: theme.spacing.unit,
		fontSize:    3 * theme.spacing.unit,
	},
});

export type Props = {
	classes: Object, // injected by withStyles()
	switchSideBar: () => void,
};

// Header renders the header of the dashboard.
class Header extends Component<Props> {
	render() {
		const {classes} = this.props;

		return (
			<AppBar position='static' className={classes.header} style={styles.header}>
				<Toolbar className={classes.toolbar} style={styles.toolbar}>
					<IconButton onClick={this.props.switchSideBar}>
						<Icon>
							<MenuIcon />
						</Icon>
					</IconButton>
					<Typography type='title' color='inherit' noWrap className={classes.title}>
						Go fafereum Dashboard
					</Typography>
				</Toolbar>
			</AppBar>
		);
	}
}

export default withStyles(themeStyles)(Header);
