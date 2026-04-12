#!/usr/bin/env bash
set -euo pipefail

mkdir -p ../../Temp/logs
npm run dev 2>&1 | tee ../../Temp/logs/launchpad-dev.log
