#!/bin/bash

echo "Stopping TinyLlama service..."
sudo systemctl stop tinyllama || true
sudo systemctl disable tinyllama || true

echo "Removing service file..."
sudo rm -f /etc/systemd/system/tinyllama.service

echo "Removing TinyLlama installation..."
sudo rm -rf /opt/tinyllama

echo "Removing Go installation..."
sudo rm -rf /usr/local/go
sudo rm -f /usr/local/go1.21.6.linux-arm64.tar.gz

echo "Cleaning up system packages..."
sudo apt-get remove -y python3-pip build-essential git wget || true
sudo apt-get autoremove -y || true
sudo apt-get clean || true

echo "Reloading systemd..."
sudo systemctl daemon-reload

echo "Cleanup complete!" 