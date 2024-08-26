from typing import *
import sys
import os
import yaml

def validate_args(args) -> bool:
  if len(args) != 2:
    print(f"[ERROR] Insuffient arguments. Given: {len(args)} needed 2")

    return False

  output_file = args[0].split(".")
  if len(output_file) != 2: 
    print(f"[ERROR] Incorrect file format, should be <file_name>.yaml, not: {args[0]}")

    return False
  
  _, extension = output_file
  if extension != "yaml":
    print(f"[ERROR] File needs to be .yaml, not {extension}")

    return False

  try:
    int(args[1])
  except ValueError:
    print(f"[ERROR] Second argument has to be numeric, {args[1]} was given instead")

    return False
  
  return True
  
def create_file(output, file_name):
  with open(file_name, 'w') as output_file:
    yaml.safe_dump(output, output_file, sort_keys=False, default_flow_style=False)

def add_server(services):
  services["server"] = {}
  services["server"]["container_name"] = "server"
  services["server"]["image"] = "server:latest"
  services["server"]["entrypoint"] = "python3 /main.py"
  services["server"]["environment"] = ["PYTHONUNBUFFERED=1", "LOGGING_LEVEL=DEBUG"]
  services["server"]["networks"] = ["testing_net"]
  services["server"]["volumes"] = ["./server/config.ini:/config.ini"]

def add_clients(services, clients):
  for i in range(1, clients + 1):
    current_client = f"client{i}"
    services[current_client] = {}
    services[current_client]["container_name"] = current_client
    services[current_client]["image"] = "client:latest"
    services[current_client]["entrypoint"] = "/client"
    services[current_client]["environment"] = [f"CLI_ID={i}", "LOGGING_LEVEL=DEBUG"]
    services[current_client]["networks"] = ["testing_net"]
    services[current_client]["depends_on"] = ["server"]
    services[current_client]["volumes"] = ["./client/config.yaml:/config.yaml"]

def add_networks(networks):
  networks["testing_net"] = {}
  networks["testing_net"]["ipam"] = {}
  networks["testing_net"]["ipam"]["driver"] = "default"
  networks["testing_net"]["ipam"]["config"] = [{"subnet": "172.25.125.0/24"}]

def generate_output(clients: int):
  output = {}

  output["name"] = "tp0"

  output["services"] = {}
  add_server(output["services"])
  add_clients(output["services"], clients)

  output["networks"] = {}
  add_networks(output["networks"])

  return output


def overwrite_existing_file(output_file_name) -> bool:
  overwrite = input(f"File {output_file_name} already exists in the current working directory, do you want to overwrite it? (y/n): ").lower()
  while overwrite != "y" and overwrite != "n":
    overwrite = input("Enter y/n: ").lower()

  if overwrite == 'n': 
    return False
  
  return True

def main(args):
  if not validate_args(args): return
  
  output_file_name, clients = args[0], int(args[1])
  if os.path.exists(output_file_name):
    if not overwrite_existing_file(output_file_name): return

  output = generate_output(clients)

  create_file(output, output_file_name)

  print(f"{output_file_name} was successfully created")

if __name__ == '__main__':
  main(sys.argv[1:])