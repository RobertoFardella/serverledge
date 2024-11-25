#!/bin/sh

echo "Invoking the function"

if [ -z "$1" ]; then
  echo "Usage: $0 <function_name> [num_instances]"
  echo "Example: $0 Func 5"
  exit 1
fi

FUNCTION_NAME="$1"
NUM_INSTANCES="${2:-1}"  # Default a 1 se non specificato

# Avvia le istanze della funzione
for i in $(seq 1 $NUM_INSTANCES); do
  echo "Invoking instance $i of $NUM_INSTANCES"
  gnome-terminal -- bash -c "./../../bin/serverledge-cli invoke -f \"$FUNCTION_NAME\"; exec bash"
done

wait 
