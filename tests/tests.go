package main

import (
	"fmt"
	"os"
)

func main()  {
	root, _ := os.Getwd()
	fmt.Println(root)
}
