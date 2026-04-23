#!/usr/bin/env bash
set -euo pipefail

kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

kubectl wait --for=condition=Established crd/applications.argoproj.io --timeout=5m
kubectl -n argocd wait --for=condition=Available deployment/argocd-server --timeout=10m
kubectl -n argocd wait --for=condition=Available deployment/argocd-repo-server --timeout=10m
kubectl -n argocd wait --for=condition=Available deployment/argocd-application-controller --timeout=10m
