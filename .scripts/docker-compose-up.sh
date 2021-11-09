#!/bin/bash
set -e

# Runs 'docker compose up' with arguments for the CI.
# Checks the output of the tests to verify that they succeeded.
# The string 'OK (1 test)' has to occur four times in the output.
test $(docker-compose up --force-recreate --remove-orphans --abort-on-container-exit --exit-code-from tester 2>&1 | tee full.log | grep -c 'OK (1 test)') -eq 4 || (cat full.log; exit 1)
