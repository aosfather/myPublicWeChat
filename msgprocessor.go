package main

import (
	"github.com/aosfather/bingo"
	"github.com/aosfather/bingo/openapi"
	"github.com/aosfather/bingo/utils"
)

//消息处理
type myprocessor struct {
	logger utils.Log
	sdk    *openapi.TulingSDK
}

func (this *myprocessor) Init(context *bingo.ApplicationContext) {
	this.logger = context.GetLog("processor")
	this.sdk = &openapi.TulingSDK{"808811ad0fd34abaa6fe800b44a9556a"}
}

//事件响应
func (this *myprocessor) OnEvent(event XMLEvent) IReplyMsg {
	this.logger.Debug("on event %v", event)
	var reply IReplyMsg = nil
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
	this.logger.Debug("reply msg %v", reply)
	return reply
}

//消息响应
func (this *myprocessor) OnMessage(msgtype string, msg interface{}) IReplyMsg {
	if msg == nil {
		this.logger.Debug("msg is nil")
	}
	switch msgtype {
	case MSGTYPE_TEXT:
		realmsg, _ := msg.(*WXxmlTextMessage)
		text := realmsg.Content
		this.logger.Debug("text msg: %s", text)
		msg := WXxmlReplyTextMessage{}
		msg.MsgType = MSGTYPE_TEXT
		msg.ToUserName = realmsg.FromUserName
		msg.FromUserName = realmsg.ToUserName
		msg.Content = CDATA(this.Reply(msg.ToUserName, text))
		return &msg

	}
	return nil
}

//调用机器人进行回复
func (this *myprocessor) Reply(user, msg string) string {
	if this.sdk != nil {
		return this.sdk.QueryAsString(user, msg)
	}

	return "[自动回复] 暂时不在线！"

}
