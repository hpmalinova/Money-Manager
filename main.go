package main

import (
	"fmt"
	"github.com/hpmalinova/Money-Manager/rest"
)

func main() {
	fmt.Println("Staring REST User Service ...")
	a := rest.App{}
	a.Init("root", "1234", "money_manager")
	a.Run(":8080")
}
