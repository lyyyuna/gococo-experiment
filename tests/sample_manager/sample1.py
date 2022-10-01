from . import sample


s1 = sample.Sample()

s1.files["./go.mod"] = \
'''module lyyyuna.com/gococo/test

go 1.18
'''

s1.files["./main.go"] = \
'''package main

import "fmt"

func main() {
    fmt.Println("hello, world")
}
'''