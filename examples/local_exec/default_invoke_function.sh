#!/bin/sh

if [ -z "$1" ];  then
  echo "Usage: $0 <function_name>"
  echo "Example: $0 Func 3"
  exit 1
fi

echo "Invoking the function"

./../../bin/serverledge-cli invoke -f "$1" --params_file isPrime/input.json


