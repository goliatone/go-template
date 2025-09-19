package {{ package_name }}

type {{ struct_name }} struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func New{{ struct_name }}() *{{ struct_name }} {
	return &{{ struct_name }}{
		Name:    "{{ name }}",
		Version: "{{ version }}",
	}
}