# Grammar Clarification Proposal: Lambda Bodies and Match Blocks

## Current Confusion

The grammar and documentation have conflicting information about blocks:

**Documentation (line 52-64) claims:**
> Every function body `{ ... }` is actually a match expression

**But the grammar (line 617) says:**
```ebnf
lambda_body = block | expression [ match_block ]
```

**Reality:**
- `x => { println("hi") }` - works (statement block)
- `x => x { 0 -> ... }` - works (expression + match block)  
- `x => { 0 -> ... }` - **DOESN'T work** (tries to parse as block, fails on `->`)

## The Root Problem

There are THREE different uses of `{ }`:

1. **Statement Block:** `{ stmt; stmt; expr }`
2. **Match Block:** `condition { pattern -> result ~> default }`
3. **Map Literal:** `{ "key": value, "key2": value2 }`

These are disambiguated by:
- Map literal: contains `:`
- Match block: must follow an expression
- Statement block: standalone or after `=>`

**The confusion:** Documentation says ALL blocks in lambdas are match blocks, but they're not.

---

## Proposal: Three-Tier Lambda Syntax (Minimal Changes)

Keep it simple and clear with three forms:

### 1. Expression Lambda (Single Expression)
```flap
square := x => x * x
add := (x, y) => x + y
```

**Grammar:**
```ebnf
lambda_expr = parameter_list "=>" expression
```

### 2. Block Lambda (Multiple Statements)
```flap
compute := x => {
    temp := x * 2
    result := temp + 10
    result
}
```

**Grammar:**
```ebnf
lambda_expr = parameter_list "=>" block
block = "{" { statement } [ expression ] "}"
```

**Key:** No `->` or `~>` allowed inside. Pure statements.

### 3. Match Lambda (Pattern Matching)
```flap
classify := x => match x {
    0 -> "zero"
    ~> x > 0 { -> "positive" ~> "negative" }
}
```

**Grammar:**
```ebnf
lambda_expr = parameter_list "=>" "match" expression match_block
match_block = "{" { match_clause } [ default_arm ] "}"
```

**Key:** Explicit `match` keyword before the condition.

---

## Benefits

1. **No Ambiguity:** Each form is syntactically distinct
2. **Clear Intent:** Developer explicitly chooses match or block
3. **Minimal Change:** Just add `match` keyword, everything else stays the same
4. **Backwards Compatible:** Existing code still works (just without the implicit match feature)

---

## Alternative: Keep Current Grammar, Fix Documentation

If adding `match` keyword is too much, simply **fix the documentation** to match reality:

**Change line 52-64 from:**
> Every function body `{ ... }` is actually a match expression

**To:**
> Lambda bodies can be either statement blocks or match expressions:
> ```flap
> // Statement block
> process := x => {
>     println(x)
>     x * 2
> }
> 
> // Match expression (requires condition before block)
> classify := x => x {
>     0 -> "zero"
>     ~> "other"
> }
> ```

Then update grammar to be clearer:

```ebnf
lambda_expr = parameter_list "=>" lambda_body

lambda_body = statement_block 
            | expression [ match_block ]

statement_block = "{" { statement } [ expression ] "}"

match_block = "{" { match_clause } [ default_arm ] "}"
```

---

## Recommendation

**Go with Alternative (Fix Documentation)** because:
1. No breaking changes
2. Grammar already supports it correctly
3. Just need to clarify documentation
4. Current implementation matches this interpretation

**Changes needed:**
1. Update LANGUAGE.md line 52-64 (remove false claim about ALL blocks being match)
2. Add clear examples showing both forms
3. Update grammar comments to explain disambiguation
4. Add note that match blocks MUST follow an expression

---

## Updated Grammar Section

Replace lines 614-624 with:

```ebnf
lambda_expr = parameter_list "=>" lambda_body
            | "==>" lambda_body ;  // Shorthand for () =>

parameter_list = identifier { "," identifier }
               | "(" [ identifier { "," identifier } ] ")" ;

lambda_body = statement_block 
            | expression [ match_block ] ;

// Lambda body semantics:
// - statement_block: Regular statements, final value is last expression
//   Example: x => { y := x + 1; y * 2 }
// 
// - expression + match_block: Pattern matching on expression result
//   Example: x => x { 0 -> "zero" ~> "other" }
//
// Note: Match blocks REQUIRE a preceding expression. You cannot write:
//   x => { 0 -> "zero" }  // ERROR: treated as statement block, -> invalid
// Must write:
//   x => x { 0 -> "zero" }  // OK: x is the match condition

statement_block = "{" { statement } [ expression ] "}" ;

match_block = "{" { match_clause } [ default_arm ] "}" ;

match_clause = expression [ "->" expression ] ;

default_arm = "~>" expression ;
```

---

## Summary

**Current State:** Documentation lies, grammar is correct
**Fix:** Update documentation to match reality
**Impact:** Zero code changes, just clearer docs
**Result:** Developers understand what actually works
