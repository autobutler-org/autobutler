#!/bin/bash

cd "$REPO" || exit
eval $(minikube docker-env)
echo "Building blog"
cd "$REPO"/birchwood-blog || exit
docker build -t "birchwood-blog" -f "$REPO/birchwood-blog/Dockerfile" .
echo "Building server"
cd "$REPO"/birchwood-server || exit
docker build -t "birchwood-server" -f "$REPO/birchwood-server/Dockerfile" .
echo "Building CMS"
cd "$REPO"/birchwood-CMS || exit
docker build -t "birchwood-cms" -f "$REPO/birchwood-CMS/Dockerfile" .
echo "Done building"