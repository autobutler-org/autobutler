#!/bin/bash

export NODES_DIR="$REPO"/birchwood-k8s/cluster-nodes

# Install blog
echo "Installing blog..."
helm install "birchwood-blog" -n "blog"

# Install CMS
echo "Installing CMS..."
helm install birchwood-cms -n cms "$NODES_DIR"/bw-cms

# Install server
echo "Installing server..."
helm install birchwood-server -n server "$NODES_DIR"/bw-server

# Install MongoDB
echo "Installing MongDB..."
helm install mongodb-bw \
    --set auth.rootPassword=secretpassword,auth.username=my-user,auth.password=my-password,auth.database=my-database \
    oci://registry-1.docker.io/bitnamicharts/mongodb