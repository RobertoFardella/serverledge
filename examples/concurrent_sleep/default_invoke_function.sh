#!/bin/sh

echo "Invoking the function"

if [ -z "$1" ]; then
  echo "Usage: $0 <function_name>"
  echo "Example: $0 Func"
  exit 1
fi

./../../bin/serverledge-cli invoke -f "$1"

