# Flap Compiler TODO

## Completed
- [x] Redesign strings as map[uint64]float64 (indices to char codes) ✓
- [x] Debug string indexing issue (fixed: check for both "map" and "string" types in IndexExpr) ✓
- [x] Implement compile-time string concatenation for literals ✓

## In Progress

## Polymorphic Operators
- [ ] Implement string concatenation with + operator (map merging with offset)
- [ ] Implement list concatenation with + operator
- [ ] Implement map union with + operator
- [ ] Implement ++ operator:
  - Increment numbers by 1.0
  - Append single values to lists/maps
- [ ] Implement -- operator:
  - Decrement numbers by 1.0
  - Pop from end of lists/maps
- [ ] Implement - operator for intersection (strings/lists/maps)
- [ ] Add tests for all polymorphic operators

## String Improvements
- [ ] Implement runtime string concatenation (not just compile-time literals)
- [ ] Implement runtime map-to-C-string conversion for println(string_var)
- [ ] Handle string concatenation results in println directly

## Future Enhancements
- [ ] ARM64 NEON SIMD for map lookups
- [ ] RISC-V Vector extension support
- [ ] Perfect hashing for compile-time constant maps
- [ ] Binary search for large maps (32+ keys)
