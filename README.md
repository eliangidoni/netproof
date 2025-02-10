# netproof

Common network patterns used in microservices (concurrency safe).
  * Bulkhead
  * Circuit Breaker
  * Retry
  * Throttling
  * Rate limiting
  * Priority Queue
  * Request Hedging

To run tests:
```shell
go test ./... -v
go test -benchmem -bench=. -count=2 ./... -v
```
