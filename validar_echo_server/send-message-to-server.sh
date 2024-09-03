#!/bin/bash
timeout=20
response=$(echo "$3" | nc -w $timeout "$1" "$2")
echo "$response"