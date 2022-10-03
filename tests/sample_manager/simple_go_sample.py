from . import sample


simple_go_mod_sample = sample.Sample()

simple_go_mod_sample.files["./go.mod"] = '''
module lyyyuna.com/gococo/test

go 1.18
'''
simple_go_mod_sample.files["./main.go"] = '''
package main

import "fmt"

func main() {
    fmt.Println("hello, world")
}
'''

simple_go_work_sample = sample.Sample()
simple_go_work_sample.files["./go.work"] = '''
go 1.18

use (
    ./hello
    ./example
)
'''
simple_go_work_sample.files["./hello/go.mod"] = '''
module example.com/hello

go 1.19

require (
    gittt.net/example v1.5.0
)
'''
simple_go_work_sample.files["./hello/main.go"] = '''
package main

import "gittt.net/example"

func main() {
    example.Print()
}
'''
simple_go_work_sample.files["./example/go.mod"] = '''
module gittt.net/example

go 1.19
'''
simple_go_work_sample.files["./example/print.go"] = '''
package example

import "fmt"

func Print() {
    fmt.Println("hello, world")
}
'''