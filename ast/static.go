package ast

// StaticVarEntry represents a single static variable with optional initializer
type StaticVarEntry struct {
	Name string
	Init Node // may be nil
	Pos  Position
}

// StaticVarDeclNode represents a 'static $a = 1, $b;' declaration inside a function
type StaticVarDeclNode struct {
	Vars []StaticVarEntry
	Pos  Position
}

func (s *StaticVarDeclNode) NodeType() string     { return "StaticVarDecl" }
func (s *StaticVarDeclNode) GetPos() Position     { return s.Pos }
func (s *StaticVarDeclNode) SetPos(pos Position)  { s.Pos = pos }
func (s *StaticVarDeclNode) String() string       { return "static vars" }
func (s *StaticVarDeclNode) TokenLiteral() string { return "static" }
