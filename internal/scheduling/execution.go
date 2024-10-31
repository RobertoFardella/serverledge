package scheduling

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/grussorusso/serverledge/internal/function"

	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/executor"
)

const HANDLER_DIR = "/app"

func Execute(contID container.ContainerID, r *scheduledRequest, isWarm bool) (function.ExecutionReport, error) {
	var req executor.InvocationRequest
	var Results, Outputs []string

	//var invocationWait time.Duration
	var err error
	var mutex sync.Mutex // Mutex per proteggere l’accesso ai risultati concorrenti
	var totalInvocationWait time.Duration
	var wg sync.WaitGroup
	errChan := make(chan error, r.Istance_number) // Canale per errori dalle goroutine

	log.Printf("[%s] Executing on container: %v", r.Fun, contID)

	if r.Fun.Runtime == container.CUSTOM_RUNTIME {
		req = executor.InvocationRequest{
			Params:       r.Params,
			ReturnOutput: r.ReturnOutput,
		}
	} else {
		cmd := container.RuntimeToInfo[r.Fun.Runtime].InvocationCmd
		req = executor.InvocationRequest{
			Command:      cmd,
			Params:       r.Params,
			Handler:      r.Fun.Handler,
			HandlerDir:   HANDLER_DIR,
			ReturnOutput: r.ReturnOutput,
		}
	}

	t0 := time.Now()
	initTime := t0.Sub(r.Arrival).Seconds()

	// Lancia esecuzioni parallele
	for i := 0; i < int(r.Istance_number); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			response, invocationWait, err := container.Execute(contID, &req)
			if err != nil {
				errChan <- fmt.Errorf("[%s] Execution failed: %v", r, err)
				return
			}
			if !response.Success {
				errChan <- fmt.Errorf("Function execution failed")
				return
			}

			// Protezione concorrente dei risultati
			mutex.Lock()
			defer mutex.Unlock()
			Results = append(Results, response.Result)
			Outputs = append(Outputs, response.Output)
			totalInvocationWait += invocationWait
		}()
	}

	// Attendi il completamento di tutte le goroutine
	wg.Wait()
	close(errChan)

	// Gestione errori se presenti
	if len(errChan) > 0 {
		err = <-errChan
		// notify scheduler
		completions <- &completionNotification{fun: r.Fun, contID: contID, executionReport: nil}
		return function.ExecutionReport{}, err
	}

	report := function.ExecutionReport{
		Result:       Results,
		Output:       Outputs,
		IsWarmStart:  isWarm,
		Duration:     time.Now().Sub(t0).Seconds() - totalInvocationWait.Seconds(),
		ResponseTime: time.Now().Sub(t0).Seconds() - totalInvocationWait.Seconds(),
	}

	// Tempo di inizializzazione, considerando i ritardi
	report.InitTime = initTime + totalInvocationWait.Seconds()

	// notifica il completamento al gestore
	completions <- &completionNotification{fun: r.Fun, contID: contID, executionReport: &report}

	return report, nil
}

// Execute serves a request on the specified container.
/*func Execute(contID container.ContainerID, r *scheduledRequest, isWarm bool) (function.ExecutionReport, error) {
	var req executor.InvocationRequest
	var Results, Outputs []string

	var response *executor.InvocationResult
	var invocationWait time.Duration
	var err error

	log.Printf("[%s] Executing on container: %v", r.Fun, contID)

	if r.Fun.Runtime == container.CUSTOM_RUNTIME {
		req = executor.InvocationRequest{
			Params:       r.Params,
			ReturnOutput: r.ReturnOutput,
		}
	} else {
		cmd := container.RuntimeToInfo[r.Fun.Runtime].InvocationCmd
		req = executor.InvocationRequest{
			Command:      cmd,
			Params:       r.Params,
			Handler:      r.Fun.Handler,
			HandlerDir:   HANDLER_DIR,
			ReturnOutput: r.ReturnOutput,
		}
	}

	t0 := time.Now()
	initTime := t0.Sub(r.Arrival).Seconds()

	for i := 0; i < int(r.Istance_number); i++ {
		response, invocationWait, err = container.Execute(contID, &req)
		if err != nil {
			// notify scheduler
			completions <- &completionNotification{fun: r.Fun, contID: contID, executionReport: nil}
			return function.ExecutionReport{}, fmt.Errorf("[%s] Execution failed: %v", r, err)
		}

		if !response.Success {
			// notify scheduler
			completions <- &completionNotification{fun: r.Fun, contID: contID, executionReport: nil}
			return function.ExecutionReport{}, fmt.Errorf("Function execution failed")
		}

		Results = append(Results, response.Result)
		Outputs = append(Outputs, response.Output)

	}

	report := function.ExecutionReport{Result: Results,
		Output:       Outputs,
		IsWarmStart:  isWarm,
		Duration:     time.Now().Sub(t0).Seconds() - invocationWait.Seconds(),
		ResponseTime: time.Now().Sub(t0).Seconds() - invocationWait.Seconds()}

	// initializing containers may require invocation retries, adding
	// latency
	report.InitTime = initTime + invocationWait.Seconds()

	// notify scheduler
	completions <- &completionNotification{fun: r.Fun, contID: contID, executionReport: &report}

	return report, nil
}*/
