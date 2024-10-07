# Thought Process

## Overview

The goal is to create a high-performance REST service in Go that can handle at least 10K requests per second. The service has a single GET endpoint that processes requests and logs unique request counts every minute. Additionally, it can send HTTP requests to a provided endpoint with the count of unique requests.

## Design Considerations

1. **Concurrency and Performance:**
   - Use a map with a mutex for thread-safe operations.
   - Use goroutines for non-blocking operations (e.g., sending HTTP requests).

2. **Logging:**
   - Use Go's standard `log` package for simplicity and performance.

3. **HTTP Requests:**
   - Use Go's `net/http` package to handle incoming and outgoing HTTP requests.

4. **Extensions:**
   - **Extension 1:** Change the HTTP method to POST and include a JSON payload.
   - **Extension 2:** Use Redis for deduplication across instances.
   - **Extension 3:** Integrate with a distributed streaming service (e.g., Kafka).

## Implementation Steps

1. **Setup HTTP Server:**
   - Create a handler for the `/api/verve/accept` endpoint.
   - Parse query parameters (`id` and optional `endpoint`).
   - Log the request and update the unique ID map.
   - Respond with "ok" or "failed".

2. **Logging and HTTP Requests:**
   - Every minute, log the count of unique requests.
   - If an endpoint is provided, send an HTTP POST request with the count as JSON.
   - Log the HTTP status code of the response.

3. **Concurrency and Load Balancer Handling:**
   - Use Redis for deduplication across instances.
