`flapc` is an experiment in creating a vibecoded compiler for the Flap programming language.

`.flap` code is compiled directly to machine code.

`x86_64`, 64-bit ARM and 64-bit RISC-V is supported.

This is a work in progress!

The Flap language is outlined in `language.md`.

The main features are:

* Modern CPU instructions are used whenever possible.
* `map[uint64]float64` is the core data type.
* No `nil`.
* Few keywords.

* License: BSD-3
