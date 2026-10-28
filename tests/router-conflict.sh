#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
set -eu
# shellcheck source=../files/dependencies.sh
. "${1:?Usage: router-conflict.sh DEPENDENCIES}"
uci() { [ "$TEST_SWITCH" = 1 ] && echo 1 || echo 0; }
ubus() { printf '{"zirgozar":{"instances":{"core":{"running":%s}}}}\n' "$TEST_RUNNING"; }
nft() { [ "$TEST_TABLE" = 1 ] && echo 'table inet zirgozar' || true; }
TEST_SWITCH=1 TEST_RUNNING=false TEST_TABLE=0
rozaneh_proxy_conflict
[ "$conflict_name" = zirgozar ]
echo 'Saved Zirgozar switch blocks startup without firewall rules'
TEST_SWITCH=0 TEST_RUNNING=true
rozaneh_proxy_conflict
[ "$conflict_name" = zirgozar ]
echo 'Running Zirgozar instance blocks startup with its switch off'
TEST_RUNNING=false TEST_TABLE=1
rozaneh_proxy_conflict
echo 'Existing firewall rules block startup'
TEST_TABLE=0
if rozaneh_proxy_conflict; then echo 'False conflict' >&2; exit 1; fi
echo 'Inactive Zirgozar permits startup'
