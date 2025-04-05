#!/bin/bash

while true
do
  output=$(kubectl get pods)
  tput cuu 3
  tput el
  tput el
  tput el
  tput el
  echo "$output"
  sleep 2
done
