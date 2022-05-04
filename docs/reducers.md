# Reducers

Fx takes a few arguments after the file name, and converts them to a reducer. Currently `javascript` is the default reducer language. This is NOT the same as the `node` language type. The `node` language type is also the only reducer language type that supports the [`.fxrc.js`](#using-fxrcjs) file so it is required to specify the usage of `node` when this is wanted or required.

## Node

To operate with nodejs interpreter you will need to specify it using the `FX_LANG` environment variable.

```bash
$ export FX_LANG=node
```

### using-fxrc.js

The `.fxrc.js` file is javasciprt that is evaluated in the same scope and namespace that your reducer and data will be in. It can be used to preload helper functions, require/include additional node/npm modules or even have additional external data available to your reducers. 

Access all lodash (or ramda, etc) methods by using [.fxrc.js](#example-usages) file.

### `$HOME/.fxrc.js`

```js
Object.assign(global, {globalData: require('/home/user/.fxdata.js')}}
Object.assign(global, {myCustomFn: function(x){ return JSON.stringify(x) }})
Object.assign(global, require('lodash/fp'))

```

### example usages

```bash
$ export FX_LANG=node
$ fx data.json 'zip(this, globalData)' # using globalData injected from .fxrc.js
$ fx data.json 'myCustomFn(this)' # using myCustomFn defined in .fxrc.js
$ fx data.json 'groupBy("commit.committer.name")' 'mapValues(_.size)' # uses lodash functions applied to the global object
```

## Python

TODO

## Ruby

TODO
