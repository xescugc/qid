# Functions

HCL functions are available in pipeline definitions for string manipulation, collection operations, encoding, and more.

## String functions

| Function      | Description                              | Example                                  |
|---------------|------------------------------------------|------------------------------------------|
| `chomp`       | Remove trailing newline                  | `chomp("hello\n")` -> `"hello"`          |
| `format`      | Format a string (printf-style)           | `format("v%s-%s", "1.0", "prod")` -> `"v1.0-prod"` |
| `formatlist`  | Format each element in a list            | `formatlist("-%s", ["a","b"])` -> `["-a","-b"]` |
| `indent`      | Indent all lines of a string             | `indent(2, "a\nb")` -> `"a\n  b"`       |
| `join`        | Join list elements with separator        | `join(",", ["a","b","c"])` -> `"a,b,c"`  |
| `lower`       | Convert to lowercase                     | `lower("HELLO")` -> `"hello"`           |
| `replace`     | Replace substring                        | `replace("hello", "l", "r")` -> `"herro"` |
| `split`       | Split string into list                   | `split(",", "a,b,c")` -> `["a","b","c"]` |
| `substr`      | Extract substring                        | `substr("hello", 1, 3)` -> `"ell"`      |
| `title`       | Capitalize first letter of each word     | `title("hello world")` -> `"Hello World"` |
| `trim`        | Remove characters from both ends         | `trim("?!hello?!", "!?")` -> `"hello"`  |
| `trimprefix`  | Remove prefix                            | `trimprefix("helloworld", "hello")` -> `"world"` |
| `trimsuffix`  | Remove suffix                            | `trimsuffix("helloworld", "world")` -> `"hello"` |
| `trimspace`   | Remove leading/trailing whitespace       | `trimspace("  hello  ")` -> `"hello"`   |
| `upper`       | Convert to uppercase                     | `upper("hello")` -> `"HELLO"`           |

## Collection functions

| Function    | Description                              | Example                                  |
|-------------|------------------------------------------|------------------------------------------|
| `concat`    | Concatenate lists                        | `concat(["a"], ["b"])` -> `["a","b"]`    |
| `contains`  | Check if list/set contains value         | `contains(["a","b"], "a")` -> `true`     |
| `distinct`  | Remove duplicates from list              | `distinct(["a","a","b"])` -> `["a","b"]` |
| `flatten`   | Flatten nested lists                     | `flatten([["a"],["b"]])` -> `["a","b"]`  |
| `keys`      | Get map keys                             | `keys({a=1, b=2})` -> `["a","b"]`       |
| `length`    | Get length of list, map, or string       | `length(["a","b"])` -> `2`               |
| `lookup`    | Look up value in map with default        | `lookup({a="1"}, "a", "0")` -> `"1"`    |
| `merge`     | Merge maps                               | `merge({a=1}, {b=2})` -> `{a=1, b=2}`   |
| `reverse`   | Reverse a list                           | `reverse(["a","b"])` -> `["b","a"]`      |
| `sort`      | Sort a list of strings                   | `sort(["b","a"])` -> `["a","b"]`         |
| `values`    | Get map values                           | `values({a=1, b=2})` -> `[1, 2]`        |

## Numeric functions

| Function | Description                | Example                    |
|----------|----------------------------|----------------------------|
| `abs`    | Absolute value             | `abs(-5)` -> `5`          |
| `ceil`   | Round up to nearest integer| `ceil(1.2)` -> `2`        |
| `floor`  | Round down to nearest integer| `floor(1.8)` -> `1`     |
| `max`    | Maximum of given numbers   | `max(1, 3, 2)` -> `3`     |
| `min`    | Minimum of given numbers   | `min(1, 3, 2)` -> `1`     |

## Encoding functions

| Function     | Description                | Example                              |
|--------------|----------------------------|--------------------------------------|
| `jsonencode` | Encode value as JSON       | `jsonencode({a=1})` -> `"{\"a\":1}"` |
| `jsondecode` | Decode JSON string         | `jsondecode("{\"a\":1}")` -> `{a=1}` |
| `csvdecode`  | Decode CSV string          | `csvdecode("a,b\n1,2")` -> list of maps |

## Regex functions

| Function       | Description                         | Example                                   |
|----------------|-------------------------------------|-------------------------------------------|
| `regex`        | Match regex, return first match     | `regex("v(\\d+)", "v123")` -> `"123"`     |
| `regexall`     | Match regex, return all matches     | `regexall("\\d+", "a1b2")` -> `["1","2"]` |
| `regexreplace` | Replace regex matches               | `regexreplace("hello", "l+", "r")` -> `"hero"` |

## Practical examples

Building docker args:

```hcl
task "test" {
  run "docker" {
    image = "golang:1.23"
    cmd   = "make test"
    args  = concat(
      ["-e", "CI=true"],
      ["-v", "/cache:/cache"],
    )
  }
}
```

String formatting:

```hcl
variable "version" {
  type    = string
  default = "1.0"
}

variable "env" {
  type    = string
  default = "prod"
}

task "tag" {
  run "exec" {
    path = "echo"
    args = [format("v%s-%s", var.version, var.env)]
  }
}
```

Joining values:

```hcl
task "echo" {
  run "exec" {
    path = "echo"
    args = [join(",", ["a", "b", "c"])]
  }
}
```
