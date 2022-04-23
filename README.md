# sgbl

A lightweight cli tool to interact with [Sourcegraph](https://sourcegraph.com).

## Installation

```bash
go get github.com/beyang/sgbl
```

## Configuration

You can configure `sgbl` to work with private [Sourcegraph](https://about.sourcegraph.com) instances in addition to Sourcegraph.com.

To do this, save a JSON file like the following to `~/.sgbl-config`:

```json
{
  "sourcegraphs": [
    {
      "url": "https://sourcegraph.mycompany.com",
      "repos": [
          "github.com/mycompany/private-repo",
          ...
       ]
    },
    ...
  ]
}
```

## Usage

```bash
sgbl [options] path/to/file
```

### Examples

#### Open a local file

```bash
sgbl main.go
```

#### Open a local file at a certain position

```bash
sgbl --line=22 main.go
# or
sgbl --line=22 --col=6 main.go
# or
sgbl --pos=22:6 main.go
```

#### Execute a search

```bash
sgbl --search="repo:gorilla/mux ^func Test"
```

#### Execute a search over the current repo

```bash
sgbl --search="^func main" .
```
