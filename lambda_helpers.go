package main

// hasLocalVariables checks if a lambda body defines any local variables
// Used to detect unsupported lambda patterns and give clear error messages
func hasLocalVariables(expr Expression) bool {
	found := false
	
	var scan func(Expression)
	scan = func(e Expression) {
		if e == nil || found {
			return
		}
		
		switch ex := e.(type) {
		case *BlockExpr:
			for _, stmt := range ex.Statements {
				if assign, ok := stmt.(*AssignStmt); ok {
					if !assign.IsUpdate && !assign.IsReuseMutable {
						found = true
						return
					}
				}
			}
			
		case *MatchExpr:
			for _, clause := range ex.Clauses {
				scan(clause.Result)
			}
			scan(ex.DefaultExpr)
		}
	}
	
	scan(expr)
	return found
}
