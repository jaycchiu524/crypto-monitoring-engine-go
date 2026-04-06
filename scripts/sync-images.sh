#!/bin/bash

# --- Crypto Monitoring Image Sync Script ---
# This script builds the Go application on your Mac and pipes the resulting 
# Docker image directly into the K3s (containerd) storage on your RHEL homelab.
# This bypasses the need for a public Docker registry.

set -e

# Configuration
IMAGE_NAME="crypto-monitor:latest"
REMOTE_HOST="homelab-ansible" # Uses the entry from your ~/.ssh/config

echo "--- 1. Building Docker image on Mac (Target: linux/amd64) ---"
# Build the binary and container for the RHEL architecture (amd64)
docker build --platform linux/amd64 -t $IMAGE_NAME ./go

echo "--- 2. Transferring and Importing Image to K3s ---"
# This is a high-performance "piped" command explaining it step-by-step:
# a. 'docker save' converts the image into a tar stream and sends it to stdout.
# b. '|' (the pipe) sends that stream over the network via SSH.
# c. 'ssh $REMOTE_HOST' executes the import command on your RHEL machine.
# d. '/usr/local/bin/k3s ctr images import -' reads the stream from stdin and loads it into containerd.
docker save $IMAGE_NAME | ssh $REMOTE_HOST "sudo /usr/local/bin/k3s ctr images import -"

echo "--- 3. Sync Complete ---"
echo "You can now run: kubectl apply -f k8s/ to deploy the latest version."
