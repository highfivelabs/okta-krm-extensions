#!/usr/bin/env bash
# Copyright 2026 High Five Labs
# SPDX-License-Identifier: Apache-2.0
# Download the vendor's OpenAPI specs at the commits vendor.yaml pins.
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p specs
[ -s specs/okta.yaml ] || curl -fsSL -o specs/okta.yaml "https://raw.githubusercontent.com/okta/okta-sdk-golang/3926996023472119e0aa9a3ca33151a511709fed/okta/api/openapi.yaml"
[ -s specs/okta-governance.yaml ] || curl -fsSL -o specs/okta-governance.yaml "https://raw.githubusercontent.com/okta/okta-governance-sdk-golang/c872f2f5e33797f6fc12b0a5bafe99a3bf76a1cf/governance/api/openapi.yaml"
