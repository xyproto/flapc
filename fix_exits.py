#!/usr/bin/env python3
import re

# Read the file
with open('parser.go', 'r') as f:
    content = f.read()

# Pattern to match fmt.Fprintf(os.Stderr, "Error: ...") followed by os.Exit(1)
# This handles both single-line and multi-line fprintf statements
lines = content.split('\n')
new_lines = []
i = 0

while i < len(lines):
    line = lines[i]

    # Check if this line has os.Exit(1)
    if 'os.Exit(1)' in line and 'fmt.Fprintf(os.Stderr' not in line:
        # Look backwards to find the fmt.Fprintf statement
        j = i - 1
        fprintf_lines = []

        while j >= 0:
            prev_line = lines[j]
            if 'fmt.Fprintf(os.Stderr' in prev_line:
                # Found the start of the fprintf
                # Collect all lines from j to i-1
                fprintf_lines = lines[j:i]

                # Extract the error message and args from fprintf
                fprintf_text = '\n'.join(fprintf_lines)

                # Match: fmt.Fprintf(os.Stderr, "Error: format string", args...)
                match = re.search(r'fmt\.Fprintf\(os\.Stderr,\s*"Error:\s*([^"]+)"([^)]*)\)', fprintf_text)

                if match:
                    format_str = match.group(1)
                    args = match.group(2).strip()

                    # Get the indentation from the fprintf line
                    indent = re.match(r'^(\s*)', lines[j]).group(1)

                    # Create the compilerError call
                    if args and args.startswith(','):
                        new_line = f'{indent}compilerError("{format_str}"{args})'
                    else:
                        new_line = f'{indent}compilerError("{format_str}")'

                    # Remove the fprintf lines and the os.Exit line
                    # Add everything up to j
                    # Then add the new compilerError line
                    # Then skip to i+1
                    new_lines.append(new_line)
                    i += 1
                    j = -1  # Break the backwards search
                    break

            # Check if we've gone too far back (e.g., hit another statement)
            if j < i - 10:  # Arbitrary limit
                break
            j -= 1

        if j != -1:
            # Couldn't find fprintf, just comment out the os.Exit
            new_lines.append(line.replace('os.Exit(1)', '// os.Exit(1) - FIXME: replace with compilerError'))
            i += 1
    elif 'fmt.Fprintf(os.Stderr' in line and i + 1 < len(lines) and 'os.Exit(1)' in lines[i+1]:
        # This fprintf is followed by os.Exit on the next line
        # Skip this line for now, it will be handled when we hit the os.Exit line
        pass  # Don't add this line yet
    elif i > 0 and 'os.Exit(1)' in lines[i-1]:
        # Previous line was os.Exit, skip this if it was already handled
        pass
    else:
        new_lines.append(line)
        i += 1

# Write the result
with open('parser.go', 'w') as f:
    f.write('\n'.join(new_lines))

print("Replaced os.Exit(1) calls with compilerError()")
