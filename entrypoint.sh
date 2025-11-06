#!/bin/sh
set -e

echo "Starting Tailscale daemon..."
# Start tailscaled in the background
tailscaled --state=/var/lib/tailscale/tailscaled.state --socket=/var/run/tailscale/tailscaled.sock &
TAILSCALED_PID=$!

# Wait for tailscaled to be ready
sleep 2

echo "Connecting to Tailscale network..."
# Build the tailscale up command
TAILSCALE_CMD="tailscale up --authkey=${TS_AUTHKEY} --ephemeral --accept-routes"

# Add exit-node if EXIT_NODE_IP is provided
if [ -n "${EXIT_NODE_IP}" ]; then
    TAILSCALE_CMD="${TAILSCALE_CMD} --exit-node=${EXIT_NODE_IP}"
    echo "Using exit node: ${EXIT_NODE_IP}"
fi

# Connect to Tailscale
eval ${TAILSCALE_CMD}

echo "Tailscale connected successfully!"
echo "Starting marmithon..."

# Start marmithon as nonroot user in the foreground
exec su-exec nonroot /app/marmithon -config /app/marmithon.toml
