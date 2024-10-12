# function.py
import time

def handler(params, context):
    try:
        # Ottiene la durata del sonno dai parametri, con un valore predefinito di 5 secondi
        sleep_duration = params.get("sleep_duration", 5)  # Usa 5 secondi se "sleep_duration" non è presente
        print(f"Pausing for {sleep_duration} seconds...")
        time.sleep(sleep_duration)
        result = {"Success": True, "Message": f"Slept for {sleep_duration} seconds"}
    except Exception as e:
        print(e)
        result = {"Success": False, "Error": str(e)}
    return result
