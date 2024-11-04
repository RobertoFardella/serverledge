echo "Creating the sleep function using serverledge-cli"
#!/bin/sh

# Se non sono stati forniti i parametri richiesti, stampa l'usage e termina lo script
if [ -z "$1" ] || [ -z "$2" ]; then
  echo "Usage: $0 <function_name> <max_istances>"
  echo "Example: $0 funcName 5"
  exit 1
fi

# Esegui il comando con i parametri forniti
./../../bin/serverledge-cli create -f "$1" --memory 256 --max_istances "$2" --runtime python310 --handler sleep.handler --src sleep_function/sleep.py
