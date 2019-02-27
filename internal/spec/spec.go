package spec

type Spec struct {
	Git Git
}

type Git struct {
	SHA       string
	Author    string
	Committer string
	Message   string
}

func Get(path string) (Spec, error) {
	return Spec{}, nil
}
