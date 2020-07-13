# localdata

... is meant to run as `DaemonSet` on your EKS cluster and ensures that a copy of some data is mounted on the cluster node
via a local EBS device. This allows to expose the data to `Pods` via `hostPath`. This can provide a few benefits compared to
other storage solutions:

- _Performance_: EBS usually is significantly faster than EFS.
- _Redundatcy_: If the data managed by `localdata` is used by multiple `Pods` then multiple copies of the data can be avoided

## State

This project is a _PROOF OF CONCEPT_ just barely works and should not be used in production.

## TODO

Future improvements can be:

- _Dashboard_ to monitor the state of things.
- _Concurrency_ while creating/attaching/mounting devices.
- _DataManagement_ such as syncing or prewarming.
- _Config Watching_ to hot reload on config changes.
- _Further Data Sources_ such as EFS or S3
