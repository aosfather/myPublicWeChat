package main

import (
	"github.com/aosfather/bingo"
)

func main() {
	app := bingo.TApplication{}
	app.SetHandler(onServiceLoad, onHandlerLoad)
	app.Run("config.json")
}

//load函数，如果加载成功返回true，否则返回FALSE
func onServiceLoad(context *bingo.ApplicationContext) bool {
	//构造processor
	p := myprocessor{}
	p.Init(context)
	context.RegisterService("processor", &p)
	return true
}

//controller load
func onHandlerLoad(mvc *bingo.MvcEngine, context *bingo.ApplicationContext) bool {
	wxcontroll := WXPublicApplication{}
	mvc.AddController(&wxcontroll)
	return true
}
