#!/bin/bash
set -e

# This script starts a container with the --privileged flag and
# then checks the permissions of the created container.
#
# Use this in CI Pipelines to check for a valid setup.

CID=$(docker run -d --rm --privileged alpine sleep 5)

if [[ $(docker inspect --format='{{.HostConfig.Privileged}}' $CID) != "true" ]]; then
    echo "Could not start privileged container."
    exit 1
else
    echo "Docker privilege looks good."
fi
