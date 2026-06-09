# Distributed Real-Time Ride-Sharing Backend Platform

A high-throughput, fault-tolerant distributed system mimicking a production-scale ride-sharing backend (like Uber). Built as a containerized mesh of microservices using Go, RabbitMQ, WebSockets, and MongoDB, fully orchestrated via Kubernetes and Tilt.

## 🏗️ System Architecture

The platform is designed around decoupled microservices interacting through synchronous **gRPC** calls for internal operations and asynchronous **RabbitMQ topic exchanges** for event-driven workflows. Persistent communication with clients is sustained using stateful **WebSockets**.

(https://private-user-images.githubusercontent.com/152910199/604974989-25317c94-30ca-4b3f-be29-0be4bfcb644a.jpeg?jwt=eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJnaXRodWIuY29tIiwiYXVkIjoicmF3LmdpdGh1YnVzZXJjb250ZW50LmNvbSIsImtleSI6ImtleTUiLCJleHAiOjE3ODA5OTU0MzEsIm5iZiI6MTc4MDk5NTEzMSwicGF0aCI6Ii8xNTI5MTAxOTkvNjA0OTc0OTg5LTI1MzE3Yzk0LTMwY2EtNGIzZi1iZTI5LTBiZTRiZmNiNjQ0YS5qcGVnP1gtQW16LUFsZ29yaXRobT1BV1M0LUhNQUMtU0hBMjU2JlgtQW16LUNyZWRlbnRpYWw9QUtJQVZDT0RZTFNBNTNQUUs0WkElMkYyMDI2MDYwOSUyRnVzLWVhc3QtMSUyRnMzJTJGYXdzNF9yZXF1ZXN0JlgtQW16LURhdGU9MjAyNjA2MDlUMDg1MjExWiZYLUFtei1FeHBpcmVzPTMwMCZYLUFtei1TaWduYXR1cmU9MzgwYmNjYmRhMzVjMmMwYTBjYzNjNTZlNjkzYTAyZWE2YThiZDNiNDA5Yzg5ZGZlM2UyNWE2N2FlMTcwODUzNCZYLUFtei1TaWduZWRIZWFkZXJzPWhvc3QmcmVzcG9uc2UtY29udGVudC10eXBlPWltYWdlJTJGanBlZyJ9.LVX2ClmkmdAHDIEX6G0kIHvpw-E3avosI0kN_kCVvwQ)

### 🎬 System Interaction Lifecycle

1. **Rider Requests Ride:** Handled by the **API Gateway** over WebSockets, which forwards the payload to the **Trip Service**. The Trip Service generates spatial routes via the **OSRM API** and saves an upfront pricing log schema to **MongoDB**.
2. **Match Event Dispatched:** The Trip Service publishes a `trip.event.created` notification to **RabbitMQ**.
3. **Driver Allocation:** The **Driver Service** consumes the event, filters nearby available operators matching the requested vehicle type (e.g., SUV, Sedan), and streams a notification payload back down the assigned driver's live WebSocket thread via the API Gateway.
4. **Resilient Settlement:** Once accepted, the **Payment Service** handles transaction requests with **Stripe**, backed by custom error-handling middleware.

---

## 🛠️ Tech Stack & Infrastructure

- **Language:** Golang (Core Engine)
- **Message Broker:** RabbitMQ (AMQP 0-9-1)
- **Databases:** MongoDB (Core persistence layer)
- **Communication Protocols:** gRPC (Internal IPC), WebSockets (Real-time client duplexes), REST/HTTP
- **Observability:** OpenTelemetry Core, Jaeger Distributed Tracing Matrix
- **Orchestration & Tooling:** Kubernetes (K8s), Docker, Tilt Engine

---

# Distributed Real-Time Ride-Sharing Backend Platform

A high-performance backend platform built to handle the core operational lifecycle of an Uber-style ride-sharing system. This project focuses on solving complex real-time marketplace challenges—such as instant driver matching, live location distribution, and resilient payment settlement—using a decoupled microservices architecture.

## 🚀 Core Features Built & Fully Operational

### 1. Real-Time Event Distribution (WebSockets)
* **Persistent Handset Pipelines:** Built a stateful connection manager in the API Gateway that holds open permanent WebSockets to riders and drivers.
* **Instant Client Dispatches:** Allows the backend to instantly push ride invitations and location updates down to a user's device without waiting for a manual page refresh.

### 2. Automated Route & Upfront Price Generation (OSRM API)
* **Spatial Calculations:** Integrates directly with the Open Source Routing Machine (OSRM) API to calculate precise driving distances and durations between pickup and destination coordinates.
* **Dynamic Pricing Engine:** Implements a pricing algorithm that uses the route metadata to generate upfront fare quotes simultaneously for multiple vehicle tiers (`SUV`, `Sedan`, `Van`, `Luxury`) in cents to prevent floating-point math errors.

### 3. Asynchronous Workflow Orchestration (RabbitMQ)
* **Decoupled Chain Reactions:** Uses RabbitMQ topic exchanges so services can pass work to each other smoothly. For example, when a trip is requested, the Trip Service drops a message in the broker, allowing the Driver Service to pick it up and process driver matchmaking completely in the background.
* **Fair Dispatching Throttling:** Configured explicit prefetch limits (`Qos(1)`) on message consumers to ensure a single microservice container handles exactly one trip event at a time, keeping system memory completely stable.

### 4. Smart Integration Resilience (Exponential Backoff)
* **Fault-Tolerant Middleware:** Engineered a custom, abstract retry engine wrapped around third-party network channels (Stripe API). 
* **Intelligent Backoff Scaling:** If Stripe is temporarily down or slow, the system automatically pauses, doubles its wait timer, and retries the operation up to 3 times before failing, hiding minor internet hiccups from the end user.
* **Context-Aware Aborts:** Integrated native Go `context.Context` channels so that if a rider cancels their request mid-retry, the backend instantly stops wasting execution resources.

---

## 💻 Local Development Setup

The repository utilizes **Tilt** to automate compilation, Docker multi-stage virtualization, and deployment updates directly into a local Kubernetes namespace loop.

### Prerequisites

- Docker Desktop & Kubernetes activated (or Minikube)
- Go 1.23+
- Tilt CLI (`brew install tilt` or equivalent)

### Run the Cluster Locally

1. **Clone the Repository:**
   ```bash
   git clone [https://github.com/your-username/ride-sharing-backend.git](https://github.com/your-username/ride-sharing-backend.git)
   cd ride-sharing-backend
   Boot up the Dev Environment:
   ```

Bash
tilt up
This single command triggers the Tiltfile blueprint to compile your Go binaries, build local container layers, mount configuration volumes, and launch the real-time development UI.

Verify running status:

Bash
kubectl get pods -n default
Local Admin Dashboards
Tilt Web UI: http://localhost:10350 (Live code reload and compilation logger)

RabbitMQ Dashboard: http://localhost:15672 (Username: guest / Password: guest)

Jaeger Tracing View: http://localhost:16686 (Analyze end-to-end trace flows)
