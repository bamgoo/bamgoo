package base

//

type (
	Any = interface{}
	Map = map[string]Any

	Res interface {
		OK() bool
		Fail() bool
		Code() int
		State() string
		Args() []Any
		With(...Any) Res
		Error() string
	}

	Vars = map[string]Var
	Var  struct {
		nil      bool
		Type     string
		Required bool
		Nullable bool
		Name     string
		Text     string
		Default  Any
		Setting  Map
		Options  Map
		Children Vars
		Encode   string
		Decode   string
		Empty    Res
		Error    Res
		Check    func(Any, Var) bool
		Convert  func(Any, Var) Any
	}
)

var (
	Nil = Var{nil: true}
)

func (vvv *Var) Nil() bool {
	if vvv == nil {
		return true
	}
	return vvv.nil
}
