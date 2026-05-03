# goftp #

[![Units tests](https://github.com/simple-utils/ftp/actions/workflows/unit_tests.yaml/badge.svg)](https://github.com/simple-utils/ftp/actions/workflows/unit_tests.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/simple-utils/ftp.svg)](https://pkg.go.dev/github.com/simple-utils/ftp)

A FTP client package for Go.

## About this fork ##

This is a fork of [`github.com/jlaffaye/ftp`](https://github.com/jlaffaye/ftp),
created because the upstream repository has become inactive.

The goal of this fork is to keep the package maintained: integrate useful
community patches, keep dependencies and the supported Go versions up to date,
and continue accepting fixes.

## Install ##

```
go get -u github.com/simple-utils/ftp
```

## Documentation ##

https://pkg.go.dev/github.com/simple-utils/ftp

## Example ##

```go
c, err := ftp.Dial("ftp.example.org:21", ftp.DialWithTimeout(5*time.Second))
if err != nil {
    log.Fatal(err)
}

err = c.Login("anonymous", "anonymous")
if err != nil {
    log.Fatal(err)
}

// Do something with the FTP conn

if err := c.Quit(); err != nil {
    log.Fatal(err)
}
```

## Store a file example ##

```go
data := bytes.NewBufferString("Hello World")
err = c.Stor("test-file.txt", data)
if err != nil {
	panic(err)
}
```

## Read a file example ##

```go
r, err := c.Retr("test-file.txt")
if err != nil {
	panic(err)
}
defer r.Close()

buf, err := ioutil.ReadAll(r)
println(string(buf))
```

## License ##

Distributed under the ISC license, the same as the upstream project.
See [LICENSE](LICENSE) for details.
