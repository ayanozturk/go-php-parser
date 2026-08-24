package ast

// InlineHTMLNode represents a run of literal (non-PHP) content that appears
// outside of <?php ... ?> tags, such as HTML markup mixed into a template.
type InlineHTMLNode struct {
	Value  string
	Pos    Position
	EndPos Position
}

func (h *InlineHTMLNode) NodeType() string       { return "InlineHTML" }
func (h *InlineHTMLNode) GetPos() Position       { return h.Pos }
func (h *InlineHTMLNode) SetPos(pos Position)    { h.Pos = pos }
func (h *InlineHTMLNode) GetEndPos() Position    { return h.EndPos }
func (h *InlineHTMLNode) SetEndPos(pos Position) { h.EndPos = pos }
func (h *InlineHTMLNode) String() string         { return "inline html" }
func (h *InlineHTMLNode) TokenLiteral() string   { return h.Value }
