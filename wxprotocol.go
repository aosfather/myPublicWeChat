//微信公众号接入口：响应公众号的校验等
package main

import (
	"bytes"
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
	"time"
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

func (this *ApplicationEncrypt) EncryptMsg(replyMsg, nonce, timestamp string) (int, string) {
	msg := wxcorp.CorpOutputMessage{}
	msg.Encrypt = this.encrypt(replyMsg)
	msg.Nonce = nonce
	msg.MsgSignature = wx.Signature(this.token, timestamp, nonce, msg.Encrypt)
	msg.TimeStamp = timestamp
	outxml, _ := xml.Marshal(msg)
	return 0, string(outxml)
}

//加密
func (this *ApplicationEncrypt) encrypt(text string) string {
	var byteGroup bytes.Buffer
	//格式 16位的random字符+文本长度（4位的网络字节序列）+文本+企业id
	randStr := wxcorp.RandomStr(16)
	byteGroup.Write([]byte(randStr))
	byteGroup.Write(NumberToBytesOrder(len(text)))
	byteGroup.Write([]byte(text))

	byteGroup.Write([]byte(this.appId))

	encryptedText := this.theAES.encrypt(byteGroup.Bytes())
	//转base64
	return base64.StdEncoding.EncodeToString(encryptedText)
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
type XMLEvent struct {
	XMLName      xml.Name `xml:"xml"`
	FromUserName string
	ToUserName   string
	CreateTime   int64
	MsgType      string //事件类型，subscribe(订阅)、unsubscribe(取消订阅),SCAN(扫描二维码)
	//如果用户已经关注事件类型，SCAN，未关注时候类型为subscribe
	//	事件类型，跳转 VIEW，菜单点击 CLICK
	Event string

	EventKey string //事件KEY值，qrscene_为前缀，后面为二维码的参数值；VIEW和CLICK时候，与自定义菜单接口中KEY值对应|事件KEY值，设置的跳转URL

	Ticket string //二维码的ticket，可用来换取二维码图片
	//	事件类型，LOCATION
	Latitude  float64 //地理位置纬度
	Longitude float64 //地理位置经度
	Precision float64 //地理位置精度
}

type WXPublicApplication struct {
	Token  string `Values:""`
	AppId  string `Values:""`
	AESKey string `Values:""`
	mvc.SimpleController
	logger    utils.Log
	encryted  *ApplicationEncrypt //加解密
	processor WXProcessor         `Inject:""` //处理器
}

func (this *WXPublicApplication) Init() {
	this.logger = this.GetBeanFactory().GetLog("wx_public")
	this.encryted = &ApplicationEncrypt{}
	this.encryted.Init(this.Token, this.AppId, this.AESKey)
}

func (this *WXPublicApplication) GetUrl() string {
	return "/wx/public_msg"
}

func (this *WXPublicApplication) GetParameType(method string) interface{} {
	if method == "GET" {
		return &wxValidateRequest{}
	} else {
		return &wxInputMsg{}
	}
}

func (this *WXPublicApplication) Get(c mvc.Context, p interface{}) (interface{}, mvc.BingoError) {
	if q, ok := p.(*wxValidateRequest); ok {
		this.logger.Info("wx validate %v", q)
		ret := this.encryted.VerifyURL(q.Signature, q.Timestamp, q.Nonce, q.Echostr)
		if ret {
			return q.Echostr, nil
		} else {
			this.logger.Info("wx validate failed！")
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
			var replymsg interface{}
			rmsg := xmlBaseMessage{}
			//消息处理
			if strings.Contains(result, "ToUserName") {
				if this.processor != nil {
					//解析result，压入对象

					msgdata := []byte(result)
					xml.Unmarshal(msgdata, &rmsg)
					//根据msgtype类型构造对应的消息结构
					var realmsg interface{}
					switch rmsg.MsgType {
					case "text":
						realmsg = WXxmlTextMessage{}
					case "image":
						realmsg = WXxmlImageMessage{}
					case "voice":
						realmsg = WXxmlVoiceMessage{}
					case "video", "shortvideo":
						realmsg = WXxmlVideoMessage{}
					case "location":
						realmsg = WXxmlLocationMessage{}
					case "link":
						realmsg = WXxmlLocationMessage{}

					}
					//重新解析消息
					xml.Unmarshal(msgdata, &realmsg)
					replymsg = this.processor.OnMessage(rmsg.MsgType, realmsg)

				}

			} else { //事件处理
				if this.processor != nil {
					event := XMLEvent{}
					xml.Unmarshal([]byte(result), &event)
					replymsg = this.processor.OnEvent(event)
				}
			}

			//如果给的返回消息不为空回复微信
			if replymsg != nil {

				//输出xml格式，加密返回
				if replyBaseMsg, ok := replymsg.(*xmlReplyBaseMessage); ok {
					replyBaseMsg.CreateTime = time.Now().Unix()
					enmsg, _ := xml.Marshal(msg)
					_, result := this.encryted.EncryptMsg(string(enmsg), "wxcorpxingyun", fmt.Sprintf("%d", replyBaseMsg.CreateTime))
					return result, nil
				}
				return "", nil
			} else {
				return "success", nil
			}
		}

	}

	return "hi", nil

}

//消息处理接口，用于实现应用自身的逻辑
type WXProcessor interface {
	OnEvent(event XMLEvent) interface{}                    //事件响应
	OnMessage(msgtype string, msg interface{}) interface{} //消息响应
}
