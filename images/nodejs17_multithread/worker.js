const { workerData, parentPort } = require('worker_threads');
const path = require('path');

try {
    // Estrai i dati passati dal thread principale
    const { handler, handler_dir, params, context, return_output } = workerData;

    // Carica il gestore dinamicamente
    const handlerModule = require(path.join(handler_dir, handler));

    // Esegui il gestore
    const result = handlerModule(params, context);

    // Prepara la risposta
    const resp = {
        Result: JSON.stringify(result),
        Success: true,
        Output: return_output
            ? "Output capture not supported for this runtime yet."
            : ""
    };

    // Invia il risultato al thread principale
    parentPort.postMessage(resp);
} catch (error) {
    // Gestione errori
    const resp = {
        Success: false,
        Output: "Output capture not supported for this runtime yet.",
        Error: error.message
    };
    parentPort.postMessage(resp);
}
