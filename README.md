# Beeline checker

> Simple API for checking balance and counters for multiple beeline accounts.

## Configuration
Use `config.yml` for configuration:
```yaml
---
listen_port: 9000
beeline_api: https://my.beeline.ru/api/1.0/
accounts:
  account_1:
    login: 9091234567
    password: superpass
  account_2:
    login: 9099876543
    password: megapass
debug: false
...
```

## API endpoints
Endpoint            |Methods
--------------------|-------
`/accounts`         |[GET](#get-accounts)
`/accounts/:name`   |[GET](#get-accountsname)

### `GET` /accounts
> Get list of all accounts with names and balances
#### Example request:
```shell
curl -s -X GET 127.0.0.1:9000/accounts
```
#### Example response:
```json
[
  {
    "Name": "account_1",
    "Number": 9091234567,
    "Status": "A",
    "Gigabytes": 29.58,
    "Minutes": 0,
    "SMS": 0,
    "Balance": 50
  },
  {
    "Name": "account_2",
    "Number": 9099876543,
    "Status": "A",
    "Gigabytes": 15.49,
    "Minutes": 118,
    "SMS": 97,
    "Balance": 10
  }
]

```
[↑](#api-endpoints)
### `GET` /accounts/:name
> Get balances of account with name `:name`.
#### Example request:
```shell
curl -s -X GET 127.0.0.1:9000/accounts/account_1
```
#### Example response:
```json
{
  "Name": "account_1",
  "Number": 9091234567,
  "Status": "A",
  "Gigabytes": 29.58,
  "Minutes": 0,
  "SMS": 0,
  "Balance": 50
}
```
[↑](#api-endpoints)