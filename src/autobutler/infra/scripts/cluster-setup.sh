#!/bin/bash

set -e

BW_INFRA=/Users/brandonapol/Documents/code/birchwood-infra

# kubectl create ns argo-cd
# kubectl create ns istio-system
# kubectl create ns istio-gateway
# kubectl label ns argo-cd istio-injection=enabled
# kubectl label ns istio-gateway istio-injection=enabled

# helm repo add istio https://istio-release.storage.googleapis.com/charts
# helm dependency build ${BW_INFRA}/cluster-nodes/platform/istio/istio-base
# helm dependency build ${BW_INFRA}/cluster-nodes/platform/istio/istio-gateway-ingress
# helm dependency build ${BW_INFRA}/cluster-nodes/platform/istio/istiod
# helm install istio-base ${BW_INFRA}/cluster-nodes/platform/istio/istio-base -n istio-system
# helm install istiod ${BW_INFRA}/cluster-nodes/platform/istio/istiod -n istio-system --wait
# helm install istio-gateway-ingress ${BW_INFRA}/cluster-nodes/platform/istio/istio-gateway-ingress -n istio-gateway

helm repo add argo https://argoproj.github.io/argo-helm 2>/dev/null || true
helm dependency build "${BW_INFRA}/cluster-nodes/platform/argo-cd"
helm install argo-cd "${BW_INFRA}/cluster-nodes/platform/argo-cd" -n argo-cd