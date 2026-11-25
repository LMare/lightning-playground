# Architecture Lightning Playground

## Vue d'ensemble du système

```
┌─────────────────────────────────────────────────────────┐
│                    UTILISATEUR                          │
└────────────────────────┬────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│          BACKEND (HTTP, Go) - port app 8080             │
│  ├─ pages, assets, fragments HTMX (http/web, ...)       │
│  ├─ http/lightning (nodes, channels, balance, payments) │
│  ├─ http/user (demo users)                              │
│  ├─ SSE (real-time payment events)                      │
│  ├─ client gRPC vers LND                                │
│  └─ port ops 9090 (/health, /ready, /metrics)           │
└────────────────┬────────────────────────────────────────┘
                 │
        ┌────────┴─────────┐
        │                  │              
        ▼                  ▼              
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  LND-0       │  │  LND-1       │  │  LNDTower    │
│  (Node 1)    │  │  (Node 2)    │  │  (Watchtower)│
│  Port: 9735  │  │  Port: 9735  │  │  Port: 9911  │
│  RPC: 10009  │  │  RPC: 10009  │  │  RPC: 10009  │
└──────┬───────┘  └──────┬───────┘  └───────┬──────┘
       │                 │                  │
       └─────────────────┼──────────────────┘
                         │
                         ▼
                  ┌──────────────┐
                  │     BTCD     │
                  │  Blockchain  │
                  │  (Bitcoin)   │
                  │ RPC: 18556   │
                  │ P2P: 18555   │
                  └──────────────┘
```

## Composants principaux

### 1. **Interface web (dans le backend)**
- Il n'y a plus de service frontend : le backend sert les pages, les assets statiques et les fragments HTMX
- Port app: `BACKEND_PORT` (8080), publié en `3000` par docker compose et routé par l'ingress Kubernetes (`app-ingress`)

### 2. **Backend**
- Serveur d'application Go, architecture hexagonale (voir [BACKEND_ARCHITECTURE.md](BACKEND_ARCHITECTURE.md))
- Expose une API HTTP (JSON, ou fragments HTML si la requête vient de HTMX) et un flux SSE (`/api/v1/stream-event`)
- Port ops: `BACKEND_OPS_PORT` (9090), `/health`, `/ready`, `/metrics`, interne au cluster uniquement
- Parle gRPC (avec macaroon + TLS) aux nœuds LND ; il n'expose pas de gRPC
- Persiste les utilisateurs de démonstration dans PostgreSQL (migrations au démarrage)

### 2 bis. **PostgreSQL**
- Table `users` (données de démo), accessible uniquement par le backend
- Kubernetes : chart Bitnami (`make deploy-postgres`) ; Docker Compose : service `postgres`

### 3. **Bitcoin Network (BTCD)**
- Implémentation blockchain Bitcoin (simnet)
- Simnet: Réseau de simulation Bitcoin pour testing
- RPC: Port 18556 (communication avec LND)
- P2P: Port 18555 (réseau pair-à-pair)
- Mining automatique avec adresse définie

### 4. **Lightning Network Nodes**

#### **LND-0** et **LND-1**
- Nœuds Lightning Network Daemon
- Gestion des canaux de paiement
- Fonctionnalités:
  - **P2P Network**: Port 9735 (connectivité réseau Lightning)
  - **RPC**: Port 10009 (API gRPC)
  - **Watchtower**: Surveillance des canaux (security)
  - **TLS**: Certificats auto-signés

#### **LNDTower** (Watchtower)
- Surveillance des pénalités (penalty transactions)
- Port 9911: Communication avec autres nœuds
- Sécurité: Prévient les fraudes sur les canaux

## Flux de données

### Création d'un canal de paiement
```
LND-0 ──(P2P 9735)──> LND-1
  │                      │
  └──────> BTCD <────────┘
           (RPC 18556)
```

### Paiement sur Lightning
```
User (navigateur) → Backend (UI + API) → LND-0 ──(P2P)──> LND-1 → Destination
                    ↓                         ↓
                  Router (payment path)    Router validation
```

## Dépendances
- `LND-0` et `LND-1` dépendent de `BTCD`
- `BTCD` doit être opérationnel avant les nœuds Lightning

## Volumes persistants
| Service | Volume | Chemin |
|---------|--------|--------|
| BTCD | btcd_data | /root/.btcd |
| LND-0 | lnd-0_data | /root/.lnd |
| LND-1 | lnd-1_data | /root/.lnd |
| LNDTower | tower_data | /root/.lnd |

## Configuration réseau (Simnet)
- **Simnet**: Réseau Bitcoin de simulation rapide
- **Génération auto de blocs**: Mining automatique sur BTCD
- **Pas de vraies transactions**: Données de test uniquement
