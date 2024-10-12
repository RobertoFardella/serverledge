echo "Creating the function using serverledge-cli"
#!/bin/sh

# Se non sono stati forniti i parametri richiesti, stampa l'usage e termina lo script
if [ -z "$1" ] || [ -z "$2" ]; then
  echo "Usage: $0 <function_name> <custom_image>"
  echo "Example: $0 sleepFunc2 prova"
  exit 1
fi

# Esegui il comando con i parametri forniti
./../../bin/serverledge-cli create -f "$1" --memory 256 --runtime custom --custom_image "$2"

