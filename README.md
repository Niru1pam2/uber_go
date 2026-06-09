# Distributed Real-Time Ride-Sharing Backend Platform

A high-throughput, fault-tolerant distributed system mimicking a production-scale ride-sharing backend (like Uber). Built as a containerized mesh of microservices using Go, RabbitMQ, WebSockets, and MongoDB, fully orchestrated via Kubernetes and Tilt.

## 🏗️ System Architecture

The platform is designed around decoupled microservices interacting through synchronous **gRPC** calls for internal operations and asynchronous **RabbitMQ topic exchanges** for event-driven workflows. Persistent communication with clients is sustained using stateful **WebSockets**.

                       +─────────────────+
                       |   Web Frontend  |
                       +────────┬────────+
                                |
                                | WebSockets / HTTP
                                ▼
                       +─────────────────+
                       |   API Gateway   |
                       +───┬─────────┬───+
                           |         |
                  gRPC IPC |         | AMQP Events
                           ▼         ▼

+──────────────────────────+ +─────────────────────────+
| Driver Service | | RabbitMQ Broker |
+──────────────────────────+ +────────────┬────────────+
|
+────────────────┴────────────────+
▼ ▼
+─────────────────────────+ +─────────────────────────+
| Trip Service | | Payment Service |
+────────────┬────────────+ +────────────┬────────────+
| |
▼ ▼
+─────────────────────────+ +─────────────────────────+
| MongoDB | | External Stripe API |
+─────────────────────────+ +─────────────────────────+

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

## ⚡ Key Engineering Implementation Details

### 1. Production-Grade Event Processing & Error Handling

- **Quality of Service (QoS):** Configured fair prefetch limits (`Qos(1)`) on RabbitMQ channels to prevent memory congestion and ensure deterministic resource balance across replica pods.
- **Dead Letter Exchanges (DLX):** Implemented a programmatic quarantine strategy. Failed queue processing iterations automatically append custom metadata strings directly into the `amqp.Table` headers (acting as an audit death certificate) before calling explicit un-requeued rejections (`d.Reject(false)`).

### 2. Resilient Integration Middleware

- **Exponential Backoff:** Engineered abstract retry wrappers incorporating custom duration scaling and safe execution ceilings (`MaxWait`).
- **Context Preservation:** Integrates native Go `context.Context` channels into retry execution frameworks, ensuring background request attempts abort immediately if a client safely disconnects or cancels the original intent.

### 3. Distributed Observability Matrix

- **Context Propagation:** Configured `TextMapPropagator` wrappers to transparently inject and extract transactional headers across physical network boundaries (RabbitMQ message headers and gRPC metadata payloads).
- **High-Precision Span Nesting:** Leverages automated Interceptors (`otelgrpc`) to build structural hierarchical trace trees inside **Jaeger**, isolating upstream gRPC server lifecycles from deep database bottlenecks down to the nanosecond.

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
