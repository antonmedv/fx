# fx

The JavaScript version of the **fx**. Short for _Function eXecution_ or _f(x)_.

```sh
npm i -g fx
```

Or use **npx**:

```sh
cat file.json | npx fx .param
```

Or use **node**:

```sh
cat file.json | node <(curl -s https://fx.wtf) .param
```

Or use **deno**:

```sh
cat file.json | deno run https://fx.wtf .param
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
echo `{"name": "world"}` | fx 'Object.keys'
```

## Advanced Usage

Fx has a shortcut for the map function. Fox example `.map(x => x.commit.message)`
can be written without leading dot and without `x => x` parts.  

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
