const path = require('path');
const http = require('http');
const { Worker } = require('worker_threads');

// Funzione per eseguire l'elaborazione in un Worker
function runWorker(data) {
    return new Promise((resolve, reject) => {
        const worker = new Worker('./worker.js', { workerData: data }); // Invia i dati al Worker

        worker.on('message', resolve); // Quando il Worker termina con successo
        worker.on('error', reject);   // Gestione errori dal Worker
        worker.on('exit', (code) => {
            if (code !== 0) reject(new Error(`Worker terminato con codice ${code}`));
        });
    });
}

// Server HTTP
http.createServer(async (request, response) => {
    if (request.method !== 'POST') {
        response.writeHead(404);
        response.end('Invalid request method');
    } else {
        const buffers = [];

        for await (const chunk of request) {
            buffers.push(chunk);
        }

        const data = Buffer.concat(buffers).toString();
        const contentType = 'application/json';

        try {
            const reqbody = JSON.parse(data);

            // Prepara i dati da inviare al Worker
            const workerData = {
                handler: reqbody["Handler"],
                handler_dir: reqbody["HandlerDir"],
                params: reqbody["Params"],
                context: process.env.CONTEXT !== "undefined" ? process.env.CONTEXT : {},
                return_output: reqbody["ReturnOutput"]
            };

            // Esegui il Worker per l'elaborazione
            const result = await runWorker(workerData);

            response.writeHead(200, { 'Content-Type': contentType });
            response.end(JSON.stringify(result), 'utf-8');
        } catch (error) {
            const resp = {
                Success: false,
                Output: "Output capture not supported for this runtime yet.",
                Error: error.message
            };
            response.writeHead(500, { 'Content-Type': contentType });
            response.end(JSON.stringify(resp), 'utf-8');
        }
    }
}).listen(8080);

console.log('Server in ascolto sulla porta 8080');





