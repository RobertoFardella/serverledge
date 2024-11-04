#!/bin/sh

if [ -z "$1" ] || [ -z "$2" ]; then
  echo "Usage: $0 <function_name> <number_of_instances>"
  echo "Example: $0 Func 3"
  exit 1
fi

echo "Invoking the function"

./../../bin/serverledge-cli invoke -f "$1" -i "$2" --params_file isPrime/input.json


