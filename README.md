[![codecov](https://codecov.io/gh/MauveSoftware/http-check/branch/main/graph/badge.svg)](https://codecov.io/gh/MauveSoftware/http-check)
[![Go ReportCard](http://goreportcard.com/badge/MauveSoftware/http-check)](http://goreportcard.com/report/MauveSoftware/http-check)

# http-check
Easy to use http(s) check for nagios/icinga

http-check is a distributed application consisting of a server and and a client component. The client requets a check from the server. The server processes the check, validates the result und returns it to the client.

## Install

### Client
```
go get -u github.com/MauveSoftware/http-check/cmd/http-check
```

### Server
```
go get -u github.com/MauveSoftware/http-check/cmd/http-check-server
```

## Run the server
```
./http-check-server
```

After starting the server listens for connections from the client on a unix socket (default: ``/tmp/http-check.sock``).

## Client usage
In this example we check if our homepage is available and if the closing body is present

```
./http-check -h www.mauve.de -s 200 -b '</body>'
```

## License
(c) Mauve Mailorder Software GmbH & Co. KG, 2020. Licensed under [Apache 2.0](LICENSE) license.
