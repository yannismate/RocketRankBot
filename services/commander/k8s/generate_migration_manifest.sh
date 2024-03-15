#!/bin/bash

kubectl create configmap commander-migrations-configmap -n rocketrankbot --from-file=../migrations --dry-run=client -o yaml