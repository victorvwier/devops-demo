Here’s a proper spec, tuned for **small, cheap, explainable, and demoable**.

# Demo spec — GitOps + custom operator + observability on a tiny cluster

## 1. Goal

Build a small end-to-end platform demo that shows:

* infra created from code
* Kubernetes bootstrapped onto fresh machines
* ArgoCD owning cluster add-ons and app delivery
* a **custom operator** managing a tiny AI-ish workload
* observability layered in incrementally, with room to explore metrics, logs, and traces
* one coherent story, not a pile of tools ([Argo CD][1])

## 2. Core idea

The demo should feel like this:

1. Terraform creates 1 or 2 cheap Linux VMs
2. k3s boots on them
3. ArgoCD gets installed once, then becomes the delivery mechanism
4. ArgoCD syncs:

   * namespaces
   * operator
   * demo app
   * observability stack
5. the custom operator reconciles a CRD into a runnable tiny LLM service
6. you hit the service
7. Grafana shows what happened
8. you change the CR or Git, and the system updates live ([Argo CD][1])

## 3. Design choices

### Cloud

Use **whatever gives free trial credits or an effectively free tiny VM fastest**. Don’t optimize for purity here. This spec stays provider-neutral on purpose. The infra only needs:

* 1 public control-plane VM
* optional 1 worker VM
* SSH access
* public IP
* basic firewall rules

### Cluster

Default to **1-node k3s** for the first version. k3s is intentionally lightweight and designed to reduce control-plane complexity, which makes it a strong fit for a fast demo. A 2-node version is a stretch goal, mainly for “that’s cooler” points, not because the demo needs it. K3s’ own requirements docs note the baseline requirements are only for Kubernetes itself, not your workloads, which matters here because observability plus an AI-ish app will dominate resource usage. ([K3s][2])

### GitOps scope

ArgoCD should manage **almost everything inside the cluster**:

* operator
* observability
* demo namespaces
* sample workload
* optional ingress bits

Terraform should **not** manage in-cluster workloads. Terraform stops at infra and bootstrap. Argo CD is explicitly built around declarative, version-controlled application definitions and automated lifecycle management, so this split keeps the demo story clean. ([Argo CD][1])

### Operator story

Use a **custom operator only**, built with Kubebuilder. Kubebuilder’s official getting-started flow is specifically aimed at scaffolding APIs/controllers and reconciling CRs into standard Kubernetes resources, which is exactly the story you want to tell. ([book.kubebuilder.io][3])

### App concept

Use a **tiny LLM runner façade**, not a “real” heavyweight AI workload. The app should feel AI-related, but it must fit on tiny infra.

Best approach:

* very small HTTP service
* accepts prompt input
* optionally runs a tiny local model only on demand, or
* more realistically, simulates “LLM runner orchestration” with a pluggable backend
* includes endpoints that are easy to observe

This keeps the AI angle without letting compute cost hijack the demo.

## 4. Recommended final architecture

### Infra layer

Managed by Terraform:

* network / firewall
* 1–2 Linux VMs
* SSH key injection
* public IP
* optional DNS record

### Cluster layer

Managed by bootstrap script or cloud-init:

* k3s server on node 1
* optional k3s agent on node 2
* default local-path storage is good enough
* keep ingress simple; k3s defaults are fine initially ([K3s][2])

### GitOps layer

Managed by ArgoCD:

* `infra-bootstrap` app
* `operator` app
* `observability` app
* `demo-app` app
* optional app-of-apps root application ([Argo CD][1])

### Operator layer

Custom operator built with Kubebuilder:

* CRD: `TinyLLMService`
* reconciles:

  * Deployment
  * Service
  * ConfigMap
  * optional Ingress
* updates status with:

  * phase
  * ready replicas
  * backend mode
  * last reconcile time ([book.kubebuilder.io][3])

### Observability layer

Open-ended by design, but start with:

* Grafana
* Prometheus
* Beyla

Then optionally add:

* Loki
* Tempo

Beyla is positioned by Grafana as eBPF-based auto-instrumentation that can export RED metrics and traces for Linux HTTP/S and gRPC services with minimal or no code changes, which makes it ideal for the “we want observability fast” demo angle. ([Grafana Labs][4])

## 5. Why this is the right scope

This scope stays small enough to finish, but still demonstrates:

* infra as code
* cluster bootstrap
* GitOps
* operator pattern
* observability
* AI-ish workload packaging

That’s enough to map directly onto the themes you pulled from the talks, without turning into a full platform rebuild.

## 6. Custom resource design

Define a CRD like this:

```yaml
apiVersion: demo.platform/v1alpha1
kind: TinyLLMService
metadata:
  name: tiny-llm
spec:
  replicas: 1
  model:
    repository: bartowski/SmolLM2-135M-Instruct-GGUF
    file: SmolLM2-135M-Instruct-Q4_K_M.gguf
    revision: main
  promptPrefix: "Be brief and helpful."
  resources:
    cpu: "250m"
    memory: "512Mi"
  ingress:
    enabled: true
    host: tiny-llm.demo.example.com
  observability:
    beylaEnabled: true
status:
  phase: Ready
  readyReplicas: 1
  backendURL: http://tiny-llm.tiny-llm.svc.cluster.local
  frontendURL: https://tiny-llm.demo.example.com
  lastReconcileTime: "2026-04-23T10:00:00Z"
```

### Spec fields

Suggested fields:

* `replicas`
* `model.repository`
* `model.file`
* `model.revision`
* `promptPrefix`
* `resources`
* `ingress.enabled`
* `ingress.host`
* `observability.beylaEnabled`

### Tiny model refs

Use tiny quantized GGUF models from Hugging Face.

Suggested examples:

* `bartowski/SmolLM2-135M-Instruct-GGUF` + `SmolLM2-135M-Instruct-Q4_K_M.gguf`
* `Qwen/Qwen2.5-0.5B-Instruct-GGUF` + `qwen2.5-0.5b-instruct-q5_k_m.gguf`

## 7. Operator behavior

The operator should reconcile a `TinyLLMService` into:

* a backend Deployment running a tiny LLM server
* a backend Service exposing it internally
* an optional backend Ingress
* a shared frontend Deployment and catalog ConfigMap
* status updates on the CR

### Reconciliation rules

* if CR does not exist, nothing exists
* if `replicas` changes, scale Deployment
* if `model.*` changes, roll Deployment
* if `ingress.enabled` changes, add/remove Ingress
* if Service endpoint becomes healthy, mark CR `Ready`
* if pods are pending or crashing, set status accordingly

This gives you a very clear “operator value” story: the user declares one business object, the controller manages the underlying Kubernetes objects.

## 8. Demo app spec

The app should be **tiny**, HTTP-based, and intentionally observable.

### Required endpoints

* `GET /health`
* `POST /generate`
* `GET /slow`
* `GET /error`
* `GET /config`

### Behavior

* `/generate` returns a fake or tiny-generated response
* `/slow` sleeps 1–3 seconds
* `/error` returns a 500
* `/config` shows current mode and prefix

### Language

Use **Go**.

Reason:

* tiny binaries
* easy containers
* easy HTTP service
* easy operator-adjacent ecosystem
* clean for demos

### Tiny LLM strategy

Do **not** try to prove “real AI performance.”
Prove:

* AI-like service packaging
* operator-managed rollout
* observability of inference-ish requests

That’s enough.

## 9. Observability plan

Start with a narrow core, then expand.

### Phase A — minimum

* Grafana
* Prometheus
* Beyla

Goal:

* request count
* error count
* latency

Beyla’s value here is the “very low instrumentation effort” story. ([Grafana Labs][4])

### Phase B — logs

Add:

* Loki
* log shipping agent

Goal:

* correlate 500s and slow calls with app logs

### Phase C — traces

Add:

* Tempo

Goal:

* inspect slow requests end-to-end

Tempo remains Grafana’s tracing backend and fits naturally with Grafana/Prometheus/Loki workflows. ([Grafana Labs][5])

### Important note

Beyla relies on Linux eBPF/kernel capabilities, so it should be validated early on the chosen VM image/kernel before treating it as guaranteed. That’s the main technical risk in this stack. ([Grafana Labs][4])

## 10. Repo structure

Use one mono-repo first.

```text
platform-demo/
  terraform/
    envs/demo/
    modules/
      vm/
      firewall/
  bootstrap/
    cloud-init/
    scripts/
      install-k3s-server.sh
      install-k3s-agent.sh
      install-argocd.sh
  gitops/
    root/
    argocd/
    apps/
      operator/
      observability/
      tiny-llm/
    clusters/
      demo/
  operator/
    api/
    controllers/
    config/
    hack/
  app/
    cmd/server/
    internal/
    charts/tiny-llm/
  docs/
    spec.md
    runbook.md
    demo-script.md
```

## 11. Bootstrap flow

### Step 1

Run Terraform:

* create VM(s)
* output IPs
* output SSH command

### Step 2

Bootstrap k3s:

* node 1 becomes server
* optional node 2 joins as agent

### Step 3

Install ArgoCD once:

* create `argocd` namespace
* install pinned version
* register root application

Argo CD’s docs still show the standard namespace + official install manifest flow and recommend pinning versions rather than blindly tracking latest stable for real environments. That fits well here too. ([Argo CD][1])

### Step 4

ArgoCD syncs:

* namespaces
* operator
* observability
* tiny-llm app

### Step 5

Apply a `TinyLLMService` CR

## 12. GitOps layout

Use **app-of-apps**.

### Root app

Owns:

* operator app
* observability app
* tiny-llm app

### Operator app

Installs:

* CRD
* controller Deployment
* RBAC
* webhook only if truly needed

### Tiny-llm app

Installs:

* namespace
* one or more sample CRs

### Observability app

Installs:

* Grafana
* Prometheus
* optional Loki
* optional Tempo
* Beyla

This keeps the live demo very nice: “merge to Git, ArgoCD syncs, cluster changes.”

## 13. Demo scenarios

### Scenario 1 — bootstrap

Show:

* fresh VM
* cluster comes up
* ArgoCD healthy
* apps synced

### Scenario 2 — operator

Apply:

```yaml
kind: TinyLLMService
spec:
  replicas: 1
  model:
    repository: bartowski/SmolLM2-135M-Instruct-GGUF
    file: SmolLM2-135M-Instruct-Q4_K_M.gguf
    revision: main
```

Show:

* operator creates backend Deployment + Service + optional Ingress
* operator also ensures the shared frontend and catalog
* status moves to Ready

### Scenario 3 — rollout through CR

Change:

* `promptPrefix`
* `replicas`
* `model.repository`
* `model.file`
* `model.revision`

Show:

* operator reconciles
* backend Deployment rolls
* status updates

### Scenario 4 — observability

Hit:

* `/generate`
* `/api/chat`
* `/api/services`
* `/slow`
* `/error`

Show in Grafana:

* request rate
* error rate
* latency
* optional traces
* optional logs

### Scenario 5 — GitOps

Commit change to sample CR or Helm values.
Show:

* ArgoCD sync
* system changes itself

## 14. Acceptance criteria

The demo is complete when:

* Terraform provisions infra from zero
* k3s cluster is reachable
* ArgoCD is healthy
* operator is installed by ArgoCD
* applying a `TinyLLMService` creates runnable app resources
* service is reachable over HTTP
* Grafana is reachable
* Prometheus shows traffic metrics
* Beyla captures useful app telemetry
* a config change to the CR visibly reconciles
* a Git change syncs through ArgoCD successfully ([Grafana Labs][4])

## 15. Constraints and tradeoffs

### One node vs two

* **1 node**

  * easier
  * cheaper
  * better first milestone
* **2 nodes**

  * cooler visually
  * better scheduling story
  * more moving parts

My recommendation:

* build for **1 node first**
* keep Terraform and bootstrap ready for optional node 2

### Tiny model reality

A true local LLM on tiny free infra is likely to be more pain than value. So the spec should treat real inference as optional, not mandatory.

### Observability stack weight

Grafana + Prometheus + Loki + Tempo + Beyla on tiny hardware can get chunky fast. So make observability **progressive**, not all-or-nothing. k3s’ low overhead helps, but workload resource use is still your real budget. ([K3s][2])

## 16. Phased implementation plan

### Phase 1 — minimal happy path

* Terraform
* 1-node k3s
* ArgoCD
* tiny app without operator

Purpose:

* verify infra + GitOps path fast

### Phase 2 — operator

* Kubebuilder scaffold
* CRD + controller
* reconcile Deployment + Service + ConfigMap
* sample CR

Purpose:

* make the operator story real ([book.kubebuilder.io][3])

### Phase 3 — observability baseline

* Prometheus
* Grafana
* Beyla

Purpose:

* show useful telemetry with low app changes ([Grafana Labs][4])

### Phase 4 — exploration

* Loki
* Tempo
* node 2
* optional tiny-local model backend

## 17. Live demo script

Suggested live flow:

1. show repo layout
2. run `terraform apply`
3. show SSH into server
4. show `kubectl get nodes`
5. show ArgoCD UI with synced apps
6. apply `TinyLLMService`
7. show created resources
8. hit `/generate`, `/slow`, `/error`
9. open Grafana
10. show RED metrics
11. change CR replicas or model
12. show reconcile + rollout
13. commit Git change
14. show ArgoCD applying it

## 18. Final recommendation

Build **this exact version**:

* provider-neutral Terraform
* **1-node k3s first**, optional second node later
* ArgoCD manages all in-cluster components
* Kubebuilder custom operator
* `TinyLLMService` CRD
* tiny Go frontend + operator-managed llama.cpp backends
* Grafana + Prometheus + Beyla first
* Loki + Tempo as follow-up

That gives you:

* least risk
* best demo value
* strongest operator story
* enough AI flavor without compute pain

## 19. Open questions I’d decide myself unless you want to steer them

* use Traefik from k3s defaults, not nginx
* use Go for both app and operator
* use Helm chart for app packaging
* use app-of-apps in ArgoCD
* keep the models tiny and quantized for v1

If you want, next thing I can do is turn this into a **concrete implementation backlog** with repo skeleton, milestones, and first CRD/API shape.

[1]: https://argo-cd.readthedocs.io/en/stable/getting_started/?utm_source=chatgpt.com "Getting Started - Argo CD - Declarative GitOps CD for Kubernetes"
[2]: https://docs.k3s.io/installation/requirements?utm_source=chatgpt.com "Requirements - K3s"
[3]: https://book.kubebuilder.io/quick-start.html?utm_source=chatgpt.com "Quick Start - The Kubebuilder Book"
[4]: https://grafana.com/docs/beyla/latest/?utm_source=chatgpt.com "Grafana Beyla | Grafana Beyla documentation"
[5]: https://grafana.com/docs/opentelemetry/instrument/beyla/?utm_source=chatgpt.com "Instrument an application with Beyla - Grafana Labs"
