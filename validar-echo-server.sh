#!/bin/bash

docker build ./validar_echo_server -t validador_echo_server 

config_file="./server/config.ini"

server_tag=$(grep -i 'SERVER_IP' "$config_file" | awk -F'=' '{print $2}' | xargs)
server_port=$(grep -i 'SERVER_PORT' "$config_file" | awk -F'=' '{print $2}' | xargs)

message_sent="Hello echo server!"

docker run -d --name validador --network tp0_testing_net validador_echo_server "$server_tag" "$server_port" "$message_sent" > /dev/null 2>&1

response=$(docker logs validador)

if [[ "$response" == "$message_sent" ]]; then
  echo "action: test_echo_server | result: success"
else 
  echo "action: test_echo_server | result: fail."
fi

docker stop validador > /dev/null 2>&1 #Discard both stdout and err
docker rm validador > /dev/null 2>&1
docker rmi validador_echo_server > /dev/null 2>&1