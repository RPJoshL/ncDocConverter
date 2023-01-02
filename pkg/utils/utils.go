package utils

import (
	"reflect"
	"unicode"
)

// Makes the first character of the given string to uppercase
func FirstCharToUppercase(text string) string {
	a := []rune(text)
	a[0] = unicode.ToLower(a[0])
	return string(a)
}

// Copys a map. The structs are also cloned
func CopyMap[T comparable, Val any](m map[T]Val) map[T]Val {
	cp := make(map[T]Val)
	for k, v := range m {
		var u Val
		Copy(&v, &u)
		cp[k] = u
	}

	return cp
}

// Copies a struct
func Copy(source interface{}, destin interface{}) {
	x := reflect.ValueOf(source)
	if reflect.ValueOf(destin).Kind() != reflect.Ptr {
		return
	}
	if x.Kind() == reflect.Ptr {
		reflect.ValueOf(destin).Elem().Set(x.Elem())
	} else {
		reflect.ValueOf(destin).Elem().Set(x)
	}
}
