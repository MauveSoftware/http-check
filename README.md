[![CircleCI](https://circleci.com/gh/MauveSoftware/http-check.svg?style=shield)](https://circleci.com/gh/MauveSoftware/http-check/tree/master)
[![codecov](https://codecov.io/gh/MauveSoftware/http-check/branch/main/graph/badge.svg)](https://codecov.io/gh/MauveSoftware/http-check)
[![Go ReportCard](http://goreportcard.com/badge/MauveSoftware/http-check)](http://goreportcard.com/report/MauveSoftware/http-check)

# http-check
Easy to use http(s) check for nagios/icinga

## Install
```
go get -u github.com/MauveSoftware/http-check
```

## Usage
In this example we check if our homepage is available and if the closing body is present

```
./http-check -h www.mauve.de -s 200 -b '</body>'
```

## License
(c) Mauve Mailorder Software GmbH & Co. KG, 2020. Licensed under [Apache 2.0](LICENSE) license.
