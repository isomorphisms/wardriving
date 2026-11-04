/* Host C ABI observer; no Android execution is claimed by this fixture. */
#include <dlfcn.h>
#include <stdint.h>
#include <stdio.h>

int main(int argc, char **argv) {
    if (argc != 2) return 2;
    void *library = dlopen(argv[1], RTLD_NOW | RTLD_LOCAL);
    if (!library) { fprintf(stderr, "%s\n", dlerror()); return 1; }
    uint32_t (*version)(void) = (uint32_t (*)(void))dlsym(library, "WD_ABIVersion");
    int32_t (*observe)(uint64_t,uint64_t,uint32_t) =
        (int32_t (*)(uint64_t,uint64_t,uint32_t))dlsym(library,"WD_ObserveNetwork");
    uint64_t (*generation)(void) = (uint64_t (*)(void))dlsym(library,"WD_CurrentGeneration");
    uint64_t (*handle)(void) = (uint64_t (*)(void))dlsym(library,"WD_CurrentHandle");
    if (!version || !observe || !generation || !handle) return 1;
    if (version()!=1 || observe(1,42,1)!=0 || generation()!=1 || handle()!=42) return 1;
    if (observe(2,43,1)!=0 || observe(1,42,1)!=-1 || generation()!=2 || handle()!=43) return 1;
    if (observe(3,0,0)!=-2 || generation()!=3 || handle()!=0) return 1;
    puts("PASS host C-to-Go ABI: current/loss/stale generation");
    /* Go runtime shared-library unloading is outside the contract. */
    return 0;
}
