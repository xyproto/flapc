#!/usr/bin/env python3
import re
import sys

# Read file
with open('LANGUAGESPEC.md', 'r') as f:
    content = f.read()

# List patterns to fix: lambda definitions should use ->
patterns = [
    # Pattern: name = params => body (should be name = params -> body)
    (r'(\w+)\s*=\s*(\w+)\s*=>', r'\1 = \2 ->'),
    (r'(\w+)\s*=\s*\(([^)]+)\)\s*=>', r'\1 = (\2) ->'),
    
    # Pattern: name := params => body (should be name := params -> body)
    (r'(\w+)\s*:=\s*(\w+)\s*=>', r'\1 := \2 ->'),
    (r'(\w+)\s*:=\s*\(([^)]+)\)\s*=>', r'\1 := (\2) ->'),
]

# Apply fixes
for pattern, replacement in patterns:
    content = re.sub(pattern, replacement, content)

print("Fixed patterns. Manual review required for:", file=sys.stderr)
print("- Match arms (should keep =>)", file=sys.stderr)
print("- Guard matches (should use =>)", file=sys.stderr)
print(content)
