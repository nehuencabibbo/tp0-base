#!/bin/bash

# Get the list of all image IDs
image_ids=$(docker images -q)

# Check if there are any image IDs to remove
if [ -n "$image_ids" ]; then
    docker rmi $image_ids
else
    echo "No Docker images found."
fi