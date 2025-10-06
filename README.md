`flapc` is an experiment in creating a vibecoded compiler for the Flap programming language.

Flap (`.flap`) code is compiled directly to machine code.

`x86_64`, `aarch64` and `riscv64` are supported.

This is a work in progress!

The Flap language is outlined in `language.md`.

The main features are:

* Modern CPU instructions are used whenever possible.
* `map[uint64]float64` is the core data type.
* No `nil`.
* Few keywords.

* License: BSD-3
