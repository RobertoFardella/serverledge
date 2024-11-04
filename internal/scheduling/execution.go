package scheduling

import (
	"fmt"
	"log"
	"time"

	"github.com/grussorusso/serverledge/internal/function"

	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/executor"
)

const HANDLER_DIR = "/app"

// Execute serves a request with multiple instances of the function on the specified container.
func Execute(contID container.ContainerID, r *scheduledRequest, isWarm bool) (function.ExecutionReport, error) {
	var req executor.InvocationRequest
	var results, outputs []string

	var err error
	var totalDuration float64
	var invocationWaits []time.Duration
	var responseTimes []float64
	errChan := make(chan error, r.Istance_number) // Channel for errors from executions

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

	// Single execution for each instance
	for i := 0; i < int(r.Istance_number); i++ {
		rtime := time.Now()
		response, invocationWait, err := container.Execute(contID, &req)

		responseTime := time.Since(rtime).Seconds() - invocationWait.Seconds()
		//duration := time.Since(t0).Seconds() - invocationWait.Seconds()

		if err != nil {
			errChan <- fmt.Errorf("[%s] Execution failed: %v", r, err)
			break
		}

		if !response.Success {
			errChan <- fmt.Errorf("Function execution failed")
			break
		}

		results = append(results, response.Result)
		outputs = append(outputs, response.Output)
		responseTimes = append(responseTimes, responseTime)

		invocationWaits = append(invocationWaits, invocationWait)
	}

	// Handle errors if present
	close(errChan)
	if len(errChan) > 0 {
		err = <-errChan
		// notify scheduler
		completions <- &completionNotification{fun: r.Fun, contID: contID, executionReport: nil}
		return function.ExecutionReport{}, err
	}

	for i := 0; i < int(r.Istance_number); i++ {
		totalDuration += responseTimes[i]
	}

	report := function.ExecutionReport{
		Result:       results,
		Output:       outputs,
		IsWarmStart:  isWarm,
		Duration:     totalDuration + initTime + invocationWaits[0].Seconds(), // Total response time of the entire request
		ResponseTime: responseTimes,                                           // Response time of individual requests
		InitTime:     initTime + invocationWaits[0].Seconds(),                 // Consider only the wait time of the first instance of the request as this is where the cold start time is taken into account
	}
	// Notify the handler of completion
	completions <- &completionNotification{fun: r.Fun, contID: contID, executionReport: &report}
	return report, nil
}

/*
	// Lancia esecuzioni parallele
	for i := 0; i < int(r.Istance_number); i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			response, invocationWait, err := container.Execute(contID, &req)

			responseTime := time.Now().Sub(t0).Seconds() - invocationWait.Seconds()
			duration := time.Now().Sub(t0).Seconds() - invocationWait.Seconds()

			if err != nil {
				errChan <- fmt.Errorf("[%s] Execution failed: %v", r, err)
				return
			}

			if !response.Success {
				errChan <- fmt.Errorf("Function execution failed")
				return
			}

			mutex.Lock()
			defer mutex.Unlock()
			results = append(results, response.Result)
			outputs = append(outputs, response.Output)
			responseTimes = append(responseTimes, responseTime)
			invocationWaits = append(invocationWaits, invocationWait)
			durations = append(durations, duration)
		}(i)
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

	for i := 0; i < int(r.Istance_number); i++ {
		totalDuration += durations[i]
	}

	report := function.ExecutionReport{
		Result:       results,
		Output:       outputs,
		IsWarmStart:  isWarm,
		Duration:     totalDuration,
		ResponseTime: responseTimes,
		InitTime:     initTime + invocationWaits[0].Seconds(), //// initializing containers may require invocation retries, adding latency
	}

	// notifica il completamento al gestore
	completions <- &completionNotification{fun: r.Fun, contID: contID, executionReport: &report}

	return report, nil
*/
