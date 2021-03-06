## stkcli state

Get or restore state

### Synopsis

The state namespace contains commands used to manage the state of the node: dump to a file, or restore from a local file.

The node's state is a JSON document containing the list of apps and sites configured. You can dump it to a local file and restore it on the same node or a different one.

The state file can contains secrets (e.g self-signed TLS certificates) which are encrypted with the symmetric key from the node's configuration file.


### Options

```
  -h, --help   help for state
```

### SEE ALSO

* [stkcli](stkcli.md)	 - Manage a Statiko node
* [stkcli state get](stkcli_state_get.md)	 - Retrieve state and save to file
* [stkcli state set](stkcli_state_set.md)	 - Restores the state of a node
* [stkcli state sync](stkcli_state_sync.md)	 - Triggers a sync of the node

