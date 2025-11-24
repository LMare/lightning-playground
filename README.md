
# Lightning Playground

Personnal projet to discover and improve skill on  :
  - Golang
  - gRPC
  - Lnd
  - HTMX
  - SSE
  - dockerfile
  - docker compose
  - docker bake
  - TU in Go

TODO :
  - CI/CD
  - extend gRPC API
  - kubernetes
  - infra as code (Terraform + Kind)
  - do module with Go

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
docker exec -it lightning-playground_lnd1_1 lncli --network=simnet create
docker exec -it lightning-playground_lnd2_1 lncli --network=simnet create
```

To mine with one of these address do :
```bash
docker exec -it lightning-playground_lnd1_1 lncli --network=simnet newaddress np2wkh
```
Copy the address then replace the value of `miningaddr` in the service `btcd` of `docker-compose.yml`.
And reload the containers
```bash
docker compose up -d
```

Mine enough block to activate taproot
```bash
docker exec -it lightning-playground_btcd_1 btcctl --simnet generate 1500
```

### Unlock the wallet
After each up of the lnd containers, the wallet must be unlock
```bash
docker exec -it lightning-playground_lnd1_1 lncli --network=simnet unlock
docker exec -it lightning-playground_lnd2_1 lncli --network=simnet unlock
```

### Generate a new block
In the simnet network the news blocks must to be mine manually. Run this cmd (every 10 minutes) to keep the lnd node synchronised with the btcd node    
```bash
docker exec -it lightning-playground_btcd_1 btcctl --simnet generate 1
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
docker exec -it lightning-playground_btcd_1 btcctl --simnet generate 10
```
  3. Generate an invoice
  4. Import the invoice on another node and pay-it



## TODO List of Ideas :
  - update the label and the color of a node (http://github.com/LMare/lightning-playground/issues/1):
    * Need to update the code of lnd to expose a gRPC method to do that
