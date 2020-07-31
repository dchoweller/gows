# gows
## Cryptocurency Server Micro-Service

Makes of the free API from https://api.hitbtc.com/ to return real-time prices of currency symbols.

### ENDPOINTS:

#### GET /currency/{symbol}

Returns the real-time crypto prices of the given currency symbol.

Sample Response:
```{
    "id": "ETH",
    "fullName": "Ethereum",
    "ask": "0.054464", "bid": "0.054463",
    "last": "0.054463", "open": "0.057133",
    "low": "0.053615",
    "high": "0.057559",
    "feeCurrency": "BTC"
}
```

#### GET /currency/all

Returns the real-time crypto prices of all the supported currencies.
Response:
 
  
```{
    "currencies": [
        {
            "id": "ETH",
            "fullName": "Ethereum",
            "ask": "0.054464", "bid": "0.054463",
            "last": "0.054463", "open": "0.057133",
            "low": "0.053615",
            "high": "0.057559",
            "feeCurrency": "BTC"
    },
    {
            "id": "BTC",
            "fullName": "Bitcoin",
            "ask":"7906.72", "bid":"7906.28",
            "last":"7906.48",
            "open":"7952.3", "low":"7561.51",
            "high":"8107.96",
            "feeCurrency": "USD"
        }
    ]
}
```
## ENVIRONMENT VARIABLES

### Specify symbols to be included
DCHOWELLER_CRYPTO_SYMBOLS="ETHBTC,BTCUSD"
### Specify hostname for web server
DCHOWELLER_CRYPTO_HOSTNAME="localhost"
### Specify port for web server
DCHOWELLER_CRYPTO_PORT="8080"
