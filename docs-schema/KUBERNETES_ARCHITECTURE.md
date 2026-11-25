# Kubernetes Architecture - Lightning Playground

## Vue d'ensemble

```
┌─────────────────────────────────────────────────────────────┐
│                  KIND Cluster (Local K8s)                   │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              Namespace: lightning-playground            │ │
│  │                                                          │ │
│  │  ┌─────────────────────────────────────────────────┐   │ │
│  │  │  API Deployment (1 pod - 2 containers)          │   │ │
│  │  │  ├─ Container: api (app :8080, ops :9090)       │   │ │
│  │  │  ├─ Container: sidecar (cert/secret mgmt)       │   │ │
│  │  │  ├─ Volume: lnd-shared-data (emptyDir)          │   │ │
│  │  │  ├─ Service: api (ClusterIP, ports api          │   │ │
│  │  │  │   and ops)                                   │   │ │
│  │  │  └─ Ingress: app-ingress (HTTP 80 -> api)       │   │ │
│  │  └─────────────────────────────────────────────────┘   │ │
│  │                                                          │ │
│  │  ┌─────────────────────────────────────────────────┐   │ │
│  │  │  BTCD StatefulSet (1 pod)                       │   │ │
│  │  │  ├─ Pod: btcd-0                                 │   │ │
│  │  │  ├─ PVC: btcd-data (persistent blockchain)      │   │ │
│  │  │  ├─ Service: btcd-headless (DNS: btcd-0)        │   │ │
│  │  │  └─ ConfigMap: btcd-config                      │   │ │
│  │  └─────────────────────────────────────────────────┘   │ │
│  │                                                          │ │
│  │  ┌─────────────────────────────────────────────────┐   │ │
│  │  │  LND StatefulSet (4 pods)                       │   │ │
│  │  │  ├─ Pod: lnd-0 (LND node + sidecar)             │   │ │
│  │  │  ├─ Pod: lnd-1 (LND node + sidecar)             │   │ │
│  │  │  ├─ Pod: lnd-2 (LND node + sidecar)             │   │ │
│  │  │  ├─ Pod: lnd-3 (LND tower)                      │   │ │
│  │  │  ├─ PVCs: lnd-0-data, lnd-1-data, etc...       │   │ │
│  │  │  └─ Service: lnd-headless (DNS: lnd-0.lnd-...) │   │ │
│  │  └─────────────────────────────────────────────────┘   │ │
│  │                                                          │ │
│  │  ┌─────────────────────────────────────────────────┐   │ │
│  │  │  LNDTower Deployment (1 pod)                    │   │ │
│  │  │  ├─ Pod: lndtower-xxx                           │   │ │
│  │  │  ├─ PVC: tower-data (watchtower backups)        │   │ │
│  │  │  └─ Service: lndtower (ClusterIP)               │   │ │
│  │  └─────────────────────────────────────────────────┘   │ │
│  │                                                          │ │
│  │  ┌─────────────────────────────────────────────────┐   │ │
│  │  │  RBAC (Role-Based Access Control)               │   │ │
│  │  │  ├─ ServiceAccount: api-service-account         │   │ │
│  │  │  ├─ ServiceAccount: btcd-service-account        │   │ │
│  │  │  ├─ ServiceAccount: lnd-service-account         │   │ │
│  │  │  ├─ Roles: pod-reader, secret-reader            │   │ │
│  │  │  └─ RoleBindings: Connect roles to accounts     │   │ │
│  │  └─────────────────────────────────────────────────┘   │ │
│  │                                                          │ │
│  │  ┌─────────────────────────────────────────────────┐   │ │
│  │  │  Secrets (mTLS Certificates)                    │   │ │
│  │  │  ├─ btcd-secret: RPC certs                      │   │ │
│  │  │  ├─ lnd-secret: TLS certs                       │   │ │
│  │  └─────────────────────────────────────────────────┘   │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Différences Docker-Compose vs Kubernetes

| Aspect | Docker-Compose | Kubernetes |
|--------|---|---|
| **Interface** | Servie par l'API (port 8080, publié en 3000) | Servie par l'API, exposée par `app-ingress` (pas de service frontend) |
| **API** | Single container | Deployment with sidecar (2 containers) |
| **BTCD** | Service | StatefulSet + PVC (persistent blockchain) |
| **LND** | 2 services | StatefulSet with 4 replicas (lnd-0, 1, 2, 3) |
| **Réseau** | Docker bridge | Service DNS (lnd-0.lnd-headless.svc.cluster.local) |
| **Volumes** | Named volumes | PersistentVolumeClaim (PVC) |
| **Certificats** | Fichiers partagés | Kubernetes Secrets |
| **Sécurité** | - | RBAC + ServiceAccounts |

---

## Types de ressources Kubernetes utilisées

### 1. **Deployment** (Stateless)
```yaml
# API (UI + API) - répliques identiques
kind: Deployment
replicas: 1
```

**Usage** : Services sans état (peuvent être redémarrés/replaced)

### 2. **StatefulSet** (Stateful)
```yaml
# BTCD & LND - identité stable, persistence
kind: StatefulSet
replicas: 4  # lnd-0, lnd-1, lnd-2, lnd-3
```

**Usage** : Services avec état persistent (blockchain, wallets, channels)

**Avantages** :
- Noms de pods stables (`lnd-0`, `lnd-1`, etc.)
- DNS stable (`lnd-0.lnd-headless`)
- PVC liés à chaque pod

### 3. **Service**
```yaml
# Types utilisés:
- ClusterIP (default) - api (ports `api` 8080 et `ops` 9090)
- Headless - btcd-headless, lnd-headless (DNS direct)
```

### 4. **Ingress**
```yaml
# Expose l'application (port app de l'API) au monde extérieur
app-ingress  (host lightning-playground.local, service api, port api)
  ├─ HTTP: /
  └─ annotations: proxy-buffering off, proxy-read-timeout 3600 (SSE)
# Le port ops (/health, /ready, /metrics) n'est pas routé par l'ingress
```

### 5. **PersistentVolumeClaim (PVC)**
```yaml
# Stockage persistant
- btcd-data (blockchain)
- lnd-0-data, lnd-1-data, lnd-2-data, lnd-3-data (wallets)
- tower-data (watchtower backups)
```

### 6. **ServiceAccount + RBAC**
```yaml
# Sécurité: Chaque service a son identité
- api-service-account
- btcd-service-account
- lnd-service-account

# Avec Roles/RoleBindings pour accéder aux secrets
```

### 7. **Secret**
```yaml
# Certificats TLS/mTLS
- btcd-secret (RPC certificates)
- lnd-secret (TLS certificates)
```

### 8. **ConfigMap**
```yaml
# Configuration BTCD
- btcd-cm1-configmap
```

---

## Architecture réseau Kubernetes

### DNS Resolution
```
API pod appelle LND:
  api → lnd-0.lnd-headless.lightning-playground.svc.cluster.local:10009
            ↓
  Kubernetes DNS résout en IP du pod lnd-0
            ↓
  gRPC connection établie

Format DNS: <pod-name>.<headless-service>.<namespace>.svc.cluster.local
```

### Service Types utilisés

**1. Headless Service (StatefulSet)**
```yaml
clusterIP: None  # Pas de load balancing, DNS direct
# Utilisé pour: btcd-headless, lnd-headless
# Raison: Besoin identités stables et DNS pod-level
```

**2. ClusterIP Service (Deployment)**
```yaml
clusterIP: <auto>  # Load balancer sur pods
# Utilisé pour: api, lndtower
# Raison: Pods peuvent être remplacés, service IP reste stable
```

---

## Container Sidecars

### API Pod (2 containers)
```yaml
containers:
  - name: api
    image: LMare/lightning-playground-api:v0.2.0-14
    ports:
      - containerPort: 8080  # app (pages, fragments, JSON, SSE, assets)
      - containerPort: 9090  # ops (/health, /ready, /metrics)
    volumeMounts:
      - name: lnd-shared-data
        mountPath: /app/node-storage
  
  - name: sidecar
    image: LMare/lightning-playground-sidecar-api:v0.2.0-14
    # Role: Gère les certificats/secrets LND
    #       Partage les données avec l'API via volume emptyDir
```

**Pourquoi 2 containers ?**
- API: Logique application
- Sidecar: Gestion certificats TLS/mTLS (injection sécurisée)
- Volume partagé: Communication inter-container

### LND Pod (2 containers) - Optionnel
LND containers peuvent aussi avoir des sidecars pour les certificats.

---

## Volumes & Persistence

### emptyDir (Éphémère)
```yaml
# API deployment
volumes:
  - name: lnd-shared-data
    emptyDir: {}  # Créé à la création du pod, détruit à la suppression
```

**Usage** : Données temporaires entre containers

### PersistentVolumeClaim (Persistant)
```yaml
# BTCD StatefulSet
volumeClaimTemplates:
  - metadata:
      name: btcd-data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 10Gi

# LND StatefulSet (1 par replica)
volumeClaimTemplates:
  - metadata:
      name: lnd-data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 5Gi
```

**Avantage** : Si un pod redémarre, les données persistent

---

## RBAC & Sécurité

### ServiceAccounts
Chaque service a son identité Kubernetes :

```yaml
# api-service-account.yaml
kind: ServiceAccount
metadata:
  name: api-service-account
  namespace: lightning-playground

# btcd-service-account.yaml
kind: ServiceAccount
metadata:
  name: btcd-service-account

# lnd-service-account.yaml
kind: ServiceAccount
metadata:
  name: lnd-service-account
```

### Roles & RoleBindings
```yaml
# lnd-role.yaml
kind: Role
metadata:
  name: lnd-role
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list", "watch"]

# lnd-rolebinding.yaml
kind: RoleBinding
metadata:
  name: lnd-rolebinding
roleRef:
  kind: Role
  name: lnd-role
subjects:
  - kind: ServiceAccount
    name: lnd-service-account
```

**Sécurité** : Chaque service accède **seulement** aux secrets dont il a besoin

---

## Secrets (Certificats TLS)

```yaml
# btcd-secret.yaml
kind: Secret
metadata:
  name: btcd-secret
  namespace: lightning-playground
type: Opaque
data:
  rpc.cert: <base64-encoded-cert>
  rpc.key: <base64-encoded-key>

# lnd-secret.yaml
kind: Secret
metadata:
  name: lnd-secret
type: Opaque
data:
  tls.cert: <base64-encoded-cert>
  tls.key: <base64-encoded-key>
```

**Montage dans pods** :
```yaml
volumeMounts:
  - name: btcd-certs
    mountPath: /root/.btcd
    readOnly: true
```

---

## Scalabilité & Limitations

### Actuellement
```yaml
# api-deployment.yaml
replicas: 1  # Pas de scalabilité horizontale
# Raison: SSE (Server-Sent Events) nécessite un broker (Redis/Kafka)

# lnd-statefulset.yaml
replicas: 4  # Peut scaler
# - lnd-0, lnd-1, lnd-2, lnd-3 = nœuds Lightning
# (Watchtower sera ajoutée plus tard)
```

### Pour scaler l'API
```yaml
# Si vous ajoutiez un broker de messages:
replicas: 3  # Plusieurs instances peuvent être déployées
# Avec Redis/Kafka pour les SSE events
```

---

## Flux de déploiement

```
1. Créer namespace
   kubectl create namespace lightning-playground

2. Créer ConfigMaps & Secrets
   kubectl apply -f btcd-secret.yaml
   kubectl apply -f lnd-secret.yaml
   kubectl apply -f btcd-cm1-configmap.yaml

3. Créer RBAC
   kubectl apply -f *-service-account.yaml
   kubectl apply -f *-role*.yaml

4. Créer services (headless)
   kubectl apply -f *-service-headless.yaml

5. Créer PVCs
   kubectl apply -f *-persistentvolumeclaim.yaml

6. Créer workloads
   kubectl apply -f btcd-statefulset.yaml
   kubectl apply -f lnd-statefulset.yaml
   kubectl apply -f api-deployment.yaml

7. Créer Ingress
   kubectl apply -f api-ingress.yaml

8. Vérifier
   kubectl get pods -n lightning-playground
   kubectl logs pod/api-xxx -n lightning-playground
```

---

## Accès local (KIND)

### Port Forwarding
```bash
# Accéder à l'application (UI + API, port app)
kubectl port-forward service/api 8080:8080 -n lightning-playground

# Accéder aux sondes et métriques (port ops, interne)
kubectl port-forward service/api 9090:9090 -n lightning-playground
```

### Ingress (cluster.yaml)
```yaml
nodes:
  - role: control-plane
    extraPortMappings:
      - containerPort: 80    # HTTP
        hostPort: 80
      - containerPort: 443   # HTTPS
        protocol: TCP
```

Permet d'accéder via `localhost` directement

---

## Différences K8s vs Docker-Compose clés

| Point | Docker-Compose | Kubernetes |
|-------|---|---|
| **Découverte service** | Hostname DNS | Kubernetes DNS + Service IP |
| **Stockage** | Volumes nommés | PersistentVolume + PersistentVolumeClaim |
| **Sécurité** | - | RBAC + Secrets + ServiceAccounts |
| **Scalabilité** | Replicas manuels | ReplicaSets/StatefulSets auto |
| **Rolling updates** | Manuel | Automatique |
| **Health checks** | Optionnel | Liveness/Readiness probes |
| **Réseau** | Bridge unique | Multi-pod, Services, Ingress |
