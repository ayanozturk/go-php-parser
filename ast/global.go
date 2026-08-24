package ast

// GlobalVarEntry represents a single variable named in a 'global' statement.
type GlobalVarEntry struct {
	Name   string
	Pos    Position
	EndPos Position
}

// GlobalVarDeclNode represents a 'global $a, $b;' declaration, typically used
// inside a function to import variables from the global scope.
type GlobalVarDeclNode struct {
	Vars   []GlobalVarEntry
	Pos    Position
	EndPos Position
}

func (g *GlobalVarDeclNode) NodeType() string       { return "GlobalVarDecl" }
func (g *GlobalVarDeclNode) GetPos() Position       { return g.Pos }
func (g *GlobalVarDeclNode) SetPos(pos Position)    { g.Pos = pos }
func (g *GlobalVarDeclNode) GetEndPos() Position    { return g.EndPos }
func (g *GlobalVarDeclNode) SetEndPos(pos Position) { g.EndPos = pos }
func (g *GlobalVarDeclNode) String() string         { return "global vars" }
func (g *GlobalVarDeclNode) TokenLiteral() string   { return "global" }
