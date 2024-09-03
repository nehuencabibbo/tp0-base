#!/bin/bash
timeout=5
response=$(echo "$3" | nc -w $timeout "$1" "$2")
echo "$response"