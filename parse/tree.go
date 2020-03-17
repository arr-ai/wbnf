package parse

func (Scanner) IsTreeElement() {}

type TreeElement interface {
	IsTreeElement()
}

type Extra interface {
	IsExtra()
}
