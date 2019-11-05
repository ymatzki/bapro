# Bapro

Bapro(backup prometheus) provides following feature.
- Export prometheus snapshot data to remote object storage.
- Import prometheus snapshot data from remote object storage.

### Usage

```
$ bapro
Export/Import prometheus snapshot data to remote object storage.

Usage:
  bapro [command]

Available Commands:
  help        Help about any command
  load        Import prometheus snapshot data to remote object storage.
  save        Export prometheus snapshot data to remote object storage.

Flags:
  -h, --help   help for bapro

Use "bapro [command] --help" for more information about a command.
```

### Reference

- https://prometheus.io/docs/prometheus/latest/querying/api/#snapshot
- https://prometheus.io/docs/prometheus/latest/storage/
- https://github.com/kubernetes/kubernetes/tree/master/cluster/addons/prometheus
- https://github.com/nginxinc/kubernetes-ingress/blob/master/docs/installation.md