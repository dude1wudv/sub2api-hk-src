/** Keep asynchronous validation and the following request in one submission. */
export function singleFlight<T extends unknown[], R>(action: (...args: T) => Promise<R>) {
  let pending = false
  return async (...args: T): Promise<R | undefined> => {
    if (pending) return
    pending = true
    try { return await action(...args) }
    finally { pending = false }
  }
}
