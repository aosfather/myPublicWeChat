package main

//消息处理
type myprocessor struct {
}

//事件响应
func (this *myprocessor) OnEvent(event XMLEvent) interface{} {
	var reply interface{} = nil
	switch event.Event {
	case EVENT_SUB:
		msg := WXxmlReplyTextMessage{}
		msg.MsgType = MSGTYPE_TEXT
		msg.ToUserName = event.FromUserName
		msg.FromUserName = event.ToUserName
		msg.Content = `
                  欢迎您的到来！
你可以直接回复主题类别来查询主题，例如：输入"悟道"
当然你也可以随便说些什么，来调戏我们的小机器人！
              `
		reply = &msg

	case EVENT_UNSUB:
		msg := WXxmlReplyTextMessage{}
		msg.MsgType = MSGTYPE_TEXT
		msg.ToUserName = event.FromUserName
		msg.FromUserName = event.ToUserName
		msg.Content = "对您的取消关注，我们表示非常的遗憾！诚挚邀请您以后常来看看"
		reply = &msg

	}

	return reply
}

//消息响应
func (this *myprocessor) OnMessage(msgtype string, msg interface{}) interface{} {

	return nil
}
