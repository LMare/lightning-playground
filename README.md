
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

**TODO** :
  - CD / infra as code (Terraform + kubernetes with Kind) -> Working in progress [feature/kubernetes](https://github.com/LMare/lightning-playground/tree/feature/kubernetes)
  - modules with Go
  - increase the test cover by implementing TI

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
