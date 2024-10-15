echo "Invoking the function"


if [ -z "$1" ]; then
  echo "Usage: $0 <function_name> [num_invocations]"
  echo "Example: $0 sleepFunc2 5"
  exit 1
fi

FUNCTION_NAME="$1"
NUM_INVOCATIONS="${2:-1}"  # Se non specificato, default a 1


for i in $(seq 1 $NUM_INVOCATIONS); do
  echo "Invocazione $i di $NUM_INVOCATIONS"
  ./../../bin/serverledge-cli invoke -f "$FUNCTION_NAME"
done


