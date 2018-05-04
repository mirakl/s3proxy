#! /bin/bash

set -e

trap "docker-compose down" SIGINT SIGQUIT SIGTERM TERM EXIT

docker-compose up
