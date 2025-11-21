#!/bin/bash
# Fix all exit() calls to use Windows calling convention

# Fix line 560 (first exit call)
sed -i '560s|fc.out.XorRegWithReg("rdi", "rdi") // exit code 0|firstArgReg := "rdi"\n\t\tif fc.eb.target.OS() == OSWindows {\n\t\t\tfirstArgReg = "rcx"\n\t\t}\n\t\tfc.out.XorRegWithReg(firstArgReg, firstArgReg) // exit code 0\n\t\t_ = fc.allocateShadowSpace()|' codegen.go

# Fix line 993 (second exit call)
sed -i '993s|fc.out.XorRegWithReg("rdi", "rdi") // exit code 0|firstArgReg := "rdi"\n\t\tif fc.eb.target.OS() == OSWindows {\n\t\t\tfirstArgReg = "rcx"\n\t\t}\n\t\tfc.out.XorRegWithReg(firstArgReg, firstArgReg) // exit code 0\n\t\t_ = fc.allocateShadowSpace()|' codegen.go
