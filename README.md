# Readme

## Builds

Create `.rcon-yml` file similar to:

```txt
host: localhost
password: ***
port: 25575
```

then run

```sh
build.sh github.com/StarForger/neb-rcon version
```

or for docker-compose create

Create `.env` file with VERSION=_version#_

and run:

```sh
docker-compose build build
```
