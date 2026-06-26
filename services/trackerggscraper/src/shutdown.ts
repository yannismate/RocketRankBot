let shutdownFn: ((exitCode: number) => void) | undefined;

export function registerShutdown(fn: typeof shutdownFn) {
    shutdownFn = fn;
}

export function shutdown(exitCode: number) {
    if (!shutdownFn) {
        throw new Error("Shutdown not initialized");
    }
    shutdownFn(exitCode);
}