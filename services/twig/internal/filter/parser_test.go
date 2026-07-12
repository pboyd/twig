package filter

import (
	"testing"
)

func TestLexQuoted(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"simple quoted", `"hello"`, "hello", false},
		{"quoted with spaces", `"hello world"`, "hello world", false},
		{"quoted with AND", `"AND review"`, "AND review", false},
		{"empty quoted", `""`, "", false},
		{"unterminated", `"hello`, "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lex := newLexer(tc.input)
			tok, err := lex.nextToken()
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tok.typ != tokenString {
				t.Errorf("type = %v, want tokenString", tok.typ)
			}
			if tok.value != tc.want {
				t.Errorf("value = %q, want %q", tok.value, tc.want)
			}
		})
	}
}

func TestLexOperators(t *testing.T) {
	cases := []struct {
		input string
		typ   tokenType
	}{
		{"=", tokenEq},
		{"!=", tokenNe},
		{"<", tokenLt},
		{"<=", tokenLe},
		{">", tokenGt},
		{">=", tokenGe},
		{"^", tokenHat},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			lex := newLexer(tc.input)
			tok, err := lex.nextToken()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tok.typ != tc.typ {
				t.Errorf("type = %v, want %v", tok.typ, tc.typ)
			}
		})
	}
}

func TestParseBareWord(t *testing.T) {
	expr, err := Parse("review")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(expr.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(expr.Conditions))
	}
	cond, ok := expr.Conditions[0].(*TextCondition)
	if !ok {
		t.Fatalf("expected *TextCondition, got %T", expr.Conditions[0])
	}
	if cond.Term != "review" {
		t.Errorf("term = %q, want %q", cond.Term, "review")
	}
}

func TestParseQuoted(t *testing.T) {
	expr, err := Parse(`"AND review"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(expr.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(expr.Conditions))
	}
	cond, ok := expr.Conditions[0].(*TextCondition)
	if !ok {
		t.Fatalf("expected *TextCondition, got %T", expr.Conditions[0])
	}
	if cond.Term != "AND review" {
		t.Errorf("term = %q, want %q", cond.Term, "AND review")
	}
}

func TestParseMultiWordBareword(t *testing.T) {
	_, err := Parse("buy milk")
	if err == nil {
		t.Fatal("expected error for missing AND, got nil")
	}
}

func TestParseEmptyExpression(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Fatal("expected error for empty expression, got nil")
	}
}

func TestParseWhitespaceOnly(t *testing.T) {
	_, err := Parse("   ")
	if err == nil {
		t.Fatal("expected error for whitespace-only expression, got nil")
	}
}

func TestParseAnd(t *testing.T) {
	expr, err := Parse(`foo AND bar`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(expr.Conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(expr.Conditions))
	}
	for i, cond := range expr.Conditions {
		tc, ok := cond.(*TextCondition)
		if !ok {
			t.Fatalf("condition[%d]: expected *TextCondition, got %T", i, cond)
		}
		if tc.Term != "foo" && tc.Term != "bar" {
			t.Errorf("condition[%d]: term = %q, want foo or bar", i, tc.Term)
		}
	}
}

func TestParseCompletedEq(t *testing.T) {
	expr, err := Parse(`completed=true`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(expr.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(expr.Conditions))
	}
	cond, ok := expr.Conditions[0].(*BoolCondition)
	if !ok {
		t.Fatalf("expected *BoolCondition, got %T", expr.Conditions[0])
	}
	if cond.Field != BoolFieldCompleted {
		t.Errorf("field = %v, want BoolFieldCompleted", cond.Field)
	}
	if cond.Op != OpEq {
		t.Errorf("op = %v, want OpEq", cond.Op)
	}
	if !cond.Value {
		t.Error("value = false, want true")
	}
}

func TestParseCompletedDate(t *testing.T) {
	expr, err := Parse(`completed < 2026-01-01`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(expr.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(expr.Conditions))
	}
	cond, ok := expr.Conditions[0].(*DateCondition)
	if !ok {
		t.Fatalf("expected *DateCondition, got %T", expr.Conditions[0])
	}
	if cond.Field != DateFieldCompleted {
		t.Errorf("field = %v, want DateFieldCompleted", cond.Field)
	}
	if cond.Op != OpLt {
		t.Errorf("op = %v, want OpLt", cond.Op)
	}
	if cond.Day != "2026-01-01" {
		t.Errorf("day = %q, want %q", cond.Day, "2026-01-01")
	}
}

func TestParseSnoozedEq(t *testing.T) {
	expr, err := Parse(`snoozed=true`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(expr.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(expr.Conditions))
	}
	cond, ok := expr.Conditions[0].(*BoolCondition)
	if !ok {
		t.Fatalf("expected *BoolCondition, got %T", expr.Conditions[0])
	}
	if cond.Field != BoolFieldSnoozed {
		t.Errorf("field = %v, want BoolFieldSnoozed", cond.Field)
	}
	if !cond.Value {
		t.Error("value = false, want true")
	}
}

func TestParseParentId(t *testing.T) {
	expr, err := Parse(`parent_id=42`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(expr.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(expr.Conditions))
	}
	cond, ok := expr.Conditions[0].(*RelCondition)
	if !ok {
		t.Fatalf("expected *RelCondition, got %T", expr.Conditions[0])
	}
	if cond.Field != RelFieldParentID {
		t.Errorf("field = %v, want RelFieldParentID", cond.Field)
	}
	if cond.ID != 42 {
		t.Errorf("id = %d, want 42", cond.ID)
	}
	if cond.Transitive {
		t.Error("transitive = true, want false")
	}
}

func TestParseTransitiveParentId(t *testing.T) {
	expr, err := Parse(`^parent_id=1`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(expr.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(expr.Conditions))
	}
	cond, ok := expr.Conditions[0].(*RelCondition)
	if !ok {
		t.Fatalf("expected *RelCondition, got %T", expr.Conditions[0])
	}
	if !cond.Transitive {
		t.Error("transitive = false, want true")
	}
	if cond.Field != RelFieldParentID {
		t.Errorf("field = %v, want RelFieldParentID", cond.Field)
	}
}

func TestParseMixedExpression(t *testing.T) {
	expr, err := Parse(`completed=true AND parent_id=1`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(expr.Conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(expr.Conditions))
	}
	// First condition: completed=true
	c0, ok := expr.Conditions[0].(*BoolCondition)
	if !ok {
		t.Fatalf("condition[0]: expected *BoolCondition, got %T", expr.Conditions[0])
	}
	if c0.Field != BoolFieldCompleted || !c0.Value {
		t.Errorf("condition[0]: field=%v value=%v", c0.Field, c0.Value)
	}
	// Second condition: parent_id=1
	c1, ok := expr.Conditions[1].(*RelCondition)
	if !ok {
		t.Fatalf("condition[1]: expected *RelCondition, got %T", expr.Conditions[1])
	}
	if c1.Field != RelFieldParentID || c1.ID != 1 {
		t.Errorf("condition[1]: field=%v id=%d", c1.Field, c1.ID)
	}
}

func TestParseUnknownField(t *testing.T) {
	// "bogus" is not a known field keyword, so it's a text term.
	// "=true" then starts with =, which is not a valid condition start.
	_, err := Parse(`bogus=true`)
	if err == nil {
		t.Fatal("expected error for bareword followed by =, got nil")
	}
}

func TestParseFieldConditions(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantTyp string // "bool", "date", "rel", or "" for error
		wantErr bool
	}{
		// Bool conditions
		{"completed true", "completed=true", "bool", false},
		{"completed false", "completed=false", "bool", false},
		{"completed ne true", "completed!=true", "bool", false},
		{"snoozed true", "snoozed=true", "bool", false},
		{"snoozed false", "snoozed=false", "bool", false},
		{"snoozed ne false", "snoozed!=false", "bool", false},

		// Date conditions on completed
		{"completed lt date", "completed < 2026-01-01", "date", false},
		{"completed le date", "completed <= 2026-06-15", "date", false},
		{"completed gt date", "completed > 2025-12-31", "date", false},
		{"completed ge date", "completed >= 2026-01-01", "date", false},

		// Rel conditions
		{"parent_id eq", "parent_id=1", "rel", false},
		{"parent_id ne", "parent_id!=1", "rel", false},
		{"goal_id eq", "goal_id=5", "rel", false},
		{"transitive parent_id", "^parent_id=1", "rel", false},
		{"transitive goal_id", "^goal_id=1", "rel", false},

		// Operator spacing (optional whitespace around op)
		{"spaced completed", "completed = true", "bool", false},
		{"spaced parent_id", "parent_id = 1", "rel", false},

		// Mixed with AND
		{"bool AND rel", "completed=true AND parent_id=1", "", false},
		{"text AND bool", "groceries AND completed=false", "", false},

		// Error: unknown field followed by = (lexes as text then = is unexpected)
		{"unknown field eq", "bogus=true", "", true},

		// Error: wrong op for snoozed (bool only accepts = !=)
		{"snoozed lt", "snoozed < 2026-01-01", "", true},

		// Error: wrong op for parent_id (rel only accepts = !=)
		{"parent_id lt", "parent_id<5", "", true},

		// Error: hat on non-rel field
		{"hat on completed", "^completed=true", "", true},

		// Error: hat on text term
		{"hat on bareword", "^foo", "", true},

		// Error: non-integer id
		{"parent_id non-int", "parent_id=abc", "", true},

		// Error: completed with != and date value (expects bool after !=)
		{"completed ne date", "completed!=2026-01-01", "", true},

		// Error: incomplete bool (no value after =)
		{"completed eq incomplete", "completed=", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := Parse(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil (conditions: %d)", tc.input, len(expr.Conditions))
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.input, err)
			}
			if tc.wantTyp == "" {
				// Just verifying it parses without error; check condition count
				if len(expr.Conditions) < 1 {
					t.Errorf("expected at least 1 condition, got %d", len(expr.Conditions))
				}
				return
			}
			if len(expr.Conditions) != 1 {
				t.Fatalf("expected 1 condition for %q, got %d", tc.input, len(expr.Conditions))
			}
			switch tc.wantTyp {
			case "bool":
				_, ok := expr.Conditions[0].(*BoolCondition)
				if !ok {
					t.Errorf("expected *BoolCondition, got %T", expr.Conditions[0])
				}
			case "date":
				_, ok := expr.Conditions[0].(*DateCondition)
				if !ok {
					t.Errorf("expected *DateCondition, got %T", expr.Conditions[0])
				}
			case "rel":
				_, ok := expr.Conditions[0].(*RelCondition)
				if !ok {
					t.Errorf("expected *RelCondition, got %T", expr.Conditions[0])
				}
			}
		})
	}
}

func TestParseRelWithHat(t *testing.T) {
	expr, err := Parse(`^goal_id=3`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(expr.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(expr.Conditions))
	}
	cond, ok := expr.Conditions[0].(*RelCondition)
	if !ok {
		t.Fatalf("expected *RelCondition, got %T", expr.Conditions[0])
	}
	if !cond.Transitive {
		t.Error("transitive = false, want true")
	}
	if cond.Field != RelFieldGoalID {
		t.Errorf("field = %v, want RelFieldGoalID", cond.Field)
	}
	if cond.ID != 3 {
		t.Errorf("id = %d, want 3", cond.ID)
	}
}
