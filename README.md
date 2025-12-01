[![Go CI](https://github.com/xyproto/flapc/actions/workflows/ci.yml/badge.svg)](https://github.com/xyproto/flapc/actions/workflows/ci.yml)

Flapc is a compiler for the Flap programming language.

See [GRAMMAR.md](GRAMMAR.md) and [LANGUAGESPEC.md](LANGUAGESPEC.md) for more information about the Flap programming language.

### Installation

`go install github.com/xyproto/flapc@latest`

### Example use for Linux

```sh
flapc sdl3example.flap -o sdl3example
./sdl3example
```

### Example use for Linux + Wine

```sh
flapc sdl3example.flap -o sdl3example.exe
wine sdl3example.exe
```

### Example programs

* [sdl3example.flap](sdl3example.flap)

### General info

* License: BSD-3
* Version: 1.5.0
