package filter

import (
	"fmt"
	"strconv"
	"strings"
)

// Parse parses a filter expression string into an Expression AST.
// Returns an error with a human-readable message for invalid expressions.
func Parse(expr string) (Expression, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return Expression{}, fmt.Errorf("filter expression cannot be empty")
	}

	lex := newLexer(expr)
	p := &parser{lexer: lex}

	conditions, err := p.parseConditions()
	if err != nil {
		return Expression{}, err
	}

	if len(conditions) == 0 {
		return Expression{}, fmt.Errorf("filter expression cannot be empty")
	}

	return Expression{Conditions: conditions}, nil
}

type parser struct {
	lexer *lexer
}

func (p *parser) parseConditions() ([]Condition, error) {
	var conditions []Condition

	cond, err := p.parseCondition()
	if err != nil {
		return nil, err
	}
	conditions = append(conditions, cond)

	for {
		tok, err := p.lexer.nextToken()
		if err != nil {
			return nil, err
		}
		if tok.typ == tokenEOF {
			break
		}
		if tok.typ != tokenAnd {
			return nil, fmt.Errorf("expected AND between conditions, got %q", tok.value)
		}
		cond, err = p.parseCondition()
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, cond)
	}

	return conditions, nil
}

func (p *parser) parseCondition() (Condition, error) {
	tok, err := p.lexer.nextToken()
	if err != nil {
		return nil, err
	}

	switch tok.typ {
	case tokenEOF:
		return nil, fmt.Errorf("expected condition but found end of expression")
	case tokenString:
		return &TextCondition{Term: tok.value}, nil
	case tokenHat:
		return p.parseRelConditionWithHat()
	case tokenIdent:
		return p.parseFieldOrText(tok.value)
	default:
		return nil, fmt.Errorf("unexpected token %q", tok.value)
	}
}

func (p *parser) parseFieldOrText(word string) (Condition, error) {
	// Check if this looks like a field condition: the next token should be an operator
	// or the word should be a known field name followed by an operator.
	lower := strings.ToLower(word)

	switch lower {
	case "completed":
		return p.parseBoolOrDateCondition(BoolFieldCompleted, DateFieldCompleted)
	case "snoozed":
		return p.parseBoolCondition(BoolFieldSnoozed)
	case "parent_id":
		return p.parseRelCondition(RelFieldParentID, false)
	case "goal_id":
		return p.parseRelCondition(RelFieldGoalID, false)
	}

	// Not a known field — treat as a text term
	return &TextCondition{Term: word}, nil
}

func (p *parser) parseBoolOrDateCondition(bf BoolField, df DateField) (Condition, error) {
	// Peek next token to decide: bool or date
	tok, err := p.lexer.nextToken()
	if err != nil {
		return nil, err
	}

	switch tok.typ {
	case tokenEq, tokenNe:
		op := tokenToOp(tok.typ)
		val, err := p.parseBoolValue()
		if err != nil {
			return nil, err
		}
		return &BoolCondition{Field: bf, Op: op, Value: val}, nil
	case tokenLt, tokenLe, tokenGt, tokenGe:
		op := tokenToOp(tok.typ)
		// Date comparison on completed
		dateTok, err := p.lexer.nextToken()
		if err != nil {
			return nil, err
		}
		if dateTok.typ != tokenDate {
			return nil, fmt.Errorf("expected date value after %s, got %q", tok.value, dateTok.value)
		}
		return &DateCondition{Field: df, Op: op, Day: dateTok.value}, nil
	default:
		return nil, fmt.Errorf("expected operator after %q, got %q", "completed", tok.value)
	}
}

func (p *parser) parseBoolCondition(bf BoolField) (Condition, error) {
	tok, err := p.lexer.nextToken()
	if err != nil {
		return nil, err
	}

	if tok.typ != tokenEq && tok.typ != tokenNe {
		return nil, fmt.Errorf("expected = or != after %q, got %q", "snoozed", tok.value)
	}

	op := tokenToOp(tok.typ)
	val, err := p.parseBoolValue()
	if err != nil {
		return nil, err
	}
	return &BoolCondition{Field: bf, Op: op, Value: val}, nil
}

func (p *parser) parseBoolValue() (bool, error) {
	tok, err := p.lexer.nextToken()
	if err != nil {
		return false, err
	}

	switch strings.ToLower(tok.value) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("expected true or false, got %q", tok.value)
	}
}

func (p *parser) parseRelConditionWithHat() (Condition, error) {
	tok, err := p.lexer.nextToken()
	if err != nil {
		return nil, err
	}

	if tok.typ != tokenIdent {
		return nil, fmt.Errorf("^ must be followed by parent_id or goal_id, got %q", tok.value)
	}

	lower := strings.ToLower(tok.value)
	var field RelField
	switch lower {
	case "parent_id":
		field = RelFieldParentID
	case "goal_id":
		field = RelFieldGoalID
	default:
		return nil, fmt.Errorf("^ must be followed by parent_id or goal_id, got %q", tok.value)
	}

	return p.parseRelCondition(field, true)
}

func (p *parser) parseRelCondition(field RelField, transitive bool) (Condition, error) {
	tok, err := p.lexer.nextToken()
	if err != nil {
		return nil, err
	}

	if tok.typ != tokenEq && tok.typ != tokenNe {
		return nil, fmt.Errorf("expected = or != after %q, got %q", "parent_id", tok.value)
	}

	op := tokenToOp(tok.typ)

	idTok, err := p.lexer.nextToken()
	if err != nil {
		return nil, err
	}

	if idTok.typ != tokenInt {
		return nil, fmt.Errorf("expected integer ID, got %q", idTok.value)
	}

	id, err := strconv.ParseInt(idTok.value, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid ID %q: %w", idTok.value, err)
	}

	return &RelCondition{Field: field, Transitive: transitive, ID: id, Op: op}, nil
}

func tokenToOp(typ tokenType) Op {
	switch typ {
	case tokenEq:
		return OpEq
	case tokenNe:
		return OpNe
	case tokenLt:
		return OpLt
	case tokenLe:
		return OpLe
	case tokenGt:
		return OpGt
	case tokenGe:
		return OpGe
	}
	return OpEq
}
