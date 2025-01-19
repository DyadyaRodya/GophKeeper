# GophKeeper
GophKeeper is a client-server system that allows the user to securely store logins, passwords, binary data and other private information.

## install dependencies
```shell
make install
```

## linter
Install `statictest` [отсюда](https://github.com/Yandex-Practicum/go-autotests?tab=readme-ov-file#%D1%82%D1%80%D0%B5%D0%BA-%D1%81%D0%B5%D1%80%D0%B2%D0%B8%D1%81-%D1%81%D0%BE%D0%BA%D1%80%D0%B0%D1%89%D0%B5%D0%BD%D0%B8%D1%8F-url)

```shell
make lint
```

## tests
```shell
make tests
```

## build binary
server:
```shell
make build-server VERSION=0.0.1
```
client:
```shell
make build-client VERSION=0.0.1
```
