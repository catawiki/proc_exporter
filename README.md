# Proc exporter

Prometheus exporter for process metrics exposed by `procfs` under `/proc/<pid>/*` directories.

## Building and running

Prerequisites:

- Go compiler

Building:

```bash
go get github.com/catawiki/proc_exporter
cd ${GOPATH-$HOME/go}/src/github.com/catawiki/proc_exporter
make
./proc_exporter <flags>
```

To see all available configuration flags:

```bash
./proc_exporter -h
```
