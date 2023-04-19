# fx

A non-interactive, JavaScript version of the [**fx**](https://fx.wtf). 
Short for _Function eXecution_ or _f(x)_.

```sh
npm i -g fx
```

Or use **npx**:

```sh
cat file.json | npx fx .field
```

Or use **node**:

```sh
cat file.json | node <(curl -s https://fx.wtf) .field
```

Or use **deno**:

```sh
cat file.json | deno run https://fx.wtf .field
```

## Usage

Fx treats arguments as JavaScript functions. Fx passes the input data to the first
function and then passes the result of the first function to the second function 
and so on.

```sh
echo '{"name": "world"}' | fx 'x => x.name' 'x => `Hello, ${x}!`'
```

Use `this` to access the input data. Use `.` at the start of the expression to 
access the input data without a `x => x` part.

```sh
echo '{"name": "world"}' | fx '.name' '`Hello, ${this}!`'
```

Use other JS functions to process the data.

```sh
echo '{"name": "world"}' | fx 'Object.keys'
```

## Advanced Usage

Fx can process a stream of json objects. Fx will apply arguments to each object.

```sh
echo '{"name": "hello"}\n{"name": "world"}' | fx '.name'
```

If you want to process a stream of json objects as a single array, 
use the **--slurp** or **-s** flag.

```sh
echo '{"name": "hello"}\n{"name": "world"}' | fx --slurp '.map(x => x.name)' '.join(", ")'
```

If you want to process non-JSON data, use the **--raw** or **-r** flag.

```sh
ls | fx -r '[this, this.includes(".md")]'
```

You can use **--raw** and **--slurp** (or **-rs**) together to get a single array of strings.

```sh
ls | fx -rs '.filter(x => x.includes(".md"))'
```

Fx has a special symbol **skip** for skipping the printing of the result.

```sh
ls | fx -r '.includes(".md") ? this : skip'
```

Fx comes with a set of useful functions: **uniq**, **sort**, **groupBy**.

```sh
cat file.json | fx 'uniq' 'sort' 'groupBy(x => x.name)'
```

Fx works with promises.

```sh
echo '"https://medv.io/*"' | fx 'fetch' '.text()'
```

### Syntactic Sugar

Fx has a shortcut for the map function. Fox example, `this.map(x => x.commit.message)`
can be rewritten without leading dot and without `x => x` parts.  

```sh
curl https://api.github.com/repos/antonmedv/fx/commits | fx 'map(.commit.message)'
```

```sh
echo '[{"name": "world"}]' | fx 'map(`Hello, ${x.name}!`)'
```

Fx has a special syntax for the flatMap function. Fox example,
`.flatMap(x => x.labels.flatMap(x => x.name))` can be rewritten in the next way.

```sh
curl https://api.github.com/repos/kubernetes/kubernetes/issues | fx '.[].labels[].name'
```

## License

[MIT](../LICENSE)
