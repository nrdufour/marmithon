#!/bin/sh
set -e

# Check if Gluetun VPN sidecar is present (via health endpoint)
if wget --spider -q http://localhost:9999 2>/dev/null; then
    echo "Gluetun VPN sidecar detected. Waiting for VPN to be ready..."
    until wget --spider -q http://localhost:9999 2>/dev/null; do
        echo "VPN not ready, waiting 5s..."
        sleep 5
    done
    echo "VPN is ready!"
else
    echo "No Gluetun VPN sidecar detected, starting directly..."
fi

echo "Starting Marmithon..."
exec /app/marmithon "$@"
