// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

// fa-only-woff-loader removes the .eot, .ttf, .svg dependencies of the FontAwesome library,
// because they produce unused extra blobs.
module.exports = function(content) {
	return content
		.replace(/src.*url(?!.*url.*(\.eot)).*(\.eot)[^;]*;/,'')
		.replace(/url(?!.*url.*(\.eot)).*(\.eot)[^,]*,/,'')
		.replace(/url(?!.*url.*(\.ttf)).*(\.ttf)[^,]*,/,'')
		.replace(/,[^,]*url(?!.*url.*(\.svg)).*(\.svg)[^;]*;/,';');
};
