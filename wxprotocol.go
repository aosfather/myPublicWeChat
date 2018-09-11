//微信公众号接入口：响应公众号的校验等
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"github.com/aosfather/bingo/mvc"
	"github.com/aosfather/bingo/utils"
	"github.com/aosfather/bingo/wx"
	"github.com/aosfather/bingo/wx/wxcorp"
	"strings"
)

//微信的验证请求
type wxValidateRequest struct {
	Signature string `Field:"signature"`
	Timestamp string `Field:"timestamp"`
	Nonce     string `Field:"nonce"`
	Echostr   string `Field:"echostr"`
}

type WxEncryptInputMessage struct {
	XMLName    xml.Name `xml:"xml"`
	ToUserName string
	AppId      string
	Encrypt    string
}

//微信发送过来的消息
type wxInputMsg struct {
	Signature   string `Field:"msg_signature"`
	Timestamp   string `Field:"timestamp"`
	Nonce       string `Field:"nonce"`
	EncryptType string `Field:"encrypt_type"`
	data        WxEncryptInputMessage
}

func (this *wxInputMsg) GetInput() WxEncryptInputMessage {
	return this.data
}
func (this *wxInputMsg) GetData() interface{} {
	return &this.data
}
func (this *wxInputMsg) GetDataType() string {
	return "xml"
}

//秘钥校验
type ApplicationEncrypt struct {
	token          string
	appId          string
	encodingAESKey string
	theAES         AES
}

func (this *ApplicationEncrypt) Init(token, appid, aeskey string) {
	this.token = token
	this.appId = appid
	if len(aeskey) != 43 { //密码长度不对失败
		panic(aeskey)
	}
	this.encodingAESKey = aeskey
	asekey, _ := base64.URLEncoding.DecodeString(this.encodingAESKey + "=")
	this.theAES = AES{}
	this.theAES.Init(asekey)

}

func (this *ApplicationEncrypt) VerifyURL(msg_signature, timestamp, nonce, echostr string) bool {

	signature := wx.Signature(this.token, timestamp, nonce, echostr)
	fmt.Println("signature:" + signature + "|" + msg_signature)
	if signature != msg_signature {
		return false

	}
	return true

}

func (this *ApplicationEncrypt) DecryptInputMsg(msg_signature, timestamp, nonce string, postdata WxEncryptInputMessage) (int, string) {
	signature := wx.Signature(this.token, timestamp, nonce, postdata.Encrypt)
	if msg_signature != signature {
		return 40001, "signature validate failed!"
	}

	msg := this.decrypt(postdata.Encrypt, postdata.ToUserName)
	if msg == "" {
		return 40005, ""
	}
	return 0, msg
}

func (this *ApplicationEncrypt) decrypt(encryptstr string, targetcorpid string) string {
	//1、解析base64
	str, _ := base64.StdEncoding.DecodeString(encryptstr)
	sourcebytes := this.theAES.decrypt(str)
	//取字节数组长度
	orderBytes := sourcebytes[16:20]
	msgLength := BytesOrderToNumber(orderBytes)

	//取文本
	text := sourcebytes[20 : 20+msgLength]
	//取应用id
	appid := sourcebytes[20+msgLength:]
	fmt.Println("the id:[" + string(appid) + "]")
	corp := strings.TrimSpace(string(appid))
	fmt.Printf("%s:", this.appId)
	if this.appId == corp {
		fmt.Println("yes!")
		return string(text)
	}

	return ""

}

//---------------协议算法------------------//
// 生成4个字节的网络字节序
func NumberToBytesOrder(number int) []byte {
	var orderBytes []byte
	orderBytes = make([]byte, 4, 4)
	orderBytes[3] = byte(number & 0xFF)
	orderBytes[2] = byte(number >> 8 & 0xFF)
	orderBytes[1] = byte(number >> 16 & 0xFF)
	orderBytes[0] = byte(number >> 24 & 0xFF)

	return orderBytes
}

// 还原4个字节的网络字节序
func BytesOrderToNumber(orderBytes []byte) int {
	var number int = 0

	for i := 0; i < 4; i++ {
		number <<= 8
		number |= int(orderBytes[i] & 0xff)
	}
	return number
}

type AES struct {
	key       []byte
	block     cipher.Block
	blockSize int
}

func (this *AES) Init(key []byte) {
	this.key = key
	block, err := aes.NewCipher(this.key)
	if err != nil {
		return
	}
	this.block = block
	this.blockSize = this.block.BlockSize()
}

func (this *AES) encrypt(sourceText []byte) []byte {
	sourceText = wxcorp.PKCS7Padding(sourceText, this.blockSize)

	blockModel := cipher.NewCBCEncrypter(this.block, this.key[:this.blockSize])

	ciphertext := make([]byte, len(sourceText))

	blockModel.CryptBlocks(ciphertext, sourceText)
	return ciphertext
}

func (this *AES) decrypt(encryptedText []byte) []byte {
	blockMode := cipher.NewCBCDecrypter(this.block, this.key[:this.blockSize])
	origData := make([]byte, len(encryptedText))
	blockMode.CryptBlocks(origData, encryptedText)
	origData = wxcorp.PKCS7UnPadding(origData, this.blockSize)
	return origData
}

//-----------微信应用---------------------//
//基本消息格式
type xmlBaseMessage struct {
	FromUserName string
	ToUserName   string
	CreateTime   int64
	MsgType      string
	MsgId        int64
}

//文本消息
type WXxmlTextMessage struct {
	XMLName xml.Name `xml:"xml"`
	xmlBaseMessage
	Content string
}

//图片消息
type WXxmlImageMessage struct {
	XMLName xml.Name `xml:"xml"`
	xmlBaseMessage
	PicUrl  string //图片路径
	MediaId string //媒体id
}

//语音消息
type WXxmlVoiceMessage struct {
	XMLName xml.Name `xml:"xml"`
	xmlBaseMessage
	MediaId string //媒体id
	Format  string //文件格式
}

//视频/小视频消息
type WXxmlVideoMessage struct {
	XMLName xml.Name `xml:"xml"`
	xmlBaseMessage
	MediaId      string //视频媒体文件id，可以调用获取媒体文件接口拉取数据，仅三天内有效
	ThumbMediaId string //视频消息缩略图的媒体id，可以调用获取媒体文件接口拉取数据，仅三天内有效
}

//地址消息
type WXxmlLocationMessage struct {
	XMLName xml.Name `xml:"xml"`
	xmlBaseMessage
	X     float64 `xml:"Location_X"` //地理位置纬度
	Y     float64 `xml:"Location_Y"` //地理位置经度
	Scale int64   //地图缩放大小
	Label string  //地理位置信息
}

//连接消息
type WXxmlLinkMessage struct {
	XMLName xml.Name `xml:"xml"`
	xmlBaseMessage
	Title       string //标题
	Description string //描述
	Url         string //封面缩略图的url
}

//被动回复消息格式
type xmlReplyBaseMessage struct {
	FromUserName string
	ToUserName   string
	CreateTime   int64
	MsgType      string
}

//文本回复消息
type WXxmlReplyTextMessage struct {
	XMLName xml.Name `xml:"xml"`
	xmlReplyBaseMessage
	Content string
}

type WXxmlReplyImageMessage struct {
	XMLName xml.Name `xml:"xml"`
	xmlReplyBaseMessage
	Content xmlReplyImageContent
}

//语音
type WXxmlReplyVoiceMessage struct {
	XMLName xml.Name `xml:"xml"`
	xmlReplyBaseMessage
	Content xmlReplyVoiceContent
}

//视频
type WXxmlReplyVideoMessage struct {
	XMLName xml.Name `xml:"xml"`
	xmlReplyBaseMessage
	Content xmlReplyVideoContent
}

//音乐
type WXxmlReplyMusicMessage struct {
	XMLName xml.Name `xml:"xml"`
	xmlReplyBaseMessage
	Content xmlReplyMusicContent
}

//图文
type WXxmlReplyArticleMessage struct {
	XMLName xml.Name `xml:"xml"`
	xmlReplyBaseMessage
	Content xmlReplyArticlesContent
}

type xmlReplyImageContent struct {
	XMLName xml.Name `xml:"Image"`
	MediaId string   //媒体id
}

type xmlReplyVoiceContent struct {
	XMLName xml.Name `xml:"Voice"`
	MediaId string   //媒体id
}

type xmlReplyVideoContent struct {
	XMLName     xml.Name `xml:"Video"`
	MediaId     string   //媒体id
	Title       string   //标题
	Description string   //描述

}

type xmlReplyMusicContent struct {
	XMLName      xml.Name `xml:"Music"`
	Title        string   //标题
	Description  string   //描述
	MusicURL     string   //音乐链接
	HQMusicUrl   string   //高质量音乐链接，WIFI环境优先使用该链接播放音乐
	ThumbMediaId string   //必填，缩略图的媒体id，通过素材管理中的接口上传多媒体文件，得到的id

}

type xmlReplyArticlesContent struct {
	XMLName xml.Name `xml:"Articles"`
	Items   []xmlReplyArticlesItem
}

type xmlReplyArticlesItem struct {
	XMLName     xml.Name `xml:"item"`
	Title       string   //标题
	Description string   //描述
	PicUrl      string   //图片链接
	Url         string   //跳转
}

//事件
type xmlBaseEvent struct {
	FromUserName string
	ToUserName   string
	CreateTime   int64
	MsgType      string
	Event        string
}

//订阅、取消订阅消息
type SubscribeEvent struct {
	XMLName      xml.Name `xml:"xml"`
	xmlBaseEvent          //	事件类型，subscribe(订阅)、unsubscribe(取消订阅)
}

//二维码关注事件
type SubscribeQREvent struct {
	XMLName      xml.Name `xml:"xml"`
	xmlBaseEvent          //如果用户已经关注事件类型，SCAN，未关注时候类型为subscribe
	EventKey     string   //事件KEY值，qrscene_为前缀，后面为二维码的参数值
	Ticket       string   //二维码的ticket，可用来换取二维码图片
}

//上报地址事件
type LocationEvent struct {
	XMLName      xml.Name `xml:"xml"`
	xmlBaseEvent          //	事件类型，LOCATION
	Latitude     float64  //地理位置纬度
	Longitude    float64  //地理位置经度
	Precision    float64  //地理位置精度
}

//跳转事件
type UrlEvent struct {
	XMLName      xml.Name `xml:"xml"`
	xmlBaseEvent          //	事件类型，跳转 VIEW，菜单点击 CLICK
	EventKey     string   //事件KEY值，与自定义菜单接口中KEY值对应|事件KEY值，设置的跳转URL
}

type WXPublicApplication struct {
	mvc.SimpleController
	logger   utils.Log
	encryted ApplicationEncrypt
}

func (this *WXPublicApplication) Get(c mvc.Context, p interface{}) (interface{}, mvc.BingoError) {
	if q, ok := p.(*wxValidateRequest); ok {
		this.logger.Info("wx validate %v", q)
		ret := this.encryted.VerifyURL(q.Signature, q.Timestamp, q.Nonce, q.Echostr)
		if ret {
			return q.Echostr, nil
		}

	}

	return "hello", nil

}

//正常的访问消息处理
func (this *WXPublicApplication) Post(c mvc.Context, p interface{}) (interface{}, mvc.BingoError) {
	if msg, ok := p.(*wxInputMsg); ok {
		this.logger.Debug("msg:%s", msg)
		ret, result := this.encryted.DecryptInputMsg(msg.Signature, msg.Timestamp, msg.Nonce, msg.GetInput())
		this.logger.Debug("msg result:%i,%s", ret, result)
		if ret == 0 {
			//消息处理
			if strings.Contains(result, "ToUserName") {

			} else { //事件处理

			}

		}

	}

	return "hi", nil

}

//消息处理接口，用于实现应用自身的逻辑
type WXProcessor interface {
	OnEvent() interface{}   //事件响应
	OnMessage() interface{} //消息响应
}
