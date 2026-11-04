// This library proves only the C-to-Go event ABI. It performs no networking.
package main

/*
#include <stdint.h>
*/
import "C"

import "sync"

var slot struct {
	sync.Mutex
	generation uint64
	handle     uint64
}

//export WD_ABIVersion
func WD_ABIVersion() C.uint32_t { return 1 }

//export WD_ObserveNetwork
func WD_ObserveNetwork(generation C.uint64_t, handle C.uint64_t, wifi C.uint32_t) C.int32_t {
	slot.Lock()
	defer slot.Unlock()
	if uint64(generation) <= slot.generation {
		return -1
	}
	slot.generation = uint64(generation)
	slot.handle = 0
	if wifi != 1 || handle == 0 {
		return -2
	}
	slot.handle = uint64(handle)
	return 0
}

//export WD_CurrentGeneration
func WD_CurrentGeneration() C.uint64_t {
	slot.Lock()
	defer slot.Unlock()
	return C.uint64_t(slot.generation)
}

//export WD_CurrentHandle
func WD_CurrentHandle() C.uint64_t {
	slot.Lock()
	defer slot.Unlock()
	return C.uint64_t(slot.handle)
}

func main() {}
