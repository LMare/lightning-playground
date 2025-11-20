
# LIGHTNING-TEST

Little personnal projet to discover and improve skill on  :
  - Golang
  - gRPC
  - Lnd
  - HTMX
  - SSE
  - dockerfile & docker compose


## Prupose
Be able to do a little web application to interract with and a lightning serveur running on simnet

## Lauch the app
```bash
docker-compose up -d
# or for development
docker-compose up -d --build
```
Go to : http://localhost:3000/

### First launch
To use the lnd fonctionalities, you will need at least 2 lnd nodes with a wallet :
```bash
docker exec -it lnd1 lncli --network=simnet create
docker exec -it lnd2 lncli --network=simnet create
```

To mine with one of these address do :
```bash
docker exec -it lnd1 lncli --network=simnet newaddress np2wkh
```
Copy the address then replace the value of `miningaddr` in the service `btcd` of `docker-compose.yml`.

Mine enough block to activate taproot
```bash
docker exec -it btcd btcctl --simnet generate 1500
```

### Unlock the wallet
After each up of the lnd containers, the wallet must be unlock
```bash
docker exec -it lnd1 lncli --network=simnet unlock
docker exec -it lnd2 lncli --network=simnet unlock
```

### Generate a new block
In the simnet network the news blocks must to be mine manually. Run this cmd (every 10 minutes) to keep the lnd node synchronised with the btcd node    
```bash
docker exec -it btcd btcctl --simnet generate 1
```


## Stop the app
```bash
docker-compose down
```


## TODO List of Ideas :
  - update the label and the color of a node (http://github.com/LMare/lightning-test/issues/1):
    * Need to update the code of lnd to expose a gRPC method to do that
