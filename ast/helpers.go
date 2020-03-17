package ast

// Which returns the first child of the branch from the list of names supplied
func Which(b Branch, names ...string) (string, Children) {
	if len(names) == 0 {
		panic("wat?")
	}
	for _, name := range names {
		if children, has := b[name]; has {
			return name, children
		}
	}
	return "", nil
}

// First finds the first child of the given node with the named tag. nil if the named node does not exist
func First(n Node, name string) Node {
	if n == nil {
		return nil
	}
	if one := n.One(name); one != nil {
		return one
	}
	if many := n.Many(name); len(many) > 0 {
		return many[0]
	}
	return nil
}

// All returns a list of Nodes for the given name even if only a single Node is found
func All(n Node, name string) []Node {
	if n == nil {
		return nil
	}
	if one := n.One(name); one != nil {
		return []Node{one}
	}
	if many := n.Many(name); len(many) > 0 {
		return many
	}
	return nil
}
