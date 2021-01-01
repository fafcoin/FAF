#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
fafdir="$workspace/src/github.com/fafereum"
if [ ! -L "$fafdir/go-fafereum" ]; then
    mkdir -p "$fafdir"
    cd "$fafdir"
    ln -s ../../../../../. go-fafereum
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$fafdir/go-fafereum"
PWD="$fafdir/go-fafereum"

# Launch the arguments with the configured environment.
exec "$@"
