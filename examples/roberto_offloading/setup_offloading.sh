#!/bin/bash

# setup_offloading.sh
# Questo script automatizza il processo di costruzione di un'immagine Docker,
# l'avvio dei nodi edge e cloud in finestre di terminale separate,
# la creazione e l'invocazione di una funzione, e fornisce istruzioni per visualizzare i log.

# Interrompi immediatamente se un comando esce con uno stato non zero
set -e

# ----------------------------
# Funzioni di supporto
# ----------------------------

# Funzione per visualizzare messaggi informativi
echo_info() {
    echo -e "\e[32m[INFO]\e[0m $1"
}

# Funzione per visualizzare messaggi di errore
echo_error() {
    echo -e "\e[31m[ERROR]\e[0m $1"
}

# Funzione per verificare se un comando esiste
command_exists() {
    command -v "$1" &> /dev/null
}

# ----------------------------
# Variabili di configurazione
# ----------------------------

# Percorsi ai binari di Serverledge sulla macchina locale
SERVERLEDGE_BIN="./../../bin/serverledge"
SERVERLEDGE_CLI_BIN="./../../bin/serverledge-cli"

# File di configurazione
CONF_EDGE="./confEdge.yaml"
CONF_CLOUD="./confCloud.yaml"

# Dettagli dell'immagine Docker
IMAGE_NAME="roberto-image"

# Dettagli della funzione
# Se FUNCTION_NAME non è stato passato come parametro, utilizza il valore predefinito "robertoFunc"
FUNCTION_NAME="${1:-robertoFunc}"
MEMORY="256"
RUNTIME="custom"
CUSTOM_IMAGE="$IMAGE_NAME"
PARAMS_FILE="./encoded_JSON_parameters/input_hello.json"

# File di log per i nodi edge e cloud
EDGE_LOG="./log/edge_serverledge.log"
CLOUD_LOG="./log/cloud_serverledge.log"

# Emulatore di terminale
TERMINAL="gnome-terminal"

# ----------------------------
# Pre-controlli
# ----------------------------

# Verifica se Docker è installato
if ! command_exists docker; then
    echo_error "Docker non è installato. Per favore installa Docker e riprova."
    exit 1
fi

# Verifica se l'emulatore di terminale è installato
if ! command_exists "$TERMINAL"; then
    echo_error "$TERMINAL non è installato. Per favore installalo o modifica lo script per utilizzare il tuo emulatore di terminale preferito."
    exit 1
fi

# Verifica se i binari di Serverledge esistono
if [ ! -x "$SERVERLEDGE_BIN" ] || [ ! -x "$SERVERLEDGE_CLI_BIN" ]; then
    echo_error "I binari di Serverledge non sono stati trovati o non sono eseguibili. Per favore verifica i percorsi."
    exit 1
fi

# Verifica se i file di configurazione esistono
if [ ! -f "$CONF_EDGE" ]; then
    echo_error "Il file di configurazione '$CONF_EDGE' non è stato trovato."
    exit 1
fi

if [ ! -f "$CONF_CLOUD" ]; then
    echo_error "Il file di configurazione '$CONF_CLOUD' non è stato trovato."
    exit 1
fi

# Verifica se il file dei parametri esiste
if [ ! -f "$PARAMS_FILE" ]; then
    echo_error "Il file dei parametri '$PARAMS_FILE' non è stato trovato."
    exit 1
fi

# ----------------------------
# Passo 1: Costruzione dell'immagine Docker
# ----------------------------

echo_info "Costruzione dell'immagine Docker '$IMAGE_NAME'..."
sudo docker build -t "$IMAGE_NAME" .
echo_info "Immagine Docker '$IMAGE_NAME' costruita con successo."

# ----------------------------
# Passo 2: Avvio dei nodi Edge e Cloud
# ----------------------------

start_serverledge() {
    NODE_TYPE=$1        # "Edge" o "Cloud"
    CONFIG_FILE=$2      # Percorso al file di configurazione
    LOG_FILE=$3         # Percorso al file di log

    echo_info "Avvio del nodo $NODE_TYPE..."

    # Apri una nuova finestra di terminale ed esegui il comando serverledge
    "$TERMINAL" -- bash -c "$SERVERLEDGE_BIN $CONFIG_FILE > $LOG_FILE 2>&1; exec bash" &

    echo_info "Nodo $NODE_TYPE avviato. I log sono scritti in '$LOG_FILE'."
}

# Avvio del nodo Edge
start_serverledge "Edge" "$CONF_EDGE" "$EDGE_LOG"

# Avvio del nodo Cloud
start_serverledge "Cloud" "$CONF_CLOUD" "$CLOUD_LOG"

# Attendi alcuni secondi per assicurarti che i nodi siano attivi
sleep 5

# ----------------------------
# Passo 3: Creazione della Funzione
# ----------------------------

echo_info "Creazione della funzione '$FUNCTION_NAME'..."
"$SERVERLEDGE_CLI_BIN" create -f "$FUNCTION_NAME" --memory "$MEMORY" --runtime "$RUNTIME" \
    --custom_image "$CUSTOM_IMAGE"
echo_info "Funzione '$FUNCTION_NAME' creata con successo."

# ----------------------------
# Passo 4: Invocazione della Funzione
# ----------------------------

echo_info "Invocazione della funzione '$FUNCTION_NAME' con il file dei parametri '$PARAMS_FILE'..."
"$SERVERLEDGE_CLI_BIN" invoke -f "$FUNCTION_NAME" --params_file "$PARAMS_FILE"
echo_info "Funzione '$FUNCTION_NAME' invocata con successo."

# ----------------------------
# Passo 5: Istruzioni per visualizzare i log
# ----------------------------

echo_info "Per controllare i log per eventuali problemi con la funzione, utilizza il seguente comando:"
echo "  docker container logs <Nome-Contenitore>"

# Opzionale: Chiedi all'utente se desidera visualizzare i log
read -p "Vuoi visualizzare ora i log di un container in particolare? (s/N): " VIEW_LOGS
if [[ "$VIEW_LOGS" =~ ^[Ss]$ ]]; then
    echo_info "Ecco la lista dei container Docker attualmente presenti:"
    docker ps -a
    read -p "Inserisci il nome o l'ID del container: " CONTAINER_NAME
    docker container logs "$CONTAINER_NAME"
fi

echo_info "Configurazione completata con successo."


