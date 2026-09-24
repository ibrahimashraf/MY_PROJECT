package main

import (
	"fmt"
	"reflect"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/ext"
)

type MyStruct struct {
	Field int
}

func main() {
	env, err := cel.NewEnv(
		ext.NativeTypes(reflect.TypeOf(MyStruct{})),
		cel.Variable("msg", cel.ObjectType("main.MyStruct")),
	)
	if err != nil {
		fmt.Println("Err:", err)
		return
	}
	
	ast, iss := env.Compile(`msg.Field == 1`)
	if iss.Err() != nil {
		fmt.Println("Compile Err:", iss.Err())
		return
	}
	prg, _ := env.Program(ast)
	out, _, _ := prg.Eval(map[string]interface{}{"msg": MyStruct{Field: 1}})
	fmt.Println("Success:", out.Value())
}
