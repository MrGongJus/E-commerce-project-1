package controllers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"dailyFresh/models"
	"strconv"
	"github.com/gomodule/redigo/redis"
	"time"
	"strings"
	"github.com/smartwalle/alipay"
	"github.com/KenmyZhang/aliyun-communicate"
	"encoding/json"
	" github.com/smartwalle/alipay"
)

type OrderController struct {
	beego.Controller
}

func(this*OrderController)HandleShowOrder(){
	//获取数据
	ids := this.GetStrings("select")
	//校验数据
	if len(ids) == 0{
		beego.Error("传输数据错误")
		return
	}
	//处理数据
	//1.获取地址信息
	userName := this.GetSession("userName")
	o := orm.NewOrm()
	//获取当前用户所有地址信息
	var adds []models.Address
	o.QueryTable("Address").RelatedSel("User").Filter("User__UserName",userName.(string)).All(&adds)
	this.Data["adds"] = adds
	//2.支付方式
	var goods []map[string]interface{}
	conn,err :=redis.Dial("tcp","192.168.110.81:6379")
	if err != nil{
		beego.Error("redis链接失败")
		return
	}
	defer conn.Close()

	//3.获取商品信息和商品数量
	totalCount := 0
	totalPrice := 0

	i := 1
	for _,id := range ids{
		skuid,_ :=strconv.Atoi(id)
		temp := make(map[string]interface{})
		//获取商品信息
		var goodsSku models.GoodsSKU
		goodsSku.Id = skuid
		o.Read(&goodsSku)

		temp["goodsSku"] = goodsSku

		//获取商品数量
		resp,err :=conn.Do("hget","cart_"+userName.(string),skuid)
		count,_ :=redis.Int(resp,err)
		temp["count"] = count
		//计算商品小计
		littlePrice := goodsSku.Price * count
		temp["littlePrice"] = littlePrice
		totalCount += 1
		totalPrice += littlePrice

		temp["i"]  = i
		i +=1

		goods = append(goods,temp)
	}

	//定义运费
	transPrice := 10
	truePrice := transPrice + totalPrice
	this.Data["totalCount"] = totalCount
	this.Data["totalPrice"] = totalPrice
	this.Data["transPrice"] = transPrice
	this.Data["truePrice"] = truePrice

	//返回数据
	this.Data["ids"] = ids
	this.Data["goods"] = goods
	this.TplName = "place_order.html"
}

//处理添加订单业务
func(this*OrderController)HandleAddOrder(){
	resp := make(map[string]interface{})
	defer AJAXRESP(&this.Controller,resp)


	//获取数据
	addrId,err1 :=this.GetInt("addId")
	skuids := this.GetString("skuids")
	payId,err2 := this.GetInt("payId")
	totalCount ,err3 := this.GetInt("totalCount")
	totalPrice,err4 := this.GetInt("totalPrice")
	transPrice,err5 := this.GetInt("transPrice")

	//校验数据u
	if err1 != nil ||  err2 != nil || err3 != nil || err4 != nil || err5 != nil{
		resp["errno"] = 1
		resp["errmsg"] = "传输数据错误"
		return
	}

	//beego.Info("addrId=",addrId,"   skuids=",skuids,"    payId=",payId,"   totalCount=",totalCount, "   totalPrice=",totalPrice,"   transPrice = ",transPrice)
	//处理数据

	//1.把获取到的数据插入到订单表
	o := orm.NewOrm()

	var orderInfo models.OrderInfo
	//插入地址信息
	var addr models.Address
	addr.Id = addrId
	o.Read(&addr)
	orderInfo.Address = &addr

	//插入用户信息
	var user models.User
	userName := this.GetSession("userName")
	user.UserName = userName.(string)
	o.Read(&user,"UserName")
	orderInfo.User = &user

	orderInfo.TransitPrice = transPrice
	orderInfo.TotalPrice = totalPrice
	orderInfo.TotalCount = totalCount
	orderInfo.PayMethod = payId
	orderInfo.OrderId = time.Now().Format("20060102150405"+strconv.Itoa(user.Id))

	//插入
	o.Begin()

	_,err := o.Insert(&orderInfo)
	if err != nil{
		resp["errno"] = 3
		resp["errmsg"] = "订单表插入失败"
	}
	//对商品Id做处理   [1   3    6    8]   字符串  string
	ids :=strings.Split(skuids[1:len(skuids)-1]," ")
	conn,err := redis.Dial("tcp","192.168.110.81:6379")
	if err != nil{
		resp["errno"] = 2
		resp["errmsg"] = "redis连接诶错误"
		return
	}
	defer conn.Close()

	var history_Stock int   //原有库存量

	for _,id := range ids{
		skuid,_ :=strconv.Atoi(id)
		var goodsSku models.GoodsSKU
		goodsSku.Id = skuid
		for i:=0;i<3;i++{
			o.Read(&goodsSku)
			history_Stock = goodsSku.Stock


			//获取商品数量
			re,err :=conn.Do("hget","cart_"+userName.(string),skuid)
			count,_ :=redis.Int(re,err)

			var orderGoods models.OrderGoods
			orderGoods.GoodsSKU = &goodsSku
			orderGoods.Price = count * goodsSku.Price
			orderGoods.Count = count
			orderGoods.OrderInfo = &orderInfo
			if goodsSku.Stock < count{
				resp["errno"] = 4
				resp["errmsg"] = goodsSku.Name+"库存不足"
				o.Rollback()
				return
			}
			o.Insert(&orderGoods)

			//time.Sleep(time.Second * 10)

			if history_Stock != goodsSku.Stock{
				if i == 2 {
					resp["errno"] = 6
					resp["errmsg"] = "商品数量被改变，请重新选择商品"
					o.Rollback()
					return
				}else{
					continue
				}
			}else{
				goodsSku.Stock -= count
				goodsSku.Sales += count
				_,err=o.Update(&goodsSku)
				if err!= nil{
					resp["errno"] = 7
					resp["errmsg"] = "更新错误"
					return
				}
				conn.Do("hdel","cart_"+userName.(string),skuid)
				break
			}
		}
	}

	//给容器赋值
	resp["errno"] = 5
	resp["errmsg"] = "OK"
	////把容器传递给前段
	//this.Data["json"] = resp
	////告诉前端以json格式接受
	//this.ServeJSON()
	//2.把购物车中的数据清除
	o.Commit()

	//返回数据
}

//给支付宝发消息
func(this*OrderController)SendAliPay(){
	var aliPublicKey = `MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0tLPfHsf+3qc9eE4U4GA
	Xor24oeH7g5QZFqEfmjf0ZaDQNv020f9X+Qn024d2DqEXhpHchodYUWmVkeEn4lm
	QGJoL7XUA/MrOLaaKz2pzg01E4bBTsmCIpkv3XKh+FlKWM751HGWQVcuhNYqPXaS
	MkXCpr2DRdNC9ki4zdCIlVf8US2vIBYiFt8tkmawF/wy/R6LFyEXa3wLbE3oSwFf
	RmVbDUx/YvY0i/e6faUqCJqGKLhH+xjSBQWutDlVv1K0Tix7SRzwC7HA7Oa2HOQq
	sE+fx3dE7R2Lfq+lmAOrwDt8fO3TGONY1G0egQuFkdUwTnnf9q75zvH8fCVg1qlH
	IwIDAQAB`
	 // 可选，支付宝提供给我们用于签名验证的公钥，通过支付宝管理后台获取
	var privateKey = `MIIEpAIBAAKCAQEA0tLPfHsf+3qc9eE4U4GAXor24oeH7g5QZFqEfmjf0ZaDQNv0
20f9X+Qn024d2DqEXhpHchodYUWmVkeEn4lmQGJoL7XUA/MrOLaaKz2pzg01E4bB
TsmCIpkv3XKh+FlKWM751HGWQVcuhNYqPXaSMkXCpr2DRdNC9ki4zdCIlVf8US2v
IBYiFt8tkmawF/wy/R6LFyEXa3wLbE3oSwFfRmVbDUx/YvY0i/e6faUqCJqGKLhH
+xjSBQWutDlVv1K0Tix7SRzwC7HA7Oa2HOQqsE+fx3dE7R2Lfq+lmAOrwDt8fO3T
GONY1G0egQuFkdUwTnnf9q75zvH8fCVg1qlHIwIDAQABAoIBAQDMRiFu/yo9FFAz
2ncmSoukj8eqNSJbYpk4s5A/n8SGou0okjfNpRJ3sG16au8WDZUmTRY/E9i14LPM
U93Ia2ydI/zJhcgZz6tod14oWcZHdfqgoeh6O7wRZBbB3oncRkBIjrv5wdmSFDRp
183z4gjEF14FDAm/RXVTh6ExI0bEVEtEWRq/eK9c5A7KnFfzKSwqPF4D6Gr75TDo
QQzq7YVHepifyJdnAh59ezD18S33NW+9ZpUxbDxxNSdW/tYSwX0zIupKMScEaQlg
oEJu1ma4tuZD7ZGp9iWtONwNa2QucxyUP7Nuzi8I9lgkRpMXpdFGZQVevhp84cT7
0vlq4eiJAoGBAO0+dm2w9tweMtY4s9A262HD3tusU0pjqU1TZmTfYyEfmVmrJDIS
ENu9zhcAxHT50JCgrpslEx0iFS9ziamBmjqv73dCtugatrCslg2IsLmD9MzBe5pd
vVnb1ftWl1NX1U2bZ6FBLhTsCwcNdRB4nhgbPAh1M9synQ9iOPmSvB3NAoGBAON9
oMBzOXM4zbTlp0UR80otU1+soq1WlKCGYdF6ISifqvWXB19dyEMbEUqAX1jWUapX
gIgOp3TOGTuFHXPVvmCUlSUorhmf/TBb4is2ixpz4MGzdM70JW3dP1c5Gb4ulUxh
q5NO+sOGtD1cUL6RAWykRhAXiHFJj7WDc01dLIivAoGBAJA5dcNvXlMoZJ1IcT+1
81hGu9dtpmDFv2mLtubBysCbNh2F5gYuZ2M+uufPBp9aMwmJNTyJyFngm2JyaZDL
ghgFVp14yDrH6qHy+XGW1GCjMJG9WcfZDsBu3WHjHTGEZt68B77HIh2D9Zw++Rif
SvS6sb8uiOzLkyGEA8DtDEFNAoGAfIvC/poW0eY/eNJiiYYSVIIMK00wowXLyTbJ
Rw4+KSeBSYOuHaASi+q9xLQTf2eWvlO5osOjGmfbmKKARXK4D9hI71ceOhlFXLxx
TodGEO1wF5xQTx2LgGKo0vAID/8g7fhrHvMWhwWwmAd6jVqGFRy63wSDRsKnUxDs
h2aDgzECgYBb4bLGLByyqR4GpI09DMZp5ywI09pADxPx8YO6RcQbgARIh0dJbp3J
AnLk8t3VXBzHwlpg8QEKE4CX75CXPO7ay4Bkw8WkcJJSo2ibSo/y7/nrBHcCWSBJ
CjOq30k0t+eJUEfTk+x17d66zepoguoXbeFc7sTzD3TUIfYgZlVsTA==` // 必须，上一步中使用 RSA签名验签工具 生成的私钥

	appId := "2016092200569649"
	var client = alipay.New(appId, aliPublicKey, privateKey, false)
	//获取订单号和总价
	orderId := this.GetString("orderId")
	totalPrice := this.GetString("totalPrice")
	if orderId == "" || totalPrice == ""{
		beego.Error("数据传输错误")
		return
	}



	var p = alipay.AliPayTradePagePay{}
	p.NotifyURL = "http://192.168.110.81:8080/payOK"
	p.ReturnURL = "http://192.168.110.81:8080/payOK"
	p.Subject = "天天生鲜商品支付"
	p.OutTradeNo = orderId
	p.TotalAmount = totalPrice
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"

	var url, err = client.TradePagePay(p)
	if err != nil {
		beego.Error("给支付宝发请求失败",err)
	}

	var payURL = url.String()
	this.Redirect(payURL,302)
}

func(this*OrderController)HandleAli(){
	orderId := this.GetString("out_trade_no")
	tradeNo := this.GetString("trade_no")
	if tradeNo == "" || orderId == ""{
		beego.Info("交易失败")
	}else {
		beego.Info("支付成功")
		o := orm.NewOrm()
		var orderInfo models.OrderInfo
		orderInfo.OrderId = orderId
		err := o.Read(&orderInfo,"OrderId")
		if err == nil{
			orderInfo.Orderstatus = 1
			orderInfo.TradeNo = tradeNo
			o.Update(&orderInfo)
		}
	}
	//订单状态修改

	this.Redirect("/goods/userCenterOrder",302)
}




//发送短信
func(this*OrderController)SendMsg(){
	var (
		gatewayUrl      = "http://dysmsapi.aliyuncs.com/"
		accessKeyId     = "LTAIh83X7bYYTIXw"
		accessKeySecret = "fYSLqA3BI8jNviNhURKT9T9TmHeOuP"
		phoneNumbers    = "15986619789"
		signName        = "天天生鲜"
		templateCode    = "SMS_149101793"
		templateParam   = "{\"code\":\"ainio\"}"
	)

	smsClient := aliyunsmsclient.New(gatewayUrl)
	result, err := smsClient.Execute(accessKeyId, accessKeySecret, phoneNumbers, signName, templateCode, templateParam)
	beego.Info("Got raw response from server:", string(result.RawResponse))
	if err != nil {
		beego.Error("Failed to send Message: " + err.Error())
	}

	//json.Marshal() //作用是把key-value形式数据打包成json格式
	resultJson, err := json.Marshal(result)
	//beego.Info("resulSjon=",resultJson,"     result=",result)
	if err != nil {
		beego.Error(err)
	}
	if result.IsSuccessful() {
		beego.Info("A SMS is sent successfully:", resultJson)
	} else {
		beego.Info("Failed to send a SMS:", resultJson)
	}
}

