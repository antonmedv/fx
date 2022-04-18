# Reducers

Fx takes a few arguments after the file name, and converts them to a reducer.

## Node

Access all lodash (or ramda, etc) methods by using [.fxrc](#using-fxrc) file.

```bash
$ fx data.json 'groupBy("commit.committer.name")' 'mapValues(_.size)'
```

## Python

TODO

## Ruby

TODO
