# JavaScript Reducers

If any additional arguments were passed, fx converts them into a function which
takes the JSON as an argument named `x`.

By default, fx uses builtin JavaScript VM ([goja](https://github.com/dop251/goja)),
but also can be used with node.

```sh
export FX_LANG=js # Default
```

Or for usage with node:

```sh
export FX_LANG=node
```

An example of anonymous function used as a reducer:
```sh
$ echo '{"foo": [{"bar": "value"}]}' | fx 'x => x.foo[0].bar'
value
```

The same reducer function can be simplified to:

```sh
$ echo '{"foo": [{"bar": "value"}]}' | fx 'x.foo[0].bar'
value
```

Each argument treated as a reducer function.

```sh
$ echo '{"foo": [{"bar": "value"}]}' | fx 'x.foo' 'x[0]' 'x.bar'
value
```

Update JSON using the spread operator:

```sh
$ echo '{"name": "fx", "count": 0}' | fx '{...this, count: 1}'
{
  "name": "fx",
  "count": 1
}
```

## Dot

Fx supports simple JS-like syntax for accessing data, which can be used with any `FX_LANG`.

```sh
$ echo '{"foo": [{"bar": "value"}]}' | fx .foo[0].bar
value
```

## .fxrc.js

Create _.fxrc.js_ file in `$HOME` directory, and define some useful functions.

```js
// .fxrc.js
function upper(s) {
  return s.toUpperCase()
}
```

```sh
$ cat data.json | fx .name upper
ANTON
```

## Node

```sh
export FX_LANG=node
```

Use any npm package by installing it globally. Create _.fxrc.js_ file in `$HOME`
directory, and require any packages or define global functions. For example,
to access all lodash methods without `_` prefix, put next line into your 
_.fxrc.js_ file:

```js
Object.assign(global, require('lodash/fp'))
```

And now you will be able to call all lodash methods. For example, see who's been
committing to react recently:

```sh
curl 'https://api.github.com/repos/facebook/react/commits?per_page=100' \
| fx 'groupBy("commit.author.name")' 'mapValues(size)' toPairs 'sortBy(1)' reverse 'take(10)' fromPairs
```

> To be able to require global modules make sure you have correct `NODE_PATH` env variable.
> ```sh
> export NODE_PATH=`npm root -g`
> ```

The _.fxrc.js_ file supports both: `import` and `require`.

```js
// .fxrc.js
import 'zx/globals'
const _ = require('lodash')
```

> If you want to use _.fxrc.js_ for both `FX_LANG=js` and `FX_LANG=node`,
> separate parts by `// nodejs:` comment:
> ```js
> // .fxrc.js
> function upper(s) {
>   return s.toUpperCase()
> }
> // nodejs:
> import 'zx/globals'
> const _ = require('lodash')
> ```
