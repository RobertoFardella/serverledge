#!/bin/bash

# Check if the correct number of arguments is provided
if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <IMAGETAG>"
  exit 1
fi

# Set the image tag from the first argument
IMAGETAG=$1

# Step 1: Build the Docker image
echo "Building the Docker image with tag: $IMAGETAG"
docker build -t $IMAGETAG .




