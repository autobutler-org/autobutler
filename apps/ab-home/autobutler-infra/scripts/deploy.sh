#!/bin/bash

# minikube start --cpus=4 --memory=3072 --disk-size=20g

export REPO="/Users/brandonapol/Documents/code/Birchwood/"
cd "$REPO" || exit

# Set up namespaces
kubectl apply -f $REPO/birchwood-k8s/namespaces/

# setup istio
# istioctl install --set profile=demo -y
# kubectl label namespace default istio-injection=enabled



./build.sh 
./install.sh 
