#!/bin/sh

echo "Invoking the function"
# Se non è stato fornito il parametro richiesto, stampa l'usage e termina lo script
if [ -z "$1" ]; then
  echo "Usage: $0 <function_name>"
  echo "Example: $0 sleepFunc2"
  exit 1
fi

# Esegui il comando con il parametro fornito
./../../bin/serverledge-cli invoke -f "$1"

