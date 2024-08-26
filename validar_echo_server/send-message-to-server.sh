#!/bin/bash

response=$(echo "$3" | nc "$1" "$2")
echo "$response"