Problem: Bounded Concurrent Job Processor
Write a function runBatchProcessor(jobs []Job) error

that processes a list of jobs concurrently.The function must adhere to the following four requirements:

Concurrency Limit: You must run at most $K$ jobs (e.g., 8) in parallel at any given time.

Overall Timeout: The entire batch processing operation must not take longer than a specified total timeout (e.g., 5 seconds).
If it exceeds this, the operation must fail.

Early Exit on Error: If any job returns an error, the system must immediately stop processing new jobs and signal all currently running jobs to cancel.

The function must return the first error that occurred.Clean Shutdown: All goroutines (workers and producers) must shut down cleanly and exit,
ensuring no goroutine leaks, regardless of whether the operation succeeds, fails due to an error, or fails due to a timeout.

What is the best use case for this
how is rollback handled here, wt happens if some of the goroutines have commited the changes
already then cancelling the remaining goroutines makes data inconsistent right

what happens to the currently running threads - will it be stopped in between or will it let 
the current goroutines to complete and then stop assigning remaining jobs to the new goroutines