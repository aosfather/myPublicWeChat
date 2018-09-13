package main

//消息处理
type myprocessor struct {
}

//事件响应
func (this *myprocessor) OnEvent(event interface{}) interface{} {
	return nil
}

//消息响应
func (this *myprocessor) OnMessage(msg interface{}) interface{} {

	return nil
}
