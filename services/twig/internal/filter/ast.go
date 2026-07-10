package filter

// BoolField represents a boolean filter field.
type BoolField int

const (
	BoolFieldCompleted BoolField = iota
	BoolFieldSnoozed
)

// DateField represents a date filter field.
type DateField int

const (
	DateFieldCompleted DateField = iota
)

// RelField represents a relationship (integer ID) filter field.
type RelField int

const (
	RelFieldParentID RelField = iota
	RelFieldGoalID
)

// Op represents a comparison operator.
type Op int

const (
	OpEq Op = iota
	OpNe
	OpLt
	OpLe
	OpGt
	OpGe
)

// Expression is the root AST node. It contains an implicit AND of conditions.
type Expression struct {
	Conditions []Condition
}

// Condition is an interface implemented by all condition types.
type Condition interface {
	conditionNode()
}

// TextCondition matches tasks by case-insensitive substring in name + description.
type TextCondition struct {
	Term string
}

func (TextCondition) conditionNode() {}

// BoolCondition matches tasks by a boolean field.
type BoolCondition struct {
	Field BoolField
	Op    Op
	Value bool
}

func (BoolCondition) conditionNode() {}

// DateCondition matches tasks by a date field (only completed).
type DateCondition struct {
	Field DateField
	Op    Op
	Day   string // YYYY-MM-DD
}

func (DateCondition) conditionNode() {}

// RelCondition matches tasks by a relationship field (parent_id, goal_id).
type RelCondition struct {
	Field      RelField
	Transitive bool // true when prefixed with ^
	ID         int64
	Op         Op
}

func (RelCondition) conditionNode() {}
