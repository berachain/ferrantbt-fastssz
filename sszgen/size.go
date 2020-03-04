package main

import (
	"fmt"
	"strconv"
	"strings"
)

// size creates a function that returns the SSZ size of the struct. There are two components:
// 1. Fixed: Size that we can determine at compilation time (i.e. uint, fixed bytes, fixed vector...)
// 2. Dynamic: Size that depends on the input (i.e. lists, dynamic containers...)
func (e *env) size(name string, v *Value) string {
	tmpl := `// Size returns the ssz encoded size in bytes for the {{.name}} object
	func (:: *{{.name}}) Size() (size int) {
		size = {{.fixed}}{{if .dynamic}}

		{{.dynamic}}
		{{end}}
		return
	}`

	str := execTmpl(tmpl, map[string]interface{}{
		"name":    name,
		"fixed":   v.n,
		"dynamic": v.sizeContainer("size", true),
	})
	return appendObjSignature(str, v)
}

func (v *Value) sizeContainer(name string, start bool) string {
	if !start {
		return fmt.Sprintf(name+" += ::.%s.Size()", v.name)
	}
	out := []string{}
	for indx, v := range v.o {
		if !v.isFixed() {
			out = append(out, fmt.Sprintf("// Field (%d) '%s'\n%s", indx, v.name, v.size(name)))
		}
	}
	return strings.Join(out, "\n\n")
}

// 'name' is the name of target variable we assign the size too. We also use this function
// during marshalling to figure out the size of the offset
func (v *Value) size(name string) string {
	if v.isFixed() {
		if v.t == TypeContainer {
			return v.sizeContainer(name, false)
		}
		if v.n == 1 {
			return name + "++"
		}
		return name + " += " + strconv.Itoa(int(v.n))
	}

	switch v.t {
	case TypeContainer:
		return v.sizeContainer(name, false)

	case TypeBitList:
		fallthrough

	case TypeBytes:
		return fmt.Sprintf(name+" += len(::.%s)", v.name)

	case TypeList:
		fallthrough

	case TypeVector:
		if v.e.isFixed() {
			return fmt.Sprintf("%s += len(::.%s) * %d", name, v.name, v.e.n)
		}
		v.e.name = v.name + "[ii]"
		tmpl := `for ii := 0; ii < len(::.{{.name}}); ii++ {
			{{.size}} += 4
			{{.dynamic}}
		}`
		return execTmpl(tmpl, map[string]interface{}{
			"name":    v.name,
			"size":    name,
			"dynamic": v.e.size(name),
		})

	default:
		panic(fmt.Errorf("size not implemented for type %s", v.t.String()))
	}
}