package bot

type Values map[interface{}]interface{}

func NewValues() Values {
	return map[interface{}]interface{}{}
}

func (vs Values) Get(k interface{}) interface{} {
	v, ok := vs[k]
	if ok {
		return v
	}
	return nil
}

func (vs Values) Set(k, v interface{}) {
	vs[k] = v
}

func (vs Values) Delete(k interface{}) {
	delete(vs, k)
}
