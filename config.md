# Config 

- `SILENTIUM_NETWORK`: The network to connect to. This could be `mainnet`, `testnet`, or `regtest`.

- `SILENTIUM_START_HEIGHT`: The block height at which to start syncing from the blockchain.

- `SILENTIUM_RPC_COOKIE_PATH`: The path to the .cookie file for JSON-RPC authentication.

- `SILENTIUM_RPC_USER`: The username for JSON-RPC authentication. Not required if cookie path set.

- `SILENTIUM_RPC_PASS`: The password for JSON-RPC authentication. Not required if cookie path set.

- `SILENTIUM_RPC_HOST`: The host of the JSON-RPC server. 

- `SILENTIUM_PORT`: The port on which the application should run.

- `SILENTIUM_NO_TLS`: If set to `true`, the application will not use TLS for the gRPC server. Otherwise, it will.

- `SILENTIUM_CERT_FILE`: The path to the TLS certificate file.

- `SILENTIUM_KEY_FILE`: The path to the TLS key file.

- `SILENTIUM_DB_TYPE`: The type of database to use. Can be `badger` or `postgres`.

- `SILENTIUM_BADGER_DATADIR`: The directory where BadgerDB should store its data.

- `SILENTIUM_POSTGRES_DSN`: The Data Source Name (DSN) for connecting to a PostgreSQL database.
