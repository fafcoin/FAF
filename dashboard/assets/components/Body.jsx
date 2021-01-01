// @flow

// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

import React, {Component} from 'react';

import SideBar from './SideBar';
import Main from './Main';
import type {Content} from '../types/content';

// styles contains the constant styles of the component.
const styles = {
	body: {
		display: 'flex',
		width:   '100%',
		height:  '92%',
	},
};

export type Props = {
	opened:        boolean,
	changeContent: string => void,
	active:        string,
	content:       Content,
	shouldUpdate:  Object,
	send:          string => void,
};

// Body renders the body of the dashboard.
class Body extends Component<Props> {
	render() {
		return (
			<div style={styles.body}>
				<SideBar
					opened={this.props.opened}
					changeContent={this.props.changeContent}
				/>
				<Main
					active={this.props.active}
					content={this.props.content}
					shouldUpdate={this.props.shouldUpdate}
					send={this.props.send}
				/>
			</div>
		);
	}
}

export default Body;
