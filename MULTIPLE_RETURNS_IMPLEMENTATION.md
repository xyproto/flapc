# Multiple Return Values Implementation

## Status: ✅ FULLY IMPLEMENTED

Multiple return values have been successfully implemented in Flap 3.0 using tuple unpacking syntax.

## Syntax

```flap
// Basic multiple assignment
a, b = [10, 20]

// Mutable variables
x, y := [100, 200]

// Function returns
new_list, popped_value = pop([1, 2, 3])

// Works with any expression that returns a list
first, second, third = some_function()
```

## Implementation Details

### Grammar Changes (GRAMMAR.md)

Added to assignment rule:
```ebnf
assignment = ...
           | identifier_list ("=" | ":=" | "<-") expression ;

identifier_list = identifier { "," identifier } ;
```

### AST Changes (ast.go)

New AST node:
```go
type MultipleAssignStmt struct {
    Names    []string   // Variable names (left side)
    Value    Expression // Expression returning a list (right side)
    Mutable  bool       // true for :=, false for =
    IsUpdate bool       // true for <-
}
```

### Parser Changes (parser.go)

- Added `tryParseMultipleAssignment()` function
- Lookahead to distinguish from lambda parameters `(a, b) =>`
- Properly restores parser state if not a multiple assignment

### Code Generation (codegen.go)

1. **collectSymbols**: Allocates stack space for each variable
2. **compileStatement**: 
   - Evaluates the expression (must return a list)
   - Extracts elements at indices 0, 1, 2, etc.
   - Assigns to variables in order
   - Missing elements default to 0
   - Extra elements are ignored

## Semantics

### Element Assignment

Variables are assigned from list elements sequentially:
- `a` gets element at index 0
- `b` gets element at index 1
- `c` gets element at index 2
- etc.

### Missing Elements

If the list has fewer elements than variables:
```flap
a, b, c = [10, 20]  // c gets 0
```

### Extra Elements

If the list has more elements than variables:
```flap
a, b = [10, 20, 30, 40]  // 30 and 40 are ignored
```

### Null/Empty Lists

Handled safely - missing elements default to 0:
```flap
a, b = []  // Both a and b get 0
```

## Integration with Existing Features

### Works with list += operator

```flap
xs := []
xs += 1
xs += 2
a, b = xs  // a=1, b=2
```

### Works with functions

```flap
divmod = (n, d) => {
    quotient := n / d
    remainder := n % d
    [quotient, remainder]
}

q, r = divmod(17, 5)  // q=3, r=2
```

### Compatible with head/tail operators

```flap
xs = [1, 2, 3, 4, 5]
first = ^xs      // first element
rest = _xs       // remaining elements
f, r = [^xs, _xs]  // Can combine
```

## Testing

All tests pass:
```flap
// Test 1: Basic
a, b = [10, 20]
// ✅ a=10, b=20

// Test 2: Mutable
x, y := [100, 200]
// ✅ x=100, y=200

// Test 3: Function return
make_pair = () => [42, 99]
p, q = make_pair()
// ✅ p=42, q=99
```

## Benefits

### Clean pop() Usage

Instead of:
```flap
result = pop(xs)
new_list = result[0]
popped = result[1]
```

Now:
```flap
new_list, popped = pop(xs)
```

### Simpler APIs

Functions can naturally return multiple values:
```flap
// Before: awkward tuple access
result = calculate()
value = result[0]
error = result[1]

// After: clean unpacking
value, error = calculate()
```

### Better Ergonomics

```flap
// Swap values
a, b = [b, a]

// Multiple returns from one expression
min_val, max_val = find_min_max(data)

// Destructuring
x, y, z = get_coordinates()
```

## Limitations

1. Right side must be a list (or expression evaluating to a list)
2. Cannot mix with other patterns in single statement
3. No nested destructuring (future enhancement)

## Future Enhancements

Possible future additions:
- Rest operator: `a, b, ...rest = [1, 2, 3, 4, 5]`
- Nested destructuring: `a, [b, c] = [1, [2, 3]]`
- Map destructuring: `{x, y} = {x: 10, y: 20}`

## Conclusion

Multiple return values are fully implemented and production-ready. The feature integrates seamlessly with Flap's existing type system (everything is a map/list) and provides a clean, intuitive syntax for tuple unpacking.

**Status**: ✅ Ready for Flap 3.0 release
