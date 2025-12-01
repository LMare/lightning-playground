
# Lightning Playground


[![CI](https://github.com/LMare/lightning-playground/actions/workflows/ci.yml/badge.svg)](https://github.com/LMare/lightning-playground/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/LMare/lightning-playground/branch/master/graph/badge.svg)](https://codecov.io/gh/LMare/lightning-playground)



Personnal projet to discover and improve skill on  :
  - Golang
  - gRPC (use & extend gRPC API)
  - Lnd
  - HTMX
  - SSE
  - dockerfile
  - docker compose
  - docker bake
  - CI

TODO :
  - CD / infra as code (Terraform + kubernetes with Kind)
  - modules with Go
  - increase the test cover by implementing TI

---------------------------------------------------------------------------------------------

## TODO List for Kubernetes Migration (Lightning Network Project)

(generated after a brainstorming with Copilot)

### 1. Cluster Setup
- [X] Spin up a local Kubernetes cluster (Kind, Minikube, or k3s).
- [ ] Configure `kubectl` and create a dedicated namespace (e.g. `lightning`).

### 2. Core Components
- [ ] **Frontend:** Deployment + Service + Ingress (stateless, scalable).
- [ ] **Backend:** Deployment + Service (stateless, responsible for discovering LND pods and unlocking/creating wallets).
- [ ] **btcd:** StatefulSet + PVC + Service (Bitcoin full node).
- [ ] **LND:** StatefulSet + PVC + headless Service (multiple replicas, each with its own wallet/certs).

### 3. Data & Secrets Management
- [ ] Define **PersistentVolumeClaims** for each LND and btcd.
- [ ] Create a **Secret bundle** (`lnd-credentials`) to store certs/macaroons for all LND pods.
- [ ] Implement a **job or init process** that copies certs/macaroons from LND PVCs into the Secret bundle.
- [ ] Mount the Secret in the backend in read-only mode.

### 4. Backend Responsibilities
- [ ] Discover LND pods via the headless service (`lnd-0.lnd-headless`, etc.).
- [ ] Read the corresponding certs/macaroons from the Secret bundle.
- [ ] Use gRPC to **create or unlock wallets** via the `WalletUnlocker` service.
- [ ] Replace static `nodes.yaml` with dynamic discovery logic.

### 5. Networking & Service Discovery
- [ ] Configure a **headless Service** for LND to provide stable DNS per pod.
- [ ] Ensure the backend can dynamically map endpoints (`lnd-N`) to certs/macaroons.
- [ ] Use Ingress to expose frontend/backend APIs externally.

### 6. Security & Best Practices
- [ ] Restrict RBAC permissions for the job that updates the Secret bundle.
- [ ] Mount Secrets as read-only in backend pods.
- [ ] Separate configs: ConfigMaps for non-sensitive data, Secrets for sensitive data.
- [ ] Add liveness/readiness probes for backend and LND.

### 7. Scalability & Monitoring
- [ ] Test scaling: `kubectl scale statefulset lnd --replicas=5`
- [ ] Verify the backend adapts automatically to new pods.
- [ ] Add monitoring (Prometheus + Grafana) and centralized logging.
- [ ] Define NetworkPolicies to restrict communication paths (backend ↔ LND ↔ btcd).

### 8. Finalization
- [ ] Organize manifests into folders (`frontend/`, `backend/`, `lnd/`, `btcd/`).
- [ ] Deploy everything with `kubectl apply -f ./manifests`.
- [ ] Validate end-to-end flow: frontend → backend → LND → btcd.
- [ ] Document the workflow for reproducibility (CI/CD, Helm charts, etc.).

---

## ✅ Expected Result

- Scaling LND pods is done with a single command (`kubectl scale`).
- Each LND pod has its own wallet and certs, stored securely in PVCs and synced into a Secret bundle.
- The backend dynamically discovers LND pods via DNS and uses the correct certs/macaroons from the Secret bundle.
- Wallet creation/unlock is handled by the backend via gRPC (`WalletUnlocker` service).
- Frontend and backend are exposed externally via Ingress.
- Secrets are mounted read-only, RBAC is restricted, and configs are separated (ConfigMap vs Secret).
- Monitoring, probes, and NetworkPolicies ensure production-grade reliability and observability.
- The architecture is clean, Kubernetes-native, and ready to evolve toward production.


-----------------------------------------------------------------------------------------------

## Prupose
Be able to do a little web application to interract with and a lightning serveur running on simnet

## Lauch the app

```bash
docker buildx bake
docker compose up -d
```
Go to : http://localhost:3000/

### First launch
To use the lnd fonctionalities, you will need at least 2 lnd nodes with a wallet :
```bash
docker exec -it lightning-playground-lnd1-1 lncli --network=simnet create
docker exec -it lightning-playground-lnd2-1 lncli --network=simnet create
```

To mine with one of these address do :
```bash
docker exec -it lightning-playground-lnd1-1 lncli --network=simnet newaddress np2wkh
```
Copy the address then replace the value of `miningaddr` in the service `btcd` of `docker-compose.yml`.
And reload the containers
```bash
docker compose up -d
```

Mine enough block to activate taproot
```bash
docker exec -it lightning-playground-btcd-1 btcctl --simnet generate 1500
```

### Unlock the wallet
After each up of the lnd containers, the wallet must be unlock
```bash
docker exec -it lightning-playground-lnd1-1 lncli --network=simnet unlock
docker exec -it lightning-playground-lnd2-1 lncli --network=simnet unlock
```

### Generate a new block
In the simnet network the news blocks must to be mine manually. Run this cmd (every 10 minutes) to keep the lnd node synchronised with the btcd node    
```bash
docker exec -it lightning-playground-btcd-1 btcctl --simnet generate 1
```


## Stop the app
```bash
docker-compose down
```

## Using the application

![Lightning-Playground](https://github.com/LMare/lightning-playground/blob/master/Lightning-Playground.png)


Steps :
 1. Add new pair with the URI of other nodes
 2. Create channels between pairs.
    After creating the channel to pass it in `active` state, generate some blocs with :
```bash
docker exec -it lightning-playground-btcd-1 btcctl --simnet generate 10
```
  3. Generate an invoice
  4. Import the invoice on another node and pay-it



## Note :
The app works with a LND customised [LMare/lnd](https://github.com/LMare/lnd/tree/feature/gRPC-alias-color).
This version allow to modify the alias and the color of the node LND by gRPC call.
