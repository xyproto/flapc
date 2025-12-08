# Shadow Keyword Implementation

## Overview

This document describes the `shadow` keyword feature added to C67 to prevent accidental variable shadowing while allowing intentional shadowing with explicit declaration.

## Motivation

**Problem:** Accidental variable shadowing is a common source of bugs where:
- A variable in an inner scope accidentally reuses a name from an outer scope
- The programmer doesn't realize they're hiding the outer variable
- Refactoring outer code can silently break inner scopes

**Previous Solution:** Required ALL_UPPERCASE for module-level variables
- **Downside:** Unnatural naming (PORT instead of port)
- **Downside:** Still allowed shadowing within functions
- **Downside:** Didn't prevent accidental shadowing in nested scopes

**New Solution:** Require explicit `shadow` keyword when shadowing
- **Advantage:** Natural naming in all scopes
- **Advantage:** Prevents accidental shadowing at all levels
- **Advantage:** Makes programmer intent crystal clear

## Syntax

```c67
shadow identifier [: type] = expression
shadow identifier [: type] := expression  
shadow identifier [: type] <- expression
```

## Rules

### 1. Shadow Required

When declaring a variable that would hide a variable from an outer scope, `shadow` is **required**:

```c67
// Module level
port = 8080
config = { host: "localhost" }

// Function that needs same names
main = {
    shadow port = 9000        // ✓ Required: shadows module 'port'
    shadow config = {}        // ✓ Required: shadows module 'config'
    println(port)             // Prints 9000
}
```

### 2. Shadow Forbidden

`shadow` cannot be used when nothing is being shadowed:

```c67
main = {
    value = 42                // ✓ OK: first declaration
    shadow result = value * 2 // ✗ ERROR: 'result' doesn't shadow anything
}
```

### 3. Updates Don't Shadow

Variable updates (`<-`) don't create new variables, so they never need `shadow`:

```c67
counter := 0
main = {
    counter <- counter + 1    // ✓ OK: updating existing variable, not shadowing
}
```

### 4. Case-Insensitive Check

Shadow detection is case-insensitive to catch common mistakes:

```c67
PORT = 8080
main = {
    port = 9000               // ✗ ERROR: would shadow PORT (case-insensitive)
    shadow port = 9000        // ✓ OK: explicitly shadows
}
```

## Examples

### Module-Level Variables

```c67
// No restrictions on naming at module level
port = 8080
maxConnections = 100
api_key = "secret"

// Functions can use any names
initServer = { println("Server starting") }
```

### Function-Level Shadowing

```c67
config = { port: 8080, host: "localhost" }

startServer = {
    // Must use shadow to reuse module-level names
    shadow config = { port: 9000, host: "0.0.0.0" }
    println(config.port)  // Prints 9000
}
```

### Nested Scope Shadowing

```c67
process = x -> {
    result = x * 2
    
    transform = y -> {
        shadow result = y + 100   // ✓ Shadows outer 'result'
        shadow x = x * 3          // ✓ Shadows parameter 'x'
        result + x
    }
    
    transform(result)
}
```

### Parameter Shadowing

```c67
// Shadow a parameter with a transformed version
normalize = value -> {
    shadow value = value / 100.0   // ✓ Shadows parameter
    value
}

// Common pattern: shadow to accumulate
sum_list = (list, acc) -> {
    list {
        [] => acc
        _ => {
            shadow acc = acc + head(list)  // ✓ Shadows parameter
            sum_list(tail(list), acc)
        }
    }
}
```

### Error Cases

```c67
x = 100

// ERROR: Forgot shadow keyword
test = {
    x = 200    // ✗ ERROR: variable 'x' shadows an outer scope variable
}              //          use 'shadow x = ...' to explicitly shadow

// ERROR: Unnecessary shadow keyword
compute = {
    y = 42
    shadow y = 100  // ✗ ERROR: 'shadow' keyword used but 'y' doesn't
}                   //          shadow any outer variable

// OK: Properly using shadow
safe = {
    shadow x = 200  // ✓ OK: explicitly shadows module 'x'
    x + 50
}
```

## Implementation Details

### Lexer Changes

Added `TOKEN_SHADOW` token type:
- Recognized as keyword `shadow`
- Can appear before variable declarations

### Parser Changes

1. **Scope Stack**: Added `scopes []map[string]bool` to track variables in each scope
2. **Scope Management**:
   - `pushScope()`: Enter new scope (function/block)
   - `popScope()`: Exit scope
   - `declareVariable(name)`: Register variable in current scope
   - `wouldShadow(name)`: Check if name exists in outer scopes

3. **parseAssignment() Updates**:
   - Check for `shadow` keyword at start
   - Validate shadow usage:
     - Error if shadowing without `shadow` keyword
     - Error if using `shadow` when not shadowing
   - Declare variable in current scope after validation

4. **Scope Boundaries**:
   - `parseLambdaBody()`: Push/pop scope for lambda bodies
   - `parsePrimary()` (statement blocks): Push/pop scope for blocks

### Removed Features

- **ALL_UPPERCASE requirement**: No longer enforced
- Module-level variables can use natural naming
- Function depth still tracked but not used for naming rules

## Migration Guide

### Old Code (ALL_UPPERCASE)

```c67
// Old style - required uppercase
PORT = 8080
CONFIG = { host: "localhost" }

main = {
    port := 9000      // Allowed - different case
    config = {}       // Allowed - different case
}
```

### New Code (shadow keyword)

```c67
// New style - natural naming
port = 8080
config = { host: "localhost" }

main = {
    shadow port = 9000      // Required - explicit shadowing
    shadow config = {}      // Required - explicit shadowing
}
```

## Benefits

1. **Prevents Bugs**: Accidental shadowing caught at compile time
2. **Clear Intent**: Reader knows shadowing is intentional
3. **Better Errors**: Precise error messages with suggestions
4. **Natural Naming**: No artificial ALL_UPPERCASE requirement
5. **Refactoring Safe**: Adding variables to outer scopes won't silently break inner code
6. **Consistent**: Same rules apply at all scope levels

## Testing

All existing tests updated to use `shadow` keyword where needed. Type annotation tests now pass with natural variable naming.

## Documentation

- **GRAMMAR.md**: Added Shadow Keyword section with full specification
- **LANGUAGESPEC.md**: Updated Variables and Assignment section
- Both documents include comprehensive examples and rationale
