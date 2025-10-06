`flapc` is an experiment in creating a vibecoded compiler for the Flap programming language, for fun and for the educational process.

The Flap language is outlined in `language.md`.

This is a work in progress!

### Main features and limitations

* Flap (`.flap`) code is compiled directly to machine code.
* `x86_64`, `aarch64` and `riscv64` are supported.
* Only Linux and ELF executables are supported.
* Modern CPU instructions are used whenever possible.
* `map[uint64]float64` is the core data type.
* No `nil`.
* Few keywords.
* Aims to be "suckless".

### General info

* License: BSD-3
