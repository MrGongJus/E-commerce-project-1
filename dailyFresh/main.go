package main

import (
	_ "dailyFresh/routers"
	"github.com/astaxie/beego"
	_ "dailyFresh/models"
)

func main() {
	beego.Run()
}

