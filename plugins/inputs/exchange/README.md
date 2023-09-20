# Exchange Rate Input Plugin

## Configuration

```toml @sample.conf
# Gather repository information from api.apilayer.com.
[[inputs.exchange]]
  ## Required currency exchange API token. Visit https://api.apilayer.com/ to get token.
  apikey = ""

  ## Required base currency.
  base_currency = "EUR"

  ## Target currency.
  target_currencies = ["USD"]
```
