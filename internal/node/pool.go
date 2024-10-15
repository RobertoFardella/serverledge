package node

import (
	"container/list"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/function"
)

type ContainerPool struct {
	running *list.List // list of ContainerRunning
	warm    *list.List // list of warmContainer
}

type warmContainer struct {
	Expiration int64
	contID     container.ContainerID
}

type containerRunning struct {
	FuncCounter int64
	contID      container.ContainerID
}

var NoWarmFoundErr = errors.New("no warm container is available")
var NoRunningContErr = errors.New("no running container is available")

// getFunctionPool retrieves (or creates) the container pool for a function.
func getFunctionPool(f *function.Function) *ContainerPool {

	if fp, ok := Resources.ContainerPools[f.Name]; ok {
		return fp
	}

	fp := newFunctionPool(f)
	Resources.ContainerPools[f.Name] = fp
	return fp
}

func (fp *ContainerPool) getRunningContainer() (container.ContainerID, bool) {

	elem := fp.running.Front()
	if elem == nil {
		return "", false
	}

	for elem != nil {
		containerElem := elem.Value.(*containerRunning)

		if containerElem.FuncCounter < MaxFunctionInstances {
			containerElem.FuncCounter++
			log.Printf("Container %s has been reused, function instances: %d.\n", containerElem.contID, containerElem.FuncCounter)
			return containerElem.contID, true
		} else {
			log.Printf("Container %s is fully utilized, function instances limit exceeded.\n", containerElem.contID)
		}
		elem = elem.Next()
	}

	return "", false
}

func (fp *ContainerPool) putRunningContainer(contID container.ContainerID) {
	fp.running.PushBack(&containerRunning{
		contID:      contID,
		FuncCounter: 1,
	})
}

func (fp *ContainerPool) putwarmContainer(contID container.ContainerID, expiration int64) {
	fp.warm.PushBack(&warmContainer{
		contID:     contID,
		Expiration: expiration,
	})
}

func newFunctionPool(_ *function.Function) *ContainerPool {
	fp := &ContainerPool{}
	fp.running = list.New()
	fp.warm = list.New()

	return fp
}

// AcquireResources reserves the specified amount of cpu and memory if possible.
func AcquireResources(cpuDemand float64, memDemand int64, destroyContainersIfNeeded bool) bool {
	Resources.Lock()
	defer Resources.Unlock()
	return acquireResources(cpuDemand, memDemand, destroyContainersIfNeeded)
}

// acquireResources reserves the specified amount of cpu and memory if possible.
// The function is NOT thread-safe.
func acquireResources(cpuDemand float64, memDemand int64, destroyContainersIfNeeded bool) bool {

	if Resources.AvailableCPUs < cpuDemand {
		return false
	}
	if Resources.AvailableMemMB < memDemand {
		if !destroyContainersIfNeeded {
			return false
		}

		enoughMem, _ := dismissContainer(memDemand)
		if !enoughMem {
			return false
		}
	}

	Resources.AvailableCPUs -= cpuDemand
	Resources.AvailableMemMB -= memDemand

	return true
}

// releaseResources releases the specified amount of cpu and memory.
// The function is NOT thread-safe.
func releaseResources(cpuDemand float64, memDemand int64) {
	Resources.AvailableCPUs += cpuDemand
	Resources.AvailableMemMB += memDemand
}

// The acquired container is alwarm in the running pool.
// The function returns an error if either:
// (i) the container does not exist
// (ii) there are not enough resources to use the container busy with some function
func AcquireRunningContainer(f *function.Function) (container.ContainerID, error) {
	Resources.Lock()
	defer Resources.Unlock()
	//seleziono un container runnning in cui è possibile invocare un funzione concorrente
	fp := getFunctionPool(f)
	contID, found := fp.getRunningContainer()
	if !found {
		log.Printf("no running container is available for %s", f)
		return "", NoRunningContErr
	}
	//controllo se la cpu del nodo è sufficiente
	if !acquireResources(f.CPUDemand, 0, false) {
		log.Printf("Not enough CPU to start a container for %s", f)
		return "", OutOfResourcesErr
	}

	//log.Printf("Using %s for %s. Now: %v", contID, f, Resources)
	return contID, nil
}

// ReleaseResources puts a container in the warm pool for a function if the counter of istance is zero.
func ReleaseResources(containerID container.ContainerID, f *function.Function) {
	// setup Expiration as time duration from now
	d := time.Duration(config.GetInt(config.CONTAINER_EXPIRATION_TIME, 600)) * time.Second
	expTime := time.Now().Add(d).UnixNano()

	Resources.Lock()
	defer Resources.Unlock()

	fp := getFunctionPool(f)

	/*
		fp.lock.RLock()
		defer fp.lock.RUnlock()*/

	// we must update the runningContainer list by decrementing the counter of istances or removing this element in case counter_istances = 0
	var removed *containerRunning
	elem := fp.running.Front()
	for ok := elem != nil; ok; ok = elem != nil {
		container := elem.Value.(*containerRunning)
		if container.contID == containerID {
			container.FuncCounter--
			releaseResources(f.CPUDemand, 0)
			//log.Printf("the container %s has %d now", container.contID, container.FuncCounter)
			//log.Printf("Released resources. Now: %v", Resources)

			if container.FuncCounter == 0 {
				removed = fp.running.Remove(elem).(*containerRunning)
				fp.putwarmContainer(removed.contID, expTime)
				return

			}

		}
		elem = elem.Next()

		/*if removed.contID != containerID && removed.FuncCounter == 0 {
			panic("Failed to release container")
		}*/
	}
}

// NewContainer creates and starts a new container for the given function.
// The container can be directly used to schedule a request.
func NewContainer(fun *function.Function) (container.ContainerID, error) {
	Resources.Lock()
	if !acquireResources(fun.CPUDemand, fun.MemoryMB, true) {
		log.Printf("Not enough resources for the new container.")
		Resources.Unlock()
		return "", OutOfResourcesErr
	}

	//log.Printf("Acquired resources for new container. Now: %v", Resources)
	Resources.Unlock()

	return NewContainerWithAcquiredResources(fun)
}

func getImageForFunction(fun *function.Function) (string, error) {
	var image string
	if fun.Runtime == container.CUSTOM_RUNTIME {
		image = fun.CustomImage
	} else {
		runtime, ok := container.RuntimeToInfo[fun.Runtime]
		if !ok {
			log.Printf("Unknown runtime: %s\n", fun.Runtime)
			return "", fmt.Errorf("invalid runtime: %s", fun.Runtime)
		}
		image = runtime.Image
	}
	return image, nil
}

func (fp *ContainerPool) getWarmContainer() (container.ContainerID, bool) {
	// TODO: picking most-recent / least-recent container might be better?
	elem := fp.warm.Front()
	if elem == nil {
		return "", false
	}

	wc := fp.warm.Remove(elem).(*warmContainer)
	fp.putRunningContainer(wc.contID)

	return wc.contID, true
}

// AcquireWarmContainer acquires a warm container for a given function (if any).
// A warm container is in running/paused state and has already been initialized
// with the function code.
// The acquired container is already in the busy pool.
// The function returns an error if either:
// (i) the warm container does not exist
// (ii) there are not enough resources to start the container
func AcquireWarmContainer(f *function.Function) (container.ContainerID, error) {
	Resources.Lock()
	defer Resources.Unlock()

	fp := getFunctionPool(f)
	contID, found := fp.getWarmContainer()
	if !found {
		return "", NoWarmFoundErr
	}

	if !acquireResources(f.CPUDemand, 0, false) {
		log.Printf("Not enough CPU to start a warm container for %s", f)
		return "", OutOfResourcesErr
	}

	//log.Printf("Using warm %s for %s. Now: %v", contID, f, Resources)
	return contID, nil
}

/* A warm container is acquired assuming that the resources have already been obtained. */
func WarmContainerWithAcquiredResources(f *function.Function) (container.ContainerID, error) {
	fp := getFunctionPool(f)

	contID, found := fp.getWarmContainer()
	if !found {
		return "", NoWarmFoundErr
	}

	//log.Printf("Using warm %s for %s. Now: %v", contID, f, Resources)
	return contID, nil
}

// NewContainerWithAcquiredResources spawns a new container for the given
// function, assuming that the required CPU and memory resources have been
// alwarm been acquired.
func NewContainerWithAcquiredResources(fun *function.Function) (container.ContainerID, error) {
	image, err := getImageForFunction(fun)
	if err != nil {
		return "", err
	}

	contID, err := container.NewContainer(image, fun.TarFunctionCode, &container.ContainerOptions{
		MemoryMB: fun.MemoryMB,
		CPUQuota: fun.CPUDemand,
	})

	if err != nil {
		log.Printf("Failed container creation: %v\n", err)
	}

	Resources.Lock()
	defer Resources.Unlock()
	if err != nil {
		releaseResources(fun.CPUDemand, fun.MemoryMB)
		return "", err
	}

	fp := getFunctionPool(fun)
	fp.putRunningContainer(contID)

	return contID, nil
}

type itemToDismiss struct {
	contID container.ContainerID
	pool   *ContainerPool
	elem   *list.Element
	memory int64
}

// dismissContainer ... this function is used to get free memory used for a new container
// 2-phases: first, we find warm container and collect them as a slice, second (cleanup phase) we delete the container only and only if
// the sum of their memory is >= requiredMemoryMB is
func dismissContainer(requiredMemoryMB int64) (bool, error) {
	var cleanedMB int64 = 0
	var containerToDismiss []itemToDismiss
	res := false

	//first phase, research
	for _, funPool := range Resources.ContainerPools {
		if funPool.warm.Len() > 0 {
			// every container into the funPool has the same memory (same function)
			//so it is not important which one you destroy
			elem := funPool.warm.Front()
			contID := elem.Value.(*warmContainer).contID
			// container in the same pool need same memory
			memory, _ := container.GetMemoryMB(contID)
			for ok := true; ok; ok = elem != nil {
				containerToDismiss = append(containerToDismiss,
					itemToDismiss{contID: contID, pool: funPool, elem: elem, memory: memory})
				cleanedMB += memory
				if cleanedMB >= requiredMemoryMB {
					goto cleanup
				}
				//go on to the next one
				elem = elem.Next()
			}
		}
	}

cleanup: // second phase, cleanup
	// memory check
	if cleanedMB >= requiredMemoryMB {
		for _, item := range containerToDismiss {
			item.pool.warm.Remove(item.elem)      // remove the container from the funPool
			err := container.Destroy(item.contID) // destroy the container
			if err != nil {
				res = false
				return res, nil
			}
			Resources.AvailableMemMB += item.memory
		}

		res = true
	}
	return res, nil
}

// DeleteExpiredContainer is called by the container cleaner
// Deletes expired warm container
func DeleteExpiredContainer() {
	now := time.Now().UnixNano()

	Resources.Lock()
	defer Resources.Unlock()

	for _, pool := range Resources.ContainerPools {

		/*
			fp.lock.RLock()
			defer fp.lock.RUnlock()*/
		elem := pool.warm.Front()

		for ok := elem != nil; ok; ok = elem != nil {
			warmed := elem.Value.(*warmContainer)
			if now > warmed.Expiration {
				temp := elem
				elem = elem.Next()
				log.Printf("cleaner: Removing container %s\n", warmed.contID)
				pool.warm.Remove(temp) // remove the expired element

				memory, _ := container.GetMemoryMB(warmed.contID)
				releaseResources(0, memory)
				err := container.Destroy(warmed.contID)
				if err != nil {
					log.Printf("Error while destroying container %s: %s\n", warmed.contID, err)
				}

			} else {
				elem = elem.Next()
			}
		}
	}

}

// ShutdownWarmContainersFor destroys warm containers of a given function
// Actual termination happens asynchronously.
func ShutdownWarmContainersFor(f *function.Function) {
	Resources.Lock()
	defer Resources.Unlock()

	fp, ok := Resources.ContainerPools[f.Name]
	if !ok {
		return
	}

	containersToDelete := make([]container.ContainerID, 0)

	elem := fp.warm.Front()
	for ok := elem != nil; ok; ok = elem != nil {
		warmed := elem.Value.(*warmContainer)
		temp := elem
		elem = elem.Next()
		log.Printf("Removing container with ID %s\n", warmed.contID)
		fp.warm.Remove(temp)

		memory, _ := container.GetMemoryMB(warmed.contID)
		Resources.AvailableMemMB += memory
		containersToDelete = append(containersToDelete, warmed.contID)
	}

	go func(contIDs []container.ContainerID) {
		for _, contID := range contIDs {
			// No need to update available resources here
			if err := container.Destroy(contID); err != nil {
				log.Printf("An error occurred while deleting %s: %v\n", contID, err)
			} else {
				log.Printf("Deleted %s\n", contID)
			}
		}
	}(containersToDelete)
}

// ShutdownAllContainers destroys all container (usually on termination)
func ShutdownAllContainers() {
	Resources.Lock()
	defer Resources.Unlock()

	for fun, pool := range Resources.ContainerPools {
		elem := pool.warm.Front()
		for ok := elem != nil; ok; ok = elem != nil {
			warmed := elem.Value.(*warmContainer)
			temp := elem
			elem = elem.Next()
			log.Printf("Removing container with ID %s\n", warmed.contID)
			pool.warm.Remove(temp)

			memory, _ := container.GetMemoryMB(warmed.contID)
			err := container.Destroy(warmed.contID)
			if err != nil {
				log.Printf("Error while destroying container %s: %s", warmed.contID, err)
			}
			Resources.AvailableMemMB += memory
		}

		functionDescriptor, _ := function.GetFunction(fun)

		elem = pool.running.Front()
		for ok := elem != nil; ok; ok = elem != nil {
			runningCont := elem.Value.(*containerRunning)
			contID := runningCont.contID
			temp := elem
			elem = elem.Next()
			log.Printf("Removing container with ID %s\n", contID)
			pool.warm.Remove(temp)

			memory, _ := container.GetMemoryMB(contID)
			err := container.Destroy(contID)
			if err != nil {
				log.Printf("Error while destroying container %s: %s", contID, err)
			}
			Resources.AvailableMemMB += memory
			Resources.AvailableCPUs += functionDescriptor.CPUDemand
		}
	}
}

// WarmStatus foreach function returns the corresponding number of warm container available
func WarmStatus() map[string]int {
	Resources.RLock()
	defer Resources.RUnlock()
	warmPool := make(map[string]int)
	for funcName, pool := range Resources.ContainerPools {
		warmPool[funcName] = pool.warm.Len()
	}

	return warmPool
}

func PrewarmInstances(f *function.Function, count int64, forcePull bool) (int64, error) {
	image, err := getImageForFunction(f)
	if err != nil {
		return 0, err
	}
	err = container.DownloadImage(image, forcePull)
	if err != nil {
		return 0, err
	}

	var spawned int64 = 0
	for spawned < count {
		_, err = NewContainer(f)
		if err != nil {
			log.Printf("Prespawning failed: %v\n", err)
			return spawned, err
		}
		spawned += 1
	}

	return spawned, nil
}
