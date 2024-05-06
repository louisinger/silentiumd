# Silentiumd

Silentium minimizes bandwith requirements of Silent Payment [(BIP352)](https://github.com/bitcoin/bips/pull/1458) wallets.

## Silent payment "scalar"

BIP352 defines silent payments as key-spend taproot script using a tweaked key.
```
silentpay_tapkey = key + tweak
```

`tweak` is a shared secret computable by the receiver and the sender. As silent payments receiver, you must compute the following for each transactions:

```
tweak = scan_sec_key * input_hash * sum(inputs_pubkeys)
```

It means scanning all taproot transactions in every block making the wallet bandwidth requirements high. 

Silentium connects to a full node and compute the public `scalar` for each transaction containing unspent taproot outputs.

```
scalar = input_hash * sum(inputs_pubkeys)
```

 Thus, a wallet can easily fetch those scalars for each block and compute the corresponding silent payments scripts. Combined with BIP158, the wallet may limit the number of blocks to download.

 ## Run Silentium

 ### Requirements

 * go 1.21
 * full node with JSON-RPC enabled

### Build

```
make build
```

### Run

first create a configuration file `config.yaml`:

```yaml
RPC_HOST: "127.0.0.1:8332"
RPC_COOKIE_PATH: "./.cookie"
```

then run the binary:

```
./silentium-linux-amd64
```

## License

<p xmlns:cc="http://creativecommons.org/ns#" xmlns:dct="http://purl.org/dc/terms/"><a property="dct:title" rel="cc:attributionURL" href="https://github.com/louisinger/silentiumd">silentiumd</a> by <a rel="cc:attributionURL dct:creator" property="cc:attributionName" href="https://github.com/louisinger">Louis Singer</a> is licensed under <a href="https://creativecommons.org/licenses/by/4.0/?ref=chooser-v1" target="_blank" rel="license noopener noreferrer" style="display:inline-block;">Creative Commons Attribution 4.0 International<img style="height:22px!important;margin-left:3px;vertical-align:text-bottom;" src="https://mirrors.creativecommons.org/presskit/icons/cc.svg?ref=chooser-v1" alt=""><img style="height:22px!important;margin-left:3px;vertical-align:text-bottom;" src="https://mirrors.creativecommons.org/presskit/icons/by.svg?ref=chooser-v1" alt=""></a></p>
